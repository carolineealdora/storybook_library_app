# Backend Code Base

**Version:** v1.0

> **Engineers: before writing a single line of code, read [`engineering-with-claude-playbook.md`](engineering-with-claude-playbook.md).** It defines the mandatory workflow for using Claude Code on this project — plan mode, TDD, quality gates, and session management. This is not optional reading.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.21 |
| HTTP Framework | [Echo v4](https://echo.labstack.com/) |
| Database | PostgreSQL (via GORM) |
| Auth | JWT |
| Logging | Logrus + Elastic APM |

---

## Architecture

This codebase follows a **3-layer architecture** — every feature is implemented as a stack of three distinct layers, each with a single responsibility.

```
HTTP Request
     │
     ▼
┌─────────────────────────────────┐
│  Handler Layer                  │  handler/controllers/
│  Parses request, validates      │
│  input, calls service,          │
│  formats HTTP response          │
└────────────────┬────────────────┘
                 │ calls
                 ▼
┌─────────────────────────────────┐
│  Service Layer                  │  internal/services/
│  Contains business logic.       │
│  Orchestrates repository calls. │
│  No HTTP or DB concerns here.   │
└────────────────┬────────────────┘
                 │ calls
                 ▼
┌─────────────────────────────────┐
│  Repository Layer               │  internal/repositories/
│  Talks to the database.         │
│  No business logic here —       │
│  only data access (queries).    │
└─────────────────────────────────┘
```

**Rule:** Business logic only lives in the service layer. Handlers are thin HTTP adapters. Repositories are thin DB adapters. Nothing crosses these boundaries.

---

## Directory Structure

```
.
├── main.go                   # Entry point — boots server or runs migrations
├── handler/
│   ├── route.go              # Route registration
│   └── controllers/          # Handler layer — one package per domain
│       ├── auth/ account/ book/ order/ subscription/
│       └── healthcheck/
├── internal/
│   ├── services/             # Service layer — business logic
│   │   ├── auth/ account/ book/ order/ subscription/
│   │   └── healthcheck/
│   ├── repositories/         # Repository layer — DB access
│   │   ├── library/          # all storybook tables (shared for cross-table transactions)
│   │   └── healthcheck/
│   ├── helper/               # Shared internal utilities
│   └── constant/             # Shared constants
├── models/                   # Data structures, DTOs, SharedHolder
├── pkg/                      # Reusable infrastructure packages
│   ├── logger/               # Logger interface + Logrus implementation
│   ├── orm/                  # DB connection (GORM + Postgres)
│   ├── migrator/             # DB migration runner
│   ├── monitor/              # APM / observability
│   └── shared/               # Shared bootstrapping utilities
├── config/                   # Env var loading and config struct
├── migrations/               # SQL migrations + runner wiring
├── test/mock/                # Mocks for unit testing
├── docs/                     # Architecture docs, API contracts, PRD templates
├── .claude/rules/            # Claude Code behavioral rules
├── CLAUDE.md                 # Claude Code project instructions
├── PLAN.md                   # Active implementation plan
├── tech-debt.md              # Deferred work registry
└── engineering-with-claude-playbook.md   # ← READ THIS FIRST
```

---

## Running Locally

Needs Go 1.21+ and PostgreSQL 13+.

```bash
cp .env.example .env            # set DB_* and JWT_SECRET
createdb storybook
make migrate                     # schema + mock accounts + subscription plan
make run                         # API on :9999, auto-renew job runs hourly

# migrations
go run . migrate up
go run . migrate create <migration_name>   # then add it to migrations/migration.go
```

Mock logins: `parent@mainstory.test` / `Parent123!`, `admin@mainstory.test` / `Admin123!`.
API reference for the frontends: [`docs/API-contracts.md`](docs/API-contracts.md).

---

## Build for Production

```bash
go get -d -v ./...
go build -o /go/bin/app
```

---

## Run Tests

```bash
make test               # unit tests
createdb storybook_test
make test-integration   # end-to-end over HTTP against Postgres (wipes storybook_test)
```

---

## Quality Gate

Before committing anything, run:

```bash
make done
```

This runs lint, tests, and build in one command. **Never commit if it fails.**

---

## Environment Variables

Copy `.env.example` and fill in real values — never commit real credentials.

| Name | Description |
|------|-------------|
| `ENV` | Runtime environment (`development`, `production`) |
| `PORT` | HTTP port (default `9999`) |
| `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE` | PostgreSQL connection |
| `JWT_SECRET` | **Required.** HMAC key for access tokens |
| `CORS_ALLOW_ORIGINS` | Comma-separated browser origins (default `*`) |

---

## Port

| Port | Description |
|------|-------------|
| `9999` | API service |

---

## docs/

```
docs/
├── PRD_storybook-lib-app.md    # Product spec: source of truth for behaviour
├── API-contracts.md            # Endpoints and shapes, written for the Lovable frontends
├── ARCHITECTURE.md             # Layers, auth model, critical flows
├── DATA-MODELS.md              # Tables and rules (the SQL in migrations/ is authoritative)
├── DECISIONS.md                # ADR log
└── PLAN-analytics-events.md    # Proposed plan for PRD §13 metrics
```

The `.claude/rules/` files are Claude Code's working rules for this repo. Adjust them as the project evolves.

---

## Claude Code Setup (Required for AI-assisted Development)

This project uses Claude Code as a core development tool. Install the following two plugins **once** before your first session — they apply to every session after that.

### caveman — output compression

Cuts ~65–75% of Claude's output tokens by compressing prose while keeping all code, errors, and technical detail intact. Reduces noise and cost across every session.

```bash
curl -fsSL https://raw.githubusercontent.com/JuliusBrussee/caveman/main/install.sh | bash
```

Mid-session controls (type inside Claude Code chat):

| Command | Effect |
|---------|--------|
| `/caveman lite` | Light compression |
| `/caveman` | Default compression |
| `/caveman ultra` | Maximum compression |
| `normal mode` | Disable compression |

### ponytail — minimal-code discipline

Enforces "laziest senior dev" coding: stdlib before third-party, no unasked-for abstractions, deletion over addition. Keeps Claude from over-engineering.

Inside a Claude Code session, run these commands in order:

```
/plugin marketplace add DietrichGebert/ponytail
/plugin install ponytail@ponytail
```

Then go to `/hooks`, trust the two lifecycle hooks, and start a new thread.

Mid-session controls:

| Command | Effect |
|---------|--------|
| `/ponytail-review` | Scans current diff for over-engineering, returns a delete-list |
| `/ponytail ultra` | Questions whether the task needs to exist at all |

> Both tools are Claude Code plugins — they operate at the AI layer, not the code layer. They are not Go libraries and do not affect the compiled binary.

---

## Engineering Workflow

This project uses Claude Code as a core development tool. The full workflow — including plan mode, TDD, quality gates, multi-session management, and context hygiene — is documented in:

**[`engineering-with-claude-playbook.md`](engineering-with-claude-playbook.md)**

Every engineer working on this codebase is expected to read it before starting any implementation work.
