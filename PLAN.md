# Plan: Storybook Library backend
**Status:** Backend MVP done and hardened (TD-002/003/006); frontends (Lovable) next
**Last updated:** 2026-11-16
**Spec:** [docs/PRD_storybook-lib-app.md](docs/PRD_storybook-lib-app.md) · **API:** [docs/API-contracts.md](docs/API-contracts.md)

## Phases
- [x] Phase 1: Schema + seed migrations (PRD §12.2)
- [x] Phase 2: Auth: mocked login, JWT 30 min, refresh 7 days, logout, role guard
- [x] Phase 3: Library: showcase, detail, cover, EPUB content gate, My Book, PIN, settings
- [x] Phase 4: Single-book checkout + mocked payment + order history
- [x] Phase 5: Subscription: subscribe, cancel, history, hourly auto-renew
- [x] Phase 6: Back office: book CRUD (multipart, files in DB), plan config
- [x] Phase 7: Tests: unit (rules) + integration (all PRD flows / edge cases, 88% coverage)
- [x] Phase 8: Hardening: parent token on payment/settings routes (TD-002), pagination (TD-003), Forgot PIN lockout (TD-006), project docs
- [ ] Phase 9: Analytics events for PRD §13: [docs/PLAN-analytics-events.md](docs/PLAN-analytics-events.md) (proposed, 4 open questions)

## Decisions made — do not revisit
- One repository package (`internal/repositories/library`) so payment transactions can span orders, purchases and subscriptions.
- Redis, Mongo and the `x-partner-id` header were removed from the scaffold: not in the PRD architecture, and the body-dump logger stored passwords and tokens.
- Mocked payment: the client sends `result: success|failed`.
- Retrying after a failed payment creates a new order. Failed orders stay in Order History.
- "Active subscription" means `status = 'active'`, not a check on `expires_at`. Renewal always succeeds (PRD), so access never lapses between expiry and the next renewal run.
- Owners keep access to soft-deleted or unpublished books. A subscription only unlocks books that are currently published.
- Idempotency keys are unique per user (`UNIQUE(user_id, idempotency_key)`), not globally.
- Parent view is proven by a 7-day parent token in `X-Parent-Token`, issued by login / PIN / Forgot PIN, never by refresh (ADR-007).
- The seeded mock accounts (migration 000002) are also the production accounts (owner decision, TD-001 won't fix).
- Lists use offset paging, default 20, max 100, with `meta.total` (ADR-008).
- The full ADR log is in [docs/DECISIONS.md](docs/DECISIONS.md).

## Known risks / open questions
- Frontends must drop the parent token on switching to Child view, or the gate is moot.
