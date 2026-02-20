package commands

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mau-network/mau/e2e/internal/testenv"
	"github.com/spf13/cobra"
)

func NewLogsCmd() *cobra.Command {
	var follow bool
	var tail int

	cmd := &cobra.Command{
		Use:   "logs <peer-name>",
		Short: "View logs from a peer container",
		Long: `Stream or display logs from a specific peer container.

Useful for debugging and observing peer behavior in real-time.`,
		Example: `  # View logs from peer-0
  mau-e2e logs peer-0

  # Follow logs (tail -f style)
  mau-e2e logs peer-0 --follow

  # Show last 50 lines
  mau-e2e logs peer-0 --tail 50`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			peerName := args[0]

			// Load current environment
			env, err := testenv.LoadCurrentEnvironment()
			if err != nil {
				return fmt.Errorf("no active environment found: %w", err)
			}

			// Find peer
			peer := env.FindPeer(peerName)
			if peer == nil {
				return fmt.Errorf("peer '%s' not found", peerName)
			}

			// Stream logs
			fmt.Printf("Logs from %s (container: %s):\n", peerName, peer.ContainerID[:12])
			fmt.Println("---")

			logReader, err := peer.Logs(ctx, follow, tail)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}
			defer logReader.Close()

			// Copy logs to stdout
			_, err = io.Copy(os.Stdout, logReader)
			if err != nil && err != io.EOF {
				return fmt.Errorf("error reading logs: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (like tail -f)")
	cmd.Flags().IntVar(&tail, "tail", 100, "Number of lines to show from the end of the logs")

	return cmd
}
