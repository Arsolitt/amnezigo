package amnezigo

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const defaultPersistentKeepalive = 25

// Manager provides high-level operations for managing a WireGuard server
// configuration file, including client CRUD and export operations.
type Manager struct {
	ConfigPath string
}

// NewManager creates a new Manager for the given config file path.
func NewManager(configPath string) *Manager {
	return &Manager{ConfigPath: configPath}
}

// Load reads and parses the server configuration from disk.
func (m *Manager) Load() (ServerConfig, error) {
	return LoadServerConfig(m.ConfigPath)
}

// Save writes the server configuration to disk using atomic file writes.
func (m *Manager) Save(cfg ServerConfig) error {
	return SaveServerConfig(m.ConfigPath, cfg)
}

// AddClient creates a new WireGuard peer with generated keys and optional
// explicit IP address. If ip is empty, the next available IP in the server
// subnet is assigned automatically.
func (m *Manager) AddClient(name, ip string) (PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return PeerConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	for _, peer := range serverCfg.Peers {
		if peer.Name == name {
			return PeerConfig{}, fmt.Errorf("client with name '%s' already exists", name)
		}
	}

	clientIP, err := m.resolveClientIP(ip, serverCfg)
	if err != nil {
		return PeerConfig{}, err
	}

	privateKey, publicKey := GenerateKeyPair()
	psk := GeneratePSK()

	newPeer := PeerConfig{
		Name:         name,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		PresharedKey: psk,
		AllowedIPs:   clientIP + "/32",
		CreatedAt:    time.Now(),
	}

	serverCfg.Peers = append(serverCfg.Peers, newPeer)

	if err := m.Save(serverCfg); err != nil {
		return PeerConfig{}, fmt.Errorf("failed to save server config: %w", err)
	}

	return newPeer, nil
}

// resolveClientIP returns the explicit IP if provided, or finds the next
// available IP in the server subnet.
func (m *Manager) resolveClientIP(ip string, serverCfg ServerConfig) (string, error) {
	if ip != "" {
		return ip, nil
	}

	_, ipnet, err := net.ParseCIDR(serverCfg.Interface.Address)
	if err != nil {
		return "", fmt.Errorf("invalid server address: %w", err)
	}

	existingIPs := make([]string, 0, len(serverCfg.Peers))
	for _, peer := range serverCfg.Peers {
		if before, ok := strings.CutSuffix(peer.AllowedIPs, "/32"); ok {
			peerIP := net.ParseIP(before)
			if peerIP != nil && ipnet.Contains(peerIP) {
				existingIPs = append(existingIPs, before)
			}
		}
	}

	clientIP, err := FindNextAvailableIP(serverCfg.Interface.Address, existingIPs)
	if err != nil {
		return "", fmt.Errorf("failed to assign IP address: %w", err)
	}

	return clientIP, nil
}

// RemoveClient removes a peer by name from the server configuration.
func (m *Manager) RemoveClient(name string) error {
	serverCfg, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	peerIndex := -1
	for i, peer := range serverCfg.Peers {
		if peer.Name == name {
			peerIndex = i
			break
		}
	}

	if peerIndex == -1 {
		return fmt.Errorf("client '%s' not found", name)
	}

	serverCfg.Peers = append(serverCfg.Peers[:peerIndex], serverCfg.Peers[peerIndex+1:]...)

	if err := m.Save(serverCfg); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	return nil
}

// FindClient returns a pointer to the peer with the given name.
func (m *Manager) FindClient(name string) (*PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	for i := range serverCfg.Peers {
		if serverCfg.Peers[i].Name == name {
			return &serverCfg.Peers[i], nil
		}
	}

	return nil, fmt.Errorf("client '%s' not found", name)
}

// ListClients returns all peers in the server configuration.
func (m *Manager) ListClients() []PeerConfig {
	serverCfg, err := m.Load()
	if err != nil {
		return nil
	}
	return serverCfg.Peers
}

// ExportClient generates a client configuration for the named peer,
// using the specified protocol for obfuscation and the given endpoint.
func (m *Manager) ExportClient(name, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	var client PeerConfig
	found := false
	for _, peer := range serverCfg.Peers {
		if peer.Name == name {
			client = peer
			found = true
			break
		}
	}
	if !found {
		return ClientConfig{}, fmt.Errorf("client '%s' not found", name)
	}

	return m.BuildClientConfig(client, protocol, endpoint)
}

// BuildClientConfig constructs a full ClientConfig from a peer, protocol, and endpoint.
func (m *Manager) BuildClientConfig(peer PeerConfig, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	clientIP := strings.TrimSuffix(peer.AllowedIPs, "/32")
	allowedIPs := "0.0.0.0/0, ::/0"

	serverPublicKey := serverCfg.Interface.PublicKey
	if serverPublicKey == "" {
		serverPublicKey = DerivePublicKey(serverCfg.Interface.PrivateKey)
	}

	i1, i2, i3, i4, i5 := GenerateCPS(
		protocol,
		serverCfg.Interface.MTU,
		serverCfg.Obfuscation.S1,
		0,
	)

	obfConfig := ClientObfuscationConfig{
		ServerObfuscationConfig: serverCfg.Obfuscation,
		I1:                      i1,
		I2:                      i2,
		I3:                      i3,
		I4:                      i4,
		I5:                      i5,
	}

	clientConfig := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey:  peer.PrivateKey,
			Address:     clientIP + "/32",
			DNS:         "1.1.1.1, 8.8.8.8",
			MTU:         serverCfg.Interface.MTU,
			Obfuscation: obfConfig,
		},
		Peer: ClientPeerConfig{
			PublicKey:           serverPublicKey,
			PresharedKey:        peer.PresharedKey,
			Endpoint:            endpoint,
			AllowedIPs:          allowedIPs,
			PersistentKeepalive: defaultPersistentKeepalive,
		},
	}

	return clientConfig, nil
}

// LoadServerConfig reads and parses a server configuration from the given file path.
func LoadServerConfig(path string) (ServerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return ServerConfig{}, err
	}
	defer file.Close()

	return ParseServerConfig(file)
}

// SaveServerConfig writes a server configuration to the given file path
// using atomic writes (write to .tmp, then rename).
func SaveServerConfig(path string, cfg ServerConfig) error {
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if err := WriteServerConfig(file, cfg); err != nil {
		file.Close()       //nolint:gosec // error path cleanup
		os.Remove(tmpPath) //nolint:gosec // error path cleanup
		return err
	}
	file.Close() //nolint:gosec // close before rename

	return os.Rename(tmpPath, path)
}
