# KPI Journal

KPI Journal is a self-hosted full-stack app for turning daily work logs into KPI-ready quarterly reports. It is designed for lightweight edge deployments and includes:

- a Go API for auth, KPI management, achievement logging, dashboard summary, and report generation
- a Go worker for asynchronous AI enhancement and KPI mapping
- a Next.js frontend for daily entry, KPI tracking, analytics, and report export
- SQLite storage and Docker-based local deployment

## Features

- Email/password authentication with JWT
- KPI create, edit, delete, and quarterly scoping
- Achievement logging, manual KPI tagging, retagging, and delete flow
- Async AI enhancement with OpenRouter-compatible APIs
- KPI progress snapshot and quarterly dashboard metrics
- Quarterly report generation with review, Notion, and HR-friendly export formats

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
| `API_PROXY_TARGET` | Next.js proxy target for `/api/*` | `http://localhost:8080` |

## Testing and Verification

Frontend checks currently available in this repository:

```bash
cd web && npm test
cd web && npm run build
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
