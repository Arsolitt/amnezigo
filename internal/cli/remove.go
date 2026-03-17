package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
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

// NewRemoveCommand creates and returns the remove command
func NewRemoveCommand() *cobra.Command {
	return removeCmd
}

// runRemove executes the remove command
func runRemove(cmd *cobra.Command, args []string) error {
	clientName := args[0]
	configPath := cfgFile

	// Load existing server config
	serverCfg, err := loadServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	// Find peer by name
	peerIndex := -1
	for i, peer := range serverCfg.Peers {
		if peer.Name == clientName {
			peerIndex = i
			break
		}
	}

	// Check if peer was found
	if peerIndex == -1 {
		return fmt.Errorf("client '%s' not found", clientName)
	}

	// Remove peer from slice
	serverCfg.Peers = append(serverCfg.Peers[:peerIndex], serverCfg.Peers[peerIndex+1:]...)

	// Save updated config
	if err := saveServerConfig(configPath, serverCfg); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	fmt.Printf("✓ Client '%s' removed successfully\n", clientName)

	return nil
}
