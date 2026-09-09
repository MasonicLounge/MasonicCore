-- +goose Up
-- +goose StatementBegin
CREATE TABLE attachments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id      uuid REFERENCES posts(id) ON DELETE SET NULL,
    filename     text NOT NULL,
    content_type text NOT NULL DEFAULT 'application/octet-stream',
    size_bytes   bigint NOT NULL DEFAULT 0,
    storage_key  text NOT NULL UNIQUE,
    public_url   text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

CREATE INDEX idx_attachments_post ON attachments (post_id);
CREATE INDEX idx_attachments_owner ON attachments (owner_id);

-- +goose Down
DROP TABLE IF EXISTS attachments;