# Sub-Agent Workflow System Implementation Plan

## Overview

Research -> Planning -> Execute Plan -> Review -> Commit & PR 워크플로우를 서브에이전트 기반으로 구현합니다. 각 단계의 Artifact는 `.agent/artifact/{timestamp}/`에 저장됩니다.

## Key Design Decisions

1. **Artifact 전달 방식**: 각 Agent 프롬프트에 artifact 경로를 직접 명시
2. **Review-Revise Cycle**: Review에서 문제 발견 시 Execute -> Review 사이클 반복

## Artifact Structure

```
.agent/artifact/{timestamp}/
  ├── 01-research.md          # Research Agent 출력
  ├── 02-plan.md              # Planning Agent 출력
  ├── 03-review.md            # Review Agent 출력 (pass/fail + 수정 지시사항)
  └── 04-commit-pr.md         # Commit & PR 정보
```

**Artifact ID Format**: `YYYYMMDD-HHMMSS` (e.g., `20251224-143022`)

**Note**: Execute/Revise는 artifact 파일을 생성하지 않음 (실제 코드만 수정)

---

## Implementation Tasks

### Task 1: Artifact Directory Setup
**File**: N/A (Bash command in orchestrator)
**Description**: Artifact 디렉토리 생성 및 metadata.json 초기화

```bash
ARTIFACT_ID=$(date +%Y%m%d-%H%M%S)
ARTIFACT_DIR=".agent/artifact/${ARTIFACT_ID}"
mkdir -p "$ARTIFACT_DIR"
```

### Task 2: Research Agent Workflow
**File**: `.agent/workflows/research.md`
**Description**: 코드베이스 분석 및 Linear 티켓 정보 수집

**Input**: User request, optional Issue ID
**Output**: `01-research.md`

**Responsibilities**:
- Linear 티켓 정보 수집 (있는 경우)
- 관련 코드베이스 탐색
- `.agent/rules/` 규칙 파악
- 유사 구현 예제 찾기
- 기술적 제약사항 파악

---

### Task 3: Planning Agent Workflow
**File**: `.agent/workflows/planning.md`
**Description**: Research 결과를 기반으로 구현 계획 작성

**Input**: `01-research.md`
**Output**: `02-plan.md`

**Responsibilities**:
- Research 결과 분석
- 아키텍처 결정
- 구현 단계 breakdown
- 파일별 변경 사항 명세
- 테스트 계획

---

### Task 4: Execute Agent Workflow
**File**: `.agent/workflows/execute.md`
**Description**: Planning 결과를 기반으로 실제 코드 구현

**Input**: `02-plan.md`
**Output**: 실제 코드 변경 (artifact 파일 없음)

**Responsibilities**:
- 계획에 따른 코드 구현
- 테스트 작성 및 실행
- 빌드 확인

---

### Task 5: Review Agent Workflow
**File**: `.agent/workflows/review.md`
**Description**: 구현 결과 검증 및 코드 리뷰

**Input**: `02-plan.md`, git diff (현재 변경사항)
**Output**: `03-review.md`

**Responsibilities**:
- 코드 품질 검증 (rules 준수 여부)
- 테스트 커버리지 확인
- 보안 취약점 검토
- pass/fail 판정 및 수정 지시사항 작성

---

### Task 6: Revise Agent Workflow (NEW)
**File**: `.agent/workflows/revise.md`
**Description**: Review 결과를 기반으로 코드 수정

**Input**: `03-review.md`
**Output**: 실제 코드 변경 (artifact 파일 없음)

**Responsibilities**:
- Review 피드백 반영
- 코드 수정
- 테스트 재실행

---

### Task 7: Commit-PR Agent Workflow (Update)
**File**: `.agent/workflows/commit-pr.md` (기존 파일 수정)
**Description**: 모든 Artifact를 읽고 커밋 및 PR 생성

**Input**: All artifacts (`01-research.md` ~ `03-review.md`)
**Output**: `04-commit-pr.md`

**Responsibilities**:
- 커밋 메시지 생성
- PR 생성
- Linear 티켓 업데이트
- 결과 기록

---

### Task 8: Orchestrator Workflow
**File**: `.agent/workflows/full-cycle.md`
**Description**: 전체 워크플로우를 조율하는 메인 워크플로우

**Responsibilities**:
- Artifact 디렉토리 생성
- 각 단계 순차 실행
- Review-Revise 사이클 관리
- 에러 처리 및 재시도

---

## Workflow Diagram

```
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
│ Research │──▶│ Planning │──▶│ Execute  │──▶│  Review  │──▶│Commit-PR │
└──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘
     │              │              ▲              │                │
     ▼              ▼              │              ▼                ▼
01-research.md  02-plan.md         │         03-review.md    04-commit-pr.md
                                   │              │
                              ┌────┴────┐         │
                              │  Revise │◀────────┘ (if fail)
                              └─────────┘
```

**Review-Revise Cycle**:
1. Review가 `03-review.md` 작성 (pass/fail + 수정 지시사항)
2. fail인 경우: Revise가 `03-review.md` 읽고 코드 수정 → Review 재실행
3. pass인 경우: Commit-PR 진행
4. 최대 3회 반복 (무한 루프 방지)

---

## Execution Order

각 태스크는 독립적으로 구현 가능합니다. Context clear 후 다음 순서로 진행:

1. **Task 2**: Research Agent
2. **Task 3**: Planning Agent
3. **Task 4**: Execute Agent
4. **Task 5**: Review Agent
5. **Task 6**: Revise Agent
6. **Task 7**: Commit-PR Agent (기존 파일 수정)
7. **Task 8**: Orchestrator (전체 조율, Review-Revise 사이클 포함)
8. **Task 1**: Artifact Setup (Orchestrator에 통합)

---

## Notes

- **Artifact 경로 전달**: 각 Agent 프롬프트에 artifact 경로를 직접 명시
  ```
  예: "Read .agent/artifact/20251224-143022/01-research.md and create plan..."
  ```
- Execute/Revise는 artifact 파일을 생성하지 않음 (실제 코드만 수정)
- Review Agent는 git diff로 현재 변경사항 확인
- Review-Revise 사이클은 최대 3회까지 반복 (무한 루프 방지)
