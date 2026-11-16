# API Contracts — Storybook Library backend
**Base URL:** `http://localhost:9999/api/v1` · **Source of truth for behaviour:** [PRD](PRD_storybook-lib-app.md)

## Conventions

- JSON in, JSON out (except the cover and EPUB file endpoints).
- Success: `{ "data": ... }` with HTTP 200.
- Error: `{ "error": { "code": "BOOK_LOCKED", "message": "Subscribe or purchase this book to read it." } }`. Branch on `code`, show `message`.
- Auth: `Authorization: Bearer <access_token>` from a **parent** account. An admin token gets `403 FORBIDDEN`.
- This backend serves only `fe_storybook_library_app`. The back office (admin login, books, plan pricing) is storybook_management_app; see its `docs/API-contracts.md` (ADR-011).
- Money is integer IDR (`15000` = Rp 15.000). Timestamps are RFC 3339.
- IDs are UUIDs. A malformed `:id` returns `400 VALIDATION_ERROR`.
- Lists take `?page=1&page_size=20` (defaults; `page_size` max 100) and return `{ "data": [...], "meta": { "total": 42, "page": 1, "page_size": 20 } }`.

### Token lifecycle (US-01)
1. `POST /auth/login` returns an `access_token` (30 min) and a `refresh_token` (7 days, not rotated).
2. On `401 TOKEN_EXPIRED`, call `POST /auth/refresh` with the refresh token, then retry the request.
3. On `401 REFRESH_TOKEN_EXPIRED` or `401 UNAUTHORIZED`, send the user to Login.
4. Only parent refresh tokens work here. An admin refresh token (the table is shared with the back office) gets `401 UNAUTHORIZED`.

### Parent view token (US-03)
Routes marked **🔒P** below also need `X-Parent-Token: <parent_token>`.
- The `parent_token` comes back from parent login, from `POST /me/pin/verify` and from `POST /me/pin/forgot`. It lasts 7 days.
- Keep it while the app is in Parent view, persisted the same way as the view mode.
- **Delete it when switching to Child view.** That is what stops a child from paying.
- `POST /auth/refresh` never returns one.
- On `403 PARENT_MODE_REQUIRED`, show the PIN prompt (the same as tapping the toggle in Child view).

### Seeded mock accounts
Created by `make seed` in storybook_management_app from its `SEED_PARENT_*` and `SEED_ADMIN_*` env vars: one parent (Library) and one admin (Book Management). Ask the backend owner for the local values; passwords are never written in docs.

## Error codes

| HTTP | Code | When |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing/invalid field (message names the field) |
| 401 | `INVALID_CREDENTIALS` | Wrong email or password ("Email or password is incorrect") |
| 401 | `UNAUTHORIZED` | No/invalid token, or revoked refresh token |
| 401 | `TOKEN_EXPIRED` | Access token expired → refresh |
| 401 | `REFRESH_TOKEN_EXPIRED` | Refresh token expired → Login |
| 401 | `INVALID_PIN` / `INVALID_PASSWORD` | Wrong PIN / wrong account password |
| 403 | `PARENT_MODE_REQUIRED` | 🔒P route called without a valid parent token → show the PIN prompt |
| 402 | `PAYMENT_FAILED` | Mocked payment failed. The order is `failed`; retry with a **new** order |
| 403 | `FORBIDDEN` | Admin account on this app (login or token) |
| 403 | `BOOK_LOCKED` | Book content requested without access |
| 404 | `BOOK_NOT_FOUND` / `ORDER_NOT_FOUND` / `NOT_FOUND` | Missing, unpublished, or not yours |
| 409 | `PRICE_CHANGED` | Subscription `expected_price` ≠ current plan price → re-read the plan, ask the parent to confirm again (CR-001) |
| 409 | `BOOK_ALREADY_OWNED` · `SUBSCRIPTION_ACTIVE` · `BOOK_UNAVAILABLE` · `BOOK_NOT_PURCHASABLE` · `DUPLICATE_REQUEST` · `PIN_NOT_SET` · `SUBSCRIPTION_NOT_ACTIVE` | PRD edge cases E1–E14 |
| 429 | `TOO_MANY_ATTEMPTS` | 5 wrong PINs → PIN locked for 1 minute. 4 wrong passwords on Forgot PIN → that is locked for 1 minute (separately) |
| 500 | `GENERAL_ERROR` | Unexpected; details only in server logs |

## Shapes

```jsonc
// Book (list, detail, my books)
{ "id": "uuid", "title": "Si Kancil", "author": "…", "description": "…",
  "age_type": "range",          // "range" | "plus"
  "age_min": 1, "age_max": 3,   // age_max null when "plus"
  "age_label": "1–3",           // ready to display: "1–3" or "3+"
  "price": 15000,               // 0 = free
  "status": "published",        // "draft" | "published"
  "access": "locked",           // parent routes only: "free" | "owned" | "subscribed" | "locked"
  "epub_file_name": "kancil.epub", "epub_size_bytes": 123456,
  "purchased_at": "…",          // /me/books only
  "created_at": "…", "updated_at": "…" }

// BookOrder
{ "id": "uuid", "book_id": "uuid", "book_title": "…", "price": 15000,
  "status": "pending",          // "pending" | "paid" | "failed"
  "created_at": "…", "paid_at": null }

// Subscription (period)
{ "id": "uuid", "plan_id": "uuid", "renewed_from_id": null, "price_paid": 49000,
  "status": "active",           // "active" | "cancelled" | "expired"
  "started_at": "…", "expires_at": "…",  // expires_at = next renewal date while active
  "cancelled_at": null,
  "order_type": "new" }         // history only: "new" | "renewal"

// SubscriptionPlan
{ "id": "uuid", "code": "monthly", "name": "Monthly Subscription", "price": 49000,
  "benefits": "…", "duration": "1 month", "updated_at": "…" }

// GET /me (CR-001)
{ "id": "uuid", "name": "…", "email": "…", "pin_set": true,
  "plan_status": "subscribed",   // "subscribed" | "free" → badge "Subscribed" | "Free"
  "is_subscriber": true,         // same meaning, kept for older clients
  "subscription": {              // null when plan_status = "free"
    …Subscription fields…,
    "next_renewal_price": 59000, // current plan price = what the next auto-renewal charges
    "price_change_pending": true // next_renewal_price ≠ price_paid → show the price-change notice
  } }
```

