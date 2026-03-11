# KPI Journal

KPI Journal is a self-hosted multi-user work log and KPI reporting app optimized for lightweight edge deployments. The repository now includes:

- a Go API server with email/password auth, SQLite storage, KPI CRUD, achievement logging, background AI enhancement jobs, and quarterly report generation
- a Go worker process for async AI enrichment via OpenRouter-compatible APIs with a deterministic fallback provider
- a Next.js frontend for authentication, KPI entry, achievement capture, and report generation
- Docker definitions for local full-stack startup

## Local development

Backend prerequisites:

```bash
go run ./cmd/api-server
go run ./cmd/worker
```

Frontend:

```bash
cd web
npm install
npm run dev
```

Docker:

```bash
docker compose up --build
```

Set `OPENROUTER_API_KEY` to enable live AI rewriting. Without it, the worker uses a local fallback enhancement so the workflow still functions in development.
# kpi-chaser
# kpi-chaser
# kpi-chaser
