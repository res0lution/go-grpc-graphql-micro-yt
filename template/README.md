# identity-manager

Сервис аутентификации и управления сессиями на Go.

`identity-manager` отвечает за OAuth2/OIDC login flow, работу с пользовательским профилем, lifecycle сессий и internal-резолв идентичности для backend-to-backend интеграций.

## Что умеет сервис

- Инициация login и обработка callback от IdP.
- Создание, валидация, refresh и revoke сессий.
- Выдача текущего пользователя и текущей сессии.
- Internal endpoint для резолва identity-контекста по `session_id`.
- Проверка доступности БД (`/ready`, `/health`) и liveness (`/live`).

## Стек

- Go `1.24.5`
- Gin
- PostgreSQL (`pgx/v5`)
- JWT/JWKS (`lestrrat-go/jwx/v2`)
- Logrus

## Структура проекта

- `cmd/identity-manager` — точка входа приложения.
- `internal/api` — роутинг и wiring HTTP-слоя.
- `internal/handler` — обработчики HTTP-запросов.
- `internal/service` — бизнес-логика.
- `internal/repository` — доступ к данным.
- `internal/model` — модели и API-контракты.
- `internal/middleware` — auth/access/request-id/logging middleware.
- `internal/db` — подключение к Postgres и SQL-миграции.
- `docs` — архитектурные и миграционные заметки.

## Требования

- Go `1.24.5+`
- PostgreSQL `15`
- `psql` (для запуска SQL-миграций через `make migrate-up`)

## Конфигурация

Сервис читает переменные окружения (и автоматически подхватывает `.env`, если файл есть).

Минимально важные переменные:

- App: `APP_ENV`, `LOG_LEVEL`, `APP_PORT`, `GIN_MODE`, `PORTAL_UI_HOST`
- DB: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- IDP: `IDP_HOST`, `IDP_CLIENT_ID`, `IDP_CLIENT_SECRET`, `IDP_REDIRECT_URI`, `IDP_JWKS_URL`
- Core: `CORE_BACKEND_URL`, `CORE_INTERNAL_AUTH_TOKEN`
- Cookie: `SESSION_COOKIE_NAME`, `SESSION_COOKIE_SECURE`, `SESSION_COOKIE_HTTP_ONLY`, `SESSION_COOKIE_SAMESITE`

Готовый шаблон конфигурации: `.env.example`.

## Быстрый старт (локально)

1. Создайте БД `identity_manager` и укажите доступ в `.env`.
2. Примените миграции:
   - `make migrate-up DATABASE_URL="postgres://user:pass@localhost:5432/identity_manager?sslmode=disable"`
3. Запустите сервис:
   - `make run`
4. Проверьте health:
   - `GET http://localhost:8088/live`
   - `GET http://localhost:8088/ready`

## Запуск и команды

- `make run` — запуск сервиса.
- `make build` — сборка.
- `make test-unit` — тесты по `cmd` и `internal`.
- `make test-integration DATABASE_URL=...` — интеграционные тесты репозитория.
- `make migrate-up DATABASE_URL=...` — применение SQL-миграций.

## API (основные маршруты)

### Public

- `GET /live`
- `GET /ready`
- `GET /health`
- `GET /oidc/auth`
- `GET /oidc/callback`
- `GET /oidc/logout`

### v1 auth

- `GET /v1/auth/login`
- `GET /v1/auth/callback`
- `POST /v1/auth/logout`

### v1 (auth required)

- `GET /v1/users/me`
- `GET /v1/sessions/me`
- `POST /v1/sessions/refresh`
- `DELETE /v1/sessions/me`

### v1 admin (auth + admin group)

- `GET /v1/admin/jwks/status`

### v1 internal (service-to-service token)

- `POST /v1/internal/identity/resolve`

Headers:

- `Authorization: Bearer <CORE_INTERNAL_AUTH_TOKEN>`
- `X-Session-ID: <session_id>` (или body `{"session_id":"..."}`)

Успешный ответ (`200`):

```json
{
  "success": true,
  "identity": {
    "session_id": "s1",
    "user_id": "u1",
    "identity_id": "idp-123",
    "login": "alice",
    "groups": ["APP_SECURITY_PORTAL_ADMIN_MS"]
  },
  "user_info": {
    "sub": "sub-123",
    "identity_id": "idp-123",
    "login": "alice",
    "email": "alice@example.com",
    "given_name": "Alice",
    "family_name": "Liddell",
    "name": "Alice Liddell",
    "group": ["APP_SECURITY_PORTAL_ADMIN_MS"],
    "winaccountname": "ALICE",
    "employee_number": "12345"
  }
}
```

## Безопасность и эксплуатация

- В production:
  - `GIN_MODE=release`
  - `DB_SSLMODE` не должен быть `disable`
  - обязательны `IDP_*` и `CORE_INTERNAL_AUTH_TOKEN`
- Session cookie настраивается через `SESSION_COOKIE_*`.
- Для internal endpoint используйте отдельный сильный shared token.

## Примечания по миграциям

- Миграции запускаются как внешний шаг деплоя (`make migrate-up`).
- Сервис не выполняет auto-migrate при старте.
