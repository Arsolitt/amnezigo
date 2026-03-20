package cli

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Arsolitt/amnezigo/internal/config"
	"github.com/Arsolitt/amnezigo/internal/crypto"
	"github.com/Arsolitt/amnezigo/internal/network"
	"github.com/Arsolitt/amnezigo/internal/obfuscation"
	"github.com/spf13/cobra"
)

var (
	initIfaceName  string
	initEndpointV4 string
	initEndpointV6 string
	initConfigPath string
)

// initCmd represents the init command
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
	initProtocol       string
	initClientToClient bool
	initIface          string
)

func init() {
	// Add init command to root command (will be done in cli.go)
	initCmd.Flags().StringVar(&initIPAddr, "ipaddr", "", "Server IP address with subnet (e.g., 10.8.0.1/24) [required]")
	initCmd.Flags().IntVar(&initPort, "port", 0, "Listen port (default: random 10000-65535)")
	initCmd.Flags().IntVar(&initMTU, "mtu", 1280, "MTU size (default: 1280)")
	initCmd.Flags().StringVar(&initDNS, "dns", "1.1.1.1, 8.8.8.8", "DNS servers (comma-separated)")
	initCmd.Flags().IntVar(&initKeepalive, "keepalive", 25, "Persistent keepalive interval in seconds")
	initCmd.Flags().BoolVar(&initClientToClient, "client-to-client", false, "Allow client-to-client traffic")
	initCmd.Flags().StringVar(&initIface, "iface", "", "Main network interface (default: auto-detect)")
	initCmd.Flags().StringVar(&initIfaceName, "iface-name", "awg0", "Tunnel interface name")
	initCmd.Flags().StringVar(&initEndpointV4, "endpoint-v4", "", "IPv4 endpoint (auto-detect if empty)")
	initCmd.Flags().StringVar(&initEndpointV6, "endpoint-v6", "", "IPv6 endpoint (optional)")
	initCmd.Flags().StringVar(&initConfigPath, "config", "awg0.conf", "Server config file path")

	initCmd.MarkFlagRequired("ipaddr")
}

// runInit executes the init command
func runInit(cmd *cobra.Command, args []string) error {
	// Validate IP address
	if !isValidIPAddr(initIPAddr) {
		return fmt.Errorf("invalid IP address format: %s", initIPAddr)
	}

	// Extract subnet from IP address (e.g., "10.8.0.1/24" -> "10.8.0.0/24")
	subnet := extractSubnet(initIPAddr)

	// Generate random port if not specified
	if initPort == 0 {
		var err error
		initPort, err = generateRandomPort()
		if err != nil {
			return fmt.Errorf("failed to generate random port: %w", err)
		}
	}

	// Auto-detect main interface if not specified
	mainIface := initIface
	if mainIface == "" {
		mainIface = detectMainInterface()
		if mainIface == "" {
			return fmt.Errorf("failed to auto-detect main interface, please specify --iface")
		}
	}

	// Determine endpoints
	endpointV4 := initEndpointV4
	if endpointV4 == "" {
		endpointV4 = getEndpointV4(initPort)
	}

	endpointV6 := initEndpointV6
	if endpointV6 == "" {
		endpointV6 = getEndpointV6(initPort)
	}

	// Generate server keypair
	privateKey, publicKey := crypto.GenerateKeyPair()

	// Generate random s1 and jc values for obfuscation config
	s1Int, _ := rand.Int(rand.Reader, big.NewInt(65))
	s1 := int(s1Int.Int64())
	jcInt, _ := rand.Int(rand.Reader, big.NewInt(11))
	jc := int(jcInt.Int64())

	obfConfig := obfuscation.GenerateServerConfig(initMTU, s1, jc)

	// Generate iptables rules
	postUp := network.GeneratePostUp(initIfaceName, mainIface, subnet, initClientToClient)
	postDown := network.GeneratePostDown(initIfaceName, mainIface, subnet, initClientToClient)

	// Create server config
	serverCfg := config.ServerConfig{
		Interface: config.InterfaceConfig{
			PrivateKey:     privateKey,
			PublicKey:      publicKey,
			Address:        initIPAddr,
			ListenPort:     initPort,
			MTU:            initMTU,
			PostUp:         postUp,
			PostDown:       postDown,
			MainIface:      mainIface,
			TunName:        initIfaceName,
			EndpointV4:     endpointV4,
			EndpointV6:     endpointV6,
			ClientToClient: initClientToClient,
		},
		Peers:       []config.PeerConfig{},
		Obfuscation: obfConfig,
	}

	// Write config file
	if err := writeConfigFile(initConfigPath, serverCfg); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Save config path to .main.config
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

// isValidIPAddr checks if the IP address is valid
func isValidIPAddr(ipaddr string) bool {
	ip, _, err := net.ParseCIDR(ipaddr)
	return err == nil && ip != nil
}

// extractSubnet extracts the network address from a CIDR
func extractSubnet(ipaddr string) string {
	_, ipnet, err := net.ParseCIDR(ipaddr)
	if err != nil {
		return ipaddr
	}
	ones, _ := ipnet.Mask.Size()
	return ipnet.IP.String() + "/" + fmt.Sprintf("%d", ones)
}

// generateRandomPort generates a random port between 10000 and 65535
func generateRandomPort() (int, error) {
	max := big.NewInt(55536) // 65535 - 10000 + 1
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + 10000, nil
}

// detectMainInterface attempts to detect the main network interface
func detectMainInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	// Look for the first non-loopback interface that is up
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				return iface.Name
			}
		}
	}

	return ""
}

// writeConfigFile writes the server config to a file
func writeConfigFile(path string, cfg config.ServerConfig) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return config.WriteServerConfig(file, cfg)
}

// saveMainConfigPath saves the config path to .main.config file
func saveMainConfigPath(path string) error {
	return os.WriteFile(".main.config", []byte(path), 0644)
}

// getEndpointV4 gets the IPv4 endpoint
func getEndpointV4(port int) string {
	ip, err := getExternalIP()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

// getEndpointV6 gets the IPv6 endpoint
func getEndpointV6(port int) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ipv6.icanhazip.com")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return fmt.Sprintf("[%s]:%d", ip, port)
}
