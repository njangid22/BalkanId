# BalkanID File Vault

Secure file vault with deduplicated storage, granular sharing, public downloads, and a clean onboarding flow. This monorepo ships a Go backend, a Next.js frontend, and Docker assets.

- Monorepo structure:
  - [app/backend](app/backend): Go API (GraphQL + REST), services, migrations
  - [app/frontend](app/frontend): Next.js 14 + TypeScript + Tailwind
  - [app/deploy](app/deploy): Docker Compose for local orchestration
  - Env templates: [app/.env.example](app/.env.example), [app/frontend/.env.example](app/frontend/.env.example)

---

## Demo (embedded)

<!-- Replace the src with your hosted MP4 (GitHub supports <video> with mp4). Keep a poster image if you have one. -->
[![Watch the video](https://img.youtube.com/vi/8hp1gf7PEJE/maxresdefault.jpg)](https://youtu.be/8hp1gf7PEJE)

### [The video walks through: environment setup, where to get keys, how to run locally and via Docker, deploying frontend/backend, and a UX tour (uploading, sharing, public downloads).](https://youtu.be/8hp1gf7PEJE)

<!-- If you only have YouTube/Vimeo, GitHub strips iframes. As a fallback, keep a clickable thumbnail:
[![Watch the video](https://img.youtube.com/vi/YOUR_ID/hqdefault.jpg)](https://www.youtube.com/watch?v=YOUR_ID)
-->


---

## Features

- Google SSO (OAuth 2.0), secure session cookies
- File uploads with server-side size/quota limits
- Deduplicated blobs, public/private sharing, direct downloads
- GraphQL API with file uploads
- Polished Next.js UI with drag‑and‑drop uploads

Key backend files:
- HTTP server and OAuth routes: [app/backend/internal/http/server.go](app/backend/internal/http/server.go)
- OAuth helper: [app/backend/internal/auth/google.go](app/backend/internal/auth/google.go)
- Configuration: [app/backend/internal/config/config.go](app/backend/internal/config/config.go)
- SQL migrations: [app/backend/migrations](app/backend/migrations)

---

## Local Development

Default local ports:
- Frontend: http://localhost:3000
- Backend: http://localhost:8080

Option A — Docker Compose (recommended)
1) Create backend env from template
   - Copy [app/.env.example](app/.env.example) → app/.env
   - Fill values (see “Environment Variables”)
2) Run migrations against your Supabase Postgres (see “Database schema”)
3) Start services
   - Windows PowerShell:
     - cd app\deploy
     - docker compose up --build
4) Open http://localhost:3000

Option B — Run without Docker
- Backend:
  - cd app\backend
  - Ensure app\.env exists (copy from example and fill)
  - go run .\cmd\server
- Frontend:
  - cd app\frontend
  - Copy [app/frontend/.env.example](app/frontend/.env.example) → .env.local
  - Set NEXT_PUBLIC_API_URL=http://localhost:8080
  - npm install
  - npm run dev
- Open http://localhost:3000

---

## Hosted Deployment (generic)

This project supports hosting the frontend (e.g., Vercel) and backend (any host). Use placeholders below and your video explains exact steps and values.

1) Backend host (environment)
- Set variables (names come from [app/backend/internal/config/config.go](app/backend/internal/config/config.go)):
  - FRONTEND_URL = https://your-frontend-domain
  - OAUTH_REDIRECT_URL = https://your-backend-domain/auth/google/callback
  - JWT_SECRET = a long random string
  - SESSION_COOKIE_NAME = vault_session
  - SESSION_TTL = 24h
  - RATE_LIMIT_RPS = 2
  - DEFAULT_USER_QUOTA_BYTES = 10485760
  - MAX_UPLOAD_BYTES = 10485760
  - SUPABASE_URL, SUPABASE_ANON_KEY, SUPABASE_SERVICE_ROLE_KEY, SUPABASE_DB_URL
  - STORAGE_BUCKET = blobs
  - REDIS_URL = (optional if you use Redis)
- Redeploy the backend.

2) Google OAuth (Google Cloud Console → OAuth 2.0 Client)
- Authorized JavaScript origins:
  - Local dev: http://localhost:3000
  - Hosted: https://your-frontend-domain
- Authorized redirect URIs:
  - Local dev: http://localhost:8080/auth/google/callback
  - Hosted: https://your-backend-domain/auth/google/callback
- Put GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET into backend env.

3) Frontend host (environment)
- Set:
  - NEXT_PUBLIC_API_URL = https://your-backend-domain
- Redeploy the frontend.
- Do not commit app/frontend/.env.local to Git for hosted deployments.

