# Core Cutover Checklist

## Goal

Switch core backend identity resolution to `identity-manager` internal contract without breaking existing auth/session behavior.

## Contract to use

- Endpoint: `POST /v1/internal/identity/resolve`
- Auth: `Authorization: Bearer <CORE_INTERNAL_AUTH_TOKEN>`
- Session input (priority order):
  - `X-Session-ID` header
  - JSON body: `{"session_id":"..."}`
- Success response:
  - `200 OK`
  - `{"success": true, "identity": {...}}`

## Identity payload

`identity` includes:

- `session_id`
- `user_id`
- `identity_id`
- `login`
- `groups`

## Rollout steps

1. Set `CORE_INTERNAL_AUTH_TOKEN` in identity-manager and core backend.
2. Run external migration step before deploy:
   - `make migrate-up DATABASE_URL=...`
3. Deploy identity-manager with migrations already applied.
4. Update core backend middleware/client to call `/v1/internal/identity/resolve`.
5. Enable dual-read mode in core (old source + new source) and compare identity fields.
6. Switch core to identity-manager as primary source.
7. Remove old monolith identity resolution path.

## Verification

- Login/callback issues valid `session_id` cookie.
- `GET /api/v1/auth/user` and `POST /api/v1/auth/refresh` stay functional.
- `POST /v1/sessions/refresh` rotates tokens via IdP refresh flow.
- Internal resolve returns stable identity for active session.
- `GET /v1/admin/jwks/status` and `GET /api/v1/admin/jwks/status` require authenticated admin group.
- Expired/revoked sessions return non-200 with clear error code.

## Migration strategy

- Startup migration runner is disabled by design.
- Migrations are applied only as an explicit external deployment step.
- Service is expected to fail fast on SQL/runtime errors if schema is not up to date.

## Rollback

- Keep old path behind feature flag until cutover confidence is reached.
- On incident, toggle core back to old identity source and keep identity-manager running for diagnostics.
