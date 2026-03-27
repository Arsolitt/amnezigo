package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

const (
	tabPadding     = 3
	separatorWidth = 76
)

// listCmd represents the list command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured clients",
	Long: `List all WireGuard clients configured in the AmneziaWG server configuration.

Displays a table with client name, IP address, and creation time.

Example:
  amnezigo list
`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
}

// NewListCommand creates and returns a new list command instance.
// Returns a fresh command to avoid cobra's root-delegation issue when
// the shared listCmd has been added as a subcommand via NewRootCmd.
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured clients",
		Long: `List all WireGuard clients configured in the AmneziaWG server configuration.

Displays a table with client name, IP address, and creation time.

Example:
  amnezigo list
`,
		RunE: runList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

// runList executes the list command.
func runList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	clients := mgr.ListClients()
	if len(clients) == 0 {
		fmt.Println("No clients configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, peer := range clients {
		timestamp := ""
		if !peer.CreatedAt.IsZero() {
			timestamp = peer.CreatedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", peer.Name, peer.AllowedIPs, timestamp)
	}

	writer.Flush()
	return nil
}
