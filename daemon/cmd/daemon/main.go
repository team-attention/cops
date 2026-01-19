package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/team-attention/cops/daemon/cmd/internal/container"
)

// version is set via ldflags during build
var version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	container.Run()
}
