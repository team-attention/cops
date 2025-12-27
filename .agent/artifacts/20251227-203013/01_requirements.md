# Requirements

## Request Summary

Session records and project data are being successfully stored in the MongoDB database (in the `session_records` and `projects` collections), but this data is not displaying on the web dashboard. The dashboard appears empty or shows zero counts despite data existing in the database. This issue affects the main dashboard overview page and potentially individual project detail pages.

## Current Understanding

Based on code analysis, the system architecture is:

**Data Flow:**
1. Daemon watches Claude Code JSONL logs
2. Daemon parses session records and sends to API server via gRPC collector endpoint
3. API server stores records in MongoDB (`session_records` and `projects` collections)
4. Web dashboard queries API server via ConnectRPC (`GetOverview` endpoint)
5. Dashboard displays aggregated statistics and lists

**Affected Pages:**
- `/dashboard` - Main dashboard with overview stats, recent projects, and recent sessions
- Potentially `/projects` and `/projects/$projectId` pages

## Questions for Clarification

### 1. Database Verification
- Can you confirm that data exists in MongoDB by running:
  - `db.projects.count()`
  - `db.sessionRecords.count()`
  - `db.projects.findOne()`
  - `db.sessionRecords.findOne()`
- Are there any data validation or schema issues in the stored documents?

### 2. Error Console & Logs
- **Browser Console**: Open DevTools (F12) and check the Console tab. Are there any:
  - JavaScript errors or exceptions?
  - Failed network requests (red entries)?
  - CORS errors?
- **API Server Logs**: Check the API server terminal output. Are there any:
  - Errors when handling the `GetOverview` RPC call?
  - MongoDB query errors?
  - Authentication/authorization errors?
- **Network Tab**: In DevTools Network tab:
  - Is the GetOverview request being sent?
  - What is the response status code (200, 404, 500, etc.)?
  - What is the actual response body content?

### 3. Dashboard Behavior
- Does the dashboard show:
  - A loading skeleton initially?
  - An error message after loading?
  - Zero counts (0 projects, 0 sessions)?
  - Empty tables with "No projects tracked yet" / "No sessions recorded yet"?
- If you click the "Refresh" button, does anything change?

### 4. API Endpoint Testing
- Can you test the API endpoint directly using curl or similar:
  ```bash
  # Test GetOverview endpoint
  curl -X POST http://localhost:8080/cops.dashboard.v1.DashboardService/GetOverview \
    -H "Content-Type: application/json" \
    -d '{}'
  ```
- What response do you get?

### 5. Configuration & Environment
- What is the API server URL configured in the web app?
- Are the API server and MongoDB running on the expected ports?
- Is there a reverse proxy or CORS configuration between web and API?

### 6. Expected vs Actual Behavior

**Expected Behavior:**
- Dashboard shows total input/output/cache tokens
- Dashboard shows project count and session count
- Dashboard displays list of recent projects with names, paths, session counts
- Dashboard displays list of recent sessions with IDs, message counts, timestamps

**Actual Behavior:**
- Please describe exactly what you see on the dashboard currently

### 7. Data Consistency
- When were the session records created (timestamp)?
- Are the `project_id` fields in `sessionRecords` valid ObjectIDs that match `_id` fields in `projects` collection?
- Do the session records have the expected token usage fields populated (`input_tokens`, `output_tokens`, `cache_read_tokens`)?

## Acceptance Criteria

- [ ] Criterion 1: MongoDB data is confirmed to exist and is in correct format
- [ ] Criterion 2: Root cause of visibility issue is identified (frontend, API, database query, or data format)
- [ ] Criterion 3: API endpoint returns correct data when tested directly
- [ ] Criterion 4: Dashboard displays all session and project data from database
- [ ] Criterion 5: Token usage statistics are calculated and displayed correctly
- [ ] Criterion 6: Recent projects list shows up to 5 projects sorted by last activity
- [ ] Criterion 7: Recent sessions list shows up to 5 sessions sorted by start time
- [ ] Criterion 8: Clicking on a project/session navigates to detail page with data

## Scope

### In Scope
- Diagnosing why existing database data is not visible in dashboard
- Fixing data fetching/display issues in frontend, API, or query layer
- Ensuring data aggregation queries return correct results
- Verifying data format consistency between database and protobuf schemas
- Testing and validating the complete data flow from database to UI

### Out of Scope
- Adding new features or dashboard components
- Performance optimization of queries (unless directly causing the issue)
- UI/UX improvements unrelated to data visibility
- Changes to daemon data collection logic
- Database migration or schema changes (unless required for fix)

## Constraints
- Must maintain existing API contracts (protobuf schema)
- Must not modify existing database records
- Should work with current system architecture (daemon → API → web)
- Fix should not require changes to Claude Code JSONL log format

## Additional Context

**Technology Stack:**
- **Frontend**: React, TypeScript, TanStack Router, TanStack Query (Connect-Query)
- **API**: Go, Fiber (HTTP), ConnectRPC (gRPC over HTTP)
- **Database**: MongoDB
- **Data Flow**: Daemon (file watcher) → API (collector) → MongoDB → API (dashboard) → Web

**Key Files:**
- Frontend: `/web/src/route/dashboard.tsx`, `/web/src/feature/dashboard/hook/use-get-overview.ts`
- API Handler: `/api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go`
- Service: `/api/internal/service/dashboard/dashboard_service.go`
- Repository: `/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**MongoDB Collections:**
- `projects` - Project metadata (name, path, git branch, etc.)
- `sessionRecords` - Individual Claude Code session records with token usage (camelCase naming)

## Questions Resolved

| Question                                                      | Answer |
| ------------------------------------------------------------- | ------ |
| Please answer the clarifying questions above                  | [Awaiting user response] |
| What exactly do you see on the dashboard currently?           | [Awaiting user response] |
| Are there any console errors in browser DevTools?             | [Awaiting user response] |
| What does the GetOverview API response contain?               | [Awaiting user response] |
| Can you confirm data exists in MongoDB with sample queries?   | [Awaiting user response] |
