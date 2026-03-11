# Repository Guidelines

## Project Structure & Module Organization
This repository is currently specification-first. The active source files are [blueprint.txt](/config/Desktop/VibeCoding/kpi-chaser/blueprint.txt) and [requirement.txt](/config/Desktop/VibeCoding/kpi-chaser/requirement.txt), which define the KPI Journal architecture and MVP scope.

The planned application layout is:
- `cmd/api-server` and `cmd/worker` for Go entrypoints
- `internal/` for domain modules such as `auth`, `kpi`, `achievements`, `ai`, and `reports`
- `pkg/` for reusable helpers
- `web/` for the frontend app and shared UI components
- `migrations/` for SQL schema changes
- `docker/` plus `docker-compose.yml` for container deployment

Keep new files aligned to that structure instead of adding top-level directories ad hoc.

## Build, Test, and Development Commands
No executable app scaffold is committed yet, so there are no working `make`, `npm`, or `go` tasks in the repository today. When bootstrapping the project, prefer adding explicit scripts such as:
- `go test ./...` for backend tests
- `go run ./cmd/api-server` for the API
- `go run ./cmd/worker` for async jobs
- `npm run dev` from `web/` for the frontend
- `docker compose up --build` for full local integration

Document any new command in `README.md` and keep local/dev commands reproducible.

## Coding Style & Naming Conventions
Follow idiomatic Go for backend code: tabs for indentation, short package names, and lowercase module directories such as `internal/auth` and `internal/ai`. Use `gofmt` and `go test` before submitting changes.

For the planned frontend, use TypeScript, 2-space indentation, and PascalCase for components such as `KpiCard.tsx`. Keep route, API, and migration names descriptive, for example `001_users.sql`.

## Testing Guidelines
Add tests with each functional change. Place Go tests beside the code as `*_test.go`; mirror frontend component tests under `web/` with `*.test.ts(x)`. Favor table-driven tests for service and repository logic, plus integration coverage for SQLite migrations and AI job flows.

If a change cannot be tested yet because the scaffold is incomplete, state that clearly in the pull request.

## Commit & Pull Request Guidelines
This directory is not currently a Git repository, so there is no local commit history to infer conventions from. Until history exists, use short imperative commits, preferably Conventional Commits such as `feat: add achievement service skeleton` or `docs: refine deployment notes`.

Pull requests should include:
- a concise summary of behavior changed
- links to the relevant requirement or blueprint section
- test evidence or an explicit note that testing is pending
- screenshots only for UI-visible changes
