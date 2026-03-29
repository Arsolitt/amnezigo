package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

func NewRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a peer from the server configuration",
		Long: `Remove a WireGuard peer from the AmneziaWG server configuration.

Removes the peer with the specified name from the server's peer list.

Example:
  amnezigo remove laptop
`,
		Args: cobra.ExactArgs(1),
		RunE: runRemove,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemovePeer(args[0]); err != nil {
		return err
	}
	fmt.Printf("Peer '%s' removed successfully\n", args[0])
	return nil
}
