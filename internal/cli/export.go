package cli

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Arsolitt/amnezigo/internal/config"
	"github.com/Arsolitt/amnezigo/internal/crypto"
	"github.com/spf13/cobra"
)

var (
	exportEndpoint string
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Export client configuration(s)",
	Long: `Export WireGuard client configuration(s) for the specified client(s).

If a name is specified, exports only that client's configuration.
If no name is specified, exports all clients' configurations.

Example:
  amnezigo export laptop
  amnezigo export --endpoint 1.2.3.4:55424 laptop
  amnezigo export
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportEndpoint, "endpoint", "", "Server endpoint (e.g., 1.2.3.4:55424)")
	exportCmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
}

// NewExportCommand creates and returns the export command
func NewExportCommand() *cobra.Command {
	return exportCmd
}

// runExport executes the export command
func runExport(cmd *cobra.Command, args []string) error {
	configPath := cfgFile

	// Load existing server config
	serverCfg, err := loadServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	// Determine endpoint
	endpoint := exportEndpoint
	if endpoint == "" {
		// Get external IP if endpoint not specified
		externalIP, err := getExternalIP()
		if err != nil {
			externalIP = "YOUR_SERVER_IP"
		}
		endpoint = fmt.Sprintf("%s:%d", externalIP, serverCfg.Interface.ListenPort)
	}

	// Determine which clients to export
	var clientsToExport []config.PeerConfig
	if len(args) == 1 {
		// Export specific client
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
		// Export all clients
		clientsToExport = serverCfg.Peers
	}

	// Export each client
	for _, client := range clientsToExport {
		if err := exportClient(client, serverCfg, endpoint); err != nil {
			return fmt.Errorf("failed to export client '%s': %w", client.Name, err)
		}
		fmt.Printf("✓ Exported client '%s' to %s.conf\n", client.Name, client.Name)
	}

	return nil
}

// exportClient exports a single client configuration
func exportClient(client config.PeerConfig, serverCfg config.ServerConfig, endpoint string) error {
	// Extract client IP address from AllowedIPs
	clientIP := strings.TrimSuffix(client.AllowedIPs, "/32")

	// Use simple AllowedIPs for IPv4 and IPv6
	allowedIPs := "0.0.0.0/0, ::/0"

	// Get server PublicKey (derive from PrivateKey if not present)
	serverPublicKey := serverCfg.Interface.PublicKey
	if serverPublicKey == "" {
		serverPublicKey = crypto.DerivePublicKey(serverCfg.Interface.PrivateKey)
	}

	// Build client config
	clientConfig := config.ClientConfig{
		Interface: config.ClientInterfaceConfig{
			PrivateKey: client.PrivateKey,
			Address:    clientIP + "/32",
			DNS:        "1.1.1.1, 8.8.8.8",
			MTU:        serverCfg.Interface.MTU,
			Obfuscation: config.ClientObfuscationConfig{
				ServerObfuscationConfig: serverCfg.Obfuscation,
			},
		},
		Peer: config.ClientPeerConfig{
			PublicKey:           serverPublicKey,
			PresharedKey:        client.PresharedKey,
			Endpoint:            endpoint,
			AllowedIPs:          allowedIPs,
			PersistentKeepalive: 25,
		},
	}

	// Write client config file
	configPath := client.Name + ".conf"
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return config.WriteClientConfig(file, clientConfig)
}

// getExternalIP retrieves the external IP address of the server
func getExternalIP() (string, error) {
	resp, err := http.Get("https://icanhazip.com")
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

	// Trim whitespace
	ip := strings.TrimSpace(string(body))

	// Validate IP
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	return ip, nil
}
