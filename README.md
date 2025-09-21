# BalkanID File Vault

Secure file vault platform with deduplicated storage, granular sharing, public file downloads, and a clean onboarding experience. This monorepo contains the Go backend, Next.js frontend, and deployment assets.

## Project Structure

- `app/backend/`: Go services, GraphQL API, REST endpoints, and migrations
- `app/frontend/`: Next.js 14 + TypeScript client and Tailwind CSS
- `app/deploy/`: Docker Compose for local orchestration
- `.env.example`: Configuration template to copy to `.env` for development

## Prerequisites

- Go 1.22+ (Go 1.23 supported)
- Node.js 18+ (Node 20 recommended)
- Docker & Docker Compose
- Supabase project with Postgres + Storage
- Google Cloud project with OAuth 2.0 client credentials

## Configuration (.env)

Copy `app/.env.example` to `app/.env` and fill values. Key variables:

- `SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`, `SUPABASE_DB_URL`
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `OAUTH_REDIRECT_URL`
- `JWT_SECRET`, `RATE_LIMIT_RPS`, `DEFAULT_USER_QUOTA_BYTES`, `MAX_UPLOAD_BYTES`
- `STORAGE_BUCKET`, `PORT`, `REDIS_URL`

Frontend uses `NEXT_PUBLIC_API_URL` at runtime to reach the backend (defaults to `http://localhost:8080` for local dev).

## Database Migrations

Run the SQL in `app/backend/migrations/` against your Supabase Postgres instance (via SQL editor or CLI):

- `0001_init.sql`
- `0002_shares_unique.sql`
- `0003_folders.sql`

## Local Development

Option A — Docker Compose (recommended):

1. `cd app/deploy`
2. `cp ../.env.example ../.env` and edit values
3. `docker compose up --build`

Services:
- Backend: `http://localhost:8080`
- Frontend: `http://localhost:3000`
- Redis: `redis://localhost:6379`

Option B — Run without Docker:

- Backend: `cd app/backend; go run ./cmd/server`
- Frontend: `cd app/frontend; npm install; npm run dev`

Ensure `NEXT_PUBLIC_API_URL=http://localhost:8080` is set for the frontend.

## Production Builds

- Backend Docker image: `app/backend/Dockerfile` (multi-stage, distroless runtime)
- Frontend Docker image: `app/frontend/Dockerfile` (Next.js standalone runtime)

## CI

GitHub Actions workflow at `.github/workflows/ci.yml` builds backend and frontend, runs vet/lint, and verifies production builds.

## Classroom Evaluation (Docker Only)

Evaluators: use Docker Compose only. Steps:

1. Ensure `app/.env` is filled (copy from `app/.env.example`).
2. Apply migrations in `app/backend/migrations/` to your Postgres (Supabase recommended).
3. From `app/deploy`, run:

	docker compose up --build

4. Open `http://localhost:3000`.

## Public Files Download

The frontend lists publicly shared files and provides per-file download buttons. Backend serves downloads at `GET /public/files/{fileID}/download`.

## Troubleshooting

- 404 on public download: ensure the file has a valid public share; backend only lists files with a valid token.
- CORS/CSRF: set `FRONTEND_URL` on backend to your frontend origin.
- Google OAuth: set `OAUTH_REDIRECT_URL` to `{BACKEND_URL}/auth/google/callback` if not using default.

## License

Educational use only. No license provided.
