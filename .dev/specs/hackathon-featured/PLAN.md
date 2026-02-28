# Hackathon Featured Board

> Organization의 해커톤 상황판(Featured Page)을 구현한다. 멤버별 프로젝트 메트릭을 실시간 폴링으로 보여주는 공개 대시보드.

---

## Verification Summary

### Agent-Verifiable (A-items)
| ID | Criterion | Method | Related TODO |
|----|-----------|--------|-------------|
| A-1 | GetFeaturedBoard RPC가 proto에 정의됨 | `grep -c "GetFeaturedBoard" idl/protobuf/dashboard/v1/dashboard.proto` | TODO 1 |
| A-2 | Go stub 생성됨 | `grep -c "GetFeaturedBoard" shared/gen/grpcstub/dashboard/v1/dashboardv1connect/dashboard.connect.go` | TODO 1 |
| A-3 | TypeScript stub 생성됨 | `grep "getFeaturedBoard\|GetFeaturedBoard" web/src/gen/grpcstub/dashboard/v1/*.ts` | TODO 1 |
| A-4 | Go 빌드 성공 | `go build ./api/... ./shared/...` | TODO 2 |
| A-5 | FeaturedBoardGRPCHandler가 인터페이스 구현 | `go vet ./api/...` | TODO 2 |
| A-6 | 빈 slug 요청 시 InvalidArgument 반환 | `go test ./api/internal/service/featured/... -v -run TestFeaturedBoard` | TODO 2 |
| A-7 | 기존 Go 테스트 회귀 없음 | `go test ./api/... ./shared/... ./daemon/... ./cli/...` | TODO Final |
| A-8 | public_connect_handlers 그룹에 등록됨 | `grep -c "public_connect_handlers\|PublicConnectHandler" api/cmd/internal/container/module_featured.go` | TODO 2 |
| A-9 | /featured/$orgSlug 라우트 파일 존재 | `test -f web/src/route/featured/\$orgSlug.tsx` | TODO 3 |
| A-10 | TypeScript 오류 증가 없음 (기준: 22개) | `cd web && npm run check 2>&1 \| grep -c "error"` | TODO Final |
| A-11 | __root.tsx에 featured 라우트 제외 조건 있음 | `grep -c "featured" web/src/route/__root.tsx` | TODO 3 |
| A-12 | refetchInterval 30000 설정됨 | `grep "refetchInterval\|30000\|30_000" web/src/feature/featured/hook/*.ts` | TODO 3 |
| A-13 | Web lint 통과 | `cd web && npm run lint` | TODO Final |
| A-14 | Vitest 통과 | `cd web && npm run test` | TODO Final |

### Human-Required (H-items)
| ID | Criterion | Reason | Review Material |
|----|-----------|--------|----------------|
| H-1 | 풀스크린 레이아웃 시각적 완성도 | 프로젝터/대형 모니터에서 확인 필요 | 브라우저에서 /featured/$orgSlug 접속 |
| H-2 | 비활성 3분 초과 시 빨간 깜빡임 동작 | 시간 경과 후 수동 확인 필요 | 3분 대기 후 브라우저 확인 |
| H-3 | 30초 폴링 시 자연스러운 데이터 갱신 | UI 깜빡임 없는 background refetch 확인 | 브라우저 실시간 관찰 |
| H-4 | 메트릭 수치 정확성 (특히 Agent Uptime) | 실제 세션 데이터와 대조 필요 | API 응답 vs DB 직접 쿼리 비교 |
| H-5 | Org slug 기반 접근 보안 수용성 | 비즈니스 정책 판단 | 실제 URL 공유 시 노출 범위 확인 |

### Verification Gaps
- Tier 3 E2E 테스트 없음 (Playwright/Cypress 미설치)
- Tier 4 Agent Sandbox 없음
- Integration 테스트는 MONGODB_URI 환경변수 필요 (없으면 skip)
- Web `npm run check`에 기존 22개 에러 존재 — 신규 에러 증가분만 추적

