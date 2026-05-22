# Identity Manager Architecture

## Goal

`identity-manager` is a standalone REST service extracted from the monolith (`old/*`) to handle:

- authentication (`auth`)
- user identity profile (`user`)
- session lifecycle (`session`)

The service is an integration point between:

- frontend app (browser/client)
- core backend app (internal consumer)
- external Identity Provider (OIDC/OAuth2)

## Scope and Boundaries

### In scope (identity-manager)

- OIDC/OAuth2 Authorization Code flow (`/auth/login`, `/auth/callback`)
- token exchange and validation with IdP
- user upsert from ID token claims
- session issuing, refresh, revocation, back-channel logout
- secure session cookie policy
- endpoint to resolve identity context for core backend

### Out of scope

- business authorization/ACL rules (remain in core backend)
- domain-specific user enrichment (remain in core backend)
- IdP login UI pages (owned by IdP)

## High-level Components

- `api/http`  
  REST routing, validation, status mapping, error contracts.

- `domain/auth`  
  OAuth2/OIDC flow orchestration, state/nonce generation, logout token checks.

- `domain/user`  
  User creation/update from claims, read current profile.

- `domain/session`  
  Session create/get/refresh/revoke, TTL policy.

- `adapter/idp`  
  Outbound integration to IdP endpoints (`/oauth2/token`, JWKS, introspection if needed).

- `adapter/core`  
  Internal API for core backend identity context resolving.

- `storage/postgres`  
  Repositories for users/sessions, migrations, indexes.

- `platform`  
  Config, logs, metrics, tracing, health/readiness.

## Proposed API (v1)

### Auth

- `GET /v1/auth/login` - initiate login redirect to IdP
- `GET /v1/auth/callback` - handle code+state callback, create session, set cookie
- `POST /v1/auth/logout` - logout current/all sessions and/or process back-channel logout
- `GET /v1/auth/jwks/status` - ops endpoint for JWKS cache health

### User

- `GET /v1/users/me` - return current authenticated user profile

### Session

- `GET /v1/sessions/me` - return current session metadata
- `POST /v1/sessions/refresh` - rotate/refresh session
- `DELETE /v1/sessions/me` - terminate current session

### Internal (core backend)

- `POST /v1/internal/identity/resolve` - resolve identity and session context for service calls

## Data Model (PostgreSQL)

### `users`

- `id uuid pk`
- `identity_id varchar unique not null`
- `sub varchar not null`
- `login varchar`
- `email varchar`
- `employee_number varchar`
- `winaccountname varchar`
- `given_name varchar`
- `family_name varchar`
- `name varchar`
- `groups jsonb`
- `is_active boolean`
- `last_login_at timestamp`
- `created_at timestamp`
- `updated_at timestamp`

### `sessions`

- `id uuid pk`
- `user_id uuid fk -> users.id`
- `access_token text`
- `refresh_token text`
- `id_token text`
- `token_expiry timestamp`
- `expires_at timestamp`
- `created_at timestamp`
- `updated_at timestamp`

## Runtime Flow

1. Frontend calls `GET /v1/auth/login`.
2. identity-manager redirects user to IdP with generated `state` and `nonce`.
3. IdP returns to `GET /v1/auth/callback` with `code` and `state`.
4. identity-manager exchanges code to tokens via IdP.
5. identity-manager upserts user from claims and creates session in DB.
6. identity-manager sets secure `HttpOnly` cookie and redirects back to frontend.
7. Frontend uses `/v1/users/me` and `/v1/sessions/me`.
8. Core backend resolves identity context through internal endpoint.

## Security Baseline

- validate `state` and `nonce` on callback
- `HttpOnly + Secure + SameSite` session cookie
- short-lived access/session TTL and refresh rotation
- token secrets encryption/hashing at rest (recommended)
- service-to-service auth for internal endpoints (mTLS or signed JWT)
- audit logs for login/logout/session revocation

## Migration Plan (Monolith -> Service)

### Phase 1: Foundation

- bootstrap service skeleton and CI
- add config and health endpoints
- create DB migrations for users/sessions

### Phase 2: Auth + Session

- migrate auth callback/token exchange
- migrate session create/revoke logic
- integrate frontend with new auth endpoints

### Phase 3: User

- migrate user upsert and `/users/me`
- remove duplicated user/session logic from monolith

### Phase 4: Core integration and cutover

- add internal identity resolve endpoint
- switch core backend to identity-manager contract
- deprecate old monolith auth paths

## Initial Non-Functional Targets

- p95 latency:
  - read endpoints <= 150 ms
  - callback/token exchange <= 700 ms
- availability target: 99.9%
- full structured logs with correlation/request ID
- metrics: login attempts, callback errors, session refresh success rate