Optional: first‑party cookie proxy via Next.js rewrites (if your host blocks third‑party cookies)
- Add a rewrite so browser talks to /api/* on the frontend host, which proxies to your backend. Then set:
  - OAUTH_REDIRECT_URL = https://your-frontend-domain/api/auth/google/callback
  - Add this exact URL to Google’s Redirect URIs.
- This keeps cookies first‑party to the frontend origin.

---

## Environment Variables

Backend (.env) — copy [app/.env.example](app/.env.example)

Local example (ports: 3000 frontend, 8080 backend):
```
# Supabase
SUPABASE_URL=
SUPABASE_ANON_KEY=
SUPABASE_SERVICE_ROLE_KEY=
SUPABASE_DB_URL=

# Google OAuth
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
OAUTH_REDIRECT_URL=http://localhost:8080/auth/google/callback

# App
FRONTEND_URL=http://localhost:3000
SESSION_COOKIE_NAME=vault_session
SESSION_TTL=24h
JWT_SECRET=
RATE_LIMIT_RPS=2
DEFAULT_USER_QUOTA_BYTES=10485760
MAX_UPLOAD_BYTES=10485760
STORAGE_BUCKET=blobs
PORT=8080
REDIS_URL=redis://redis:6379
```

Hosted example (replace placeholders with your domains):
```
FRONTEND_URL=https://your-frontend-domain
OAUTH_REDIRECT_URL=https://your-backend-domain/auth/google/callback
# plus the Supabase and other keys as above
```

Frontend (.env) — copy [app/frontend/.env.example](app/frontend/.env.example)
- Local:
  NEXT_PUBLIC_API_URL=http://localhost:8080
- Hosted:
  NEXT_PUBLIC_API_URL=https://your-backend-domain

Never commit secrets. Keep only the .env.example files in Git. The repo already ignores .env via [.dockerignore](.dockerignore) and [app/frontend/.dockerignore](app/frontend/.dockerignore).

---

## Database schema (Supabase)

Run the SQL files in [app/backend/migrations](app/backend/migrations) using the Supabase SQL Editor (or psql), in order:
- 0001_init.sql
- 0002_shares_unique.sql
- 0003_folders.sql

Open Supabase → SQL → paste each file’s contents and run. This creates users, files, file_blobs, shares, and related indexes.

---

## How it works (high level)

- OAuth flow
  - Start: GET /auth/google/start (redirects to Google)
  - Callback: GET /auth/google/callback (verifies code, creates/updates user, sets session, redirects to FRONTEND_URL/files)
- Session
  - HttpOnly cookie; in hosted mode ensure Secure and SameSite=None so the browser sends it to the backend from the frontend origin.
- GraphQL
  - POST /graphql with credentials: include
  - Uploads via multipart; limited by MAX_UPLOAD_BYTES

Relevant code:
- Server and routes: [app/backend/internal/http/server.go](app/backend/internal/http/server.go)
- OAuth: [app/backend/internal/auth/google.go](app/backend/internal/auth/google.go)
- Config: [app/backend/internal/config/config.go](app/backend/internal/config/config.go)

---

## UI walkthrough

- Login page: Google sign‑in
- Files dashboard: upload via drag‑and‑drop, filter/search, manage shares
- Public links: create public link and share; anyone with the link can download
- Setup page: shows sample envs and local run instructions

---

## Troubleshooting

- Google 400 redirect_uri_mismatch
  - JS Origin must equal your frontend origin.
  - Redirect URI must match OAUTH_REDIRECT_URL exactly.

- Login loop on hosted
  - Backend must allow CORS from FRONTEND_URL with Allow‑Credentials=true.
  - Session cookie should be Secure and SameSite=None in production (https).
  - The frontend GraphQL client must send credentials: include.
  - Ensure hosted frontend uses NEXT_PUBLIC_API_URL for the hosted backend (and no leftover .env.local with localhost).

- CORS preflight fails
  - FRONTEND_URL must be an exact origin (scheme + host + optional port), no trailing slash.
  - AllowedHeaders should include Authorization, Content-Type.

---

## Development commands

Windows PowerShell (local without Docker)
- Backend:
  - cd app\backend
  - go run .\cmd\server
- Frontend:
  - cd app\frontend
  - npm install
  - npm run dev

Docker (local)
- cd app\deploy
- Copy env: cp ../.env.example ../.env
- docker compose up --build

Health checks
- Backend: curl http://localhost:8080/healthz
- GraphQL: open http://localhost:8080/playground

---

## Contact for environment keys

If this is for an assessment, contact the maintainer to request temporary keys:
- Email: nikhiljangid222@gmail.com

Replace with your real contact details as needed.

---

## License

Educational use only (no warranty).
