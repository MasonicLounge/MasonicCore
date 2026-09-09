# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added / Добавлено

- Backend skeleton: Go module (`github.com/masoniclounge/masoniccore`), chi router, env config, pgx pool, goose migrations on startup, healthcheck (`/healthz`, `/readyz`, `/api/v1/health`) and version endpoint (`/api/v1/version` -> `{backend, db}`), graceful shutdown.
- Скелет бэкенда: Go-модуль (`github.com/masoniclounge/masoniccore`), роутер chi, конфиг из env, пул pgx, миграции goose при старте, healthcheck (`/healthz`, `/readyz`, `/api/v1/health`) и эндпоинт версии (`/api/v1/version` -> `{backend, db}`), graceful shutdown.

- Initial schema migration `00001`: `settings` (JSONB), `users`, `roles`, `user_roles` with seeded roles (admin/moderator/member); `pgcrypto` extension.
- Начальная миграция схемы `00001`: `settings` (JSONB), `users`, `roles`, `user_roles` с предзаполненными ролями (admin/moderator/member); расширение `pgcrypto`.

- `.gitignore`, `.env.example` for local development.
- `.gitignore`, `.env.example` для локальной разработки.

- `.gitattributes` — LF normalization for all text files; eliminates CRLF warnings on Windows.
- `.gitattributes` — нормализация LF для всех текстовых файлов; устраняет CRLF-предупреждения на Windows.