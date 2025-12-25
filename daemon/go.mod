module github.com/team-attention/cops/daemon

go 1.25.5

replace github.com/team-attention/cops/shared => ../shared

require (
	connectrpc.com/connect v1.19.1
	github.com/caarlos0/env/v11 v11.3.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/google/uuid v1.6.0
	github.com/imroc/req/v3 v3.57.0
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/team-attention/cops/shared v0.0.0-00010101000000-000000000000
	go.uber.org/fx v1.24.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.14.2 // indirect
	github.com/bytedance/sonic/loader v0.4.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/icholy/digest v1.1.0 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.57.1 // indirect
	github.com/refraction-networking/utls v1.8.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
