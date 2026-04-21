---
description: Unified task completion workflow with mandatory verification and sign-off evidence
---

1. Open `TASKS.md` and choose one task only.
2. Copy acceptance criteria into a temporary checklist and convert each criterion into a test case.
3. Implement the minimal code changes required to satisfy criteria.
4. Run build/tests based on scope:
   - Backend: `go build ./...` then `go test ./...`
   - Frontend: `npm run build` and smoke run in dev mode
5. Verify auth, access boundaries, and side effects:
   - Unauthorized access blocked
   - Cross-user data access blocked
   - DB state updates correct
   - Job queue and external API effects are correct and idempotent
6. Run regression smoke on critical journeys impacted by the change.
7. Run load/stress checks for changed endpoints and capture error rate + p95/p99.
8. Fill sign-off evidence using the template in:
   - `docs/engineering/DELIVERY_VERIFICATION_PLAYBOOK.md`
9. Update task status in `TASKS.md` to Completed only if all gates passed.
10. If any gate fails, keep task open and document blockers + next action.
