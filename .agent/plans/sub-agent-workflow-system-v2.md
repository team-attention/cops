# Sub-Agent Workflow System Implementation Plan v2

## Overview

Research → Planning → Execute → Review → Commit-PR 워크플로우를 Sub-Agent 기반으로 구현합니다.
- **Sub-Agents**: Task tool로 호출되는 전문화된 AI 어시스턴트
- **Skills**: 사용자가 직접 호출하는 워크플로우 오케스트레이터
- **Artifacts**: 각 단계의 출력물을 `.agent/artifact/{timestamp}/`에 저장

## Directory Structure

```
.agent/
├── agents/                    # Sub-agent 정의 (YAML frontmatter + Markdown)
│   ├── research.md           # Research 전문가
│   ├── planning.md           # Planning 전문가
│   ├── execute.md            # Implementation 전문가
│   ├── review.md             # Code review 전문가
│   └── revise.md             # Revision 전문가
│
├── skills/                    # Skill 정의 (YAML frontmatter + Markdown)
│   └── full-cycle/
│       └── SKILL.md          # Orchestrator: 전체 워크플로우 조율
│
├── artifact/                  # Artifact 저장소
│   └── {timestamp}/          # 예: 20251225-143022/
│       ├── 01-research.md
│       ├── 02-plan.md
│       ├── 03-review.md
│       └── 04-commit-pr.md
│
├── workflows/                 # 기존 workflows (점진적으로 skills로 마이그레이션)
│   ├── commit-pr.md
│   ├── init-worktree.md
│   └── ...
│
└── rules/                     # 기존 rules (그대로 유지)
    └── ...

.claude/
├── agents -> ../.agent/agents     # 심볼릭 링크
├── skills -> ../.agent/skills     # 심볼릭 링크
├── commands -> ../.agent/workflows # 기존 심볼릭 링크
├── rules -> ../.agent/rules       # 기존 심볼릭 링크
└── memory -> ../.agent/memory     # 기존 심볼릭 링크
```

## Key Design Decisions

1. **Sub-Agent vs Skill**:
   - **Sub-Agents** (`.agent/agents/*.md`): Task tool로 호출, 독립적인 context window
   - **Skills** (`.agent/skills/*/SKILL.md`): 사용자가 `/skill-name` 호출

2. **Artifact 전달 방식**:
   - 각 Sub-Agent는 artifact 경로를 프롬프트로 전달받음
   - 예: "Read `.agent/artifact/20251225-143022/01-research.md` and create plan..."

3. **Review-Revise Cycle**:
   - Review에서 FAIL 판정 시 Revise → Review 재실행
   - 최대 3회 반복 (무한 루프 방지)

## Artifact Structure

```
.agent/artifact/{timestamp}/
  ├── 01-research.md          # Research Agent 출력
  ├── 02-plan.md              # Planning Agent 출력
  ├── 03-review.md            # Review Agent 출력 (PASS/FAIL + 수정 지시사항)
  └── 04-commit-pr.md         # Commit & PR 정보
```

**Artifact ID Format**: `YYYYMMDD-HHMMSS` (예: `20251225-143022`)

**Note**: Execute/Revise는 artifact 파일을 생성하지 않음 (실제 코드만 수정)

---

## Implementation Tasks

### Task 1: Research Sub-Agent
**File**: `.agent/agents/research.md`
**Type**: Sub-Agent (Task tool로 호출)
**Description**: 코드베이스 분석 및 Linear 티켓 정보 수집

**Configuration**:
```yaml
---
name: research
description: Codebase research and Linear ticket analysis expert. Use proactively to gather context before planning or implementation.
tools: Read, Glob, Grep, Bash, mcp__linear__get_issue, mcp__linear__list_issues, mcp__context7__resolve-library-id, mcp__context7__get-library-docs
model: sonnet
---
```

**Responsibilities**:
- Linear 티켓 정보 수집 (있는 경우)
- 관련 코드베이스 탐색
- `.agent/rules/` 규칙 파악
- 유사 구현 예제 찾기 (file:line references)
- 기술적 제약사항 파악
- Context7로 최신 라이브러리 문서 검색

