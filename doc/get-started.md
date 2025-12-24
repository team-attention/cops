# 시작하기 (Get Started)

C-Ops는 당신의 AI 코딩 파트너 Claude Code의 활동을 시각화하고 인사이트를 제공합니다.
이 가이드는 C-Ops를 설치하고 첫 프로젝트를 등록하는 과정을 안내합니다.

## 1. 설치 (Installation)

macOS 사용자는 Homebrew를 통해 CLI와 데몬 서비스를 한 번에 설치할 수 있습니다.

```bash
brew tap team-attention/cops
brew install cops
```

설치가 완료되면, C-Ops 데몬 서비스가 백그라운드에서 자동으로 시작됩니다.

> **수동 설치**:
> Go가 설치된 환경이라면 다음 명령어로 설치할 수 있습니다:
> ```bash
> go install github.com/team-attention/cops/cmd/cops@latest
> ```
> *참고: 수동 설치 시 데몬(`copsd`)을 별도로 실행해야 합니다.*

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
