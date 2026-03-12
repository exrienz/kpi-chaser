# KPI Journal

KPI Journal is a self-hosted full-stack app for turning daily work logs into KPI-ready quarterly reports. It is designed for lightweight edge deployments and includes:

- a Go API for auth, KPI management, achievement logging, dashboard summary, and report generation
- a Go worker for asynchronous AI enhancement and KPI mapping
- a Next.js frontend for daily entry, KPI tracking, analytics, and report export
- SQLite storage and Docker-based local deployment

## Features

- Email/password authentication with JWT
- Auth-gated dashboard access with login/register entry flow
- KPI create, edit, delete, and quarterly scoping
- Achievement logging, manual KPI tagging, retagging, and delete flow
- Async AI enhancement with OpenRouter-compatible APIs
- KPI hierarchy grouped by quarter with sub-KPI visibility and progress percentages
- Password-confirmed reset workflow for clearing user progress data
- Quarterly report generation with review, Notion, and HR-friendly export formats
- Yearly checkpoint view that aggregates quarterly report data

## Repository Layout

```text
cmd/                  Go entrypoints (`api-server`, `worker`)
internal/             Backend modules and services
migrations/           SQL schema files
docker/               Backend container build
web/                  Next.js frontend
blueprint.txt         System blueprint
requirement.txt       Product requirement source
```

## Prerequisites

- Node.js 22+
- npm 10+
- Go 1.22+ for local backend runs, or Docker for containerized startup

## Quick Start

1. Copy environment defaults:

```bash
cp .env.example .env
```

2. Install frontend dependencies:

```bash
cd web
npm install
cd ..
```

3. Start the stack.

With local runtimes:

```bash
go run ./cmd/api-server
go run ./cmd/worker
cd web && npm run dev
```

With Docker:

```bash
docker compose up --build
```

Frontend: `http://localhost:3000`
API: `http://localhost:8080`

If `OPENROUTER_API_KEY` is empty, the worker uses a deterministic fallback enhancer so the app still works in development.

## Development Commands

```bash
cd web && npm run dev
cd web && npm test
cd web && npm run build
go run ./cmd/api-server
go run ./cmd/worker
docker compose up --build
```

## Environment Variables

| Variable | Purpose | Default |
| --- | --- | --- |
| `HTTP_ADDRESS` | API listen address | `:8080` |
| `DATABASE_PATH` | SQLite file path | `./kpi-journal.db` |
| `JWT_SECRET` | JWT signing secret | `change-me` |
| `OPENROUTER_API_KEY` | Enables live AI calls | empty |
| `OPENROUTER_BASE_URL` | OpenRouter-compatible endpoint | `https://openrouter.ai/api/v1` |
| `OPENROUTER_MODEL` | AI model name | `openai/gpt-4o-mini` |
| `NEXT_PUBLIC_API_BASE_URL` | Frontend API base URL (skips proxy when set) | unset |
| `NEXT_PUBLIC_NEW_DASHBOARD_ENABLED` | Enables the redesigned dashboard UI. Set to `false` to roll back to the legacy dashboard. | `true` |
| `API_PROXY_TARGET` | Next.js proxy target for `/api/*` | `http://localhost:8080` |

## Dashboard Architecture

- The default frontend entrypoint is the redesigned dashboard, gated behind `NEXT_PUBLIC_NEW_DASHBOARD_ENABLED`.
- If the flag is set to `false`, the app renders the legacy dashboard without changing stored data.
- The new dashboard fetches KPI hierarchy and achievements only after successful authentication, then groups them by `YYYY-Qn` for quarter and yearly checkpoint views.

## Authentication and Dashboard Access

- Unauthenticated users are shown the login/register gate and cannot load dashboard data.
- Existing email/password accounts remain valid; authentication is still JWT-based and backward compatible with current users.
- The backend logs login success/failure and basic dashboard/report access events.

## Reset Flow

- `Reset all progress` now opens a protected confirmation modal in the frontend.
- The server requires both the literal confirmation text `RESET` and the user password before it clears progress data.
- The reset endpoint also requires the `X-Confirm-Action: reset-progress` header, which adds protection for this destructive action beyond a simple form post.
- Reset clears KPI progress snapshots and deletes the signed-in user's achievements, reports, and queued jobs.

## Reporting Views

- Quarter view groups root KPIs under each quarter and nests sub-KPIs under their parent KPI.
- Yearly checkpoint view presents quarter tabs for the selected year and renders KPI/sub-KPI report content under each parent KPI.

## Testing and Verification

Frontend checks currently available in this repository:

```bash
cd web && npm test
cd web && npm run build
```

Backend checks:

```bash
go test ./...
```

The backend source is scaffolded and Docker-ready. If Go is installed locally, run both Go services directly to verify the API and worker workflow.

## GitHub Push Checklist

Before pushing:

```bash
git status
cd web && npm test && npm run build
git add .
git commit -m "feat: initial KPI Journal full-stack app"
git push origin <branch-name>
```

This repository now ignores generated frontend artifacts, local databases, and environment files so they do not get committed accidentally.