**Input**: User request, optional Issue ID, artifact path
**Output**: `.agent/artifact/{ARTIFACT_ID}/01-research.md`

---

### Task 2: Planning Sub-Agent
**File**: `.agent/agents/planning.md`
**Type**: Sub-Agent (Task tool로 호출)
**Description**: Research 결과를 기반으로 구현 계획 작성

**Configuration**:
```yaml
---
name: planning
description: Software architect creating detailed implementation plans. Use after research phase to design implementation strategy.
tools: Read, Write
model: sonnet
---
```

**Responsibilities**:
- Research 결과 분석 (01-research.md 읽기)
- 아키텍처 결정
- 구현 단계 breakdown
- 파일별 변경 사항 명세
- 테스트 계획

**Input**: `.agent/artifact/{ARTIFACT_ID}/01-research.md`
**Output**: `.agent/artifact/{ARTIFACT_ID}/02-plan.md`

---

### Task 3: Execute Sub-Agent
**File**: `.agent/agents/execute.md`
**Type**: Sub-Agent (Task tool로 호출)
**Description**: Planning 결과를 기반으로 실제 코드 구현

**Configuration**:
```yaml
---
name: execute
description: Implementation specialist executing detailed plans. Use to implement code according to plan.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
---
```

**Responsibilities**:
- 계획에 따른 코드 구현
- 테스트 작성 및 실행
- 빌드 확인
- 규칙 준수

**Input**: `.agent/artifact/{ARTIFACT_ID}/02-plan.md`
**Output**: 실제 코드 변경 (artifact 파일 없음)

---

### Task 4: Review Sub-Agent
**File**: `.agent/agents/review.md`
**Type**: Sub-Agent (Task tool로 호출)
**Description**: 구현 결과 검증 및 코드 리뷰

**Configuration**:
```yaml
---
name: review
description: Code review specialist verifying implementation quality. Use proactively after code implementation.
tools: Read, Grep, Glob, Bash
model: sonnet
---
```

**Responsibilities**:
- 코드 품질 검증 (`.agent/rules/` 준수 여부)
- 테스트 커버리지 확인
- 보안 취약점 검토
- git diff 분석
- PASS/FAIL 판정 및 수정 지시사항 작성

**Input**: `.agent/artifact/{ARTIFACT_ID}/02-plan.md`, git diff
**Output**: `.agent/artifact/{ARTIFACT_ID}/03-review.md`

**Output Format**:
```markdown
# Review Result

## Status
PASS | FAIL

## Issues Found
(if FAIL)
- Issue 1: description
- Issue 2: description

## Required Changes
(if FAIL)
- Change 1
- Change 2

## Approval Notes
(if PASS)
- Quality verified
- Ready for commit
```

---

### Task 5: Revise Sub-Agent
**File**: `.agent/agents/revise.md`
**Type**: Sub-Agent (Task tool로 호출)
**Description**: Review 결과를 기반으로 코드 수정

**Configuration**:
```yaml
---
name: revise
description: Code revision specialist fixing review issues. Use when review fails to address feedback.
tools: Read, Edit, Write, Bash, Glob, Grep
model: sonnet
---
```

**Responsibilities**:
- Review 피드백 반영 (03-review.md 읽기)
- 코드 수정
- 테스트 재실행
- 빌드 재확인

**Input**: `.agent/artifact/{ARTIFACT_ID}/03-review.md`
**Output**: 실제 코드 변경 (artifact 파일 없음)

---

### Task 6: Full-Cycle Skill (Orchestrator)
**File**: `.agent/skills/full-cycle/SKILL.md`
**Type**: Skill (사용자가 `/full-cycle` 호출)
**Description**: 전체 워크플로우를 조율하는 오케스트레이터

**Configuration**:
```yaml
---
name: full-cycle
description: Execute complete development cycle from research to PR. Use when implementing features end-to-end with Linear tickets.
---
```

