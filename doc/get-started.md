# 시작하기 (Get Started)

C-Ops는 당신의 AI 코딩 파트너 Claude Code의 활동을 시각화하고 인사이트를 제공합니다.
이 가이드는 C-Ops를 설치하고 첫 프로젝트를 등록하는 과정을 안내합니다.

## 1. 설치 (Installation)

### 자동 설치 (권장)

다음 명령어로 CLI와 데몬 서비스를 한 번에 설치할 수 있습니다:

```bash
curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | bash
```

설치 스크립트는 다음 작업을 자동으로 수행합니다:
- 현재 플랫폼(OS/아키텍처) 감지
- 최신 바이너리 다운로드 및 설치 (`~/.cops/bin/`)
- PATH 환경변수 설정
- 데몬 서비스 자동 등록 및 시작

### 지원 플랫폼
- macOS (Intel & Apple Silicon)
- Linux (x86_64 & ARM64)

### 특정 버전 설치

특정 버전을 설치하려면 `COPS_VERSION` 환경변수를 사용하세요:

```bash
COPS_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | bash
```

### 설치 후 확인

설치가 완료되면 터미널을 재시작하거나 다음 명령어를 실행하세요:

```bash
source ~/.zprofile  # zsh 사용자
# 또는
source ~/.bash_profile  # bash 사용자
```

버전 확인:

```bash
cops version
cops-daemon version
```

## 2. 서버 연결 (Setup)

설치 후, 팀의 C-Ops 서버 주소를 설정합니다. 복잡한 환경 변수 대신 CLI로 간단히 설정하세요.

```bash
cops config --server "https://cops.team-attention.com"
```

연결 상태를 확인합니다:

```bash
cops ping
# Output: ✅ Connected to https://cops.team-attention.com (v1.0.0)
```

## 3. 첫 프로젝트 등록 (Onboarding)

이제 당신이 작업 중인 프로젝트를 C-Ops에게 알려주세요.

1.  **프로젝트 이동**:
    ```bash
    cd ~/workspace/my-awesome-project
    ```

2.  **프로젝트 추가**:
    `--sync` 옵션을 사용하면 기존의 대화 기록까지 서버로 안전하게 업로드됩니다.
    ```bash
    cops add . --sync
    ```
    > **Tip**: Git Root가 아닌 하위 디렉토리에서 실행해도, 자동으로 Git Root를 찾아 프로젝트로 등록합니다.

3.  **Wow Moment 확인**:
    대시보드(예: `https://cops.team-attention.com`)에 접속해보세요.
    방금 등록한 프로젝트의 타임라인에 Claude Code와의 대화 내역이 실시간으로 동기화되어 나타납니다.

## 4. 일상적인 사용 (Daily Usage)

C-Ops는 **"설치하고 잊어버리는(Install and Forget)"** 도구입니다.
백그라운드 데몬이 파일 시스템 이벤트를 감지하므로, 당신은 평소처럼 코딩에만 집중하면 됩니다.

### 유용한 명령어
- **`cops status`**: 현재 데몬의 상태와 수집(Ingestion) 지연 시간을 확인합니다.
- **`cops list`**: 현재 추적 중인 프로젝트 목록을 보여줍니다.
