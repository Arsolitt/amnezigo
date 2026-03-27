package amnezigo

import (
	"bytes"
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
//
//nolint:dupl // intentionally mirrors AddEdge for the Clients slice
func (m *Manager) AddClient(name, ip string) (PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return PeerConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	for _, peer := range serverCfg.Clients {
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
		Role:         RoleClient,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		PresharedKey: psk,
		AllowedIPs:   clientIP + "/32",
		CreatedAt:    time.Now(),
	}

	serverCfg.Clients = append(serverCfg.Clients, newPeer)

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

	existingIPs := make([]string, 0, len(serverCfg.Clients)+len(serverCfg.Edges))
	for _, peer := range serverCfg.Clients {
		if before, ok := strings.CutSuffix(peer.AllowedIPs, "/32"); ok {
			peerIP := net.ParseIP(before)
			if peerIP != nil && ipnet.Contains(peerIP) {
				existingIPs = append(existingIPs, before)
			}
		}
	}
	for _, edge := range serverCfg.Edges {
		if before, ok := strings.CutSuffix(edge.AllowedIPs, "/32"); ok {
			edgeIP := net.ParseIP(before)
			if edgeIP != nil && ipnet.Contains(edgeIP) {
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
	for i, peer := range serverCfg.Clients {
		if peer.Name == name {
			peerIndex = i
			break
		}
	}

	if peerIndex == -1 {
		return fmt.Errorf("client '%s' not found", name)
	}

	serverCfg.Clients = append(serverCfg.Clients[:peerIndex], serverCfg.Clients[peerIndex+1:]...)

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

	for i := range serverCfg.Clients {
		if serverCfg.Clients[i].Name == name {
			return &serverCfg.Clients[i], nil
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
	return serverCfg.Clients
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
	for _, peer := range serverCfg.Clients {
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

// AddEdge creates a new edge peer with generated keys and optional
// explicit IP address. If ip is empty, the next available IP in the server
// subnet is assigned automatically.
//
//nolint:dupl // intentionally mirrors AddClient for the Edges slice
func (m *Manager) AddEdge(name, ip string) (PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return PeerConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	for _, edge := range serverCfg.Edges {
		if edge.Name == name {
			return PeerConfig{}, fmt.Errorf("edge with name '%s' already exists", name)
		}
	}

	edgeIP, err := m.resolveClientIP(ip, serverCfg)
	if err != nil {
		return PeerConfig{}, err
	}

	privateKey, publicKey := GenerateKeyPair()
	psk := GeneratePSK()

	newEdge := PeerConfig{
		Name:         name,
		Role:         RoleEdge,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		PresharedKey: psk,
		AllowedIPs:   edgeIP + "/32",
		CreatedAt:    time.Now(),
	}

	serverCfg.Edges = append(serverCfg.Edges, newEdge)

	if err := m.Save(serverCfg); err != nil {
		return PeerConfig{}, fmt.Errorf("failed to save server config: %w", err)
	}

	return newEdge, nil
}

// RemoveEdge removes an edge peer by name from the server configuration.
func (m *Manager) RemoveEdge(name string) error {
	serverCfg, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	edgeIndex := -1
	for i, edge := range serverCfg.Edges {
		if edge.Name == name {
			edgeIndex = i
			break
		}
	}

	if edgeIndex == -1 {
		return fmt.Errorf("edge '%s' not found", name)
	}

	serverCfg.Edges = append(serverCfg.Edges[:edgeIndex], serverCfg.Edges[edgeIndex+1:]...)

	if err := m.Save(serverCfg); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	return nil
}

// FindEdge returns a pointer to the edge peer with the given name.
func (m *Manager) FindEdge(name string) (*PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	for i := range serverCfg.Edges {
		if serverCfg.Edges[i].Name == name {
			return &serverCfg.Edges[i], nil
		}
	}

	return nil, fmt.Errorf("edge '%s' not found", name)
}

// ListEdges returns all edge peers in the server configuration.
func (m *Manager) ListEdges() []PeerConfig {
	serverCfg, err := m.Load()
	if err != nil {
		return nil
	}
	return serverCfg.Edges
}

// extractHubIP extracts the IP address from a CIDR notation address string.
func extractHubIP(address string) string {
	ip, _, err := net.ParseCIDR(address)
	if err != nil {
		return address
	}
	return ip.String()
}

// BuildEdgeConfig constructs a full ClientConfig for an edge peer.
// Edge configs route only to the hub IP (not full tunnel), and have no DNS.
func (m *Manager) BuildEdgeConfig(name, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	var edge PeerConfig
	found := false
	for _, e := range serverCfg.Edges {
		if e.Name == name {
			edge = e
			found = true
			break
		}
	}
	if !found {
		return ClientConfig{}, fmt.Errorf("edge '%s' not found", name)
	}

	edgeIP := strings.TrimSuffix(edge.AllowedIPs, "/32")

	serverPublicKey := serverCfg.Interface.PublicKey
	if serverPublicKey == "" {
		serverPublicKey = DerivePublicKey(serverCfg.Interface.PrivateKey)
	}

	hubIP := extractHubIP(serverCfg.Interface.Address)

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
			PrivateKey:  edge.PrivateKey,
			Address:     edgeIP + "/32",
			DNS:         "",
			MTU:         serverCfg.Interface.MTU,
			Obfuscation: obfConfig,
		},
		Peer: ClientPeerConfig{
			PublicKey:           serverPublicKey,
			PresharedKey:        edge.PresharedKey,
			Endpoint:            endpoint,
			AllowedIPs:          hubIP + "/32",
			PersistentKeepalive: defaultPersistentKeepalive,
		},
	}

	return clientConfig, nil
}

// ExportEdge generates and serializes a client configuration for the named edge peer.
func (m *Manager) ExportEdge(name, protocol, endpoint string) ([]byte, error) {
	clientCfg, err := m.BuildEdgeConfig(name, protocol, endpoint)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := WriteClientConfig(&buf, clientCfg); err != nil {
		return nil, fmt.Errorf("failed to write edge config: %w", err)
	}

	return buf.Bytes(), nil
}
