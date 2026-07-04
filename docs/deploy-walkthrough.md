# Deploy walkthrough — free live demo

A first-time, click-by-click guide to putting Turtask online for **free**, end to end.
Every choice is already made for you. Follow the steps in order — the ordering matters
(there's a chicken-and-egg between the frontend and backend URLs).

**Stack (decided):** Vercel (frontend) · Render (backend, Docker) · Neon (Postgres) · Google OAuth on · a seeded demo account.
**Cost:** $0 — no credit card needed on any of the three.
**Time:** ~1.5–2 hours the first time.
**Region:** pick **Singapore** everywhere it's offered (lowest latency from Thailand).

Prerequisites on your machine: `git`, Go 1.25+ (for the one-time seed), and a browser. That's it.

> The code is already prepared and on `main` (`COOKIE_CROSS_SITE` flag + demo seed). Just `git pull` first.

---

## Order of operations (why this order)

```
1. Neon        → get DB_URL
2. Render       → deploy backend (needs DB_URL) → get the backend URL
3. Google OAuth → register the backend callback URL
4. Vercel       → deploy frontend (needs the backend URL) → get the frontend URL
5. Render again → set FRONTEND_URL to the Vercel URL → redeploy
6. Seed         → run once, locally, against Neon
7. Verify       → smoke test + real browser login
8. Keep-warm    → UptimeRobot ping
9. README       → demo link + screenshots + credentials
```

The backend needs the DB before it can boot; the frontend needs the backend URL baked in at
build time; and the backend needs the frontend URL for CORS + cookies. So you deploy the
backend with a placeholder frontend URL first, then circle back in step 5.

---

## Step 1 — Neon (Postgres)

1. Go to **neon.tech** → sign up (GitHub login is fine).
2. **Create project.** Name it `turtask`. Region: **Singapore (ap-southeast-1)**. Postgres 15+.
3. On the project dashboard, find the **connection string**. Choose the **Pooled connection** toggle (important — the pooler handles many short connections).
4. Copy the string. It looks like:
   ```
   postgres://<user>:<password>@<host>-pooler.ap-southeast-1.aws.neon.tech/turtask?sslmode=require
   ```
   Make sure it ends with **`?sslmode=require`**. Keep it somewhere safe — this is your `DB_URL`.

That's all for Neon. Migrations run automatically when the backend first boots (step 2).

---

## Step 2 — Render (backend)

1. Go to **render.com** → sign up with GitHub → authorize access to your repo.
2. **New +** → **Web Service** → connect the `Kanban-management` repo.
3. Configure:
   | Field | Value |
   |---|---|
   | Name | `turtask-api` |
   | Region | **Singapore** |
   | Branch | `main` |
   | **Root Directory** | `backend` |
   | **Runtime / Language** | **Docker** (it auto-detects `backend/Dockerfile`) |
   | Instance Type | **Free** |
   | Health Check Path | `/healthz` |
4. **Environment variables** (Advanced → Add Environment Variable). Add these:
   | Key | Value |
   |---|---|
   | `DB_URL` | the Neon pooled string from step 1 |
   | `JWT_SECRET` | generate one: run `openssl rand -base64 32` locally and paste it. **Keep it stable forever** — changing it logs everyone out. |
   | `ENV` | `production` |
   | `COOKIE_CROSS_SITE` | `true` |
   | `MIGRATIONS_PATH` | `/app/database/migrations` |
   | `FRONTEND_URL` | `http://localhost:3000` (temporary placeholder — fixed in step 5) |
   | `GOOGLE_CLIENT_ID` | leave blank for now (filled in step 3) |
   | `GOOGLE_CLIENT_SECRET` | blank for now |
   | `GOOGLE_REDIRECT_URL` | blank for now |

   > **Do not set `PORT`.** Render injects it and the app reads it automatically.
5. **Create Web Service.** Watch the logs. On first boot you should see `running database migrations` then `server listening`. If it says `db_connected: false` or `migrations failed`, re-check `DB_URL` (must include `?sslmode=require`).
6. Copy your backend URL from the top of the page, e.g. **`https://turtask-api.onrender.com`**. Call it `<BACKEND_URL>`.
7. Quick check — open `https://turtask-api.onrender.com/healthz` in a browser. Expect `{"status":"ok","db_connected":true,...}`.

> Free Render **spins down after ~15 min idle**; the next request cold-starts in ~50s. Fixed in step 8.

---

## Step 3 — Google OAuth

1. Go to **console.cloud.google.com** → create a project (or reuse one), name it `turtask`.
2. **APIs & Services** → **OAuth consent screen**:
   - User type: **External** → Create.
   - App name `Turtask`, your email for support + developer contact.
   - Scopes: add **`.../auth/userinfo.email`** and **`.../auth/userinfo.profile`** (these are non-sensitive — no Google verification needed).
   - **Publish app** (Publishing status → "In production"). With only email/profile scopes this needs no review.
3. **APIs & Services** → **Credentials** → **Create Credentials** → **OAuth client ID**:
   - Application type: **Web application**.
   - Name: `turtask-web`.
   - **Authorized redirect URIs** → Add:
     ```
     https://turtask-api.onrender.com/api/auth/google/callback
     ```
     (your `<BACKEND_URL>` + `/api/auth/google/callback` — exact, no trailing slash)
   - Create → copy the **Client ID** and **Client secret**.
4. Back in **Render** → your service → **Environment** → fill:
   | Key | Value |
   |---|---|
   | `GOOGLE_CLIENT_ID` | the client id |
   | `GOOGLE_CLIENT_SECRET` | the client secret |
   | `GOOGLE_REDIRECT_URL` | `https://turtask-api.onrender.com/api/auth/google/callback` |
   Save (Render will redeploy).

---

## Step 4 — Vercel (frontend)

1. Go to **vercel.com** → sign up with GitHub → import the `Kanban-management` repo.
2. Configure:
   | Field | Value |
   |---|---|
   | Framework Preset | **Next.js** (auto-detected) |
   | **Root Directory** | `frontend` |
   | Build/Output | leave defaults |
3. **Environment Variables** (Production):
   | Key | Value |
   |---|---|
   | `NEXT_PUBLIC_API_URL` | `https://turtask-api.onrender.com/api` |
   | `NEXT_PUBLIC_WS_URL` | `wss://turtask-api.onrender.com/ws` |
   Note the schemes: **`https`** for API, **`wss`** for the WebSocket. Both point at `<BACKEND_URL>`.
   > These are baked in at build time — if you change them later you must redeploy.
4. **Deploy.** When it finishes, copy the production URL, e.g. **`https://turtask.vercel.app`**. Call it `<FRONTEND_URL>`.

---

## Step 5 — Close the loop (Render `FRONTEND_URL`)

1. Back in **Render** → your service → **Environment**.
2. Set **`FRONTEND_URL`** = your `<FRONTEND_URL>` from step 4 (e.g. `https://turtask.vercel.app`, no trailing slash).
3. Save → Render redeploys. This makes CORS allow your Vercel origin and lets the auth cookie work cross-site.

---

## Step 6 — Seed the demo data (once)

Run the seeder from **your local machine**, pointed at the Neon DB (schema already exists from step 2's migrations). In a terminal at the repo root:

```bash
git pull                     # make sure you have cmd/seed
cd backend
DB_URL="<your Neon pooled string with ?sslmode=require>" go run ./cmd/seed
```

Expect a log line `seed complete` with `demo_login=demo@turtask.app`. It's idempotent — running it again does nothing.

This creates:
- a demo login **`demo@turtask.app`** / **`demodemo123`**
- a "Product Launch" board with cards across To&nbsp;Do / In&nbsp;Progress / Review / Done (some overdue, some assigned) and a planning session.

---

## Step 7 — Verify (do this in a real browser)

1. **Health:** `https://turtask-api.onrender.com/healthz` → `status: ok`, `db_connected: true`.
2. **Login (the real cookie test):** open your `<FRONTEND_URL>`, log in with `demo@turtask.app` / `demodemo123`. **Reload the page and click into a board.** You must stay logged in — if you get bounced to /login, `COOKIE_CROSS_SITE` isn't `true` on Render (re-check step 2 + 5).
3. **Google OAuth:** click "Login with Google" → should round-trip and land you logged in.
4. **Realtime:** open the same board in **two browser tabs**, drag a card in one → it moves in the other within a second.
5. Optional deeper smoke test: the curl block in [`DEPLOY.md`](DEPLOY.md#smoke-test-post-deploy) against `<BACKEND_URL>`.

If login loops or CORS errors show in the browser console, see [`DEPLOY.md` → Common breakages](DEPLOY.md#common-breakages).

---

## Step 8 — Keep it warm (free)

1. Go to **uptimerobot.com** → sign up (free).
2. **Add New Monitor** → type **HTTP(s)** → URL `https://turtask-api.onrender.com/healthz` → interval **5 minutes**.
3. This pings the backend so Render is less likely to be cold when someone opens the demo, and emails you if it goes down.

---

## Step 9 — Portfolio polish (README)

Edit `README.md` (top of the file) and add a live-demo block, then commit + push:

```markdown
## Live demo

**https://turtask.vercel.app** — try it with the demo account:

- **Email:** `demo@turtask.app`
- **Password:** `demodemo123`

> Hosted on free tiers; the backend may take ~50s to wake on the first request.
```

Also add 2–3 screenshots or a short GIF (board view, the realtime sync, the planning tab):
1. Take screenshots, drop them in a new `docs/screenshots/` folder.
2. Reference them in the README: `![Board](docs/screenshots/board.png)`.

Commit on a branch → PR → merge, same as always.

---

## If something breaks — quick map

| Symptom | Cause → fix |
|---|---|
| Login works, next click → /login | `COOKIE_CROSS_SITE` not `true` on Render, or `ENV` ≠ `production` |
| CORS error in console | `FRONTEND_URL` on Render ≠ your exact Vercel URL (step 5) |
| `/healthz` → `db_connected: false` | `DB_URL` wrong or missing `?sslmode=require` |
| Backend exits: `migrations failed` | bad `DB_URL`, or Neon not reachable |
| Google login → "redirect_uri_mismatch" | the URI in Google console ≠ `GOOGLE_REDIRECT_URL` exactly |
| WebSocket won't connect | `NEXT_PUBLIC_WS_URL` must be `wss://…/ws` (not `https`), and rebuilt on Vercel |
| First load very slow | Render cold start — expected on free; step 8 mitigates |

More detail and rollback steps: [`DEPLOY.md`](DEPLOY.md).