---

## External Dependencies Strategy

### Pre-work (user prepares before AI work)
| Dependency | Action | Command/Step | Blocking? |
|------------|--------|-------------|-----------|
| buf CLI | 설치 확인 | `buf --version` | Yes |
| MongoDB | Docker 실행 | `cd api && make dev` (또는 docker-compose up) | No (통합테스트 실행 시에만) |

### During (AI work strategy)
| Dependency | Dev Strategy | Rationale |
|------------|-------------|-----------|
| MongoDB | Go 빌드 + 유닛 테스트는 DB 불필요 | 통합테스트만 실제 DB 필요 |
| buf generate | TODO 1에서 실행 | Go/TS stub 필요 |

### Post-work (user actions after completion)
| Task | Related Dependency | Action | Command/Step |
|------|--------------------|--------|-------------|
| API 서버 기동 | MongoDB | dev 서버 실행 후 브라우저 확인 | `cd api && make dev` |
| 메트릭 정확성 확인 | MongoDB + 실제 세션 데이터 | H-4 수동 검증 | API 응답 vs mongosh 쿼리 비교 |

---

## Context

### Original Request
해커톤 참가자들의 프로젝트에 대해 상황판을 만들어 보여주기. Organization의 유저들이 각각 등록한 첫번째 프로젝트를 기준으로 토큰 사용량, 세션 상태를 리얼타임으로 업데이트. 오후 8시 기준부터 측정. 3분 이상 비활성 시 빨간 깜빡임.

### Interview Summary
**Key Discussions**:
- 인증: Public (로그인 불필요) — `/featured/$orgSlug` URL만으로 접근
- 프로젝트 선택: 멤버별 가장 먼저 등록된 프로젝트 (registered_at 기준 1개)
- 비활성 UX: 빨간 깜빡임 (3Hz 미만, WCAG 2.3.1 준수)
- 보안 게이트: Slug만으로 공개 (opt-in flag 없음)
- 아키텍처: 별도 FeaturedBoardService + FeaturedBoardGRPCHandler (기존 DashboardService는 private 유지)
- 데이터 인터페이스: v2 추상화 인터페이스만 사용 (provider-agnostic)
- 폴링 주기: 30초
- 시작 시간: since 파라미터 (int64 Unix epoch)

**Research Findings**:
- 기존 DashboardService는 PrivateConnectHandler로 등록됨 — interceptor가 handler-group 레벨로 적용되어 단일 RPC만 public으로 만들 수 없음
- `OrganizationRepositoryPort.GetBySlug`가 이미 존재 (`organization_repo.go:78`)
- v2 도메인: HumanMessage(prompt), AgentMessage.Usage(token), ToolExecution(tool calls), TreeNodeMeta.Timestamp(uptime)
- mongoschema에 필드명 상수 정의 완료 (SessionTypeField, SessionTimestampField 등)
- __root.tsx에서 경로 조건으로 사이드바 제외 패턴 존재

---

## Work Objectives

### Core Objective
Organization slug 기반으로 접근 가능한 해커톤 상황판을 구현하여, 멤버별 4개 메트릭(Prompt Submit Count, Token Usage, Tool Call Manifest, Agent Uptime)을 30초 폴링으로 실시간 표시한다.

### Concrete Deliverables
- `idl/protobuf/dashboard/v1/dashboard.proto` — GetFeaturedBoard RPC + FeaturedBoard 메시지 타입 추가
- `shared/gen/grpcstub/` + `web/src/gen/grpcstub/` — 생성된 Go/TS 스텁
- `api/internal/service/featured/` — FeaturedBoardService (서비스 + 핸들러 + 리포지토리)
- `api/cmd/internal/container/module_featured.go` — fx 모듈 등록
- `web/src/route/featured/$orgSlug.tsx` — Featured page 라우트
- `web/src/feature/featured/` — 컴포넌트 + 훅