**Account badge (CR-001-1):** show **Subscribed** when `plan_status = "subscribed"`, otherwise **Free**, in the header of both views. Read `plan_status`; fall back to `is_subscriber` if absent. Re-fetch `/me` after a successful subscription payment or cancel so the badge changes without a reload. Purchased books never change the badge.

**Price-change notice (CR-001-2):** when `subscription.price_change_pending` is true, show on the Subscription page / Settings: *"Your subscription price will change from Rp [price_paid] to Rp [next_renewal_price] on your next renewal ([expires_at date])."*

**Which button to show on Book Detail (US-06), from `access`:**
`locked` → Buy Now + subscription prompt · `subscribed` → disabled "Subscribed" + Read · `owned` / `free` → Read.

## Digital Storybook Library app (parent token)

| Method | Path | Body | Returns |
|---|---|---|---|
| POST | `/auth/login` *(public)* | `{email, password}` | `{access_token, refresh_token, token_type, expires_in, refresh_expires_at, account_type, parent_token, parent_token_expires_in}` |
| POST | `/auth/refresh` *(public)* | `{refresh_token}` | `{access_token, token_type, expires_in, account_type}` |
| POST | `/auth/logout` | `{refresh_token}` | `{logged_out: true}` |
| GET | `/me` | — | see the `GET /me` shape above |
| PATCH 🔒P | `/me` | `{name}` | same as GET /me |
| PUT 🔒P | `/me/pin` | `{pin, pin_confirmation, password?}` | `{pin_set: true}`. `pin` is 4 digits. `password` is required only when a PIN already exists |
| POST | `/me/pin/verify` | `{pin}` | `{verified: true, parent_token, parent_token_expires_in}` → switch to Parent view. Errors: `PIN_NOT_SET`, `INVALID_PIN`, `TOO_MANY_ATTEMPTS` |
| POST | `/me/pin/forgot` | `{password}` | same as verify. Errors: `INVALID_PASSWORD`, `TOO_MANY_ATTEMPTS` (after 4 wrong) |
| GET | `/books?page=&page_size=` | — | `Book[]`: published books, newest first, with `access` |
| GET | `/books/:id` | — | `Book` |
| GET | `/books/:id/cover` | — | image bytes (`image/jpeg` / `image/png`) |
| GET | `/books/:id/content` | — | EPUB bytes (`application/epub+zip`), or `403 BOOK_LOCKED` |
| GET | `/me/books?page=&page_size=` | — | `Book[]`: purchased only, newest purchase first. Includes books the admin later deleted |
| POST 🔒P | `/book-orders` | `{book_id, idempotency_key}` | `BookOrder` (pending, price snapshotted) |
| POST 🔒P | `/book-orders/:id/pay` | `{result: "success" \| "failed"}` | `BookOrder` (paid), or `402 PAYMENT_FAILED` |
| GET | `/book-orders?page=&page_size=` | — | `BookOrder[]`: Order History (paid + failed only), newest first |
| GET | `/subscription/plan` | — | `SubscriptionPlan` |
| POST 🔒P | `/subscription-orders` | `{idempotency_key, expected_price}` | subscription order `{id, price, status, order_type, …}`. `expected_price` = the price the paywall showed; mismatch → `409 PRICE_CHANGED` |
| POST 🔒P | `/subscription-orders/:id/pay` | `{result: "success" \| "failed"}` | `Subscription` (active), or `402 PAYMENT_FAILED` |
| POST 🔒P | `/subscriptions/current/cancel` | — | `Subscription` (cancelled). Access stops immediately |
| GET | `/subscriptions?page=&page_size=` | — | `Subscription[]`: Subscription History, newest first |

**Subscription paywall (CR-001):** show the price from `GET /subscription/plan` and send it back as `expected_price`. On `409 PRICE_CHANGED`, reload the plan, show the new price and ask again.

**Checkout flow (US-09):** Buy Now → `POST /book-orders` with a fresh `idempotency_key` (e.g. `crypto.randomUUID()`), then show Checkout with the order's `price`. Pay → `POST /book-orders/:id/pay`. A second Pay on the same order returns `409 DUPLICATE_REQUEST`. After `402`, retry by creating a new order. Subscriptions follow the same two steps.

**Images and EPUB need the bearer header**, so `<img src>` won't work directly. Fetch them as a blob and use `URL.createObjectURL`.

**Mocked payment:** the client chooses the outcome with `result`. Give the Checkout page "Pay (success)" and "Pay (fail)" controls for the test.
