# API Patterns

## URL Conventions

- Versioned: `/api/v1/[resource]`
- Plural nouns: `/users`, `/orders`, `/items` (never `/user`, `/getOrder`)
- Nested only one level deep: `/api/v1/[resource]/:id/[sub-resource]` (not deeper)
- Webhooks at `/webhooks/[provider]`
- Admin-only at `/api/v1/admin/[resource]`

## HTTP Method Semantics

| Method | Use | Body |
|---|---|---|
| GET | Read only, no side effects | None |
| POST | Create resource or trigger action | JSON |
| PATCH | Partial update (only fields provided change) | JSON |
| PUT | Full replacement (all fields required) | JSON |
| DELETE | Remove or soft-delete | None |

Use PATCH for partial updates (e.g. update a resource status). Use PUT only when the full resource is replaced.

## Status Codes

| Code | When |
|---|---|
| 200 | Successful GET, PATCH, PUT |
| 201 | Successful POST (resource created) — include resource in body |
| 204 | Successful DELETE with no body |
| 400 | Malformed request (bad JSON, missing required field) |
| 401 | Unauthenticated (no/invalid/expired JWT) |
| 403 | Authenticated but unauthorized for this action |
| 404 | Resource not found (also use for cross-user access — don't leak existence) |
| 409 | Conflict (duplicate, slot taken, illegal state transition) |
| 422 | Validation error (field-level — use body to describe which fields) |
| 500 | Internal error (log internally; never expose stack trace to client) |

## Error Response Shape

All errors use this shape — machine-readable code + human message:
```json
{
  "error": {
    "code": "RESOURCE_CONFLICT",
    "message": "The requested resource is no longer available.",
    "fields": {}
  }
}
```

Validation errors include per-field detail:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed.",
    "fields": {
      "start_time": "must be in the future",
      "duration_minutes": "must be 30 or 60"
    }
  }
}
```

## Success Response Shape

Single resource:
```json
{ "data": { ...resource... } }
```

Collection (paginated):
```json
{
  "data": [ ...items... ],
  "meta": {
    "total": 120,
    "page": 1,
    "page_size": 20
  }
}
```

## Authentication

All endpoints require auth middleware unless explicitly annotated `// PUBLIC`.

Public endpoints must be explicitly marked and documented. Webhooks should be public but verified (e.g. HMAC signature check).

Extract the authenticated user's ID from JWT claims via auth middleware. Never re-parse the token in handlers.

## Pagination

Use offset-based pagination for all list endpoints:

- Query params: `?page=1&page_size=20`
- Default `page_size`: 20. Max: 100.
- Always return `meta.total` for frontend pagination controls.

## Handler Pattern

1. Parse and validate input
2. Extract claims from context (set by auth middleware)
3. Call service method
4. Map service errors to HTTP status codes
5. Return response

Never put business logic in handlers. Handlers are thin HTTP adapters only.