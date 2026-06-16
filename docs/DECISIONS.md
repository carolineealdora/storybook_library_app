# Architectural Decisions
**Last updated:** [date]

This document records every significant architectural decision made for this platform using the ADR (Architectural Decision Record) format. Each decision captures what was decided, why, what alternatives were considered, and the consequences.

**Rules:**
- Add an ADR for every decision that affects data models, system boundaries, tech choices, or integration patterns
- Never delete an ADR — supersede it with a new one
- Status must be one of: `Proposed` · `Accepted` · `Deprecated` · `Superseded by ADR-[N]`

---

## Index

| ADR | Title | Status |
|---|---|---|
| [ADR-001](#adr-001) | Multi-location model — shared DB with location_id | Accepted |
| [ADR-002](#adr-002) | Court slot availability via Redis | Accepted |
| [ADR-003](#adr-003) | Payment confirmation via server-side webhook only | Accepted |
| [ADR-004](#adr-004) | JWT RS256 + Redis-stored refresh tokens | Accepted |
| [ADR-005](#adr-005) | Soft deletes on all financial and audit tables | Accepted |
| [ADR-006](#adr-006) | Money stored as INTEGER (sen) | Accepted |
| [ADR-007](#adr-007) | Background jobs via goroutines + Redis queue (Phase 1) | Accepted |
| [ADR-008](#adr-008) | PostgreSQL enums as VARCHAR + CHECK constraint | Accepted |
| [ADR-009](#adr-009) | ErrNotFound on cross-member resource access | Accepted |

---

<a name="adr-001"></a>
## ADR-001 — Multi-location model: shared database with location_id

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
All locations share a single PostgreSQL database. Every entity that belongs to a location carries a `location_id` UUID foreign key. New locations are added via admin configuration — no schema change, no code change.

### Context
The platform must support multiple physical club locations (current: one; roadmap: franchise expansion). Two structural options were evaluated.

### Options considered

| Option | Pros | Cons |
|---|---|---|
| Separate DB per location | Strict data isolation; simpler per-location queries | Cross-location reporting requires federation; N migration runs per deploy; ops complexity scales with location count |
| **Shared DB with location_id** ✅ | Single migration path; cross-location BI in one query; simpler ops | All queries must filter by location_id; slightly more complex row-level security |

### Consequences
- Every table that is location-scoped must have a `location_id` column and FK
- Every query on location-scoped data must include `WHERE location_id = $1` (enforced in repository layer)
- Cross-location reporting is a single query with `GROUP BY location_id`
- If true data isolation becomes a hard requirement (e.g. franchise ownership changes), migrate to schema-per-location or DB-per-location at that point — not before

---

<a name="adr-002"></a>
## ADR-002 — Court slot availability via Redis

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
Court slot availability is served from Redis, not from a live PostgreSQL query. Slots are stored as keys with the pattern `slot:booked:{court_id}:{date}:{slot_start_unix}`. Holds use the same key pattern with a 300-second TTL.

### Context
The availability endpoint is the highest-traffic read in the system. Under concurrent booking load (e.g. a popular Friday morning slot), multiple members hit availability simultaneously. A live DB query would be slow and create race conditions on the same slot.

### Options considered

| Option | Pros | Cons |
|---|---|---|
| PostgreSQL query with row-level locking | Single source of truth; no sync needed | Slow under concurrent load; lock contention on popular slots |
| **Redis with DB as source of truth** ✅ | Sub-10ms reads; TTL natively handles hold expiry; atomic SET NX prevents double-booking | Two sources of truth; must keep Redis in sync on booking confirmation and cancellation |

### Consequences
- Booking confirmation (from Xendit webhook) must update both PostgreSQL and Redis
- Cancellation must delete the Redis slot key
- If Redis is unavailable, availability falls back to a PostgreSQL query (degrade gracefully — do not fail)
- Hold TTL is configurable via `SLOT_HOLD_TTL` environment variable (default: 300s)

---

<a name="adr-003"></a>
## ADR-003 — Payment confirmation via server-side webhook only

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
A booking is only confirmed — and a court slot only permanently reserved — after receiving a valid, HMAC-verified webhook from Xendit. Client-reported payment status is never trusted.

### Context
Payment flows involve a client redirect after Xendit processes the payment. The client could be tampered with, the network could drop, or the redirect could be spoofed. Trusting the client creates a vector for booking without paying.

### Consequences
- The POST /bookings endpoint creates a booking with `status: pending_payment` and returns payment instructions
- The slot remains in Redis as a "hold" (TTL) until the webhook fires
- If the webhook never fires (payment abandoned), the hold expires and the slot is released automatically
- The `/webhooks/xendit` endpoint must be idempotent — duplicate webhooks (Xendit retries) must not double-confirm or double-notify
- Idempotency is enforced via `UNIQUE` constraint on `bookings.xendit_payment_id`

---

<a name="adr-004"></a>
## ADR-004 — JWT RS256 + Redis-stored refresh tokens

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
Access tokens are short-lived JWTs signed with RS256 (15-minute TTL). Refresh tokens are opaque random strings stored in PostgreSQL (`refresh_tokens` table) with a 7-day TTL. Logout revokes the refresh token in the database — the access token becomes useless after expiry.

### Context
HS256 JWTs with long TTLs cannot be revoked — logout is fake. RS256 + server-side refresh token storage gives us real revocability without a per-request database hit on every API call.

### Options considered

| Option | Pros | Cons |
|---|---|---|
| HS256 long-lived JWT | Simple; no DB hit | No revocability; logout is cosmetic |
| Session tokens in DB | Fully revocable; no JWT complexity | DB hit on every request |
| **RS256 JWT + Redis refresh token** ✅ | Stateless access token (no DB hit per request); real logout via refresh token revocation | Two-token complexity; must rotate refresh token on use |

### Consequences
- Access token TTL is 15 minutes — clients must implement silent refresh
- Refresh token rotation: a new refresh token is issued on every `/auth/refresh` call; the old one is revoked
- If a refresh token is used twice, the second use fails — this signals token theft (log and alert)
- The public RS256 key must be available to all API instances (stored in environment config)

---

<a name="adr-005"></a>
## ADR-005 — Soft deletes on financial and audit tables

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
Tables with financial or audit significance use soft deletes: either a `deleted_at TIMESTAMPTZ` column or a `status` enum that includes a cancelled/voided state. Hard `DELETE` is never used on these tables.

Affected tables: `bookings`, `lounge_orders`, `lounge_order_items`, `payment_intents`, `members`, `attendance_records`.

### Context
Deleting a booking record removes the payment trail. Deleting a member record orphans transaction history. Regulatory and business audit requirements make hard deletes unacceptable for financial records.

### Consequences
- All repository queries on soft-deletable tables must include `WHERE deleted_at IS NULL` (or equivalent status filter)
- Cancellation sets `status = 'cancelled'` and `cancelled_at = NOW()` — never `DELETE`
- The "Resolved" section of tech-debt.md is the only place where we track genuinely removed tech debt — records in the DB are never removed

---

<a name="adr-006"></a>
## ADR-006 — Money stored as INTEGER (sen)

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
All monetary values are stored and processed as `INTEGER` in the smallest currency unit (Indonesian sen: 1 Rupiah = 100 sen). `Rp 150,000` is stored as `15000000`. Floating-point types are never used for money.

### Context
Floating-point arithmetic is imprecise. `0.1 + 0.2 ≠ 0.3` in IEEE 754. Financial rounding errors compound across transactions and create discrepancies in reports and reconciliation.

### Consequences
- All monetary columns are `INTEGER NOT NULL` in PostgreSQL
- All Go domain structs use `int` or `int64` for monetary fields
- Display formatting (Rp 150.000) is the responsibility of the frontend — the API returns raw integers
- The field name convention is `[name]_idr` (e.g. `total_amount_idr`, `price_idr`) to make the unit explicit

---

<a name="adr-007"></a>
## ADR-007 — Background jobs via goroutines + Redis queue (Phase 1)

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
Phase 1 background jobs (notification delivery, slot hold expiry) use Go goroutines with a Redis-backed job queue. No external job queue infrastructure (no RabbitMQ, no SQS) in Phase 1.

### Context
Phase 1 job volume is low (notifications on booking events, hold expiry). Over-engineering with a full message broker before load justifies it adds infrastructure complexity, cost, and ops burden.

### Consequences
- If job volume grows significantly (>1000 jobs/min sustained), migrate to a proper queue — this is documented as TD-[N] in tech-debt.md
- Jobs must be idempotent — if a worker crashes and the job is retried, running it twice must not cause double-notifications
- The migration path to a proper queue (Redis Streams → SQS → RabbitMQ) is straightforward — the job interface is abstracted behind a `JobQueue` interface in `internal/queue/`

---

<a name="adr-008"></a>
## ADR-008 — Enums as VARCHAR + CHECK constraint

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
Enumerated values (status fields, type fields) are stored as `VARCHAR(N)` with a `CHECK` constraint — not as PostgreSQL native `ENUM` types.

### Context
PostgreSQL native enums require an `ALTER TYPE` migration to add new values. This is a blocking DDL operation. VARCHAR + CHECK allows adding new values by simply updating the CHECK constraint in a migration, without locking the table.

### Consequences
- All CHECK constraints must be kept in sync with Go constants in `internal/domain/`
- Type safety is enforced at the application layer — the DB constraint is a last-resort catch
- Example: `CHECK (status IN ('pending_payment','confirmed','cancelled','no_show','refunded'))`

---

<a name="adr-009"></a>
## ADR-009 — ErrNotFound on cross-member resource access

**Status:** Accepted
**Date:** [date]
**Ticket:** —

### Decision
When a member requests a resource that belongs to a different member, the service returns `ErrNotFound` (HTTP 404) — not `ErrForbidden` (HTTP 403).

### Context
Returning 403 on a cross-member access attempt confirms that the resource exists. An attacker can enumerate booking IDs or member IDs by checking whether they get 403 (exists, not yours) vs 404 (not found). Returning 404 for both cases prevents existence leakage.

### Consequences
- Repository queries always include `WHERE id = $1 AND member_id = $2`
- If the record exists but belongs to another member, the query returns no rows — same result as not found
- This means 404 can mean either "doesn't exist" or "exists but not yours" — the client cannot tell the difference (by design)
- Admin endpoints are exempt — admins query without member_id filter and may see 403

---

## Template — adding a new ADR

Copy this block. Increment the ADR number. Update the Index table above.

```markdown
<a name="adr-[N]"></a>
## ADR-[N] — [Short title]

**Status:** Proposed | Accepted | Deprecated | Superseded by ADR-[N]
**Date:** [date]
**Ticket:** PROJ-[id] (or —)

### Decision
[One paragraph: what was decided.]

### Context
[Why was this decision needed? What problem does it solve?]

### Options considered

| Option | Pros | Cons |
|---|---|---|
| Option A | ... | ... |
| **Option B ✅** | ... | ... |

### Consequences
- [What becomes easier or harder as a result]
- [What constraints this places on future decisions]
- [What must be monitored or revisited]
```