## dev-build: Build CLI with debug symbols and race detection
.PHONY: dev-build
dev-build:
	go build -race -gcflags "all=-N -l" -o cops ./cmd/cops
