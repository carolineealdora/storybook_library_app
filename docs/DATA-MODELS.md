# Data Models — Storybook Library
**Last updated:** 2026-11-16

**Source of truth is the SQL in [`storybook_management_app/migrations/`](../../storybook_management_app/migrations/)**, which owns the schema this app shares (its ADR-010). This app writes `users`, `book_orders`, `purchases`, `subscription_orders` and `subscriptions`, plus its own parent `credentials` / `refresh_tokens` rows. Books, covers, the plan and its price history are written by the back office (ADR-011). This page explains the model. Read the migration before writing a query. The ER diagram is in the [PRD §12.2](PRD_storybook-lib-app.md).

---

## Conventions

| Convention | Rule |
|---|---|
| Primary keys | `UUID DEFAULT gen_random_uuid()` |
| Money | `INT` whole IDR (`15000` = Rp 15.000). IDR has no minor unit. Never float |
| Timestamps | `TIMESTAMPTZ` |
| Enums | `VARCHAR(n)` + `CHECK`, not native Postgres enums |
| Soft delete | Only `books.deleted_at`. Orders, purchases and subscriptions are never deleted; their `status` carries history |
| Price snapshots | `book_orders.price`, `purchases.price_paid` and `subscriptions.price_paid` keep the price at order time (PRD E7, US-17) |
| Idempotency | `UNIQUE (user_id, idempotency_key)` on both order tables |
| New schema | Add a new numbered migration **in storybook_management_app** and list it in its `migrations/migration.go`. Never edit an applied one |

---

## Tables

| Table | One row = | Key rules |
|---|---|---|
| `users` | A parent (account holder) | `pin_hash` is null until set. Two lockout counters: `pin_*` (5 tries, PRD E12) and `password_*` (4 tries on Forgot PIN, TD-006) |
| `admins` | A back-office user | — |
| `credentials` | A login | Unique `lower(email)`. `account_type` is `parent`/`admin`, and exactly one of `user_id`/`admin_id` is set |
| `refresh_tokens` | A session | Only the SHA-256 of the token is stored. `revoked_at` is set on logout |
| `books` | A storybook | `status` is draft/published. `age_type` range ⇒ `age_max > age_min`, plus ⇒ `age_max` null. `epub_content` is never selected in lists |
| `book_files` | A book's cover | One per book. jpeg/png only |
| `subscription_plans` | The plan | Single row in the MVP, `code = 'monthly'` (unique). `updated_by` → admin who last changed it |
| `subscription_plan_price_history` | One price change | `old_price ≠ new_price` (a same-price save writes nothing). `changed_by` → admins (CR-001) |
| `subscriptions` | One paid monthly period (`price_paid` is locked for that period; a renewal is a new row at the then-current price) | `active` → `cancelled` or `expired`. Partial unique index: at most one `active` per user. `renewed_from_id` links the chain |
| `book_orders` | A checkout for one book | `pending` → `paid` or `failed`. Never reused after `failed` |
| `subscription_orders` | A checkout or renewal for the plan | `order_type` is new/renewal. Renewal key is `renewal:<subscription_id>` |
| `purchases` | Ownership of a book | `UNIQUE (user_id, book_id)`. Survives the book being unpublished or deleted |

---

## Relationships

```
credentials ─┬─ users ──< subscriptions >── subscription_plans
             │     ├────< subscription_orders >── subscription_plans
             │     ├────< book_orders >── books ── book_files (1:1)
             │     └────< purchases   >── books
             └─ admins
credentials ──< refresh_tokens
```

---

## Access rule (derived, not stored)

A parent can read a book when:

```
book.price = 0 (and visible)  OR  purchases row exists  OR  (active subscription AND book published, not deleted)
```

"Visible" means published and not deleted. Owners see their books even after that stops being true. See `internal/services/book/service.go`.
