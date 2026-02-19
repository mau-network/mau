package commands

import (
	"context"
	"fmt"

	"github.com/mau-network/mau/e2e/internal/testenv"
	"github.com/spf13/cobra"
)

func NewStopCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the current test environment",
		Long: `Stop and clean up the current Mau E2E test environment.

This stops all peer containers, removes the Docker network, and deletes
the environment state file.`,
		Example: `  # Stop current environment
  mau-e2e stop

  # Force stop (skip confirmation)
  mau-e2e stop --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Load current environment
			env, err := testenv.LoadCurrentEnvironment()
			if err != nil {
				return fmt.Errorf("no active environment found: %w", err)
			}

			// Confirm if not forced
			if !force {
				fmt.Printf("Stop environment '%s' with %d peers? [y/N]: ", env.Name, len(env.Peers))
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			fmt.Printf("Stopping environment '%s'...\n", env.Name)

			if err := env.Stop(ctx); err != nil {
				return fmt.Errorf("failed to stop environment: %w", err)
			}

			fmt.Println("✓ Environment stopped and cleaned up")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
