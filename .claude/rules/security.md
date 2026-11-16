# Security rules

## Secrets
- Secrets come from env only (`JWT_SECRET`, `DB_PASSWORD`). `.env.example` holds placeholders.
- The service refuses to start without `JWT_SECRET`. Never add a default.
- Never log passwords, PINs, tokens, idempotency keys or file contents. Error logs carry the code and internal info only.

## Auth and tokens
- Access tokens: HS256 only (`jwt.WithValidMethods`), expiry required. Reject `alg: none`.
- Refresh tokens are stored as SHA-256 only. Compare by hash.
- Parent token (`role: parent_mode`) must never be accepted as an access token, and the reverse.
- Passwords and PINs use bcrypt. Login runs bcrypt even for unknown emails (`dummyHash`) so response timing doesn't reveal which emails exist.
- Lockouts: PIN 5 tries / 1 min, Forgot PIN password 4 tries / 1 min. Counters update atomically in SQL.

## Data access
- Parameterized SQL only. The single exception is `library.Counter`, which picks a column name from a closed set. Don't widen it to take input.
- Every query that reads user data filters by the authenticated `user_id`.
- Money is calculated server-side from the DB (price snapshot on the order). Never trust a client amount.
- Locked EPUB bytes are never sent. The access check runs before the file is read.

## Uploads
- Cover: `.jpg/.jpeg/.png` extension and sniffed content type, ≤ 5 MB. EPUB: `.epub` extension and ZIP magic bytes, ≤ 50 MB. Request body limit is 60 MB.

## Review markers
- Auth, token, PIN and payment code carries `// SECURITY: manual review required`.

## Known gaps
- The mock accounts from migration 000002 are also used in production, by owner decision (TD-001). Don't "fix" this without asking.
- TD-007: an access token stays valid up to 30 min after logout.
