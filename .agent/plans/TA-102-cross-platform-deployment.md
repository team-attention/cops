# TA-102: Cross-Platform Deployment & Distribution System

## Overview
macOS/Linux용 CLI(`cops`)와 Daemon(`cops-daemon`) 바이너리의 자동 배포 파이프라인 구현

## Scope
- **Target platforms**: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- **Binaries**: cops (CLI), cops-daemon (Daemon)
- **Excluded**: Windows, API server (Docker 배포)

---

## Implementation Steps

### Phase 1: Build System Setup

#### 1.1 Create `.goreleaser.yaml`
**File**: `/.goreleaser.yaml` (new)

```yaml
version: 2
project_name: cops

before:
  hooks:
    - go mod tidy -C ./cli
    - go mod tidy -C ./daemon

builds:
  - id: cops
    main: ./cli/cmd/cops
    binary: cops
    env: [CGO_ENABLED=0]
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      # -s: symbol table 제거, -w: DWARF 디버깅 정보 제거 (바이너리 크기 30-50% 감소)
      - -s -w
      - -X main.version={{.Version}}

  - id: cops-daemon
    main: ./daemon/cmd/daemon
    binary: cops-daemon
    env: [CGO_ENABLED=0]
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}

archives:
  - id: cops-archive
    builds: [cops, cops-daemon]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format: tar.gz

checksum:
  name_template: "checksums.txt"

release:
  github:
    owner: team-attention
    name: cops
```

**배포 결과물** (GitHub Releases 페이지):
```
https://github.com/team-attention/cops/releases/tag/v1.0.0
├── cops_1.0.0_darwin_amd64.tar.gz   # macOS Intel (cops + cops-daemon)
├── cops_1.0.0_darwin_arm64.tar.gz   # macOS Apple Silicon
├── cops_1.0.0_linux_amd64.tar.gz    # Linux x64
├── cops_1.0.0_linux_arm64.tar.gz    # Linux ARM64
└── checksums.txt                     # SHA256 체크섬
```

#### 1.2 Add Build-time Version Variables (main.version 방식 통일)

**File**: `/cli/cmd/cops/main.go`

```go
// Add at package level
var version = "dev"

// rootCmd에 version 플래그 또는 서브커맨드 추가
```

**File**: `/cli/internal/platform/setup/config/config.go`

```go
// line 61: 하드코딩된 버전 제거, 환경변수나 외부 주입으로 변경
// 또는 main.go에서 version을 config로 전달
```

**File**: `/daemon/cmd/daemon/main.go`

```go
// Add at package level
var version = "dev"

// Add version command to rootCmd
versionCmd := &cobra.Command{
    Use:   "version",
    Short: "Print version",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println(version)
    },
}
rootCmd.AddCommand(versionCmd)
```

---

### Phase 2: CI/CD Pipeline

#### 2.1 Create GitHub Actions Workflow
**File**: `/.github/workflows/release.yaml` (new)

```yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.work

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

### Phase 3: Universal Installer Script

#### 3.1 Create Install Script
**File**: `/script/install.sh` (new)

**스크립트 동작 흐름**:

```
1. OS/Arch 감지
   └── uname -s (Darwin/Linux) → darwin/linux
   └── uname -m (x86_64/arm64/aarch64) → amd64/arm64

2. 최신 버전 조회 (GitHub API)
   └── curl https://api.github.com/repos/team-attention/cops/releases/latest
   └── JSON에서 "tag_name" 파싱 → v1.0.0

3. 아카이브 다운로드
   └── https://github.com/team-attention/cops/releases/download/v1.0.0/cops_1.0.0_darwin_arm64.tar.gz
   └── 임시 디렉토리에 저장

4. 설치
   └── ~/.cops/bin/ 디렉토리 생성
   └── tar 압축 해제 → cops, cops-daemon 바이너리 이동
   └── chmod +x 실행 권한 부여

5. PATH 업데이트 (신규 설치 시만)
   └── 현재 쉘 감지 ($SHELL → zsh/bash)
   └── zsh: ~/.zprofile 존재하면 사용, 없으면 ~/.zshrc
   └── bash: ~/.bash_profile 존재하면 사용, 없으면 ~/.bashrc
   └── 'export PATH="$HOME/.cops/bin:$PATH"' 추가
   └── 이미 설정되어 있으면 스킵

6. Daemon 서비스 등록
   └── 업그레이드인 경우: cops uninstall 먼저 실행
   └── cops install 실행 → kardianos/service가 플랫폼별 서비스 등록
       └── macOS: ~/Library/LaunchAgents/com.cops.daemon.plist
       └── Linux: ~/.config/systemd/user/cops-daemon.service

7. 완료 메시지
   └── 설치 경로, 버전 출력
   └── 쉘 재시작 안내 (source ~/.zprofile 등)
```

**Usage**:
```bash
# 최신 버전 설치
curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | bash

# 특정 버전 설치
COPS_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | ba```

---

### Phase 4: Documentation Updates

#### 4.1 Update README.md
**File**: `/README.md`

"시작하기" 섹션에 설치 가이드 추가:

```markdown
## 🚀 시작하기

### 설치

```bash
curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | bash
```

### 지원 플랫폼
- macOS (Intel & Apple Silicon)
- Linux (x86_64 & ARM64)
```

#### 4.2 Update doc/get-started.md
**File**: `/doc/get-started.md`

설치 섹션을 새로운 installer script 사용법으로 업데이트

---

## Files to Create/Modify

| Action | File Path                                       |
| ------ | ----------------------------------------------- |
| CREATE | `/.goreleaser.yaml`                             |
| CREATE | `/.github/workflows/release.yaml`               |
| CREATE | `/script/install.sh`                            |
| MODIFY | `/cli/cmd/cops/main.go`                         |
| MODIFY | `/cli/internal/platform/setup/config/config.go` |
| MODIFY | `/daemon/cmd/daemon/main.go`                    |
| MODIFY | `/README.md`                                    |
| MODIFY | `/doc/get-started.md`                           |

---

## Verification Checklist

- [ ] `goreleaser build --snapshot --clean` 성공
- [ ] GitHub Actions workflow 트리거 확인 (테스트 태그)
- [ ] macOS에서 install.sh 테스트
- [ ] Linux에서 install.sh 테스트
- [ ] `cops install` → daemon 서비스 등록 확인
- [ ] 업그레이드 시나리오 테스트

---

## Notes

- Daemon 바이너리 경로 `~/.cops/bin/cops-daemon`는 기존 설정과 일치 (config.go:68)
- `kardianos/service` 라이브러리가 macOS/Linux 서비스 등록 자동 처리
- Windows 지원은 별도 작업으로 분리