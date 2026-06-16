# PRD: [Feature Name]
**Feature ID:** [X.X]
**Ticket:** PROJ-[number]
**Branch:** `feat/PROJ-[number]-[name]`
**Phase:** [1 or 2]
**Status:** 📝 Draft | 🔄 In Progress | ✅ Done

---

## Implementation Status
<!-- Claude: check this table at session start. Update it as phases complete. -->
<!-- To verify a component: check if its key file exists (see Files section). -->

| Component | Status | PR | make done |
|---|---|---|---|
| Domain structs | ⏳ Todo | — | — |
| DB migration | ⏳ Todo | — | — |
| Repository layer | ⏳ Todo | — | — |
| Service layer | ⏳ Todo | — | — |
| Handler layer | ⏳ Todo | — | — |
| Route registration | ⏳ Todo | — | — |
| [Feature-specific, e.g. Webhook handler] | ⏳ Todo | — | — |
| Tests ≥ 80% coverage | ⏳ Todo | — | — |

**Status key:** ⏳ Todo · 🔄 In Progress · ✅ Done · 🚫 Blocked

---

## Problem statement

[What pain does this solve, and for whom? 2–3 sentences max.]

## Goal

[One sentence: what does "done" look like for the user?]

## Out of scope — do not build in this ticket
- [Item 1 — PROJ-X]
- [Item 2 — PROJ-X]

---

## User flows

### Primary: [happy path name]
```
1. [Step]
2. [Step]
3. [Step]
```

### Edge: [edge case name]
```
[Edge case flow]
```

---

## Acceptance criteria

```
Given [context]
When [action]
Then [expected outcome]

Given [context]
When [action]
Then [expected outcome]
```

---

## Data model

```sql
-- Table: [table_name]
CREATE TABLE [table_name] (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    [field]     [TYPE] NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_[table]_[field] ON [table_name]([field]);
```

**Money fields:** always INTEGER (smallest currency unit — sen). Never FLOAT.
**Soft deletes:** use `deleted_at TIMESTAMPTZ` or `status` enum. Never hard-delete.

---

## API endpoints

```
METHOD  /api/v1/[resource]           Description
METHOD  /api/v1/[resource]/:id       Description
```

---

## Key decisions — resolve before implementation
<!-- Engineer annotates these before Claude starts coding. Do not implement until resolved. -->

- [ ] [Decision 1] → `_______`
- [ ] [Decision 2] → `_______`
- [ ] [Decision 3] → `_______`

---

## Files to create / modify

```
New:
  internal/[module]/handler/[feature].go
  internal/[module]/handler/[feature]_test.go
  internal/[module]/service/[feature].go
  internal/[module]/service/[feature]_test.go
  internal/[module]/repository/[feature].go
  internal/[module]/repository/[feature]_test.go
  internal/domain/[feature].go
  db/migrations/00N_create_[table].sql

Modified:
  internal/router/router.go

Do not touch:
  internal/auth/          (out of scope)
  internal/payment/       (out of scope unless explicitly payment-related)
```

---

## Non-functional requirements

| Requirement | Target |
|---|---|
| Latency | < [N]ms p99 |
| Idempotency | [Yes/No — describe key] |
| Consistency requirement | [e.g. ACID, eventual] |

---

## Definition of Done

```bash
make done   # must output: ✅ Definition of Done: ALL CHECKS PASSED
```

All acceptance criteria covered by tests (including edge cases from `.claude/rules/testing.md`).
Coverage ≥ 80% on `internal/[module]/` package.
Implementation Status table above fully updated.
tech-debt.md updated if any shortcuts were taken.