package main

import (
	"fmt"
	"os"

	"github.com/team-attention/cops/cli/cmd/internal/container"
)

// version is set via ldflags during build
var version = "dev"

func main() {
	// Inject version via environment variable for Config to pick up
	if version != "" {
		os.Setenv("COPS_APP_VERSION", version)
	}

	if err := container.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
