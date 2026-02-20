package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mau-network/mau/e2e/internal/testenv"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current environment status",
		Long: `Display information about the current test environment.

Shows:
- Environment name and ID
- Network name
- List of peers with their status
- Container IDs and ports`,
		Example: `  # Show status
  mau-e2e status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load current environment
			env, err := testenv.LoadCurrentEnvironment()
			if err != nil {
				fmt.Println("No active environment found.")
				fmt.Println("Start one with: mau-e2e start")
				return nil
			}

			// Display environment info
			fmt.Printf("Environment: %s\n", env.Name)
			fmt.Printf("Network: %s\n", env.NetworkName)
			fmt.Printf("Peers: %d\n\n", len(env.Peers))

			// Create table writer
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATUS\tCONTAINER ID\tHTTP PORT\tFINGERPRINT")
			fmt.Fprintln(w, "----\t------\t------------\t---------\t-----------")

			for _, peer := range env.Peers {
				status := "unknown"
				if peer.Running {
					status = "running"
				} else {
					status = "stopped"
				}

				fingerprint := peer.Fingerprint
				if len(fingerprint) > 12 {
					fingerprint = fingerprint[:12] + "..."
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					peer.Name,
					status,
					peer.ContainerID[:12],
					peer.HTTPPort,
					fingerprint,
				)
			}

			w.Flush()

			return nil
		},
	}

	return cmd
}