**Workflow**:
```
1. Create Artifact Directory
   ↓
2. Invoke Research Sub-Agent (Task tool)
   → writes 01-research.md
   ↓
3. Invoke Planning Sub-Agent (Task tool)
   → reads 01-research.md
   → writes 02-plan.md
   ↓
4. Invoke Execute Sub-Agent (Task tool)
   → reads 02-plan.md
   → implements code
   ↓
5. Review-Revise Loop (max 3 iterations):
   ├─ Invoke Review Sub-Agent
   │  → reads 02-plan.md, git diff
   │  → writes 03-review.md
   │
   ├─ If FAIL:
   │  ├─ Invoke Revise Sub-Agent
   │  │  → reads 03-review.md
   │  │  → fixes code
   │  └─ Repeat Review
   │
   └─ If PASS:
      → Continue to Commit-PR
   ↓
6. Invoke Commit-PR Skill
   → reads all artifacts
   → creates commit & PR
   → writes 04-commit-pr.md
```

**Responsibilities**:
- Artifact 디렉토리 생성 (`date +%Y%m%d-%H%M%S`)
- 각 단계 순차 실행
- Review-Revise 사이클 관리 (최대 3회)
- 에러 처리 및 재시도
- 최종 결과 보고

**Arguments**:
- `$ARGUMENTS`: User request or Linear ticket ID

**Example Usage**:
```bash
# With user request
/full-cycle "implement user authentication"

# With Linear ticket
/full-cycle TA-123
```

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
                              │  Revise │◀────────┘ (if FAIL)
                              └─────────┘

                        Review-Revise Cycle (max 3회)
```

---

## Execution Order

각 태스크는 독립적으로 구현 가능합니다:

1. **Task 1**: Research Sub-Agent (`.agent/agents/research.md`)
2. **Task 2**: Planning Sub-Agent (`.agent/agents/planning.md`)
3. **Task 3**: Execute Sub-Agent (`.agent/agents/execute.md`)
4. **Task 4**: Review Sub-Agent (`.agent/agents/review.md`)
5. **Task 5**: Revise Sub-Agent (`.agent/agents/revise.md`)
6. **Task 6**: Full-Cycle Skill (`.agent/skills/full-cycle/SKILL.md`)

---

## Migration Notes

### 기존 Workflows

기존 `.agent/workflows/*.md` 파일들은 점진적으로 `.agent/skills/`로 마이그레이션:
- `commit-pr.md` → `.agent/skills/commit-pr/SKILL.md`
- `init-worktree.md` → `.agent/skills/init-worktree/SKILL.md`
- `plan-work.md` → `.agent/skills/plan-work/SKILL.md`

### Symlink 구조

```bash
.claude/agents -> ../.agent/agents     # Sub-agents 링크
.claude/skills -> ../.agent/skills     # Skills 링크
.claude/commands -> ../.agent/workflows # 기존 workflows (deprecated)
```

---

## Best Practices

1. **Sub-Agent 프롬프트**:
   - 명확한 역할 정의
   - 구체적인 입력/출력 명세
   - 예제 포함

2. **Artifact 경로 전달**:
   ```
   Use Task tool with:
   - subagent_type: "research"
   - prompt: "Research the codebase for authentication. Save results to .agent/artifact/20251225-143022/01-research.md"
   ```

3. **Review-Revise 사이클**:
   - 명확한 PASS/FAIL 기준
   - 구체적인 수정 지시사항
   - 최대 반복 횟수 제한 (3회)

4. **Tool Permissions**:
   - Research/Planning: Read-only tools
   - Execute/Revise: Write/Edit tools
   - Review: Read + Bash (git diff)

---

## Testing Strategy

각 Sub-Agent를 개별적으로 테스트:

1. **Research**: Linear 티켓 + 코드베이스 분석
2. **Planning**: Research artifact → 구현 계획
3. **Execute**: Plan artifact → 실제 코드 구현
4. **Review**: git diff → PASS/FAIL 판정
5. **Revise**: Review artifact → 코드 수정
6. **Full-Cycle**: 전체 워크플로우 end-to-end 테스트

---

## Next Steps

1. Task 1부터 순차적으로 구현
2. 각 Sub-Agent 개별 테스트
3. Full-Cycle Skill로 통합 테스트
4. 기존 workflows를 skills로 마이그레이션
