# Tech Debt Registry
Last updated: 2026-11-16

## Priority: Critical (blocks scaling or reliability)
<!-- Empty -->

## Priority: High (significant impact, address in next sprint)
- [ ] **TD-011: Demo account passwords and bcrypt hashes remain in git history**
  - Impact: The removed seed migration and old docs had the mock accounts' plaintext passwords, and the branch is pushed to origin. Anyone with repo access can log in wherever those accounts still exist with those passwords.
  - Source: feat/STORY-3-shared-schema (seed migration removed from HEAD only)
  - Created: 2026-11-16
  - Notes: Change those passwords on any non-local DB (seed values now come from the management app's `SEED_*` env vars). Rewriting history is not planned.

## Priority: Medium (worth fixing, not urgent)
- [ ] **TD-004: EPUB and covers are served from Postgres, fully buffered**
  - Impact: Each content request loads the whole EPUB (≤ 50 MB) into memory. Many concurrent readers means high RAM use and DB I/O.
  - Source: feat/storybook-backend (PRD §11 chose DB storage)
  - Created: 2026-11-16
  - Notes: Move to object storage with signed URLs when the catalogue grows.

## Priority: Low (nice to have)
- [ ] **TD-009: Library writes don't maintain `updated_at` on shared tables**
  - Impact: The shared schema (owned by storybook_management_app) has `updated_at` on `subscriptions`, `book_orders` and `subscription_orders`. This app's status updates leave it at the insert time. Nothing reads it today; it will mislead anyone who audits by it.
  - Source: feat/STORY-3-shared-schema
  - Created: 2026-11-16
  - Notes: Add `updated_at = now()` to the status-changing `UPDATE`s in `internal/repositories/library`.
- [ ] **TD-010: Schema is reached through a sibling checkout path**
  - Impact: `make test-integration` and local setup need `../storybook_management_app`. A different layout needs `SCHEMA_DIR` set.
  - Source: feat/STORY-3-shared-schema (management repo ADR-010)
  - Created: 2026-11-16
  - Notes: Move to a shared schema module or git submodule if more services join, or if CI comes back.
- [ ] **TD-005: Auto-renew runs as an in-process hourly ticker**
  - Impact: No retry/backoff or alerting beyond an error log. Renewal can land up to 1 h after expiry. Access is unaffected because an active period stays active until it's renewed.
  - Source: feat/storybook-backend
  - Created: 2026-11-16
- [ ] **TD-007: Access tokens stay valid for up to 30 min after logout**
  - Impact: Logout revokes the refresh token only (stateless JWT by design). The same holds for the parent token (7 days), but it is useless without an access token for the same user.
  - Source: feat/storybook-backend
  - Created: 2026-11-16
- [ ] **TD-008: Scaffold files use CRLF line endings, so `make lint` skips gofmt**
  - Impact: Formatting isn't enforced. Normalize line endings (`.gitattributes` + one formatting commit), then add `gofmt -l` back to `make lint`.
  - Source: feat/storybook-backend
  - Created: 2026-11-16

## Resolved
- [x] **TD-001: Mock accounts are seeded by a migration**
  - Impact: `20261116000002_seed_mock_accounts_and_plan.sql` creates accounts with known passwords. If it runs against a real environment, anyone can log in as admin.
  - Source: feat/storybook-backend (PRD: auth is mocked)
  - Created: 2026-11-16
  - Won't fix (owner decision 2026-11-16): the same mock accounts are used in production. Accepted risk: their passwords are published in the README, the PRD and docs/API-contracts.md.
- [x] **TD-002: Parent/Child view mode is enforced client-side only**
  - Impact: The server can't tell the views apart. A child holding the parent's token could call payment endpoints directly. The PIN only gates the UI.
  - Source: feat/storybook-backend (PRD US-03 keeps one token per account)
  - Created: 2026-11-16
  - Notes: Fix by having `/me/pin/verify` mint a short-lived "parent" claim that is required on payment routes.
  - Resolved 2026-11-16: parent token (`X-Parent-Token`) from login / PIN / Forgot PIN, required on payment and settings routes. It lives 7 days, not short-lived, because the PRD keeps the view across reopen. See ADR-007.
- [x] **TD-003: No pagination on list endpoints**
  - Impact: `/books`, `/me/books`, `/book-orders`, `/subscriptions`, `/admin/books` return everything. Fine for a curated MVP catalogue, slow once it reaches hundreds of rows.
  - Source: feat/storybook-backend
  - Created: 2026-11-16
  - Resolved 2026-11-16: `?page=&page_size=` (default 20, max 100) with `meta.total` on all five lists. See ADR-008.
- [x] **TD-006: "Forgot PIN?" password check has no attempt limit**
  - Impact: The account password can be brute-forced through `/me/pin/forgot` by someone already holding the session.
  - Source: feat/storybook-backend
  - Created: 2026-11-16
  - Resolved 2026-11-16: 4 wrong passwords lock Forgot PIN for 1 minute, separate counter from the PIN (migration 000003).
