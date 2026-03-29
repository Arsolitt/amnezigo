package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var peerIPAddr string

func NewAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new peer to the server configuration",
		Long: `Add a new WireGuard peer to the AmneziaWG server configuration.

Generates a keypair for the peer and adds it to the server's peer list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo add laptop
  amnezigo add phone --ipaddr 10.8.0.50
`,
		Args: cobra.ExactArgs(1),
		RunE: runAdd,
	}
	cmd.Flags().StringVar(&peerIPAddr, "ipaddr", "", "Peer IP address (e.g., 10.8.0.5)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peer, err := mgr.AddPeer(args[0], peerIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Peer '%s' added successfully\n", peer.Name)
	fmt.Printf("  IP Address: %s\n", peer.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", peer.PublicKey)

	return nil
}
