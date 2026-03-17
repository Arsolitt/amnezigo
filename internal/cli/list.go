package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
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

// NewListCommand creates and returns the list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured clients",
		Long: `List all WireGuard clients configured in the AmneziaWG server configuration.

Displays a table with client name, IP address, and creation time.

Example:
  gawg list
`,
		RunE: runList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

// runList executes the list command
func runList(cmd *cobra.Command, args []string) error {
	configPath := cfgFile

	// Load existing server config
	serverCfg, err := loadServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	// Check if there are any clients
	if len(serverCfg.Peers) == 0 {
		fmt.Println("No clients configured")
		return nil
	}

	// Create table writer
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Write header
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", 76))

	// Write each client
	for _, peer := range serverCfg.Peers {
		// Format timestamp as "YYYY-MM-DD HH:MM"
		timestamp := ""
		if !peer.CreatedAt.IsZero() {
			timestamp = peer.CreatedAt.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\n", peer.Name, peer.AllowedIPs, timestamp)
	}

	writer.Flush()

	return nil
}
