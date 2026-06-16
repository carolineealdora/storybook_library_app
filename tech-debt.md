## Template
# Tech Debt Registry
Last updated: [date]

## Priority: Critical (blocks scaling or reliability)
<!-- Empty -->

## Priority: High (significant impact, address in next sprint)
- [ ] **TD-001: N+1 query in GET /users endpoint**
  - Impact: Page load >3s under load (>100 concurrent users)
  - Source: feat/user-list — identified during Phase 2
  - Created: [date]
  - Notes: Need to batch-load with JOIN or use DataLoader pattern

## Priority: Medium (worth fixing, not urgent)
- [ ] **TD-002: Test coverage below 80% in auth module**
  - Impact: Regression risk on next auth change
  - Source: feat/auth-refactor — deferred to meet deadline
  - Created: [date]

## Priority: Low (nice to have)
- [ ] **TD-003: Hard-coded timeout values in HTTP client**
  - Impact: No per-endpoint tuning possible
  - Source: feat/external-api-integration
  - Created: [date]

## Resolved
- [x] **TD-000: Missing index on users.email**
  - Resolved: [date] in PR #47