### Definition of Done
- [ ] `/featured/$orgSlug` URL로 로그인 없이 접근 가능
- [ ] 참가자별 4개 메트릭이 카드에 표시됨
- [ ] 30초마다 자동 데이터 갱신
- [ ] `since` 파라미터로 특정 시간 이후 데이터만 집계
- [ ] 3분 이상 비활성 프로젝트 카드에 빨간 깜빡임 표시
- [ ] 사이드바 없는 풀스크린 레이아웃
- [ ] `go build ./api/... ./shared/...` 성공
- [ ] `cd web && npm run check` 기존 대비 에러 증가 없음
- [ ] 기존 Go 테스트 회귀 없음

### Must NOT Do (Guardrails)
- 기존 GetOverview 등 DashboardService RPC 수정 금지
- raw 데이터(v1 transcript) 사용 금지 — v2 추상화 인터페이스만 사용
- 기존 DashboardGRPCHandler 수정 금지 (별도 FeaturedBoardGRPCHandler)
- checkRBAC를 featured 핸들러에서 호출 금지 (public 엔드포인트)
- mongoschema 필드명 상수를 새로 정의하지 않기 (기존 상수 재사용)
- DashboardService 생성자 시그니처 변경 금지
- Web에서 별도 publicTransport 생성 금지 (기존 transport 사용)
- count-up 라이브러리 등 외부 의존성 추가 금지
- 깜빡임 주기 3Hz 이상 금지 (WCAG 2.3.1)

---

## Task Flow

```
TODO-1 (Proto + Generate) → TODO-2 (API Backend)  ─┐
                          → TODO-3 (Web Frontend) ─┤→ TODO-Final (Verification)
```

## Dependency Graph

| TODO | Requires (Inputs) | Produces (Outputs) | Type |
|------|-------------------|-------------------|------|
| 1 | - | `proto_path` (file), `go_stubs_dir` (file), `ts_stubs_dir` (file) | work |
| 2 | `todo-1.go_stubs_dir` | `featured_service_dir` (file), `featured_module_path` (file) | work |
| 3 | `todo-1.ts_stubs_dir` | `featured_route_path` (file), `featured_feature_dir` (file) | work |
| Final | all outputs | - | verification |

## Parallelization

| Group | TODOs | Reason |
|-------|-------|--------|
| A | TODO-2, TODO-3 | 둘 다 TODO-1의 생성된 스텁에만 의존하며 서로 독립적 (Go backend / React frontend) |

## Commit Strategy

| After TODO | Message | Files | Condition |
|------------|---------|-------|-----------|
| 1 | `feat(proto): add GetFeaturedBoard RPC definition` | `idl/protobuf/dashboard/v1/dashboard.proto`, `shared/gen/`, `web/src/gen/` | always |
| 2 | `feat(api): implement FeaturedBoard service` | `api/internal/service/featured/`, `api/cmd/internal/container/module_featured.go` | always |
| 3 | `feat(web): add hackathon featured page` | `web/src/route/featured/`, `web/src/feature/featured/`, `web/src/route/__root.tsx` | always |

> **Note**: No commit after Final (Verification does not modify source code).

## Error Handling

### Failure Categories

| Category | Examples | Detection Pattern |
|----------|----------|-------------------|
| `env_error` | buf CLI missing, MongoDB 미실행, 네트워크 타임아웃 | `/EACCES\|ECONNREFUSED\|timeout\|not found/i` |
| `code_error` | Go 컴파일 에러, TypeScript 타입 에러, lint 실패 | `/cannot\|TypeError\|SyntaxError\|lint\|test failed/i` |
| `scope_internal` | 생성된 stub과 코드 불일치, 누락된 proto 필드 | Worker `suggested_adaptation` 존재 여부 |
| `unknown` | 분류 불가 에러 | Default fallback |

### Failure Handling Flow

| Scenario | Action |
|----------|--------|
| work fails | Retry up to 2 times → Analyze → (see below) |
| verification fails | Analyze immediately (no retry) → (see below) |
| Worker times out | Halt and report |
| Missing Input | Skip dependent TODOs, halt |

