# MasonicCore

[![CI](https://img.shields.io/github/actions/workflow/status/masoniclounge/MasonicCore/ci.yml?branch=dev&label=CI)](https://github.com/masoniclounge/MasonicCore/actions)
[![Commit activity](https://img.shields.io/github/commit-activity/m/masoniclounge/MasonicCore)](https://github.com/masoniclounge/MasonicCore/commits)
[![Contributors](https://img.shields.io/github/contributors/masoniclounge/MasonicCore)](https://github.com/masoniclounge/MasonicCore/graphs/contributors)
[![Last commit](https://img.shields.io/github/last-commit/masoniclounge/MasonicCore)](https://github.com/masoniclounge/MasonicCore/commits)

> Badges render once the real GitHub organization exists. / Значки появляются после создания организации на GitHub.

## About / О проекте

**MasonicCore** — the API engine of **Masonic Lounge**, an open-source, self-hostable
forum platform. It exposes a versioned REST API (`/api/v1`), owns the PostgreSQL
schema and data access, stores uploaded media in S3 (MinIO), and pushes realtime
events over WebSocket. **MasonicSkin** (frontend) talks to this API.

**MasonicCore** — API-движок форум-платформы **Masonic Lounge**: самодостаточный
open-source движок форумов. Сервис отдаёт версионированный REST API (`/api/v1`),
владеет схемой PostgreSQL и доступом к данным, хранит загруженные файлы в S3 (MinIO)
и рассылает realtime-события через WebSocket. Фронтенд **MasonicSkin** работает с этим API.

## Implemented features / Реализованные функции

- **Authentication** — registration, login, refresh rotation with opaque refresh
  tokens in httpOnly cookies, logout; passwords hashed with **argon2id**, JWT access
  tokens (HS256). / Регистрация, вход, ротация refresh-токенов в httpOnly-cookie,
  выход; пароли — **argon2id**, access-токены JWT (HS256).
- **Roles** — seeded `admin` / `moderator` / `member` roles, per-user assignment. /
  Роли `admin` / `moderator` / `member` с назначением пользователям.
- **Core CRUD** — nested `groups → threads → posts` (categories, topics, replies)
  with pagination, pins, locks, view/post counters and moderation rules. /
  Вложенные `groups → threads → posts` (разделы, темы, ответы) с пагинацией,
  закреплением, блокировкой, счётчиками просмотров/ответов и модерацией.
- **Media (S3/MinIO)** — avatar and attachment upload with size/type validation,
  object streaming back via `/media/{bucket}/*`. / Загрузка аватаров и вложений
  с валидацией размера/типа и отдача объектов через `/media/{bucket}/*`.
- **Realtime (WebSocket)** — `/ws` gateway with presence, live private messages
  and notifications. / Шлюз `/ws`: presence, мгновенные личные сообщения и уведомления.
- **Private messaging & notifications** — REST inbox/conversations, unread counters,
  read/read-all markers + realtime push. / REST-входящие и переписки, счётчики
  непрочитанного, отметки прочитанного + realtime-доставка.
- **Install wizard** — first-run `/install` creates the forum + admin user, guarded
  by the `system.installed` flag. / Первичная установка `/install`: форум и
  администратор, защита флагом `system.installed`.
- **Admin API** — user management (roles/status), forum settings, media listing,
  all behind `RequireAuth + RequireRole(admin)`. / Управление пользователями
  (роли/статус), настройки форума, список медиа — только для администраторов.
- **Integrity** — goose migrations with embedded SQL, health/readiness probes,
  graceful shutdown, structured JSON logs. / Миграции goose со встроенным SQL,
  health/readiness-пробы, graceful shutdown, структурированные логи.

## Tech stack / Стек

| Layer / Слой             | Technology / Технология                                  |
| ------------------------ | -------------------------------------------------------- |
| Language / Язык          | Go 1.27                                                  |
| HTTP router / Маршрутизация | `chi` (v5)                                             |
| Database / БД            | PostgreSQL 16, `pgx` v5 (explicit SQL, no ORM), `goose`  |
| Auth / Аутентификация    | JWT (HS256) access + argon2id, opaque refresh sessions   |
| Media storage / Хранилище | MinIO / S3 via `minio-go`                               |
| Realtime / Realtime      | WebSocket via `gorilla/websocket`                        |
| Auth/IDs / Идентификаторы | `google/uuid`                                           |
| Shipping / Доставка      | Multi-stage Docker image, GitHub Actions CI              |

## Repository layout / Структура репозитория

```
backend/
├── cmd/server/        # entrypoint, graceful shutdown
├── internal/
│   ├── auth/          # argon2id, JWT, refresh tokens
│   ├── config/        # env-based configuration
│   ├── db/            # pgx pool + goose migrations
│   ├── handlers/      # REST handlers, middleware
│   ├── http/          # router assembly
│   ├── integration/   # integration test suite (TEST_DATABASE_URL)
│   ├── models/        # domain structs (DTO/JSON)
│   ├── services/      # business logic
│   ├── storage/       # S3 client (MinIO)
│   ├── store/         # explicit SQL data access
│   └── ws/            # websocket hub + client
├── migrations/        # goose SQL migrations
└── .github/workflows/ # CI: test (with postgres service) + ghcr publish on tags
```

## Development / Разработка

```bash
# start local infra (Postgres + MinIO) — see the workspace-root Makefile
make up

# run the API
export DATABASE_URL=postgres://masonic:test@localhost:55432/masonic?sslmode=disable
export JWT_SECRET=0123456789abcdef0123456789abcdef
export S3_ENDPOINT=localhost:9000 S3_ACCESS_KEY=minioadmin S3_SECRET_KEY=minioadmin
go run ./cmd/server
```

See `release/docker-compose.yml` for the full orchestrated stack.

## Tests / Тесты

- Unit tests: `internal/auth` (argon2id roundtrips, JWT lifecycle).
- Integration tests: `internal/integration` — install/auth/Core CRUD/admin flows
  against a real PostgreSQL; skipped unless `TEST_DATABASE_URL` is set (CI sets it).
- Юнит-тесты: `internal/auth` (argon2id, жизненный цикл JWT).
- Интеграционные: `internal/integration` — install/auth/CRUD/admin против реального
  PostgreSQL; пропускаются без `TEST_DATABASE_URL` (выставляется в CI).

## License / Лицензия

GNU GPL v3.0 — see the repository `LICENSE` file. / GNU GPL v3.0 — см. файл `LICENSE` в репозитории.

## Contributors / Контрибьюторы

[![Contributors](https://img.shields.io/github/contributors/masoniclounge/MasonicCore)](https://github.com/masoniclounge/MasonicCore/graphs/contributors)

See the contributor graph above (renders after the repository exists).
Список контрибьюторов — по ссылке выше (отображается после создания репозитория).