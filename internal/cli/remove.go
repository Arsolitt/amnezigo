package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

// removeCmd represents the remove command.
var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a client from the server configuration",
	Long: `Remove a WireGuard client from the AmneziaWG server configuration.

Removes the peer with the specified name from the server's peer list.

Example:
  amnezigo remove laptop
`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
}

// NewRemoveCommand creates and returns the remove command.
func NewRemoveCommand() *cobra.Command {
	return removeCmd
}

// runRemove executes the remove command.
func runRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemoveClient(args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ Client '%s' removed successfully\n", args[0])
	return nil
}
