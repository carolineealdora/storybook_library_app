# Architectural Decisions — Storybook Library backend
**Last updated:** 2026-11-16

Short ADRs. Never delete one: supersede it. Status is `Accepted`, `Proposed`, or `Superseded by ADR-N`.
Product decisions (pricing, subscription rules, PIN gate) live in [PRD §11](PRD_storybook-lib-app.md).

| ADR | Title | Status |
|---|---|---|
| 001 | Postgres only; Redis, Mongo and the scaffold's payment sample removed | Accepted |
| 002 | One repository package for all storybook tables | Accepted |
| 003 | EPUB and covers stored in Postgres | Accepted |
| 004 | HS256 access JWT + opaque refresh token in Postgres | Accepted |
| 005 | Mocked payment: two steps, client picks the outcome | Accepted |
| 006 | "Active subscription" means `status = 'active'`; hourly in-process renewal | Accepted |
| 007 | Parent token proves Parent view on payment and settings routes | Accepted |
| 008 | Offset pagination with `meta.total` | Accepted |
| 009 | 404 for other users' orders | Accepted |
| 010 | CR-001: badge and plan price changes fit the existing model | Accepted |

---

### ADR-001 — Postgres only
**Decision:** Remove Redis, Mongo, the `x-partner-id` header and the sample payment module from the scaffold.
**Why:** The PRD architecture has a single Postgres. The scaffold's body-dump logger wrote passwords and tokens to logs.
**Consequence:** Locks and idempotency use Postgres row locks and unique constraints.

### ADR-002 — One repository package
**Decision:** All storybook SQL lives in `internal/repositories/library`, with `Repo.InTx`.
**Why:** Paying an order writes orders, purchases and subscriptions in one transaction.
**Alternative:** One repo per table plus a unit-of-work. That's more code for the same result.

### ADR-003 — Files in Postgres
**Decision:** EPUB goes in `books.epub_content`; covers go in `book_files`. List queries never select the bytes.
**Why:** It's a PRD §11 trade-off: one store and simple backups for a small curated catalogue.
**Consequence:** Whole files are loaded into memory per request (TD-004). Move to object storage when the catalogue grows.

### ADR-004 — Tokens
**Decision:** Access JWT HS256 (30 min). The refresh token is a random opaque string, stored as SHA-256 and valid 7 days. It is not rotated (PRD US-01).
**Why:** Logout can revoke the refresh token without Redis. HS256 is enough with a single issuer and verifier.
**Consequence:** An access token stays valid up to 30 min after logout (TD-007).

### ADR-005 — Mocked payment
**Decision:** Create an order (`pending`, price snapshotted, idempotency key unique per user), then call `/pay` with `result: success|failed`. Retrying after a failure needs a new order.
**Why:** The PRD mocks the payment gateway but asks for testable failures and protection against double payment.
**Consequence:** A real gateway would replace `/pay` with a webhook. The order and lock logic stays.

### ADR-006 — Subscription state
**Decision:** Access checks `status = 'active'`, not `expires_at`. `RunRenewals` runs hourly in the API process and is idempotent.
**Why:** Renewal always succeeds (PRD), so access never lapses between expiry and the next run.
**Consequence:** No retries or alerts beyond logs (TD-005).

### ADR-007 — Parent token (resolves TD-002)
**Decision:** Login, a correct PIN, or a correct password on Forgot PIN return a `parent_token` (JWT with role `parent_mode`, 7 days). Create/pay orders, subscribe, cancel, `PATCH /me` and `PUT /me/pin` require it in `X-Parent-Token`. Refresh never issues one.
**Why:** Without it the PIN only gated the UI, and a child holding the session could call payment endpoints.
**Alternatives:** A short TTL ("sudo mode") forces a PIN on every app reopen, which goes against PRD US-03 (view kept after reopen) and the PRD §11 decision to put the PIN on the toggle, not at checkout. Storing the view server-side adds a table plus a call on every toggle.
**Consequence:** The client must drop the token on switching to Child view, and treat `403 PARENT_MODE_REQUIRED` as "show the PIN prompt". Read-only history stays reachable without it.

### ADR-008 — Pagination
**Decision:** `?page=&page_size=` (default 20, max 100) returns `meta {total, page, page_size}`. Orders are stable (`…, id`).
**Why:** Lovable list UIs need page numbers and totals. The catalogue is small, so `count(*)` is cheap.
**Alternative:** Keyset paging. Revisit if a list grows past ~10k rows.

### ADR-009 — Other users' resources
**Decision:** Paying someone else's order returns `404 ORDER_NOT_FOUND`, not 403.
**Why:** It doesn't reveal that the order exists.

### ADR-010 — CR-001 on the existing subscription model
**Decision:** The badge (`plan_status`) uses the same rule as book access (`status = 'active'`). A plan price change applies at the next **automatic** renewal; there is no user-triggered renew endpoint. Each renewal stays a new period row (no `renewed_at`), duration stays one calendar month (no `duration_days`), the stale-price check runs when the subscription order is created (`expected_price`), and existing endpoint paths are kept. History foreign keys point at `admins`, not `users`.
**Why:** The PRD requires auto-renew and one history row per period (US-10, US-13); a manual renew while active would charge the new price before `expires_at` (against CR-001 P2); the badge must not say "Free" while books are still unlocked; the frontend contract already uses these paths.
**Consequence:** CR-001 E-CR5 / T10 (`NO_ACTIVE_SUBSCRIPTION`) do not apply. `is_subscriber` stays next to `plan_status` for older clients.

### ADR-011 — Parent-only backend
**Decision:** This backend serves only `fe_storybook_library_app`. The admin routes and code (login, book CRUD, plan pricing, price history) were removed; storybook_management_app serves the back office and owns the shared schema. `/auth/refresh` returns `401 UNAUTHORIZED` for a non-parent credential, because the `refresh_tokens` table is shared. Use a different `JWT_SECRET` from the back office. The hourly renewal job runs only here.
**Why:** the back office already called only its own backend, but the duplicate routes stayed reachable and the two copies of each rule were drifting. In the other direction, the back office's parent routes let a parent skip this app's parent-mode token and Forgot PIN lockout; those routes are gone too.
**Consequence:** the integration suite creates books and plan-price changes with SQL fixtures. Cross-app flows are not covered by any automated test (management repo TD-14).