### After Analyze

| Category | Action |
|----------|--------|
| `env_error` | Halt + log to `issues.md` |
| `code_error` | Create Fix Task (depth=1 limit) |
| `scope_internal` | Adapt → Dynamic TODO (depth=1) |
| `unknown` | Halt + log to `issues.md` |

## Runtime Contract

| Aspect | Specification |
|--------|---------------|
| Working Directory | Repository root (`/Users/hoyeonlee/team-attention/cops`) |
| Network Access | Allowed (buf registry, npm registry) |
| Package Install | Denied (use existing deps only) |
| File Access | Repository only |
| Max Execution Time | 5 minutes per TODO |
| Git Operations | Denied (Orchestrator handles) |

---

## TODOs

### [x] TODO 1: Define GetFeaturedBoard proto and generate stubs

**Type**: work

**Required Tools**: `buf`

**Inputs**: (none - first task)

**Outputs**:
- `proto_path` (file): `idl/protobuf/dashboard/v1/dashboard.proto` - Updated proto file with FeaturedBoard RPC
- `go_stubs_dir` (file): `shared/gen/grpcstub/dashboard/v1/` - Generated Go stubs
- `ts_stubs_dir` (file): `web/src/gen/grpcstub/dashboard/v1/` - Generated TypeScript stubs

**Steps**:
- [ ] Read existing `idl/protobuf/dashboard/v1/dashboard.proto` to understand current message/service structure
- [ ] Add new message types to `dashboard.proto`:
  - `GetFeaturedBoardReq`: `string org_slug = 1; int64 since_unix = 2;`
  - `GetFeaturedBoardRes`: `repeated FeaturedMember members = 1; string organization_name = 2;`
  - `FeaturedMember`: `string user_id = 1; string user_name = 2; string project_id = 3; string project_name = 4; TokenUsageSummary usage = 5; int32 prompt_count = 6; repeated ToolCallSummary tool_calls = 7; google.protobuf.Timestamp last_activity_at = 8;`
  - `ToolCallSummary`: `string tool_name = 1; int32 count = 2;`
- [ ] Add new service definition: `service FeaturedBoardService { rpc GetFeaturedBoard(GetFeaturedBoardReq) returns (GetFeaturedBoardRes); }`
- [ ] Reuse existing `TokenUsageSummary` message (do NOT duplicate)
- [ ] Run `cd idl/protobuf && buf generate` to generate Go and TypeScript stubs
- [ ] Verify generated files exist in `shared/gen/` and `web/src/gen/`

**Must NOT do**:
- Do not modify existing message types or service definitions in `DashboardService`
- Do not rename or reorder existing fields in any message
- Do not run git commands

**References**:
- `idl/protobuf/dashboard/v1/dashboard.proto:10-23` - TokenUsageSummary message (reuse)
- `idl/protobuf/dashboard/v1/dashboard.proto:323-344` - DashboardService definition (do not modify)
- `idl/protobuf/session/v1/session.proto:49-70` - TreeNodeMeta fields reference

**Acceptance Criteria**:

*Functional:*
- [x] `GetFeaturedBoardReq` message with `org_slug` and `since_unix` fields defined
- [x] `GetFeaturedBoardRes` message with `repeated FeaturedMember` field defined
- [x] `FeaturedBoardService` service with `GetFeaturedBoard` RPC defined
- [x] `TokenUsageSummary` reused (no new duplicate message)

*Static:*
- [x] `cd idl/protobuf && buf lint` → exit 0

*Runtime:*
- [x] `go build ./shared/...` → exit 0 (generated code compiles)

```yaml
Verify:
  acceptance:
    - given: ["dashboard.proto exists with existing DashboardService"]
      when: "FeaturedBoardService and messages are added and buf generate runs"
      then: ["Go stubs compile", "TS stubs generated", "existing DashboardService unchanged"]
  integration:
    - "buf generate produces valid Go and TypeScript code"
  commands:
    - run: "grep -c 'GetFeaturedBoard' idl/protobuf/dashboard/v1/dashboard.proto"
      expect: "exit 0 with count >= 1"
    - run: "cd idl/protobuf && buf lint"
      expect: "exit 0"
    - run: "go build ./shared/..."
      expect: "exit 0"
  risk: LOW
```

