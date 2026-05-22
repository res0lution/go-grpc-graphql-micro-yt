# Migration Map: monolith -> identity-manager

## File mapping

- `old/auth_handler.go` -> `internal/handler/auth_handler.go`, `internal/service/auth_service.go`
- `old/user_handler.go` -> `internal/handler/user_handler.go`
- `old/user_service.go` -> `internal/service/user_service.go`
- `old/session_repository.go` -> `internal/repository/session_repository.go`
- `old/user_repository.go` -> `internal/repository/user_repository.go`
- `old/auth_middleware.go` -> `internal/middleware/auth.go`
- `old/config.go` -> `internal/config/config.go`
- `old/database.go` -> `internal/db/postgres.go`
- `old/logger.go` -> `internal/logger/logger.go`
- `old/models.go` -> `internal/model/*`

## Migration order

1. Auth + Session: `/v1/auth/*`, `/v1/sessions/*`.
2. User profile: `/v1/users/me`.
3. Internal contract for core backend: `/v1/internal/identity/resolve`.
4. Replace remaining monolith entrypoints and remove duplicated code.

## Stage note

- Stabilization stage completed:
  - `/v1/sessions/refresh` now follows real token refresh flow.
  - JWKS status moved under admin-protected routes.
  - Migration strategy is external-only (`make migrate-up DATABASE_URL=...`).

## Test coverage status

- Unit tests (`make test-unit`) cover:
  - `internal/service`: auth/session/state-store critical paths (callback, refresh, logout token, fallback user-info).
  - `internal/middleware`: auth required/optional, RBAC (`RequireAnyGroup`, `RequireAllGroups`), internal bearer token middleware.
  - `internal/handler`: auth/session/user/internal identity + health checks.
  - `internal/api`: router smoke contracts (admin JWKS protected, internal resolve requires token, legacy auth routes protected).
  - `internal/model`: group helper methods.
  - `internal/config`: required env + defaults behavior.
- Integration tests (`make test-integration DATABASE_URL=...`) cover SQL paths in `internal/repository`:
  - user upsert create/update and active users listing.
  - session create/get/update tokens/delete by user.
  - tests auto-skip when `DATABASE_URL` is not provided.
