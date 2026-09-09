-- +goose Up
-- +goose StatementBegin
CREATE TABLE groups (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL UNIQUE,
    slug        text NOT NULL UNIQUE,
    description text NOT NULL DEFAULT '',
    parent_id   uuid REFERENCES groups(id) ON DELETE SET NULL,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE threads (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id     uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    author_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        text NOT NULL,
    pinned       boolean NOT NULL DEFAULT false,
    locked       boolean NOT NULL DEFAULT false,
    views        integer NOT NULL DEFAULT 0,
    post_count   integer NOT NULL DEFAULT 0,
    last_post_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

CREATE INDEX idx_threads_group ON threads (group_id, pinned DESC, last_post_at DESC NULLS LAST, created_at DESC);
CREATE INDEX idx_threads_author ON threads (author_id);

-- +goose StatementBegin
CREATE TABLE posts (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id    uuid NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    author_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body         text NOT NULL,
    edited_by_id uuid REFERENCES users(id) ON DELETE SET NULL,
    edited_at    timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

CREATE INDEX idx_posts_thread ON posts (thread_id, created_at);
CREATE INDEX idx_posts_author ON posts (author_id);

-- +goose Down
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS threads;
DROP TABLE IF EXISTS groups;