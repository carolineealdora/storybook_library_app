# Plan: Analytics events for PRD §13 success metrics
**Status:** Proposed, not started · **Created:** 2026-11-16
**Note (2026-11-16, STORY-3):** the schema now lives in storybook_management_app. The `events` migration in Phase 1 must be added there, not in this repo.
**Spec:** [PRD §13](PRD_storybook-lib-app.md) · **Size:** about half a day of backend work plus frontend wiring in Lovable

## Goal
Every metric in PRD §13 can be answered with one SQL query, using the event names the PRD uses.

## Key decision: log only what the database doesn't already know
Half of the §13 events are already rows in the order and subscription tables. Writing them again as events would create a second copy of the financial data that could disagree with the first. So there are two sources:

| Source | PRD events | Why |
|---|---|---|
| **Derived** from existing tables (no new writes) | `purchase_completed` / `book_payment_succeeded`, `subscription_started` / `subscription_payment_succeeded`, `subscription_renewed`, `subscription_cancelled`, `payment_failed`, `checkout_started` | `book_orders`, `subscription_orders`, `subscriptions` already record who, what, how much and when |
| **Server-written** `events` rows | `pin_failed`, `pin_locked` | The PIN counter resets, so failures are not kept anywhere |
| **Client-sent** via `POST /events` | `book_detail_viewed`, `subscription_page_viewed`, `subscription_prompt_clicked`, `book_opened`, `book_finished`, `view_mode_switched` | Only the app sees screens, taps, the reader and Child view |

A single view, `analytics_events`, combines all three into one stream `(user_id, name, occurred_at, properties)`, so analysts never need to know which source a row came from.

## Phases

### Phase 1: Schema (migration `…_create_events.sql`)
```sql
CREATE TABLE events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- the client sends its own id for retries
    user_id     UUID NOT NULL REFERENCES users (id),
    name        VARCHAR(40) NOT NULL CHECK (name IN ('pin_failed','pin_locked','book_detail_viewed',
                  'subscription_page_viewed','subscription_prompt_clicked','book_opened','book_finished','view_mode_switched')),
    properties  JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX events_name_time_idx ON events (name, occurred_at);
CREATE INDEX events_user_time_idx ON events (user_id, occurred_at);

ALTER TABLE book_orders         ADD COLUMN failed_at TIMESTAMPTZ;  -- payment_failed needs a time
ALTER TABLE subscription_orders ADD COLUMN failed_at TIMESTAMPTZ;

CREATE VIEW analytics_events AS
  SELECT user_id, name, occurred_at, properties FROM events
  UNION ALL SELECT user_id, 'checkout_started', created_at, jsonb_build_object('kind','book','book_id',book_id,'price',price) FROM book_orders
  UNION ALL SELECT user_id, 'purchase_completed', paid_at, jsonb_build_object('book_id',book_id,'price',price) FROM book_orders WHERE status = 'paid'
  UNION ALL SELECT user_id, 'payment_failed', failed_at, jsonb_build_object('kind','book','price',price) FROM book_orders WHERE status = 'failed'
  -- … same three for subscription_orders (order_type new → subscription_started, renewal → subscription_renewed)
  UNION ALL SELECT user_id, 'subscription_cancelled', cancelled_at, '{}' FROM subscriptions WHERE status = 'cancelled';
```
`*_payment_succeeded` are aliases of `purchase_completed` and `subscription_started`/`subscription_renewed`. See open question 1.

### Phase 2: Server-written PIN events
- `account.attempt()`: insert `pin_failed` on a wrong PIN and `pin_locked` when the lock triggers, in the same statement path as `RecordFailure`. Properties: `{"via": "pin" | "password"}`.
- Order services: set `failed_at` when marking an order failed. That's one line in each `Pay`.