---

### [x] TODO 2: Implement FeaturedBoard API service

**Type**: work

**Required Tools**: (none)

**Inputs**:
- `go_stubs_dir` (file): `${todo-1.outputs.go_stubs_dir}` - Generated Go ConnectRPC stubs

**Outputs**:
- `featured_service_dir` (file): `api/internal/service/featured/` - FeaturedBoard service implementation
- `featured_module_path` (file): `api/cmd/internal/container/module_featured.go` - fx module registration

**Steps**:
- [ ] Read existing patterns:
  - `api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go` - repository port pattern
  - `api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go` - handler pattern
  - `api/internal/service/dashboard/dashboard_service.go` - service pattern
  - `api/cmd/internal/container/module_dashboard.go` - fx module pattern
  - `api/cmd/internal/container/register_connectrpc.go` - public handler registration pattern
  - `shared/domain/mongoschema/session.go` - field name constants
- [ ] Create `api/internal/service/featured/featured_service.go`:
  - `FeaturedBoardService` struct with `orgRepo OrganizationRepositoryPort` and `dashboardRepo FeaturedBoardRepositoryPort`
  - `GetFeaturedBoard(ctx, orgSlug, sinceUnix)` method:
    1. orgRepo.GetBySlug(slug) → org (return NotFound if nil)
    2. Get org members list from org.Members
    3. For each member, get their earliest registered project (registered_at ASC, limit 1)
    4. dashboardRepo.GetFeaturedBoardStats(ctx, projectIDs, memberIDs, since) → per-member metrics
    5. Return assembled response
- [ ] Create `api/internal/service/featured/outbound/repository/featured_repo_port.go`:
  - `FeaturedBoardRepositoryPort` interface with `GetFeaturedBoardStats` method
  - `FeaturedMemberStats` struct: UserID, ProjectID, PromptCount, Usage (TokenUsageSummary), ToolCalls (map[string]int32), LastActivityAt
- [ ] Create `api/internal/service/featured/outbound/repository/mongodb/featured_repo.go`:
  - MongoDB aggregation pipeline:
    1. `$match`: projectId in projectIDs AND timestamp >= since
    2. `$facet` for parallel aggregations:
       - prompts: `$match` type="human", `$group` by projectId, count
       - tokens: `$match` type="agent", `$group` by projectId, sum usage fields
       - tools: `$match` type="tool_execution", `$group` by {projectId, toolName}, count
       - activity: `$group` by projectId, `$max` timestamp
  - Use existing mongoschema field constants (SessionTypeField, SessionTimestampField, etc.)
- [ ] Create `api/internal/service/featured/inbound/grpc/connectrpc/handler.go`:
  - `FeaturedBoardGRPCHandler` struct implementing generated `FeaturedBoardServiceHandler`
  - `GetFeaturedBoard` method: validate org_slug not empty, call service, convert to proto response
  - NO RBAC check (public endpoint)
- [ ] Create `api/cmd/internal/container/module_featured.go`:
  - fx module providing FeaturedBoardService, FeaturedBoardRepository, FeaturedBoardGRPCHandler
  - Register handler as `PublicConnectHandler` (fx.ResultTags `group:"public_connect_handlers"`)
- [ ] Register module in `api/cmd/internal/container/application.go`

**Must NOT do**:
- Do not modify existing DashboardService, DashboardGRPCHandler, or DashboardRepositoryPort
- Do not modify DashboardService constructor signature
- Do not call checkRBAC from the featured handler (public endpoint, no JWT)
- Do not redefine mongoschema field name constants (import and reuse)
- Do not create a separate publicTransport
- Do not install new Go packages
- Do not run git commands

