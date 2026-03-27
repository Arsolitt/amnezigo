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

// NewExportCommand creates and returns the export command.
func NewExportCommand() *cobra.Command {
	return exportCmd
}

// runExport executes the export command.
func runExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := serverCfg.Interface.EndpointV4
	if endpoint == "" {
		endpoint = serverCfg.Interface.EndpointV6
		if endpoint == "" {
			externalIP, err := getExternalIP()
			if err != nil {
				externalIP = "YOUR_SERVER_IP"
			}
			endpoint = fmt.Sprintf("%s:%d", externalIP, serverCfg.Interface.ListenPort)
		}
	}

	var clientsToExport []amnezigo.PeerConfig
	if len(args) == 1 {
		clientName := args[0]
		found := false
		for _, peer := range serverCfg.Peers {
			if peer.Name == clientName {
				clientsToExport = append(clientsToExport, peer)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("client '%s' not found", clientName)
		}
	} else {
		clientsToExport = serverCfg.Peers
	}

	for _, client := range clientsToExport {
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
