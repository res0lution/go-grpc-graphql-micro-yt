# Identity Resolver Template

This folder contains a production-oriented template for integrating `core` with `identity-manager` using feature flags.

## Included files

- `config.go` - feature flags and client configuration with defaults
- `client.go` - resilient internal API client for `/v1/internal/identity/resolve`
- `middleware.go` - middleware with primary source selection, dual-read, and fail-open fallback
- `context.go` - request context helpers to pass resolved identity downstream
- `errors.go` - internal helper errors

## Typical rollout setup

1. Start with:
   - `Enabled=false`
   - `DualReadEnabled=true`
   - `PrimarySource=legacy`
   - `FailOpen=true`
2. Compare old and new identity resolution in logs.
3. Switch `PrimarySource` to `idm`.
4. Disable `DualReadEnabled` after confidence is high.
