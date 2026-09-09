-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE settings (
    key        text        PRIMARY KEY,
    value      jsonb       NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      text        NOT NULL,
    email         text        NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    display_name  text        NOT NULL DEFAULT '',
    avatar_url    text        NOT NULL DEFAULT '',
    status        text        NOT NULL DEFAULT 'active',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz
);

CREATE UNIQUE INDEX users_username_lower_idx ON users (lower(username));

CREATE TABLE roles (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    key         text        NOT NULL UNIQUE,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT ''
);

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

INSERT INTO roles (key, name, description) VALUES
    ('admin',      'Administrator', 'Full access to administration and moderation'),
    ('moderator',  'Moderator',     'Content moderation, no system settings'),
    ('member',     'Member',        'Regular registered user');

-- +goose Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS settings;