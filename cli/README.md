# C-Ops

Claude Code Ops

## Features
- 유저가 지정한 Repository에서 Claude Code의 세션 기록을 모두 Recording하고 가시적으로 보여주는 서비스

## CLI

### cops add [directory]
프로젝트의 Session 트래킹을 시작. 다음 동작을 수행해야 함.

1. Git이 사용되는 프로젝트인지 체크
  - 맞다면 main 브랜치 경로를 Global Config에 프로젝트 디렉토리로 기록 (`gitProject: true`)
  - 아니라면 현재 디렉토리 경로를 프로젝트 디렉토리로 기록 (`gitProject: false`)
2. 디렉토리에 Local Config 세팅이 있는지 확인
  - 있다면 생략 (Fetch 받은 것이거나 Worktree인 경우로 가정)
  - 없다면 UUID를 부여해 Local Config로 저장 (`.cops/config.json`)
3. Global Config에 프로젝트 ID로 등록된 프로젝트 경로가 있는지 확인
   - 있다면 생략 (이미 설정된 적 있음)
   - 없다면 Global Config에 프로젝트 ID - 프로젝트 디렉토리 기록

#### Parameters
- `--no-git`: 초기화 과정에 이 디렉토리엔 Git이 없다고 가정하고 진행
- `--sync/-s`: 이 디렉토리에 저장된 모든 세션을 한 차례 Sync함. 이때 Collector가 없다면 에러를 보여줌

### cops list
등록된 프로젝트 목록을 테이블로 표시. Git 프로젝트인 경우 Worktree도 함께 표시됨.