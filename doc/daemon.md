# C-Ops Daemon 가이드

Daemon은 C-Ops 시스템의 "눈" 역할을 하는 백그라운드 서비스입니다. 사용자가 신경 쓰지 않아도 조용히 Claude Code의 활동을 관찰하고 보고합니다.

## 1. 역할 및 동작 원리

### 역할
- **File Watcher**: 등록된 프로젝트의 `.claude` 디렉토리에 생성되는 JSONL 로그 파일의 변경 사항을 실시간으로 감지합니다.
- **Log Parsing**: 추가된 로그 라인(JSON)을 파싱하여 구조화된 Go 구조체(`SessionRecord`)로 변환합니다.
- **Forwarder**: 파싱된 데이터를 주기적으로(또는 실시간으로) API 서버(Collector)로 전송합니다.
- **Config Watcher**: `~/.cops/config.json` 파일의 변경을 감시하여, 런타임에 감시 대상 프로젝트 목록을 동적으로 갱신합니다.

### 동작 원리
데몬은 시작 시 `~/.cops/config.json`을 로드하고, `Projects` 목록에 있는 경로들의 `.claude` 디렉토리에 대해 `fsnotify` 왓처를 생성합니다. Claude Code가 로그 파일에 내용을 쓰면(`write` 이벤트), 데몬은 파일의 변경된 부분을 읽어(tailing) 처리합니다.

## 2. 서비스 등록 및 제거 프로세스

데몬 자체는 Stateless에 가깝게 설계되어 있으며, 상태의 원본(Source of Truth)은 `~/.cops/config.json` 파일입니다.

### 서비스 등록 (Service Registration)
1. 사용자가 CLI(`cops add`)를 통해 config 파일을 수정합니다.
2. 데몬의 **Config Watcher**가 파일 변경 이벤트를 감지합니다.
3. 데몬은 설정을 다시 로드(Reload)합니다.
4. 새로운 프로젝트 경로가 발견되면, 해당 경로에 대한 **Log Watcher** 고루틴(Goroutine)을 스폰합니다.
5. 이제부터 해당 경로의 로그가 수집됩니다.

### 서비스 제거 (Service Removal)
1. 사용자가 CLI(`cops remove`)를 통해 config 파일에서 항목을 삭제합니다.
2. 데몬이 설정 변경을 감지하고 다시 로드합니다.
3. 기존 감시 목록에는 있었으나 새 설정에는 없는 경로를 식별합니다.
4. 해당 경로를 감시하던 Log Watcher 고루틴에게 종료 시그널(Context Cancellation)을 보냅니다.
5. 리소스가 해제되고 더 이상 해당 경로를 감시하지 않습니다.

## 3. 구체적인 동작 로직 (Concrete Behavior)

### 로그 처리 파이프라인
1. **Detect**: `fsnotify`가 `CREATE` 또는 `WRITE` 이벤트를 감지.
2. **Read**: 변경된 파일의 마지막 오프셋부터 새로운 라인을 읽음.
3. **Parse**: 각 라인을 JSON 디코딩하여 `SessionRecord` 객체 생성.
4. **Enrich**: 필요한 경우 메타데이터 보강 (예: Git 브랜치 정보, Project ID).
5. **Buffer**: `LogProcessor` 서비스 내부 버퍼에 추가.
6. **Flush**: 
    - 버퍼가 가득 차거나(예: 100건), 
    - 일정 시간(예: 1초)이 지나면,
    - gRPC `CollectorService.SendRecords`를 호출하여 API 서버로 일괄 전송.

### 예외 처리
- **네트워크 오류**: API 서버 일시 중단 시, 일정 횟수 재시도(Retry)하거나 로컬에 임시 버퍼링(Backpressure) 합니다.
- **파일 로테이션**: Claude Code가 로그 파일을 로테이션(삭제 후 재생성 등)하는 경우를 대비해 파일 핸들을 적절히 관리합니다.
