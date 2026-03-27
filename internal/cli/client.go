package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	clientIPAddr   string
	clientProtocol string
)

// NewClientCommand creates the client command group.
func NewClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Manage client peers",
	}

	cmd.AddCommand(NewClientAddCommand())
	cmd.AddCommand(NewClientListCommand())
	cmd.AddCommand(NewClientExportCommand())
	cmd.AddCommand(NewClientRemoveCommand())

	return cmd
}

// NewClientAddCommand creates the client add subcommand.
func NewClientAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new client to the server configuration",
		Long: `Add a new WireGuard client to the AmneziaWG server configuration.

Generates a keypair for the client and adds it to the server's peer list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo client add laptop
  amnezigo client add phone --ipaddr 10.8.0.50
`,
		Args: cobra.ExactArgs(1),
		RunE: runClientAdd,
	}
	cmd.Flags().StringVar(&clientIPAddr, "ipaddr", "", "Client IP address (e.g., 10.8.0.5)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peer, err := mgr.AddClient(args[0], clientIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Client '%s' added successfully\n", peer.Name)
	fmt.Printf("  IP Address: %s\n", peer.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", peer.PublicKey)

	return nil
}

// NewClientListCommand creates the client list subcommand.
func NewClientListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured clients",
		Long: `List all WireGuard clients configured in the AmneziaWG server configuration.

Displays a table with client name, IP address, and creation time.

Example:
  amnezigo client list
`,
		RunE: runClientList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	clients := mgr.ListClients()
	if len(clients) == 0 {
		fmt.Println("No clients configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, peer := range clients {
		timestamp := ""
		if !peer.CreatedAt.IsZero() {
			timestamp = peer.CreatedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", peer.Name, peer.AllowedIPs, timestamp)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}

// NewClientExportCommand creates the client export subcommand.
func NewClientExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export client configuration(s)",
		Long: `Export WireGuard client configuration(s) for the specified client(s).

If a name is specified, exports only that client's configuration.
If no name is specified, exports all clients' configurations.

Example:
  amnezigo client export laptop
  amnezigo client export --protocol quic laptop
  amnezigo client export
`,
		Args: cobra.MaximumNArgs(1),
		RunE: runClientExport,
	}
	cmd.Flags().StringVar(&clientProtocol, "protocol", "random", "Obfuscation protocol")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := resolveExportEndpoint(serverCfg)

	clientsToExport, err := selectClientsToExport(serverCfg.Clients, args)
	if err != nil {
		return err
	}

	return writeClientConfigs(mgr, clientsToExport, endpoint)
}

// NewClientRemoveCommand creates the client remove subcommand.
func NewClientRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a client from the server configuration",
		Long: `Remove a WireGuard client from the AmneziaWG server configuration.

Removes the peer with the specified name from the server's peer list.

Example:
  amnezigo client remove laptop
`,
		Args: cobra.ExactArgs(1),
		RunE: runClientRemove,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemoveClient(args[0]); err != nil {
		return err
	}
	fmt.Printf("Client '%s' removed successfully\n", args[0])
	return nil
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
		clientCfg, err := mgr.BuildClientConfig(client, clientProtocol, endpoint)
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

		fmt.Printf("Exported client '%s' to %s.conf\n", client.Name, client.Name)
	}
	return nil
}

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
