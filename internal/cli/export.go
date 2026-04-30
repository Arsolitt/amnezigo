package cli

import (
	"bytes"
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

var peerProtocol string
var peerEndpoint string

func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export peer configuration(s)",
		Long: `Export WireGuard peer configuration(s).

If a name is specified, exports only that peer's configuration.
If no name is specified, exports all peers' configurations.

Example:
  amnezigo export laptop
  amnezigo export --protocol quic laptop
  amnezigo export
`,
		Args: cobra.MaximumNArgs(1),
		RunE: runExport,
	}
	cmd.Flags().StringVar(
		&peerProtocol, "protocol", "random",
		"Obfuscation protocol: random, quic, dns, dtls, stun, sip",
	)
	cmd.Flags().StringVar(&peerEndpoint, "endpoint", "", "Override endpoint (skip auto-detection)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	var endpoint string
	if peerEndpoint != "" {
		endpoint = peerEndpoint
	} else {
		endpoint = resolveExportEndpoint(serverCfg)
	}

	peersToExport, err := selectPeersToExport(serverCfg.Peers, args)
	if err != nil {
		return err
	}

	return writePeerConfigs(mgr, peersToExport, endpoint)
}

func selectPeersToExport(peers []amnezigo.PeerConfig, args []string) ([]amnezigo.PeerConfig, error) {
	if len(args) == 0 {
		return peers, nil
	}
	peerName := args[0]
	for _, peer := range peers {
		if peer.Name == peerName {
			return []amnezigo.PeerConfig{peer}, nil
		}
	}
	return nil, fmt.Errorf("peer '%s' not found", peerName)
}

func writePeerConfigs(mgr *amnezigo.Manager, peers []amnezigo.PeerConfig, endpoint string) error {
	for _, peer := range peers {
		peerCfg, err := mgr.BuildPeerConfig(peer, peerProtocol, endpoint)
		if err != nil {
			return fmt.Errorf("failed to export peer '%s': %w", peer.Name, err)
		}

		var buf bytes.Buffer
		if err := amnezigo.WriteClientConfig(&buf, peerCfg); err != nil {
			return fmt.Errorf("failed to write peer config: %w", err)
		}

		configPath := peer.Name + ".conf"
		if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Exported peer '%s' to %s\n", peer.Name, configPath)
	}
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
