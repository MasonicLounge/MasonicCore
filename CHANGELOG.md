# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added / Добавлено

- Install wizard: `GET /api/v1/install` returns setup status (`installed`, `forum_name`); `POST /api/v1/install` runs first-time setup — creates the admin user (member + admin roles) and writes `system.installed`/`forum.name` settings. Guarded: a second POST returns `409 already_installed`.
- Мастер установки: `GET /api/v1/install` возвращает статус настройки (`installed`, `forum_name`); `POST /api/v1/install` выполняет первичную настройку — создаёт пользователя-админа (роли member + admin) и записывает настройки `system.installed`/`forum.name`. Защита: повторный POST вернёт `409 already_installed`.

- Realtime WebSocket gateway at `/ws` (JWT via `access_token` query parameter for browser sockets): per-user event fan-out, presence broadcast (`join`/`leave`), hub with connection registry and per-connection send buffer. Built on `gorilla/websocket`.
- Realtime WebSocket-шлюз на `/ws` (JWT через query-параметр `access_token` для браузерных сокетов): раздача событий по пользователям, трансляция presence (`join`/`leave`), hub с реестром подключений и буфером отправки на соединение. Реализовано на `gorilla/websocket`.

- Private messaging: send (`POST /api/v1/pms`, validates recipient, live push to recipient socket), conversation (`GET /api/v1/pms/with?with={userID}`), inbox (`GET /api/v1/pms`, one latest message per conversation), mark read (`PATCH /api/v1/pms/{id}/read`), unread counters (`GET /api/v1/pms/unread-count`).
- Приватные сообщения: отправка (`POST /api/v1/pms`, проверка получателя, live-доставка в сокет получателя), переписка (`GET /api/v1/pms/with?with={userID}`), inbox (`GET /api/v1/pms`, последнее сообщение по каждому диалогу), отметка прочитанным (`PATCH /api/v1/pms/{id}/read`), счётчики непрочитанного (`GET /api/v1/pms/unread-count`).

- Notifications: list (`GET /api/v1/notifications`), mark one read (`PATCH /api/v1/notifications/{id}/read`), mark all read (`PATCH /api/v1/notifications/read-all`); new PMs create a notification row and push a live `notification` event.
- Уведомления: список (`GET /api/v1/notifications`), отметка одного (`PATCH /api/v1/notifications/{id}/read`), отметка всех (`PATCH /api/v1/notifications/read-all`); новые PM создают запись уведомления и отправляют live-событие `notification`.

- Presence: online list (`GET /api/v1/presence`) and per-user lookup (`GET /api/v1/presence/{userID}`); hub tracks online users and broadcasts join/leave.
- Presence: список онлайн-пользователей (`GET /api/v1/presence`) и проверка конкретного (`GET /api/v1/presence/{userID}`); hub отслеживает онлайн-участников и транслирует join/leave.

- Migration `00005`: `private_messages` (sender/recipient, `read_at`, indexes on recipient and sender) and `notifications` (per-user, type, JSONB payload, read marker) tables.
- Миграция `00005`: таблицы `private_messages` (отправитель/получатель, `read_at`, индексы по получателю и отправителю) и `notifications` (по пользователям, тип, JSONB-нагрузка, отметка прочитанности).

- Media storage via S3 (MinIO): avatar upload (`POST /api/v1/media/avatar`, 2 MiB cap, `image/*` only), post attachments (`POST /api/v1/media/attachments`, 20 MiB cap, optional `post_id`), attachment deletion (`DELETE /api/v1/media/attachments/{id}`, owner or moderator), public media serving at `/media/{bucket}/...` with long-lived cache headers. Built on `minio-go`, bucket auto-created on startup.
- Хранилище медиа на S3 (MinIO): загрузка аватара (`POST /api/v1/media/avatar`, до 2 МиБ, только `image/*`), вложения к постам (`POST /api/v1/media/attachments`, до 20 МиБ, опциональный `post_id`), удаление вложения (`DELETE /api/v1/media/attachments/{id}`, владелец или модератор), публичная раздача медиа по `/media/{bucket}/...` с долгим кэшированием. Реализовано на `minio-go`, бакет создаётся автоматически при старте.

- Migration `00004`: `attachments` table (S3 metadata + `post_id`/`owner_id` relations, unique `storage_key`, `public_url`).
- Миграция `00004`: таблица `attachments` (метаданные S3 + связи `post_id`/`owner_id`, уникальный `storage_key`, `public_url`).

