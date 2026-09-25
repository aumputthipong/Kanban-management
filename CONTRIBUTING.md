# Contributing

Thanks for taking a look. This doc covers the dev workflow, branch / commit conventions, and the PR checklist.

## Local setup

```bash
git clone <repo>
cd mini-erp-kanban
make dev          # spins up Postgres + backend + frontend (see Makefile)
```

If `make` isn't available, follow the full Quick Start in [`README.md`](README.md). Verify with:

```bash
curl http://localhost:8080/healthz | jq
# expect db_connected: true
```

## Branch naming

Use a short prefix that signals intent — same set we already use throughout the repo:

| Prefix      | When to use                                                  |
|-------------|--------------------------------------------------------------|
| `feat/...`  | new user-visible feature                                     |
| `fix/...`   | bug fix                                                      |
| `refactor/...` | internal restructure with no behaviour change            |
| `perf/...`  | performance work                                             |
| `docs/...`  | docs / godoc / JSDoc / README only                           |
| `chore/...` | tooling, config, deps, repo polish                           |
| `ci/...`    | CI / build pipeline only                                     |
| `test/...`  | tests only                                                   |

Keep names kebab-case and tight: `feat/my-tasks-redesign`, not `feat/redesign-my-tasks-page-to-add-due-soon-tabs`.

## Commit messages

Conventional-style prefix matching the branch:

```
feat: add no-date tab to /my-tasks
fix: prevent double-broadcast on optimistic card move
refactor: extract permission check into middleware
chore(deps): bump pgx to v5.9.2
```

Subject under 70 chars. Body for the *why*, not the *what* — diffs already describe the what.

## Before opening a PR

Run locally — same checks CI runs:

```bash
make verify       # go vet + go test + tsc + vitest
```

Or piecemeal:

```bash
cd backend  && go vet ./... && go test -race ./...
cd frontend && npx tsc --noEmit && npm test -- --run
```

If you touched API handlers, regenerate the OpenAPI spec:

```bash
make swag
```

## Pull request checklist

- [ ] Branch follows the prefix table above
- [ ] CI is green on the PR
- [ ] Tests added or updated for the change
- [ ] Docs updated if behaviour changed (`docs/`, `docs/CODE-NOTES.md`, `README`)
- [ ] Migration files included if the schema changed (additive forward, see `docs/DATABASE.md`)
- [ ] OpenAPI spec regenerated if handler annotations changed
- [ ] No secrets committed (`.env`, credentials)

## What goes where

If you're touching code, glance at [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) first — it lists the layering rules and the things that are *intentionally* not in the project (no ORM, no GraphQL, no microservices). Useful before reaching for a new abstraction.

Schema changes: [`docs/DATABASE.md`](docs/DATABASE.md) explains the position-gap strategy, soft-delete pattern, and migration rules ("never edit a migration that's already shipped").

Deploying: [`docs/DEPLOY.md`](docs/DEPLOY.md) covers the pre-flight checklist, three deploy paths, and the rollback playbook.

## Code style

- **Go** — `gofmt` clean, lint-clean. Keep Swagger `// @...` annotations on handlers. Prefer concrete types over `interface{}`. Service-layer methods take `context.Context` first.
- **TypeScript** — strict mode is on; don't add `any` to silence the compiler.
- **Comments** — default to none; explanations go in `docs/CODE-NOTES.md` or an ADR. See AGENTS.md "Comment & doc conventions" ([ADR 0011](docs/adr/0011-minimal-comments.md)).
- **CSS / Tailwind v4** — utility classes in JSX; arbitrary values only when the design literally requires it. No emoji in code unless explicitly requested.
- Keep changes minimal and on-topic. A bug fix is not a license to refactor surrounding code.

## Reporting issues

Open a GitHub issue with:

- repro steps
- expected vs. actual
- `version` from `/healthz` (so we know the build), browser / OS, and any console errors
