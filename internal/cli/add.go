package cli

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo/internal/config"
	"github.com/Arsolitt/amnezigo/internal/crypto"
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
	clientName := args[0]
	configPath := cfgFile

	// Load existing server config
	serverCfg, err := loadServerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	// Check if client name already exists
	for _, peer := range serverCfg.Peers {
		if peer.Name == clientName {
			return fmt.Errorf("client with name '%s' already exists", clientName)
		}
	}

	// Determine client IP address
	clientIP := addIPAddr
	if clientIP == "" {
		// Extract subnet prefix from server address for filtering
		_, ipnet, err := net.ParseCIDR(serverCfg.Interface.Address)
		if err != nil {
			return fmt.Errorf("invalid server address: %w", err)
		}

		// Collect existing IPs within the server's subnet
		existingIPs := make([]string, 0, len(serverCfg.Peers))
		for _, peer := range serverCfg.Peers {
			if before, ok := strings.CutSuffix(peer.AllowedIPs, "/32"); ok {
				ipStr := before
				peerIP := net.ParseIP(ipStr)
				if peerIP != nil && ipnet.Contains(peerIP) {
					existingIPs = append(existingIPs, ipStr)
				}
			}
		}

		// Debug: log what we found
		// fmt.Fprintf(os.Stderr, "DEBUG: configPath=%s, serverAddress=%s, existingIPs=%v\n",
		//	configPath, serverCfg.Interface.Address, existingIPs)

		// Find next available IP
		clientIP, err = findNextAvailableIP(serverCfg.Interface.Address, existingIPs)
		if err != nil {
			return fmt.Errorf("failed to assign IP address: %w", err)
		}
	}

	// Generate keypair for client
	privateKey, publicKey := crypto.GenerateKeyPair()

	// Generate preshared key
	psk := crypto.GeneratePSK()

	// Create new peer
	newPeer := config.PeerConfig{
		Name:         clientName,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		PresharedKey: psk,
		AllowedIPs:   clientIP + "/32",
		CreatedAt:    time.Now(),
	}

	// Add peer to config
	serverCfg.Peers = append(serverCfg.Peers, newPeer)

	// Save updated config
	if err := saveServerConfig(configPath, serverCfg); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	fmt.Printf("✓ Client '%s' added successfully\n", clientName)
	fmt.Printf("  IP Address: %s\n", clientIP)
	fmt.Printf("  Public Key: %s\n", publicKey)

	return nil
}

// loadServerConfig loads the server configuration from a file.
func loadServerConfig(path string) (config.ServerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return config.ServerConfig{}, err
	}
	defer file.Close()

	return config.ParseServerConfig(file)
}

// saveServerConfig saves the server configuration to a file.
func saveServerConfig(path string, cfg config.ServerConfig) error {
	// Write to temporary file first
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if err := config.WriteServerConfig(file, cfg); err != nil {
		//nolint:errcheck // Best effort cleanup on error
		file.Close()
		//nolint:errcheck // Best effort cleanup on error
		os.Remove(tmpPath)
		return err
	}
	//nolint:errcheck // Write completed successfully, Close should not fail meaningfully
	file.Close()

	// Rename temporary file to actual file (atomic operation)
	return os.Rename(tmpPath, path)
}

// findNextAvailableIP finds the next available IP address in the server's subnet
// Skips .0, .1 (server) and any already used IPs.
func findNextAvailableIP(serverAddress string, existingIPs []string) (string, error) {
	// Parse server address to get subnet
	ip, ipnet, err := net.ParseCIDR(serverAddress)
	if err != nil {
		return "", fmt.Errorf("invalid server address: %w", err)
	}

	// Create a map of existing IPs for quick lookup
	existing := make(map[string]bool)
	for _, ipStr := range existingIPs {
		existing[ipStr] = true
	}

	// Iterate through IPs in the subnet
	// Start from .2 (skip .0 and .1)
	for i := 2; i <= 254; i++ {
		// Create IP address
		ipBytes := ip.To4()
		if ipBytes == nil {
			return "", errors.New("not an IPv4 address")
		}

		ipBytes[3] = byte(i)
		candidateIP := ipBytes.String()

		// Skip if already used
		if existing[candidateIP] {
			continue
		}

		// Check if IP is in the subnet
		if !ipnet.Contains(net.ParseIP(candidateIP)) {
			continue
		}

		return candidateIP, nil
	}

	return "", fmt.Errorf("no available IP addresses in subnet %s", ipnet.String())
}
