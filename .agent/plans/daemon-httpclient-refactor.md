# Daemon HTTP Client를 imroc/req/v3로 변경

## 요약
daemon의 adapter에서 `http.Client`를 직접 사용하는 대신, CLI와 동일하게 `imroc/req/v3` 패키지를 사용하도록 변경

## 현재 상태
- **CLI**: `cli/internal/platform/setup/httpclient/httpclient.go`에서 `req.C()`로 HTTP 클라이언트 생성
- **Daemon**: `adapter.go`에서 `&http.Client{Timeout: ...}` 직접 생성

## 변경 사항

### 1. 새 파일 생성: `daemon/internal/platform/setup/copsapi/copsapi.go`

```go
package copsapi

import (
    "net/http"
    "github.com/imroc/req/v3"
    "github.com/team-attention/cops/daemon/internal/platform/setup/config"
)

type APIClient struct {
    *req.Client
}

func InitAPIClient(cfg *config.Config) *APIClient {
    client := req.C().
        SetBaseURL(cfg.API.URL).
        SetTimeout(cfg.API.Timeout)
    return &APIClient{Client: client}
}

func (c *APIClient) StandardHTTPClient() *http.Client {
    return c.Client.GetClient()
}
```

### 2. 수정: `daemon/internal/service/logprocessor/outbound/api/connectrpc/adapter.go`

```go
// Before
import (
    "net/http"
    ...
)

func NewAdapter(l *slog.Logger, cfg *config.Config) *Adapter {
    client := collectorv1connect.NewCollectorServiceClient(
        &http.Client{Timeout: cfg.API.Timeout},
        cfg.API.URL,
    )
    ...
}

// After
import (
    "github.com/team-attention/cops/daemon/internal/platform/setup/copsapi"
    ...
)

func NewAdapter(l *slog.Logger, apiClient *copsapi.APIClient, cfg *config.Config) *Adapter {
    client := collectorv1connect.NewCollectorServiceClient(
        apiClient.StandardHTTPClient(),
        cfg.API.URL,
    )
    ...
}
```

### 3. 수정: `daemon/cmd/internal/container/module_platform.go`
- `copsapi.InitAPIClient` provider 추가

### 4. 의존성 추가
```bash
cd daemon && go get github.com/imroc/req/v3
```
