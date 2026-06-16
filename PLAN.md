## Template
# Plan: [Feature Name]
**Status:** In Progress — Phase 2 of 4
**Last updated:** [date]

## Phases
- [x] Phase 1: Database layer — DONE
- [ ] Phase 2: API endpoints — IN PROGRESS
- [ ] Phase 3: Auth integration — TODO
- [ ] Phase 4: Tests + cleanup — TODO

## Phase 2 detail
- [x] POST /users
- [ ] GET /users/:id
- [ ] PUT /users/:id (PATCH semantics — partial update only)

## Decisions made — do not revisit
- Using PATCH not PUT for user updates
- Return 409 on duplicate email (not 400)
- Storing refresh tokens in Redis (not DB) — see docs/DECISIONS.md

## Known risks / open questions
- Rate limiting on auth endpoints — deferred to Phase 4
- Need to confirm Keycloak realm config with DevOps before Phase 3