### Phase 3: `POST /api/v1/events` (parent token; also allowed in Child view, so no `X-Parent-Token`)
```jsonc
{ "events": [ { "id": "uuid", "name": "book_opened", "occurred_at": "RFC3339",
                "properties": { "book_id": "uuid", "view_mode": "child" } } ] }
→ { "data": { "accepted": 1 } }
```
- Up to 50 events per call. `properties` ≤ 2 KB and flat (string/number/bool values only).
- If any name is not in the client list, the whole batch is rejected with `400 VALIDATION_ERROR`.
- `user_id` always comes from the token, never the body.
- `occurred_at` outside [now − 7 days, now + 5 min] is replaced with the server time (protects against a wrong device clock).
- `INSERT … ON CONFLICT (id) DO NOTHING`, so a client retry never double-counts.
- Properties per event:

| Event | When the app sends it | Properties |
|---|---|---|
| `book_detail_viewed` | Book Detail opens | `book_id`, `access` |
| `subscription_page_viewed` | Subscription page opens | — |
| `subscription_prompt_clicked` | Subscription prompt tapped on a locked book | `book_id` |
| `book_opened` | Reader shows the first page | `book_id`, `view_mode` |
| `book_finished` | Reader reaches the last page | `book_id`, `view_mode` |
| `view_mode_switched` | Toggle succeeds | `to` (`parent` / `child`) |

### Phase 4: Metric queries (`docs/metrics.sql`)
One commented query per row of the §13 table, all over `analytics_events`, for example the north star:
```sql
-- Books read per active family per week (active family = ≥1 book_opened that week)
SELECT date_trunc('week', occurred_at) AS week,
       count(*) FILTER (WHERE name = 'book_opened')::numeric / count(DISTINCT user_id) AS books_per_family
FROM analytics_events WHERE name = 'book_opened' GROUP BY 1 ORDER BY 1;
```

### Phase 5: Docs and frontend
- Add `POST /events` and the event table above to `docs/API-contracts.md`, for Lovable.
- Lovable: a small `track(name, props)` helper that queues events, flushes every 10 s or on `visibilitychange`, and generates ids with `crypto.randomUUID()`.

## Tests (TDD, written first)
- Integration, `/events`: a valid batch returns accepted; replaying the same ids changes no counts; an unknown name returns 400 and inserts nothing; 51 events returns 400; oversized properties returns 400; an admin token returns 403; the user comes from the token even if the body names another user.
- Integration, view: after the existing checkout story, `analytics_events` contains `checkout_started`, `purchase_completed` and `payment_failed` (with `failed_at`) and `subscription_cancelled`, with the right users.
- Integration, PIN: after the existing PIN story, `pin_failed` = 4 rows and `pin_locked` = 1 row.
- `docs/metrics.sql`: every query runs without error against the test DB, as a smoke check.

## Files
- **Touched:** `migrations/` (new file, listed in `migration.go`), `internal/repositories/library/` (new `events.go`; `failed_at` in the order status setters), `internal/services/account/service.go`, `internal/services/order`, `internal/services/subscription`, new `internal/services/event` + `handler/controllers/event`, `handler/route.go`, `models/dto/storybook.go`, `handler/integration_test.go`, `docs/API-contracts.md`, new `docs/metrics.sql`.
- **Not touched:** auth and token code, access rules, the payment logic itself (only `failed_at` is added).

## Risks
- **Client events can be faked or lost.** That's acceptable: they only feed engagement and funnel ratios. Revenue numbers come from the order tables, never from client events.
- **Volume.** Expect about 10 events per family per day. Postgres handles this for years. Revisit (partitioning or an external tool) above ~10M rows.
- **Privacy.** Children are not identified separately, and there is no free text in properties. Add a retention job (e.g. delete events older than 13 months) before launch.
- **`book_finished` needs the EPUB reader to detect the last page.** Check this in Lovable first.

## Open questions (need an answer before Phase 1)
1. The PRD uses two names for the same fact (`purchase_completed` vs `book_payment_succeeded`, `subscription_started` vs `subscription_payment_succeeded`). Should we keep one name each and update §13? Recommended: yes.
2. What does "active family" mean? ≥1 `book_opened` in the week (assumed above), or ≥1 login?
3. Where will metrics be read? Plain SQL (this plan), or a BI tool such as Metabase on a read replica (later, no code change)?
4. Should failed Forgot PIN passwords count toward the safety metric (`via: password`), or only the PIN?
