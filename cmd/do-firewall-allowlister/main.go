package main

import (
	"fmt"
	"os"

	"github.com/kholisrag/do-firewall-allowlister/pkg/commands"
)

// Build information set by GoReleaser
var (
	version = "dev"
)

func main() {
	// Create and execute root command with build information
	rootCmd := commands.NewRootCommand()
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