**References**:
- `api/internal/service/dashboard/inbound/grpc/connectrpc/handler.go:37-71` - ConnectRPC handler pattern
- `api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go:12-77` - Repository port + aggregation structs
- `api/internal/service/dashboard/dashboard_service.go` - Service pattern (skip RBAC for featured)
- `api/cmd/internal/container/module_dashboard.go` - fx module pattern
- `api/cmd/internal/container/register_connectrpc.go:44-60` - PublicConnectHandler group tagging
- `api/internal/service/organization/outbound/repository/mongodb/organization_repo.go:78` - GetBySlug method
- `shared/domain/mongoschema/session.go` - Field name constants (SessionTypeField, etc.)
- `shared/domain/v2/session_agent.go:54-65` - TokenUsage fields
- `shared/domain/v2/session_tool.go:23-45` - ToolExecution fields
- `shared/domain/v2/session_human.go:1-32` - HumanMessage type
- `shared/domain/v2/session.go:68-81` - TreeNodeMeta.Timestamp

**Acceptance Criteria**:

*Functional:*
- [x] `FeaturedBoardGRPCHandler` implements generated `FeaturedBoardServiceHandler` interface
- [x] Empty org_slug returns ConnectRPC `CodeInvalidArgument`
- [x] Non-existent slug returns ConnectRPC `CodeNotFound`
- [x] Handler registered in `public_connect_handlers` group (no JWT interceptor)

*Static:*
- [x] `go build ./api/...` → exit 0
- [x] `go vet ./api/internal/service/featured/...` → exit 0

*Runtime:*
- [x] `go test ./api/... ./shared/...` → all pass (no regression)

```yaml
Verify:
  acceptance:
    - given: ["Generated Go stubs for FeaturedBoardService exist"]
      when: "API service, handler, repository, and module are implemented"
      then: ["Go builds", "handler implements interface", "public group registration"]
    - given: ["FeaturedBoardGRPCHandler is registered"]
      when: "Request with empty org_slug"
      then: ["Returns CodeInvalidArgument"]
  integration:
    - "FeaturedBoardService calls OrganizationRepositoryPort.GetBySlug"
    - "FeaturedBoardService calls FeaturedBoardRepositoryPort.GetFeaturedBoardStats"
    - "MongoDB aggregation uses existing mongoschema field constants"
  commands:
    - run: "go build ./api/..."
      expect: "exit 0"
    - run: "go vet ./api/internal/service/featured/..."
      expect: "exit 0"
    - run: "go test ./api/... ./shared/..."
      expect: "exit 0"
  risk: MEDIUM
```

---

### [x] TODO 3: Implement Web featured page

**Type**: work

**Required Tools**: (none)

**Inputs**:
- `ts_stubs_dir` (file): `${todo-1.outputs.ts_stubs_dir}` - Generated TypeScript ConnectRPC stubs

**Outputs**:
- `featured_route_path` (file): `web/src/route/featured/$orgSlug.tsx` - Featured page route
- `featured_feature_dir` (file): `web/src/feature/featured/` - Components and hooks

**Steps**:
- [ ] Read existing patterns:
  - `web/src/route/__root.tsx` - sidebar exclusion pattern
  - `web/src/route/dashboard.tsx` - route loader pattern
  - `web/src/feature/dashboard/hook/use-get-overview.ts` - ConnectRPC query hook pattern
  - `web/src/feature/dashboard/component/overview-stats.tsx` - StatCard component pattern
  - `web/src/feature/project/component/project-header.tsx:131-136` - animate-pulse dot pattern
  - `web/src/shared/util/format.ts` - formatTokenCount utility
- [ ] Create `web/src/feature/featured/hook/use-get-featured-board.ts`:
  - `useGetFeaturedBoard({ orgSlug, sinceUnix })` hook
  - Use `useQuery` from `@connectrpc/connect-query` with generated `getFeaturedBoard` descriptor
  - Set `refetchInterval: 30_000` for 30-second polling
  - Set `enabled: !!orgSlug`
