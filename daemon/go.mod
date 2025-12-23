module github.com/team-attention/cops/daemon

go 1.25.5

replace github.com/team-attention/cops/shared => ../shared

require (
	github.com/caarlos0/env/v11 v11.3.1
	github.com/spf13/cobra v1.10.2
	go.uber.org/fx v1.24.0
)

require (
	connectrpc.com/connect v1.19.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
