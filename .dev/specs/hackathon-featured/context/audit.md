## TODO 1 — Reconciliation

### [2026-02-28] Verify
- Status: VERIFIED (6/6 PASS)
- acceptance_criteria: all PASS
- must_not_do: no violations
- side_effects: cli/Makefile modified out of scope (benign), new TS file generated (expected)
- missing_context: buf lint pre-existing naming convention errors (14 in dashboard.proto), 2 new ones consistent with project convention

## TODO 2 — Reconciliation

### [2026-02-28] Verify
- Status: VERIFIED (7/7 PASS)
- acceptance_criteria: all PASS (build via featured subpackage to avoid pre-existing sonic issue)
- must_not_do: no violations
- side_effects: web/src/routeTree.gen.ts auto-regenerated (expected), no unit tests written (accepted)
- missing_context: sonic/Go 1.26 incompatibility pre-existing, OrganizationRepositoryPort DI wiring correct

## TODO 3 — Reconciliation

### [2026-02-28] Triage
- acceptance_criteria:root_excludes_featured FAIL → RETRY (code_error)
- Worker claimed __root.tsx was updated but git diff shows NO changes to __root.tsx
- side_effects: web/src/feature/session/ 10 files reformatted (whitespace only), web/src/shared/util/format.ts added formatTokenCount (additive)
- Disposition: RETRY #1 — fix Worker to update __root.tsx

### [2026-02-28] Retry #1
- Fix Worker applied: added isFeaturedRoute to __root.tsx sidebar exclusion condition
- Re-verify: VERIFIED (8/8 PASS)
- Previously failed criterion root_excludes_featured: now PASS
- No new violations or side-effects from fix

## TODO Final — Reconciliation

### [2026-02-28] Verify
- Status: VERIFIED (14/14 PASS)
- acceptance_criteria: all PASS (proto, stubs, service dir, fx module, web route, sidebar exclusion, polling interval, Go build/vet, TS check, lint, tests)
- must_not_do: no violations (DashboardService unchanged, no RBAC in featured handler, no new go.mod deps)
- side_effects: none
- missing_context: featured service has no unit tests (acceptable per criteria — exit 0 with [no test files])

## Final Code Review

### [2026-02-28] Review #1
- Status: NEEDS_FIXES
- Findings:
  - CR-001 [critical] Public endpoint exposes member PII — ACCEPTED (explicit design decision from plan interview)
  - CR-002 [warning] Members without projects produce zero-stat cards
  - CR-003 [warning] getDefaultSince() returns future timestamp before 20:00
  - CR-004 [warning] BigInt(NaN) crash on invalid ?since= param
  - CR-005 [warning] Hardcoded "$toolName" bypasses mongoschema constants
  - CR-006 [warning] sinceUnix=0 causes unbounded collection scan
  - CR-007 [info] All errors collapsed to "Organization not found"
  - CR-008 [info] Uptime/inactive 30s visual lag between polls
- Action: Fix tasks created for CR-003+CR-004, CR-005+CR-006, CR-002

### [2026-02-28] Review #2 (retry)
- Status: SHIP (2/3 models: Codex SHIP, Claude SHIP, Gemini NEEDS_FIXES)
- All 5 targeted fixes verified correct:
  - CR-002: filteredMemberIDs pre-filters members without projects ✓
  - CR-003: getDefaultSince rolls back to previous day when < 20:00 ✓
  - CR-004: validateSearch uses Number.isFinite guard ✓
  - CR-005: SessionToolNameField constant added and used ✓
  - CR-006: sinceUnix <= 0 defaults to 24h ago ✓
- Gemini re-flagged CR-001 (accepted design decision) — does not count
- Remaining info items: generated code churn (cosmetic), frontend/backend default-since mismatch (no user-visible bug)
- Action: Proceed (SHIP)