- [ ] Create `web/src/feature/featured/component/featured-board.tsx`:
  - Main board component receiving members data
  - Grid layout: `grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4`
  - Map each member to `<MemberCard />`
- [ ] Create `web/src/feature/featured/component/member-card.tsx`:
  - Props: member data (name, project, metrics, last_activity_at)
  - 4 metric displays: Prompt Count, Token Usage (formatTokenCount), Tool Calls (top 5), Uptime
  - Uptime: calculate `now - last_activity_at` client-side, display as "Xm ago" or "Active"
  - Inactive state (diff > 3 min): card background with red blinking animation
    - Use CSS `animate-pulse` variant with `bg-red-500/10` → `bg-red-500/30` at < 3Hz
    - Or `@keyframes blink` with `animation-duration: 1s` (1Hz, well within WCAG 3Hz limit)
  - Active state: subtle cyan left border + green pulse dot (emerald-400)
  - Use dark theme: zinc-950 background, cyan/violet accents (match existing dashboard)
- [ ] Create `web/src/route/featured/$orgSlug.tsx`:
  - TanStack Router route component
  - Extract `orgSlug` from route params
  - Extract `since` from search params (`?since=1740700800`), default to today 8PM (20:00 local)
  - Render `<FeaturedBoard />` with full-screen layout (no sidebar, no header)
  - Error state: show org not found message
  - Loading state: skeleton cards
- [ ] Update `web/src/route/__root.tsx`:
  - Add `const isFeaturedRoute = pathname.startsWith('/featured')`
  - Add to condition: `if (isAuthRoute || isOrganizationNewRoute || isInviteRoute || isFeaturedRoute)`
  - This renders featured route without sidebar

**Must NOT do**:
- Do not modify existing dashboard components or hooks
- Do not add external npm dependencies (count-up libs, animation libs)
- Do not create a separate ConnectRPC transport for public endpoints
- Do not add auth guards to the featured route
- Do not use `animate-blink` or any animation > 3Hz (WCAG 2.3.1)
- Do not run git commands

**References**:
- `web/src/route/__root.tsx:25-48` - Sidebar exclusion pattern (add isFeaturedRoute)
- `web/src/feature/dashboard/hook/use-get-overview.ts:1-19` - ConnectRPC query hook (add refetchInterval)
- `web/src/feature/dashboard/component/overview-stats.tsx:1-144` - StatCard + formatTokenCount pattern
- `web/src/feature/project/component/project-header.tsx:131-136` - animate-pulse emerald dot
- `web/src/shared/util/format.ts` - formatTokenCount helper
- `web/src/route/dashboard.tsx` - Route with loader pattern
- `web/src/shared/component/app-sidebar.tsx` - Sidebar (NOT used in featured)

**Acceptance Criteria**:

*Functional:*
- [x] Route file `web/src/route/featured/$orgSlug.tsx` exists
- [x] Hook file `web/src/feature/featured/hook/use-get-featured-board.ts` exists with `refetchInterval: 30_000`
- [x] `__root.tsx` excludes `/featured` from sidebar layout
- [x] MemberCard renders 4 metrics: prompt count, token usage, tool calls, uptime
- [x] Inactive (> 3min) cards have red blinking animation (< 3Hz)
- [x] Default `since` is today 20:00 when query param absent

*Static:*
- [x] `cd web && npm run check` → no new errors beyond baseline (22)
- [x] `cd web && npm run lint` → passes for new files

*Runtime:*
- [x] `cd web && npm run test` → passes

