package main

import (
	"os"

	"github.com/team-attention/cops/api/cmd/internal/container"
)

// version is set via ldflags during build
var version = "dev"

func main() {
	// Inject version via environment variable for Config to pick up
	if version != "" {
		os.Setenv("APP_VERSION", version)
	}

	container.Run()
}
