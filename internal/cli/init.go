package cli

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	initIfaceName  string
	initEndpointV4 string
	initEndpointV6 string
	initConfigPath string
)

const (
	defaultHTTPTimeout = 5 * time.Second
	defaultMTU         = 1280
	defaultKeepalive   = 25
	s1Range            = 65
	jcRange            = 11
)

// initCmd represents the init command.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new AmneziaWG server configuration",
	Long: `Initialize a new AmneziaWG v2.0 server configuration.

Generates:
- Server keypair (X25519)
- Preshared key
- Obfuscation config
- iptables rules
- Writes config file

 Example:
  amnezigo init --ipaddr 10.8.0.1/24 [--port 55424] [--mtu 1280] [--dns "1.1.1.1, 8.8.8.8"] [--keepalive 25] [--client-to-client] [--iface-name awg0] [--endpoint-v4 1.2.3.4] [--endpoint-v6 "[::1]"] [--config awg0.conf]
 `,
	RunE: runInit,
}

var (
	initIPAddr         string
	initPort           int
	initMTU            int
	initDNS            string
	initKeepalive      int
	initClientToClient bool
	initIface          string
)

func init() {
	initCmd.Flags().StringVar(&initIPAddr, "ipaddr", "", "Server IP address with subnet (e.g., 10.8.0.1/24) [required]")
	initCmd.Flags().IntVar(&initPort, "port", 0, "Listen port (default: random 10000-65535)")
	initCmd.Flags().IntVar(&initMTU, "mtu", defaultMTU, "MTU size (default: 1280)")
	initCmd.Flags().StringVar(&initDNS, "dns", "1.1.1.1, 8.8.8.8", "DNS servers (comma-separated)")
	initCmd.Flags().IntVar(&initKeepalive, "keepalive", defaultKeepalive, "Persistent keepalive interval in seconds")
	initCmd.Flags().BoolVar(&initClientToClient, "client-to-client", false, "Allow client-to-client traffic")
	initCmd.Flags().StringVar(&initIface, "iface", "", "Main network interface (default: auto-detect)")
	initCmd.Flags().StringVar(&initIfaceName, "iface-name", "awg0", "Tunnel interface name")
	initCmd.Flags().StringVar(&initEndpointV4, "endpoint-v4", "", "IPv4 endpoint (auto-detect if empty)")
	initCmd.Flags().StringVar(&initEndpointV6, "endpoint-v6", "", "IPv6 endpoint (optional)")
	initCmd.Flags().StringVar(&initConfigPath, "config", "awg0.conf", "Server config file path")

	//nolint:gosec,errcheck // required by Cobra, error only occurs on misconfiguration
	initCmd.MarkFlagRequired("ipaddr")
}

// runInit executes the init command.
func runInit(_ *cobra.Command, _ []string) error {
	if !amnezigo.IsValidIPAddr(initIPAddr) {
		return fmt.Errorf("invalid IP address format: %s", initIPAddr)
	}

	subnet := amnezigo.ExtractSubnet(initIPAddr)

	if initPort == 0 {
		var err error
		initPort, err = amnezigo.GenerateRandomPort()
		if err != nil {
			return fmt.Errorf("failed to generate random port: %w", err)
		}
	}

	mainIface := initIface
	if mainIface == "" {
		mainIface = amnezigo.DetectMainInterface()
		if mainIface == "" {
			return errors.New("failed to auto-detect main interface, please specify --iface")
		}
	}

	endpointV4 := initEndpointV4
	if endpointV4 == "" {
		endpointV4 = getEndpointV4(initPort)
	}

	endpointV6 := initEndpointV6
	if endpointV6 == "" {
		endpointV6 = getEndpointV6(initPort)
	}

	privateKey, publicKey := amnezigo.GenerateKeyPair()

	s1Int, _ := rand.Int(rand.Reader, big.NewInt(s1Range))
	s1 := int(s1Int.Int64())
	jcInt, _ := rand.Int(rand.Reader, big.NewInt(jcRange))
	jc := int(jcInt.Int64())

	obfConfig := amnezigo.GenerateServerConfig(initMTU, s1, jc)

	postUp := amnezigo.GeneratePostUp(initIfaceName, mainIface, subnet, initClientToClient)
	postDown := amnezigo.GeneratePostDown(initIfaceName, mainIface, subnet, initClientToClient)

	serverCfg := amnezigo.ServerConfig{
		Interface: amnezigo.InterfaceConfig{
			PrivateKey:          privateKey,
			PublicKey:           publicKey,
			Address:             initIPAddr,
			ListenPort:          initPort,
			MTU:                 initMTU,
			PostUp:              postUp,
			PostDown:            postDown,
			MainIface:           mainIface,
			TunName:             initIfaceName,
			EndpointV4:          endpointV4,
			EndpointV6:          endpointV6,
			ClientToClient:      initClientToClient,
			DNS:                 initDNS,
			PersistentKeepalive: initKeepalive,
		},
		Peers:       []amnezigo.PeerConfig{},
		Obfuscation: obfConfig,
	}

	if err := amnezigo.SaveServerConfig(initConfigPath, serverCfg); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := saveMainConfigPath(initConfigPath); err != nil {
		return fmt.Errorf("failed to save config path: %w", err)
	}

	fmt.Printf("✓ AmneziaWG configuration initialized successfully\n")
	fmt.Printf("  Config: %s\n", initConfigPath)
	fmt.Printf("  Server IP: %s\n", initIPAddr)
	fmt.Printf("  Listen Port: %d\n", initPort)
	fmt.Printf("  Main Interface: %s\n", mainIface)
	if endpointV4 != "" {
		fmt.Printf("  IPv4 Endpoint: %s\n", endpointV4)
	}
	if endpointV6 != "" {
		fmt.Printf("  IPv6 Endpoint: %s\n", endpointV6)
	}

	return nil
}

// saveMainConfigPath saves the config path to .main.config file.
func saveMainConfigPath(path string) error {
	return os.WriteFile(".main.config", []byte(path), 0600)
}

// getEndpointV4 gets the IPv4 endpoint.
func getEndpointV4(port int) string {
	client := &http.Client{Timeout: defaultHTTPTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://ipv4.icanhazip.com", nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

// getEndpointV6 gets the IPv6 endpoint.
func getEndpointV6(port int) string {
	client := &http.Client{Timeout: defaultHTTPTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://ipv6.icanhazip.com", nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To4() != nil {
		return ""
	}
	return fmt.Sprintf("[%s]:%d", ip, port)
}
