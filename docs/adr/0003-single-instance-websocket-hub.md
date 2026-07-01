# 0003 — In-memory WebSocket hub, single instance

**Status:** Accepted

## Context

Realtime sync (card moves, edits, activity feed) is delivered over WebSocket. The
server keeps one "room" per board and fans a broadcast out to every client in
that room. Where the room membership and the in-flight rate-limit buckets live
determines how the backend can be scaled.

A shared layer (Redis pub/sub for fan-out, a distributed rate limiter) would let
the backend run as N replicas behind a load balancer. That is real infrastructure
to run, operate, and pay for — weight this project does not yet carry.

## Decision

Keep hub state **in-memory**, and run the backend as a **single instance**:

- Room membership is a map in the `internal/websocket` hub; broadcasts fan out
  only to sockets held by that process.
- Rate limiting uses an in-memory token bucket per process.
- There is deliberately no Redis, no shared session store, no sticky-session
  routing.

This limitation is stated up front in the README and enforced operationally by
running one replica.

## Consequences

- Scaling is **vertical only**. Running multiple replicas without a shared layer
  would silently drop broadcasts to clients connected to a different instance —
  the worst kind of bug (invisible, data-shaped). Don't do it until the shared
  layer exists.
- A server restart drops all connections; clients reconnect with exponential
  backoff (1s → 30s) and the hub starts empty. No in-memory state needs
  recovering because activities are already persisted before broadcast (the DB is
  the source of truth, the hub is just a fan-out).
- For a single-team product and a portfolio demo, one instance is plenty. The
  scaling path (Redis pub/sub + distributed limiter) is known and documented, to
  be introduced only when a real hot path demands it.

Implemented in `backend/internal/websocket/`. See also
`docs/ARCHITECTURE.md → WebSocket hub`.
