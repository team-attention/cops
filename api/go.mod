module github.com/team-attention/cops/api

go 1.25.5

replace github.com/team-attention/cops/shared => ../shared

require (
	connectrpc.com/connect v1.19.1
	github.com/bytedance/sonic v1.14.2
	github.com/caarlos0/env/v11 v11.3.1
	github.com/gofiber/adaptor/v2 v2.2.1
	github.com/gofiber/fiber/v2 v2.52.10
	github.com/samber/lo v1.52.0
	github.com/team-attention/cops/shared v0.0.0-00010101000000-000000000000
	go.mongodb.org/mongo-driver/v2 v2.4.1
	go.uber.org/fx v1.24.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic/loader v0.4.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.68.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)
