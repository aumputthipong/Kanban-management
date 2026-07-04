# Deployment runbook

How to ship a new version, recover from common breakages, and what to monitor. Treat this as a script — when something is on fire at 2 AM, you should be able to follow it without thinking.

> **First time deploying the free demo?** Follow the click-by-click [deploy walkthrough](deploy-walkthrough.md) (Vercel + Render + Neon, every choice pre-made). This runbook is the reference; that guide is the first-run script.

## Pre-flight checklist

Run through this before every production deploy.

- [ ] CI green on the target commit (`go test`, `npm test`, `tsc --noEmit`)
- [ ] Migrations are forward-compatible (no destructive change without a backfill plan)
- [ ] `JWT_SECRET` in production env is **32+ random bytes**, not a placeholder
- [ ] `ENV=production` set on the backend (turns on `Secure` cookies + HSTS)
- [ ] `FRONTEND_URL` matches the actual public URL (used for CORS allow-origin)
- [ ] `NEXT_PUBLIC_API_URL` and `NEXT_PUBLIC_WS_URL` baked into the frontend build point at the production backend
- [ ] Database backup taken in the last 24 h
- [ ] Rollback plan ready (see below)

## Deploy paths

The repo is host-agnostic. Pick whichever target matches the use case.

### A. Self-hosted VPS (Docker Compose)

Best for: portfolio demo, low-traffic team usage, full control.

```bash
ssh deploy@your-vps
cd /opt/turtask
git pull
docker compose -f docker-compose.prod.yml pull   # if images are pre-built
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml logs -f backend
```

In front of the stack you'll typically want:
- **nginx** (or Caddy) reverse proxy terminating TLS via Let's Encrypt
- **ufw** firewall: 22 + 80 + 443 only
- A `cron` entry that runs `pg_dump` daily and ships the dump to S3/R2

Sample nginx server block (path: `/etc/nginx/sites-available/turtask`):

```nginx
server {
  listen 443 ssl http2;
  server_name turtask.example.com;
  ssl_certificate     /etc/letsencrypt/live/turtask.example.com/fullchain.pem;
  ssl_certificate_key /etc/letsencrypt/live/turtask.example.com/privkey.pem;

  location /api/      { proxy_pass http://127.0.0.1:8080; include /etc/nginx/proxy_params; }
  location /ws/       { proxy_pass http://127.0.0.1:8080; proxy_http_version 1.1;
                        proxy_set_header Upgrade $http_upgrade;
                        proxy_set_header Connection "upgrade";
                        proxy_read_timeout 3600s; }
  location /healthz   { proxy_pass http://127.0.0.1:8080; access_log off; }
  location /          { proxy_pass http://127.0.0.1:3000; include /etc/nginx/proxy_params; }
}
```

### B. Vercel + Railway/Render + Neon

Best for: zero ops; managed everything.

| Component | Host                              | Notes                                                        |
|-----------|-----------------------------------|--------------------------------------------------------------|
| Frontend  | Vercel (`frontend/`)              | Connect repo; `next.config.ts` standalone is ignored on Vercel |
| Backend   | Railway / Render Docker service   | Build context: `backend/`; expose port 8080                  |
| DB        | Neon / Supabase                   | Use `?sslmode=require` and the **pooler** connection string  |

Set env vars on each platform — copy from `backend/.env.example` and `frontend/.env.example`. The Google OAuth `GOOGLE_REDIRECT_URL` must match what's registered in the Google Cloud console.

> **Cross-site cookies (required for this split):** frontend and backend are on different sites (`*.vercel.app` vs `*.onrender.com`), so set **`COOKIE_CROSS_SITE=true`** on the backend. This switches the auth cookies to `SameSite=None; Secure`. Without it the browser refuses to send the cookie cross-site and login silently loops (see "Unauthorized loop" below). Both hosts serve HTTPS, so `Secure` is satisfied.

**Seed demo data** (once, after the first successful deploy): with the production `DB_URL` set, run `make seed` (or `cd backend && go run ./cmd/seed`). It creates a `demo@turtask.app` login and a sample board with cards + a planning session so the app opens onto real content. Idempotent — re-running does nothing if the demo user already exists.

### C. Cloud Run / ECS

Use the existing Dockerfiles. Key extras:
- Cloud Run: `--allow-unauthenticated`, `--port 8080`, set `ENV=production`. WS support requires the gen2 execution environment + connection-based billing.
- ECS: build via the existing Dockerfiles, ALB target group, ALB → 80/443 → backend container. WebSocket support requires sticky sessions or a single instance per board until the hub is sharded.

## Smoke test (post-deploy)

Run within 5 minutes of every deploy:

```bash
PROD=https://turtask.example.com

# 1. Health probe
curl -s "$PROD/healthz" | jq
# expect: {status: "ok", db_connected: true, uptime_seconds: <small>}

# 2. Auth flow
curl -s -i -X POST "$PROD/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"smoke@example.com","password":"<password>"}'
# expect: 200 + Set-Cookie: auth_token=...

# 3. Validation actually rejects bad input
curl -s -i -X POST "$PROD/api/auth/register" \
  -H 'Content-Type: application/json' \
  -d '{"email":"bad","full_name":"x","password":"short"}'
# expect: 400 with field-level error

# 4. Rate limit kicks in
for i in {1..25}; do curl -s -o /dev/null -w '%{http_code}\n' "$PROD/api/auth/login" -d '{}'; done
# expect: first ~20 are 4xx, then 429
```

If any of these fail, **roll back** before debugging.

## Rollback

The fastest path is the same regardless of target:

### Docker Compose
```bash
git checkout <previous-tag>
docker compose -f docker-compose.prod.yml up -d --build
```

### Vercel/Railway
Both keep the previous build/deployment. Use the dashboard's "Promote to production" / "Rollback" button. Faster than rebuilding.

### Database
**Migrations are not auto-rolled back** by an app rollback. If a bad migration is in production:

```bash
go run github.com/golang-migrate/migrate/v4/cmd/migrate \
  -source file://backend/database/migrations \
  -database "pgx5://$DB_URL" \
  down 1
```

Before running `down`, take a fresh `pg_dump` — `down` migrations may drop columns / tables.

## Common breakages

### `/healthz` returns 503 with `db_connected: false`
- Check DB container/instance is running
- Check connection pool isn't exhausted: backend logs will show `acquire ... timeout`
- Check network/firewall between backend and DB
- If using Neon, the cold-start may take 5–10s on first request — wait, then re-probe

### Backend boots but immediately exits with `migrations failed`
- Migration version is `dirty` — a prior migration crashed mid-apply
- Fix: connect to DB, inspect `schema_migrations`, manually correct the offending migration, then `UPDATE schema_migrations SET dirty = false WHERE version = <N>`
- If the migration itself is broken, delete it from `migrations/`, restart, then commit a new migration with the fix. **Never edit a migration that's already shipped.**

### CORS errors in browser console
- `FRONTEND_URL` env on the backend doesn't match the actual origin (including protocol + port)
- Misconfigured nginx — `Access-Control-Allow-*` headers are managed by the Go service, not the proxy. Don't double-set them.

### WebSocket disconnects every ~100s
- Likely Cloudflare / proxy idle timeout. Either bump the proxy's `proxy_read_timeout` past your idle window or wait for the heartbeat work in P1.

### Rate limit too aggressive
- Defaults: 20/min/IP for `/api/auth`, 300/min/IP for protected. If a legitimate caller hits the limit, edit `internal/middleware/ratelimit.go` and redeploy.
- For a single trusted IP that needs more, bypass `httprate.LimitByIP` with a custom `KeyFunc` that returns a constant for that IP.

### "Unauthorized" loop after deploy
- **Split-domain deploy (frontend ≠ backend site):** the auth cookie is `SameSite=Lax/Strict` by default, so the browser won't send it cross-site — login succeeds, the next request is 401. Fix: set **`COOKIE_CROSS_SITE=true`** on the backend (→ `SameSite=None; Secure`). Both ends must be HTTPS.
- Old `auth_token` cookie was signed with the previous `JWT_SECRET`. Either revert the secret change or accept that all sessions are invalidated (everyone re-logs in).
- This is why `JWT_SECRET` should not rotate casually.

## Monitoring (minimum viable)

Until proper observability lands (Week 4):

- **UptimeRobot** (free) — pings `/healthz` every 5 min, alerts to email
- **Container logs** — `docker compose logs -f backend` during incidents; tail to a log aggregator if available
- **Postgres** — `SELECT pg_database_size('erp_kanban');` weekly to track growth

What to watch when adding:
- p95 request latency (target: < 500 ms)
- DB pool acquire wait (warn: > 100 ms p95)
- WebSocket open connections (per-instance)
- `/healthz` 5xx rate (target: < 0.1%)

## Secrets management

- `.env` files are git-ignored. Never commit them.
- Production secrets: store in the host's secret manager (Railway/Vercel env vars, AWS Secrets Manager, Doppler, etc.) — not in plain `.env` files on the VPS unless the VPS itself is locked down.
- Rotate `JWT_SECRET` only when there's a known leak — it invalidates every active session.
- OAuth client secrets: regenerate via Google Cloud Console; the redirect URI must stay registered there.

## Versioning

`version` is injected at build time via Docker `--build-arg VERSION=...`. Tag releases as `vX.Y.Z`:

```bash
git tag -a v0.3.0 -m "Week 3 — docs"
git push origin v0.3.0
docker build --build-arg VERSION=v0.3.0 -t turtask-backend:v0.3.0 backend/
```

`/healthz` returns this string so you can confirm which build is live.
