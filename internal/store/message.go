package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const pmColumns = `pm.id, pm.sender_id, pm.recipient_id, pm.body, pm.read_at, pm.created_at`

// MessageStore provides SQL access to private messages.
type MessageStore struct {
	db Pool
}

// NewMessageStore creates a MessageStore backed by the given Pool.
func NewMessageStore(db Pool) *MessageStore {
	return &MessageStore{db: db}
}

// Send inserts a private message and returns it.
func (s *MessageStore) Send(ctx context.Context, m models.NewPrivateMessage) (*models.PrivateMessage, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO private_messages (sender_id, recipient_id, body)
		VALUES ($1, $2, $3) RETURNING `+pmColumns, m.SenderID, m.RecipientID, m.Body)
	return scanPM(row)
}

// Conversation returns the private conversation between two users, oldest first.
func (s *MessageStore) Conversation(ctx context.Context, a, b uuid.UUID, limit, offset int) ([]models.PMConversation, int, error) {
	total := 0
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM private_messages
		WHERE (sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1)`,
		a, b).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+pmColumns+`, u.username, COALESCE(u.display_name, u.username)
		FROM private_messages pm
		JOIN users u ON u.id = pm.sender_id
		WHERE (pm.sender_id = $1 AND pm.recipient_id = $2) OR (pm.sender_id = $2 AND pm.recipient_id = $1)
		ORDER BY pm.created_at ASC
		LIMIT $3 OFFSET $4`, a, b, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []models.PMConversation{}
	for rows.Next() {
		var c models.PMConversation
		if err := rows.Scan(&c.ID, &c.SenderID, &c.RecipientID, &c.Body, &c.ReadAt, &c.CreatedAt,
			&c.SenderUsername, &c.SenderDisplay); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Inbox returns the latest message for each of the user's conversations, newest first.
func (s *MessageStore) Inbox(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.PMConversation, int, error) {
	total := 0
	if err := s.db.QueryRow(ctx, `
		SELECT count(DISTINCT
			CASE WHEN sender_id = $1 THEN recipient_id ELSE sender_id END)
		FROM private_messages
		WHERE sender_id = $1 OR recipient_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ON (partner)
			pm.id, pm.sender_id, pm.recipient_id, pm.body, pm.read_at, pm.created_at,
			u.username, COALESCE(u.display_name, u.username)
		FROM (
			SELECT *,
				CASE WHEN sender_id = $1 THEN recipient_id ELSE sender_id END AS partner
			FROM private_messages
			WHERE sender_id = $1 OR recipient_id = $1
			ORDER BY created_at DESC
		) pm
		JOIN users u ON u.id = pm.partner
		ORDER BY pm.partner, pm.created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []models.PMConversation{}
	for rows.Next() {
		var c models.PMConversation
		if err := rows.Scan(&c.ID, &c.SenderID, &c.RecipientID, &c.Body, &c.ReadAt, &c.CreatedAt,
			&c.SenderUsername, &c.SenderDisplay); err != nil {
			return nil, 0, err
		}
		c.RecipientID = userID
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// MarkRead marks a message as read if the caller is its recipient.
// Returns ErrNotFound when no row matched.
func (s *MessageStore) MarkRead(ctx context.Context, id, recipientID uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE private_messages SET read_at = now()
		WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL`, id, recipientID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountUnread counts unread messages addressed to the user.
func (s *MessageStore) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM private_messages
		WHERE recipient_id = $1 AND read_at IS NULL`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func scanPM(row pgx.Row) (*models.PrivateMessage, error) {
	var m models.PrivateMessage
	err := row.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.Body, &m.ReadAt, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// NotificationStore provides SQL access to the notifications table.
type NotificationStore struct {
	db Pool
}

// NewNotificationStore creates a NotificationStore backed by the given Pool.
func NewNotificationStore(db Pool) *NotificationStore {
	return &NotificationStore{db: db}
}

// Create inserts a notification and returns it.
func (s *NotificationStore) Create(ctx context.Context, n models.NewNotification) (*models.Notification, error) {
	if n.Payload == nil {
		n.Payload = json.RawMessage("{}")
	}
	row := s.db.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, payload)
		VALUES ($1, $2, $3) RETURNING id, user_id, type, payload, read_at, created_at`,
		n.UserID, n.Type, n.Payload)
	var out models.Notification
	if err := row.Scan(&out.ID, &out.UserID, &out.Type, &out.Payload, &out.ReadAt, &out.CreatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns the user's notifications, newest first.
func (s *NotificationStore) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Notification, int, error) {
	total := 0
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM notifications WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, type, payload, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []models.Notification{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Payload, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// MarkRead marks one notification as read for the user.
func (s *NotificationStore) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE id = $1 AND user_id = $2 AND read_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead marks all notifications of a user as read.
func (s *NotificationStore) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL`, userID)
	return err
}

// CountUnread counts unread notifications for the user.
func (s *NotificationStore) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
