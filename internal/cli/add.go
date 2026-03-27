package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	addIPAddr string
)

// addCmd represents the add command.
var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new client to the server configuration",
	Long: `Add a new WireGuard client to the AmneziaWG server configuration.

Generates a keypair for the client and adds it to the server's peer list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo add laptop
  amnezigo add phone --ipaddr 10.8.0.50
`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringVar(&addIPAddr, "ipaddr", "", "Client IP address (e.g., 10.8.0.5)")
	addCmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
}

// NewAddCommand creates and returns the add command.
func NewAddCommand() *cobra.Command {
	return addCmd
}

// runAdd executes the add command.
func runAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peer, err := mgr.AddClient(args[0], addIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Client '%s' added successfully\n", peer.Name)
	fmt.Printf("  IP Address: %s\n", peer.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", peer.PublicKey)

	return nil
}
