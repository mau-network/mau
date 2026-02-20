package main

import (
	"fmt"
	"os"

	"github.com/mau-network/mau/e2e/cmd/mau-e2e/commands"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mau-e2e",
		Short: "Mau E2E Testing Framework CLI",
		Long: `Interactive CLI tool for Mau P2P file synchronization end-to-end testing.

Provides commands to start/stop test environments, manage peers, establish
friend relationships, inject files, and simulate network conditions.

For automated testing, use 'go test' in the e2e/tests directory.`,
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
	}

	// Add subcommands
	rootCmd.AddCommand(commands.NewStartCmd())
	rootCmd.AddCommand(commands.NewStopCmd())
	rootCmd.AddCommand(commands.NewStatusCmd())
	rootCmd.AddCommand(commands.NewLogsCmd())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
