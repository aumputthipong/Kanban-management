# Architecture Decision Records

An ADR captures one architectural decision, the context that forced it, and the
consequences we accepted. They exist so a reviewer (or future me) can understand
*why* the system looks the way it does without reverse-engineering it from code
comments or git history.

Format is a trimmed [Nygard](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
style: **Context → Decision → Consequences**, plus a status line. One decision
per file, numbered, immutable once accepted — superseding a decision means a new
ADR that references the old one, not an edit.

## Index

| #                                             | Decision                                        | Status   |
|-----------------------------------------------|-------------------------------------------------|----------|
| [0001](0001-opaque-refresh-tokens.md)         | Opaque, revocable refresh tokens (not JWT)      | Accepted |
| [0002](0002-sqlc-over-orm.md)                 | sqlc for data access, no ORM                    | Accepted |
| [0003](0003-single-instance-websocket-hub.md) | In-memory WebSocket hub, single instance        | Accepted |
| [0004](0004-membership-gate-returns-404.md)   | Board membership gate returns 404, not 403      | Accepted |

## Adding an ADR

1. Copy the shape of an existing file; take the next number.
2. Keep it short — Context is the important part; Decision is usually a paragraph.
3. Link it from the table above and from the relevant code with a one-line
   `// See docs/adr/000N` pointer, so the rationale lives here, not in godoc.