```yaml
Verify:
  acceptance:
    - given: ["Generated TS stubs for FeaturedBoardService exist"]
      when: "Navigate to /featured/test-org"
      then: ["Page renders without sidebar", "Query hook fires with orgSlug and since params"]
    - given: ["Member has last_activity_at > 3 minutes ago"]
      when: "Featured board renders member card"
      then: ["Card background shows red blinking animation at < 3Hz"]
  integration:
    - "useGetFeaturedBoard hook calls generated getFeaturedBoard with correct params"
    - "__root.tsx correctly excludes /featured routes from sidebar layout"
  commands:
    - run: "test -f web/src/route/featured/\\$orgSlug.tsx"
      expect: "exit 0"
    - run: "grep -c 'refetchInterval' web/src/feature/featured/hook/use-get-featured-board.ts"
      expect: "exit 0 with count >= 1"
    - run: "grep -c 'featured' web/src/route/__root.tsx"
      expect: "exit 0 with count >= 1"
    - run: "cd web && npm run lint"
      expect: "exit 0"
  risk: LOW
```

---

### [x] TODO Final: Verification

**Type**: verification

**Required Tools**: `go`, `buf`, `npm`

**Inputs**:
- `proto_path` (file): `${todo-1.outputs.proto_path}`
- `featured_service_dir` (file): `${todo-2.outputs.featured_service_dir}`
- `featured_module_path` (file): `${todo-2.outputs.featured_module_path}`
- `featured_route_path` (file): `${todo-3.outputs.featured_route_path}`
- `featured_feature_dir` (file): `${todo-3.outputs.featured_feature_dir}`

**Outputs**: (none)

**Steps**:
- [ ] Verify all deliverables from Work Objectives exist
- [ ] Run Go build for all modules
- [ ] Run Go vet for featured service
- [ ] Run all Go tests (regression check)
- [ ] Run Web TypeScript check (compare error count to baseline 22)
- [ ] Run Web lint
- [ ] Run Web tests (Vitest)
- [ ] Verify FeaturedBoardGRPCHandler is registered as PublicConnectHandler
- [ ] Verify __root.tsx excludes featured route from sidebar
- [ ] Verify refetchInterval is set to 30000

**Must NOT do**:
- Do not use Edit or Write tools (source code modification forbidden)
- Do not add new features or fix errors (report only)
- Do not run git commands
- Bash is allowed for: running tests, builds, type checks
- Do not modify repo files via Bash (no `sed -i`, `echo >`, etc.)

**Acceptance Criteria**:

*Functional:*
- [x] Proto: `grep -c "FeaturedBoardService" idl/protobuf/dashboard/v1/dashboard.proto` → >= 1
- [x] Go stubs: `grep -c "GetFeaturedBoard" shared/gen/grpcstub/dashboard/v1/dashboardv1connect/dashboard.connect.go` → >= 1
- [x] API service dir: `test -d api/internal/service/featured` → exit 0
- [x] fx module: `grep -c "public_connect_handlers" api/cmd/internal/container/module_featured.go` → >= 1
- [x] Web route: `test -f web/src/route/featured/\$orgSlug.tsx` → exit 0
- [x] Sidebar exclusion: `grep -c "featured" web/src/route/__root.tsx` → >= 1
- [x] Polling: `grep "30.000\|30_000\|refetchInterval" web/src/feature/featured/hook/use-get-featured-board.ts` → matches

*Static:*
- [x] `go build ./api/... ./shared/...` → exit 0
- [x] `go vet ./api/...` → exit 0
- [x] `cd web && npm run check 2>&1 | grep -c "error"` → <= 22 (no increase from baseline)
- [x] `cd web && npm run lint` → exit 0

*Runtime:*
- [x] `go test ./api/... ./shared/... ./daemon/... ./cli/...` → all pass
- [x] `cd web && npm run test` → all pass

```yaml
Verify:
  acceptance:
    - given: ["All TODOs completed"]
      when: "Full verification suite runs"
      then: ["Go builds", "Go tests pass", "Web type check stable", "Web lint pass", "Web tests pass"]
  commands:
    - run: "go build ./api/... ./shared/..."
      expect: "exit 0"
    - run: "go test ./api/... ./shared/... ./daemon/... ./cli/..."
      expect: "exit 0"
    - run: "cd web && npm run lint"
      expect: "exit 0"
    - run: "cd web && npm run test"
      expect: "exit 0"
  risk: LOW
```
