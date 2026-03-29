package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured peers",
		Long: `List all WireGuard peers configured in the AmneziaWG server configuration.

Displays a table with peer name, IP address, and creation time.

Example:
  amnezigo list
`,
		RunE: runList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peers := mgr.ListPeers()
	if len(peers) == 0 {
		fmt.Println("No peers configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, peer := range peers {
		timestamp := ""
		if !peer.CreatedAt.IsZero() {
			timestamp = peer.CreatedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", peer.Name, peer.AllowedIPs, timestamp)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}
