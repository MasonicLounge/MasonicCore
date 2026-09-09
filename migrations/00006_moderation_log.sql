-- +goose Up

CREATE TABLE moderation_log (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    moderator_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action       text NOT NULL,
    target_type  text NOT NULL,
    target_id    uuid NOT NULL,
    reason       text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_moderation_log_moderator ON moderation_log (moderator_id, created_at DESC);
CREATE INDEX idx_moderation_log_target ON moderation_log (target_type, target_id);

-- +goose Down
DROP TABLE IF EXISTS moderation_log;