- Config keys for object storage (`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `S3_REGION`, `S3_USE_SSL`, `S3_FORCE_PATH_STYLE`, `MEDIA_BASE_URL`); `.env.example` extended accordingly.
- Ключи конфигурации объектного хранилища (`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`, `S3_REGION`, `S3_USE_SSL`, `S3_FORCE_PATH_STYLE`, `MEDIA_BASE_URL`); `.env.example` дополнен.

- Core CRUD: groups (REST under `/api/v1/groups`, admin-managed, hierarchical `parent_id`, unique `slug`), threads (`/api/v1/groups/{groupID}/threads`, `/api/v1/threads`), posts (`/api/v1/posts`). Paginated lists (`limit`/`offset`), view counters, pin/lock moderation, locked threads reject replies, thread post/last-activity stats maintained on post create/delete, group deletion blocked while non-empty.
- Core CRUD: группы (REST по `/api/v1/groups`, управляются админом, иерархический `parent_id`, уникальный `slug`), темы (`/api/v1/groups/{groupID}/threads`, `/api/v1/threads`), посты (`/api/v1/posts`). Пагинация списков (`limit`/`offset`), счётчик просмотров, модерация pin/lock, запрет ответов в заблокированных темах, поддержка статистики постов/последней активности темы при создании и удалении постов, блокировка удаления непустой группы.

- Migration `00003`: `groups`, `threads`, `posts` tables with indexes on group/author and thread activity ordering.
- Миграция `00003`: таблицы `groups`, `threads`, `posts` с индексами по группе/автору и сортировкой по активности.

- Authorization: role guard middleware `RequireRole` (used for admin-only routes); service layer enforces owner/moderation rules for thread and post mutation.
- Авторизация: middleware-гард ролей `RequireRole` (используется на админских маршрутах); слой сервисов проверяет права владельца/модератора при изменении тем и постов.

- Authentication: registration (`POST /api/v1/auth/register`), login (`/auth/login`), access-token refresh (`/auth/refresh`), logout (`/auth/logout`), profile (`/auth/me`). Roles: every new account gets the `member` role.
- Аутентификация: регистрация (`POST /api/v1/auth/register`), вход (`/auth/login`), обновление access-токена (`/auth/refresh`), выход (`/auth/logout`), профиль (`/auth/me`). Роли: каждая новая учётка получает роль `member`.

- Security: argon2id password hashing (OWASP params), JWT HS256 access tokens (15 min) with issued-at/expiry validation, opaque refresh tokens (30 days) stored as SHA-256 hashes and rotated on every refresh, httpOnly refresh cookie.
- Безопасность: хеширование паролей argon2id (OWASP-параметры), JWT HS256 access-токены (15 мин) с проверкой времени выпуска/истечения, opaque refresh-токены (30 дней), хранящиеся как SHA-256 хеши и ротируемые при каждом обновлении, httpOnly refresh-кука.

- Migration `00002`: `user_sessions` table (token hash, device/UA, IP, expiry, revocation) for refresh-session management.
- Миграция `00002`: таблица `user_sessions` (хеш токена, устройство/UA, IP, срок действия, отзыв) для управления refresh-сессиями.

- Auth middleware `RequireAuth` (Bearer token → user context), stable error codes for clients (`invalid_credentials`, `invalid_refresh_token`, `account_disabled`, etc.), `config.Validate()` requiring a JWT secret of at least 32 chars, `.env.example` updated with auth keys.
- Middleware `RequireAuth` (Bearer-токен → контекст пользователя), стабильные коды ошибок для клиентов (`invalid_credentials`, `invalid_refresh_token`, `account_disabled` и т.д.), `config.Validate()` требующий JWT-секрет не короче 32 символов, `.env.example` дополнен ключами аутентификации.

- Unit tests for the password hasher and JWT manager.
- Юнит-тесты для хешера паролей и JWT-менеджера.

- Backend skeleton: Go module (`github.com/masoniclounge/masoniccore`), chi router, env config, pgx pool, goose migrations on startup, healthcheck (`/healthz`, `/readyz`, `/api/v1/health`) and version endpoint (`/api/v1/version` -> `{backend, db}`), graceful shutdown.
- Скелет бэкенда: Go-модуль (`github.com/masoniclounge/masoniccore`), роутер chi, конфиг из env, пул pgx, миграции goose при старте, healthcheck (`/healthz`, `/readyz`, `/api/v1/health`) и эндпоинт версии (`/api/v1/version` -> `{backend, db}`), graceful shutdown.

- Initial schema migration `00001`: `settings` (JSONB), `users`, `roles`, `user_roles` with seeded roles (admin/moderator/member); `pgcrypto` extension.
- Начальная миграция схемы `00001`: `settings` (JSONB), `users`, `roles`, `user_roles` с предзаполненными ролями (admin/moderator/member); расширение `pgcrypto`.

- `.gitignore`, `.env.example` for local development.
- `.gitignore`, `.env.example` для локальной разработки.

- `.gitattributes` — LF normalization for all text files; eliminates CRLF warnings on Windows.
- `.gitattributes` — нормализация LF для всех текстовых файлов; устраняет CRLF-предупреждения на Windows.

### Fixed / Исправлено

- Session IP column (`inet`) is now scanned as `text`: pgx cannot decode binary `inet` into a `*string`, which broke login/refresh with a 500.
- Колонка IP сессии (`inet`) теперь сканируется как `text`: pgx не умеет декодировать бинарный `inet` в `*string`, из-за чего вход/обновление токена падали с 500.