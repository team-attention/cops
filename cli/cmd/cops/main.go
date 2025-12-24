package main

import (
	"fmt"
	"os"

	"github.com/team-attention/cops/cli/cmd/internal/container"
)

func main() {
	if err := container.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
