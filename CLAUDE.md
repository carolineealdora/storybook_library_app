# CLAUDE.md

## Project
**Name:** [Project Name]
**Purpose:** [1–2 sentences: what this does and what problem it solves]

---

## Tech stack
- Language/Runtime: [Go 1.22 / Node.js 20 / Python 3.12 / etc.]
- Framework: [Gin / Express / FastAPI / etc.]
- Database: [PostgreSQL 16 — parameterized queries only, never raw string interpolation]
- Cache: [Redis 7 — slot availability, session tokens, job queue / etc.]
- Auth: [JWT RS256 + Redis-stored refresh tokens / Keycloak — read @middleware/auth.go for pattern]
- Payments: [Stripe / etc. — if applicable]
- Notifications: [WhatsApp Business API + SMTP / etc. — if applicable]
- Message queue: [if applicable]

---

## System context — read before structural decisions
- Architecture overview: @docs/ARCHITECTURE.md
- API contracts and response shapes: @docs/API-contracts.md
- Data models and entity relationships: @docs/DATA-MODELS.md
- Feature PRDs: @docs/PRD-[feature-name].md

---

## Key files — entry points for understanding patterns
- Entry point:        [cmd/main.go]
- Config:             [internal/config/config.go]
- Handler pattern:    [@internal/handler/example.go]
- Service pattern:    [@internal/service/example.go]
- Repository pattern: [@internal/repository/example.go]
- Auth middleware:    [middleware/auth.go]
- Response helpers:   [internal/response/response.go]
- Input validation:   [internal/validator/validator.go]
- DB pool:            [internal/db/db.go]

---

## Commands
- Run:   `make run`
- Test:  `make test`  (includes -race flag)
- Lint:  `make lint`
- Build: `make build`
- **Definition of Done: `make done`** ← always run before declaring complete. Show full output.

---

## BEHAVIOR — always follow for every session

### Plan mode (mandatory for implementation tasks)
- For ANY implementation task, start in plan mode (Shift+Tab × 2 or `/plan`).
- Read the relevant PRD (`@docs/PRD-[feature].md`) and architecture doc before planning.
- Create `plan.md` with phases, files to touch, decisions, and risks.
- Do not write production code until `plan.md` has been annotated and confirmed.
- Use the phrase `"address all notes, don't implement yet"` after receiving annotations.
- Skip plan mode only for trivial changes (single-line fixes, config values, typos).

### TDD workflow (mandatory — no exceptions)
1. Write tests BEFORE production code. Always.
2. Run tests. Confirm they FAIL. Show the failure output before continuing.
3. Implement until all tests pass.
4. Commit tests and implementation separately.

See full TDD rules: @.claude/rules/testing.md

### Tech debt tracking
When making an intentional shortcut, deferring an improvement, or identifying a known limitation outside current scope — add an entry to `tech-debt.md` immediately:
```
## [Priority]
- [ ] TD-[N]: [What the shortcut is]
  - Impact: [Why it matters and when it will hurt]
  - Source: [Current feature/ticket — e.g. feat/PROJ-42 Phase 2]
  - Created: [YYYY-MM-DD]
```
Do not skip this. Undocumented shortcuts become invisible debt.

---

## HARD RULES — never break

### Secret hygiene
- Never commit secrets, API keys, tokens, or credentials to version control
- All secrets live in environment variables only
- Use `.env.example` with redacted values as the schema reference
- If a secret is ever committed — rotate it immediately and treat it as compromised
- Never log sensitive fields: tokens, payment data, passwords, user PII

### Git discipline
- Always create a feature branch before making changes — never commit to main
- Branch naming: `feat/PROJ-[id]-short-description` or `fix/PROJ-[id]-short-description`
- Commit format: `feat(scope): description [PROJ-ID]` (conventional commits)
- Separate commits for tests and implementation
- Run `make done` before every commit. Never commit if it fails.

See full git rules: @.claude/rules/git.md

### Code rules
- Never use `_ = err` to silence errors — handle or propagate with context
- Never use raw string interpolation in SQL queries — parameterized queries only
- Never trust client-reported amounts — always calculate server-side
- Never expose stack traces or internal error messages in API responses
- Webhook handlers: verify HMAC signature before processing — reject unsigned requests
- Auth and payment changes: add comment `// SECURITY: manual review required`

See full security rules: @.claude/rules/security.md

---

## SCOPE CONTROL
- Only modify files within the stated scope of the current task
- Do not touch auth code unless the task explicitly involves auth
- Do not touch payment/webhook logic unless the task explicitly involves payments
- State explicitly which files you will NOT touch before implementing

---

## Rules files (load when relevant — do not load all every session)
- Testing + TDD:    @.claude/rules/testing.md
- Security:         @.claude/rules/security.md
- API conventions:  @.claude/rules/api-patterns.md
- Git:              @.claude/rules/git.md

---

## Session handoff (use this at the start of every new session)
```
Read CLAUDE.md, PLAN.md, and tech-debt.md.
We are on [phase]. Last session completed [X].
Continue with [next task].
Only modify files in [scoped path].
Run make done when complete and show output.
```