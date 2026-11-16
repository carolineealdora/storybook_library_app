# Architecture Overview — Storybook Library backend
**Last updated:** 2026-11-16
**This is the map, not the manual.** Behaviour: [PRD](PRD_storybook-lib-app.md) · Endpoints: [API-contracts.md](API-contracts.md) · Tables: [DATA-MODELS.md](DATA-MODELS.md) · Why: [DECISIONS.md](DECISIONS.md)

---

## System purpose

This Go API is called by the **Digital Storybook Library** frontend (`fe_storybook_library_app`). It also still implements the admin endpoints below, but the Book Management frontend (`fe_storybook_management_app`) calls storybook_management_app, which owns the shared schema:

- **Digital Storybook Library** (parent account): browse the showcase, buy single books, subscribe monthly, read EPUBs. Has a Parent view and a Child view; switching from Child to Parent view needs a PIN.
- **Book Management** (admin account): book CRUD with EPUB and cover upload, subscription plan price and benefits.

Login and payment are mocked for the MVP (PRD §3).

---

## Tech stack

| Layer | Technology | Why |
|---|---|---|
| API | Go 1.21, Echo v4 | The scaffold we were given |
| Database | PostgreSQL 13+ (GORM, raw SQL) | Orders, purchases and subscriptions need transactions and constraints |
| Files | Postgres `bytea` (EPUB in `books`, covers in `book_files`) | One store for the MVP (PRD §11). See TD-004 |
| Auth | JWT HS256 + opaque refresh token in Postgres + parent token | Revocable sessions without Redis |
| Jobs | In-process hourly ticker (auto-renew) | One job, idempotent. See TD-005 |
| Frontends | Lovable (outside this repo) | — |

---

## System boundaries

```
┌──────────────────────────────┐   ┌──────────────────────────────┐
│ Library app (Lovable)        │   │ Book Management (Lovable)    │
│ parent token on Parent view  │   │ admin token                  │
└──────────────┬───────────────┘   └──────────────┬───────────────┘
               │ HTTPS / REST /api/v1             │
┌──────────────▼──────────────────────────────────▼───────────────┐
│ Go API                                                         │
│ handler/  → auth middleware, binding, JSON responses           │
│ internal/services/  → access rules, checkout, PIN, renewals    │
│ internal/repositories/library  → SQL, transactions             │
│ main.go → hourly RunRenewals goroutine                         │
└──────────────────────────────┬─────────────────────────────────┘
                               │
                     ┌─────────▼─────────┐
                     │ PostgreSQL        │
                     │ data + EPUB/cover │
                     └───────────────────┘
```

No external services. Payment is a client-chosen outcome (`result: success|failed`).

---

## Layers

| Layer | Path | Does | Never does |
|---|---|---|---|
| Handler | `handler/controllers/<domain>` | Bind and validate input, read claims, call one service, write `{data}` / `{error}` | Business rules, SQL |
| Service | `internal/services/<domain>` | Access rules, state transitions, error codes | HTTP, SQL text |
| Repository | `internal/repositories/library` | SQL, `InTx` transactions, row locks | Business decisions |

There is one repository package on purpose: paying an order touches orders, purchases and subscriptions in one transaction.

---

## Auth model

| Token | Lifetime | Where | Proves |
|---|---|---|---|
| Access (JWT) | 30 min | `Authorization: Bearer` | Who you are, and whether it's a parent or admin account |
| Refresh (opaque, SHA-256 in DB) | 7 days, not rotated | `POST /auth/refresh` body | Session; revoked on logout |
| Parent (JWT, role `parent_mode`) | 7 days | `X-Parent-Token` header | That this is Parent view. Issued on login, correct PIN and Forgot PIN. Refresh never issues one. |

Payment and settings routes need both the access token and the parent token. Child view drops the parent token, so a child holding the session cannot buy, subscribe, cancel or change settings (TD-002).

---

## Critical flows

**Single-book checkout (PRD US-09)**
```
POST /book-orders {book_id, idempotency_key}   → pending order, price snapshotted
POST /book-orders/:id/pay {result}             → in one transaction:
    lock order FOR UPDATE → re-check purchasable (published, not owned, not subscribed)
    → success: order paid + purchase row   |   failed: order failed, 402
```
A second pay on the same order returns 409 `DUPLICATE_REQUEST`. To retry after a failure, create a new order.

**Subscription**: same two steps (`/subscription-orders`). Only one active subscription per user, enforced by a partial unique index. Cancelling stops access immediately.

**Auto-renew (US-13)**: `RunRenewals` runs every hour. For each active period past `expires_at`, it expires the period with a conditional update, creates a new period from the old `expires_at`, and creates a paid renewal order with key `renewal:<old_id>`. It is safe to run twice or on two instances.

**Reading (US-07)**: `GET /books/:id/content` checks access in this order: free → owned → subscribed → `403 BOOK_LOCKED`. The EPUB bytes are sent only after that check passes.

---

## Where to look before changing…

- Access rules: `internal/services/book/service.go` (`Access`, `Visible`)
- Payment / orders: `internal/services/order`, `internal/services/subscription`, `internal/repositories/library/orders.go`
- Tokens and parent mode: `internal/services/auth`, `handler/controllers/auth/controller.go`
- PIN and Forgot PIN lockouts: `internal/services/account`
- Schema: `../storybook_management_app/migrations/*.sql` (authoritative, shared database), summarised in [DATA-MODELS.md](DATA-MODELS.md)
