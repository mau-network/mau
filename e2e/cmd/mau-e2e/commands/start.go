package commands

import (
	"context"
	"fmt"

	"github.com/mau-network/mau/e2e/internal/testenv"
	"github.com/spf13/cobra"
)

func NewStartCmd() *cobra.Command {
	var peers int
	var name string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new test environment",
		Long: `Start a new Mau E2E test environment with N peers.

This creates an isolated Docker network and starts the specified number
of Mau peer containers. Each peer gets a unique PGP account.

The environment state is saved to ~/.mau-e2e/ for use with other commands.`,
		Example: `  # Start environment with 2 peers (default)
  mau-e2e start

  # Start with 5 peers
  mau-e2e start --peers 5

  # Start with custom name
  mau-e2e start --name my-test --peers 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Create environment
			env, err := testenv.NewEnvironment(name)
			if err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}

			// Start environment with N peers
			fmt.Printf("Starting Mau E2E environment with %d peers...\n", peers)
			
			if err := env.Start(ctx, peers); err != nil {
				return fmt.Errorf("failed to start environment: %w", err)
			}

			fmt.Printf("\n✓ Environment '%s' started successfully!\n", env.Name)
			fmt.Printf("  Peers: %d\n", len(env.Peers))
			fmt.Printf("  Network: %s\n", env.NetworkName)
			fmt.Println("\nNext steps:")
			fmt.Println("  • View status:       mau-e2e status")
			fmt.Println("  • View logs:         mau-e2e logs <peer-name>")
			fmt.Println("  • Stop environment:  mau-e2e stop")

			return nil
		},
	}

	cmd.Flags().IntVarP(&peers, "peers", "n", 2, "Number of peers to start")
	cmd.Flags().StringVar(&name, "name", "", "Custom environment name (auto-generated if not specified)")

	return cmd
}
