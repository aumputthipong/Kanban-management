# 0005 — WebSocket handshake authenticates with a short-lived ticket

**Status:** Accepted

## Context

The frontend and the backend are deployed on different registrable domains
(`*.vercel.app` and `*.onrender.com`). Browsers block third-party cookies, so
the auth cookie set by the API host was dropped and login failed. The fix was
to route browser API calls same-origin at `/api` and let a Next.js rewrite
proxy them to the backend — the cookie is now first-party to the *frontend*
origin.

That leaves the WebSocket with no credential. Next.js rewrites do not proxy a
WebSocket upgrade (and cannot on Vercel without a custom server), so the socket
still dials the API host directly, cross-origin, where the cookie does not
exist. `RequireAuth` answered every handshake with 401.

This was not a degraded-realtime bug. Card create / move / delete / done-toggle
and every column mutation are sent over the socket, and the socket is what
writes them to the database, so the board silently stopped persisting while the
optimistic UI kept showing success.

A browser cannot set headers on a WebSocket, so the only channel available to
the handshake is the URL. The cookie cannot be read from JS either — it is
HttpOnly by design — so the page cannot simply copy the session token into the
URL. Something has to mint a separate, URL-safe credential.

## Decision

Authenticate the handshake with a **short-lived, WS-only JWT** passed as a
`ticket` query parameter.

- `GET /api/ws-ticket` runs on the cookie-authed path (same-origin, through the
  existing proxy — the one place the cookie is available) and mints a token
  signed with the existing `JWT_SECRET`, carrying `aud: "ws"` and a 30s TTL
  (`WS_TICKET_TTL`).
- `RequireWSTicket` gates `/ws/{boardID}`, validates the ticket, and injects the
  user ID under the same context key `RequireAuth` uses — so the board
  membership gate composes on top unchanged.
- The audience claim is enforced in **both** directions: `Parse` rejects a WS
  ticket, `ParseWSTicket` rejects an access token. Sharing a signing key between
  two credentials is only safe if neither is accepted where the other belongs.
- The client fetches a **fresh ticket per connection attempt**. Reconnect
  backoff climbs to 30s, so a ticket held across a wait would arrive expired.
- `/ws/` is added to the request logger's redaction list.

Rejected alternatives:

- **A stateful, single-use ticket store.** Stronger against replay, and the
  in-memory map would have been consistent with ADR 0003's single instance.
  Not worth a map, a mutex, a sweeper, and restart semantics for a 30s window.
- **A shared parent domain** (`app.` + `api.` on one registrable domain). This
  is the real fix — it removes the ticket *and* the proxy hop on every REST
  call. It needs a domain purchase and DNS, which does not unblock a broken
  production today. Revisit; this ADR is superseded the day it happens.

## Consequences

- The ticket travels in the URL, so it reaches upstream access logs we do not
  control (Render, Cloudflare). The 30s TTL is the mitigation, and the
  `aud` check limits a captured ticket to joining a board room the holder is
  already a member of.
- Connecting costs one extra round trip. It happens once per board open, and
  again per reconnect attempt.
- `connect()` is now async, which introduces a suspension point between "decide
  to connect" and "open the socket". The effect's cancel guard is a per-run
  local, not a ref: a ref shared across effect runs would be reset to `false` by
  the next run and let a stale connect proceed after a `boardID` change.
- If the session has expired, the ticket fetch 401s; `apiClient` attempts a
  refresh and redirects to login when that fails. The socket stops retrying
  because the page navigates away.
- There are now two ways to authenticate. That surface is held shut by the
  audience check, which is covered by tests in both directions — treat those
  tests as load-bearing.
- **This fixes the transport, not the design.** Writes still ride a
  fire-and-forget channel: `sendMessage` drops the message when the socket is
  not open, without buffering, retry, or any signal to the user. A wifi drop, a
  laptop sleep, a free-tier spin-down, or a deploy still loses work silently.
  Moving board mutations onto REST — leaving the socket as pure fan-out, which
  is all ADR 0003 ever claimed it was — is the follow-up.
