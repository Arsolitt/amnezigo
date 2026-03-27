package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	exportProtocol string
)

// exportCmd represents the export command.
var exportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Export client configuration(s)",
	Long: `Export WireGuard client configuration(s) for the specified client(s).

If a name is specified, exports only that client's configuration.
If no name is specified, exports all clients' configurations.

Example:
  amnezigo export laptop
  amnezigo export --protocol quic laptop
  amnezigo export
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportProtocol, "protocol", "random", "Obfuscation protocol")
	exportCmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
}

// NewExportCommand creates and returns a new export command instance.
// Returns a fresh command to avoid cobra's root-delegation issue when
// the shared exportCmd has been added as a subcommand via NewRootCmd.
func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export client configuration(s)",
		Long: `Export WireGuard client configuration(s) for the specified client(s).

If a name is specified, exports only that client's configuration.
If no name is specified, exports all clients' configurations.

Example:
  amnezigo export laptop
  amnezigo export --protocol quic laptop
  amnezigo export
`,
		Args: cobra.MaximumNArgs(1),
		RunE: runExport,
	}
	cmd.Flags().StringVar(&exportProtocol, "protocol", "random", "Obfuscation protocol")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

// runExport executes the export command.
func runExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := resolveExportEndpoint(serverCfg)

	clientsToExport, err := selectClientsToExport(serverCfg.Peers, args)
	if err != nil {
		return err
	}

	return writeClientConfigs(mgr, clientsToExport, endpoint)
}

func resolveExportEndpoint(serverCfg amnezigo.ServerConfig) string {
	if serverCfg.Interface.EndpointV4 != "" {
		return serverCfg.Interface.EndpointV4
	}
	if serverCfg.Interface.EndpointV6 != "" {
		return serverCfg.Interface.EndpointV6
	}
	externalIP, err := getExternalIP()
	if err != nil {
		externalIP = "YOUR_SERVER_IP"
	}
	return fmt.Sprintf("%s:%d", externalIP, serverCfg.Interface.ListenPort)
}

func selectClientsToExport(peers []amnezigo.PeerConfig, args []string) ([]amnezigo.PeerConfig, error) {
	if len(args) == 0 {
		return peers, nil
	}
	clientName := args[0]
	for _, peer := range peers {
		if peer.Name == clientName {
			return []amnezigo.PeerConfig{peer}, nil
		}
	}
	return nil, fmt.Errorf("client '%s' not found", clientName)
}

func writeClientConfigs(mgr *amnezigo.Manager, clients []amnezigo.PeerConfig, endpoint string) error {
	for _, client := range clients {
		clientCfg, err := mgr.BuildClientConfig(client, exportProtocol, endpoint)
		if err != nil {
			return fmt.Errorf("failed to export client '%s': %w", client.Name, err)
		}

		configPath := client.Name + ".conf"
		file, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		defer file.Close()

		if err := amnezigo.WriteClientConfig(file, clientCfg); err != nil {
			return fmt.Errorf("failed to write client config: %w", err)
		}

		fmt.Printf("✓ Exported client '%s' to %s.conf\n", client.Name, client.Name)
	}
	return nil
}

// getExternalIP retrieves the external IP address of the server.
func getExternalIP() (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://icanhazip.com", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get external IP: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	return ip, nil
}
