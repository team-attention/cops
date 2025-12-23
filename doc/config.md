# 설정

이 문서는 [Viper](https://github.com/spf13/viper)를 사용한 C-Ops 설정 방법을 설명합니다.

## 개요

C-Ops는 환경 변수를 지원하는 Viper를 사용하여 설정을 관리합니다. 모든 설정 값은 `COPS_` 접두사를 사용한 환경 변수로 설정할 수 있습니다.

## 환경 변수

| 변수명 | 기본값 | 설명 |
|--------|--------|------|
| `COPS_APP_NAME` | `cops` | 애플리케이션 이름 |
| `COPS_APP_VERSION` | `0.0.1` | 애플리케이션 버전 |
| `COPS_LOGGING_LEVEL` | `info` | 로그 레벨 (debug, info, warn, error) |
| `COPS_LOGGING_FORMAT` | `text` | 로그 포맷 (text, json) |
| `COPS_COLLECTOR_URL` | `http://localhost:8080` | Collector 서버 URL |
| `COPS_COLLECTOR_TIMEOUT` | `30s` | Collector 서버 타임아웃 |
| `COPS_API_URL` | `http://localhost:8081` | API 서버 URL |
| `COPS_API_TIMEOUT` | `30s` | API 서버 타임아웃 |

## Viper 동작 방식

### 환경 변수 접두사

모든 환경 변수는 `COPS_` 접두사를 사용합니다:
```go
v.SetEnvPrefix("COPS")
```

### 키 변환

설정 키의 점(.)은 환경 변수 이름에서 밑줄(_)로 변환됩니다:
```go
v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
```

예시:
- `logging.level` -> `COPS_LOGGING_LEVEL`
- `collector.url` -> `COPS_COLLECTOR_URL`
- `api.timeout` -> `COPS_API_TIMEOUT`

### 자동 환경 변수 바인딩

Viper가 환경 변수를 자동으로 바인딩합니다:
```go
v.AutomaticEnv()
```

## 설정 구조

```go
type Config struct {
    App       AppConfig
    Logging   LoggingConfig
    Collector CollectorConfig
    API       APIConfig
}

type AppConfig struct {
    Name    string
    Version string
}

type LoggingConfig struct {
    Level  string
    Format string
}

type CollectorConfig struct {
    URL     string
    Timeout time.Duration
}

type APIConfig struct {
    URL     string
    Timeout time.Duration
}
```

## 사용 예시

### 디버그 로그 레벨 설정
```bash
export COPS_LOGGING_LEVEL=debug
cops list
```

### JSON 로그 포맷 사용
```bash
export COPS_LOGGING_FORMAT=json
cops list
```

### Collector 서버 URL 설정
```bash
export COPS_COLLECTOR_URL=http://collector.example.com:8080
cops add . --sync
```

### Collector 타임아웃 설정
```bash
export COPS_COLLECTOR_TIMEOUT=60s
cops add . --sync
```

## 고정 경로

일부 경로는 설정이 불가능하며 플랫폼 기본값을 사용합니다:

| 경로 | 설명 |
|------|------|
| `~/.claude` | Claude Code 데이터 디렉토리 (읽기 전용) |
| `~/.cops/config.json` | 전역 C-Ops 설정 |
| `{project}/.cops/config.json` | 프로젝트별 로컬 설정 |

이 경로들은 `pathutil.DefaultClaudeDir()` 및 `pathutil.DefaultCopsConfigDir()` 함수에 의해 결정됩니다.
