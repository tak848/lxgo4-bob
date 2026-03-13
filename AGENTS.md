# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Multi-tenant task management system for [layerx.go #4](https://layerx.connpass.com/event/383847/) demo.
Bob ORM + PostgreSQL 17 + ogen (OpenAPI) + Next.js frontend.

## Build & Run Commands

```bash
# Toolchain (mise manages go 1.26.1, node 24.14.0, pnpm 10.32.1)
mise trust && mise install

# All tasks are defined in mise.toml [tasks] section
mise run up                   # docker compose up -d
mise run migrate              # dbmate migrations
mise run seed                 # sample data
mise run server               # start API server (:8080 default)
mise run dev                  # Next.js dev server (:3001 default)

# Code generation (requires running DB)
mise run bobgen               # bob ORM code generation
mise run ogen                 # ogen server code generation
mise run generate:api         # openapi-typescript type generation

# Direct commands
go build ./...                # compile check
go vet ./...                  # lint
cd webapp && pnpm type-check  # tsc --noEmit
cd webapp && pnpm lint        # next lint
```

## Architecture

### Backend Layers

```
HTTP Request
  → internal/oas/        (ogen generated server, DO NOT EDIT)
  → internal/handler/    (ogen Handler impl, DTO conversion)
  → internal/service/    (business logic, direct bob usage, NO repository pattern)
  → internal/infra/dbgen/ (bob generated models/where/joins/loaders, DO NOT EDIT)
  → internal/infra/hook/ (QueryHooks for tenant filtering)
  → internal/infra/db/   (DB connection, WorkspaceScopedExec/GlobalExec)
  → queries/             (sqlc-style hand-written SQL + bob generated code)
```

### Multi-tenant Design

- Tenant key is `workspace_id` (NOT tenant_id or organization_id)
- **QueryHooks** auto-inject `WHERE workspace_id = $1` on SELECT/UPDATE/DELETE for members, projects, tasks, task_comments
- Hooks do NOT apply to INSERT — Service layer must explicitly set `workspace_id` from path parameter
- Hooks do NOT apply to queries plugin output — SQL must contain explicit `WHERE workspace_id = $1`
- `WorkspaceScopedExec(ctx, exec, wsID)` sets workspace_id in context for hooks
- `GlobalExec(exec)` skips scoping (used for workspaces table itself)
- Cross-workspace references prevented by composite FK: `UNIQUE(workspace_id, id)` + `FOREIGN KEY (workspace_id, xxx_id) REFERENCES xxx(workspace_id, id)`

### Bob Nullable Types

- `null.Val[T]` — model fields (NULLABLE columns), states: value / NULL
- `omit.Val[T]` — setter NOT NULL fields, states: value / unset (omit)
- `omitnull.Val[T]` — setter NULLABLE fields, states: value / NULL / unset
- To set NULL explicitly: `var v omitnull.Val[T]; v.Null(); setter.Field = v`

### Generated Code (DO NOT EDIT)

- `internal/infra/dbgen/` — bob models, where, joins, loaders, factory, enums
- `internal/oas/` — ogen server code
- `queries/reports.bob.go`, `queries/reports.bob.sql` — queries plugin output
- `dberrors/` — bob error helpers

Regenerate after schema changes: `mise run bobgen`, `mise run ogen`

### Frontend

- Next.js 15 App Router + React Compiler enabled
- `webapp/src/app/` — all pages are Client Components
- `webapp/src/lib/api/client.ts` — openapi-fetch typed client
- `webapp/src/lib/api/schema.d.ts` — generated from `api/openapi.yaml` (gitignored)
- shadcn/ui components in `webapp/src/components/ui/`

### Environment & Ports

All ports configurable via `mise.toml` env vars. Override locally with `mise.local.toml` (gitignored).
Default ports: API=8080, Webapp=3001, PostgreSQL=5432, Grafana=3000, Loki=3100.
`PSQL_DSN` env var overrides bobgen.yaml DSN (official bob feature via koanf).

### Key Pitfall: Factory Plugin

Bob's factory `Create()` auto-generates relationships and **overwrites** `workspace_id`/`assignee_id` etc.
For seed data or tests requiring specific FK values, use direct `dbgen.Xxx.Insert(&dbgen.XxxSetter{...})` instead.

## Review

- レビューは日本語で行うこと

## File Conventions

- All files must end with a trailing newline
- UUIDv7 generated app-side via `uuid.NewV7()` (no DB DEFAULT)
- `cmd/server/main.go` — logs to both stdout and `logs/server.log` (promtail reads the file)
