package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo/internal/config"
	"github.com/Arsolitt/amnezigo/internal/network"
)

// Ensure config package is used.
var _ config.ServerConfig

var (
	editClientToClient string
	editConfigPath     string
)

// editCmd represents the edit command.
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit server configuration",
	Long:  `Edit server configuration parameters.`,
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().
		StringVar(&editClientToClient, "client-to-client", "", "Enable/disable client-to-client (true/false)")
	editCmd.Flags().StringVar(&editConfigPath, "config", "awg0.conf", "Server config file")
}

// NewEditCommand creates and returns the edit command.
func NewEditCommand() *cobra.Command {
	return editCmd
}

// runEdit executes the edit command.
func runEdit(_ *cobra.Command, _ []string) error {
	serverCfg, err := loadServerConfig(editConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	if editClientToClient != "" {
		newValue := editClientToClient == "true"
		if serverCfg.Interface.ClientToClient && !newValue {
			// Disabling - print iptables command
			tunName := serverCfg.Interface.TunName
			if tunName == "" {
				tunName = "awg0"
			}
			fmt.Printf("Run this command to disable client-to-client immediately:\n")
			fmt.Printf("  iptables -D FORWARD -i %s -o %s -j ACCEPT\n\n", tunName, tunName)
		}
		serverCfg.Interface.ClientToClient = newValue
		changed = true
	}

	if !changed {
		fmt.Println("No changes specified")
		return nil
	}

	// Regenerate iptables rules
	subnet := extractSubnet(serverCfg.Interface.Address)
	tunName := serverCfg.Interface.TunName
	if tunName == "" {
		tunName = "awg0"
	}
	serverCfg.Interface.PostUp = network.GeneratePostUp(
		tunName,
		serverCfg.Interface.MainIface,
		subnet,
		serverCfg.Interface.ClientToClient,
	)
	serverCfg.Interface.PostDown = network.GeneratePostDown(
		tunName,
		serverCfg.Interface.MainIface,
		subnet,
		serverCfg.Interface.ClientToClient,
	)

	// Save config
	if err := saveServerConfig(editConfigPath, serverCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Configuration updated")
	fmt.Println("  Restart AmneziaWG service to apply changes")
	return nil
}
