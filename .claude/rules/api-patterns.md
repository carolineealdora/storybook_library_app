# API conventions

The contract the frontends rely on is [docs/API-contracts.md](../../docs/API-contracts.md). Keep it in sync with every route change.

## URLs and methods
- Everything is under `/api/v1`. Use plural nouns. Nest at most one level (`/books/:id/content`).
- Admin routes go under `/api/v1/admin/...`.
- Actions are `POST` sub-resources (`/book-orders/:id/pay`, `/subscriptions/current/cancel`).
- `PATCH` for partial updates (`/me`), `PUT` for full replacement (`/admin/books/:id`, `/me/pin`).

## Responses
- Success is always HTTP 200 with `{ "data": ... }`. Lists add `"meta": { "total", "page", "page_size" }`.
- Errors: `{ "error": { "code": "BOOK_LOCKED", "message": "..." } }`. Codes and HTTP statuses live in `internal/constant/response/response.go`. Add a code there, never inline.
- Validation errors are `400 VALIDATION_ERROR`, and the message names the field.
- Never put internal error text in a message. Only the allowlisted codes in `internal/helper/error.go` pass their message through.
- 404 (not 403) for another user's resource.

## Lists
- Offset pagination: `?page=1&page_size=20`, max 100. Use `Base.Page` + `Base.RespondPage` in the controller and the `page[T]` helper in the repo.
- Every list query needs a deterministic `ORDER BY` (end with `id`).

## Auth
- Every route takes a role middleware (`parent` / `admin`) unless it is commented `// PUBLIC` in `handler/route.go`.
- Routes that spend money or change settings also take `parentView` (`X-Parent-Token`).
- Handlers read the account from `ec.Get(key.AccountID)`. Never re-parse the token.

## Handler shape
1. `c.Bind(ec, &req)`. Validation tags live on the DTO in `models/dto`.
2. Call one service method.
3. `c.Respond(ec, data, err)`. The service has already chosen the error code.

No business logic in handlers.
