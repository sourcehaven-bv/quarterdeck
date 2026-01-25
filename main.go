package main

import (
	"fmt"
	"os"

	"github.com/sourcehaven-bv/quarterdeck/internal/app"
	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

// Version information set via ldflags
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("quarterdeck %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
