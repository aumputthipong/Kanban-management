# 0001 — Opaque, revocable refresh tokens (not JWT)

**Status:** Accepted

## Context

Sessions use a short-lived access token (JWT, in the `auth_token` cookie) plus a
long-lived refresh token. The refresh token is the "stay logged in" credential,
so it has to survive for weeks — which makes its blast radius on theft large.

A JWT refresh token would be self-contained and need no database lookup, but JWTs
are only revocable by keeping a server-side blocklist and checking it on every
use — which reintroduces the database round-trip a JWT was supposed to avoid, and
gives no way to force-logout a user before the token's own expiry.

## Decision

The refresh token is an **opaque** 256-bit random string, not a JWT:

- 32 bytes from `crypto/rand`, base64url-encoded (well above the 128-bit floor
  for unguessable tokens).
- Only `sha256(token)` is stored in `refresh_tokens`; the raw value lives only in
  the client cookie. A leaked DB snapshot cannot be replayed.
- The cookie is `HttpOnly`, `SameSite=Strict`, and path-scoped to
  `/api/auth/refresh`, so it never travels with ordinary API calls — only the
  access token does.
- Rotation: every refresh mints a new token and revokes the old row, so an active
  session slides forward indefinitely while an idle one expires after the TTL
  (default 30d). Reuse of an already-revoked token is treated as theft and
  revokes **all** of that user's refresh tokens.

Refresh-token logic lives on `AuthService` (identity-scoped, shared with the
Register / Login / OAuth handlers); access-token issuance stays in the `token`
package shared with middleware.

## Consequences

- Every refresh is one indexed DB lookup by hash — the cost we accepted to get
  server-side revocation and single-use replay detection.
- Force-logout and "log out everywhere" are trivial (revoke rows); no blocklist,
  no JWT key rotation dance.
- The access token stays a stateless JWT, so the hot path (every authenticated
  request) still needs no DB read — only the infrequent refresh does.

Implemented in `backend/internal/token/refresh.go` and
`backend/internal/service/refresh_token_service.go`.
