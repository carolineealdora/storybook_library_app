# API conventions

- Use PATCH for partial updates, PUT for full replacement
- Error responses: `{ "error": "message", "code": "MACHINE_READABLE_CODE" }`
- Status codes: 400 validation, 401 unauthenticated, 403 unauthorized, 409 conflict, 422 unprocessable
- All endpoints require auth middleware unless explicitly annotated `// PUBLIC`
- Pagination: cursor-based, not offset — see @internal/handler/user.go for reference
- All request bodies must be validated before use — see @internal/validator/ for helpers