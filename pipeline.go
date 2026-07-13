package amnezigo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateOptions configures the generate pipeline.
type GenerateOptions struct {
	ProjectDir string
	OutputDir  string
	JpathDirs  []string
	PeerFilter []string
	DryRun     bool
	FullReset  bool
	VPNLinks   bool // generate AmneziaVPN vpn:// import links per client
}

// GenerateResult holds the output of a generate run.
type GenerateResult struct {
	ServerPeer  string
	Files       []FileOutput
	ClientPeers []string
	Findings    []Finding
}

// FileOutput represents a single file to be written.
type FileOutput struct {
	RelPath string
	Content []byte
}

// resolveObfuscation merges explicit manifest values with randomly generated ones.
// If ObfuscationManifest has no values (all nil), generates everything randomly.
// If some values are set, preserves them and generates the rest.
func resolveObfuscation(obf ObfuscationManifest) (ServerObfuscationConfig, error) {
	result := ServerObfuscationConfig{
		S1: resolveInt(obf.S1),
		S2: resolveInt(obf.S2),
		S3: resolveInt(obf.S3),
		S4: resolveInt(obf.S4),
	}

	result = fillMissingSPrefixes(result)

	result = fillMissingHeaders(obf, result)

	junkResult, err := fillMissingJunk(obf, result)
	if err != nil {
		return ServerObfuscationConfig{}, err
	}
	result = junkResult

	return result, nil
}

// fillMissingSPrefixes generates S-prefixes for any zero values.
// Retries until all zero fields get non-zero values, since GenerateSPrefixes
// can produce 0 (rand.Int [0,64)).
func fillMissingSPrefixes(cfg ServerObfuscationConfig) ServerObfuscationConfig {
	if cfg.S1 != 0 && cfg.S2 != 0 && cfg.S3 != 0 && cfg.S4 != 0 {
		return cfg
	}
	for range 100 {
		p := GenerateSPrefixes()
		cfg.S1 = pickNonZero(cfg.S1, p.S1)
		cfg.S2 = pickNonZero(cfg.S2, p.S2)
		cfg.S3 = pickNonZero(cfg.S3, p.S3)
		cfg.S4 = pickNonZero(cfg.S4, p.S4)
		if cfg.S1 != 0 && cfg.S2 != 0 && cfg.S3 != 0 && cfg.S4 != 0 {
			return cfg
		}
	}
	return cfg
}

// fillMissingHeaders generates header ranges for any nil values.
func fillMissingHeaders(obf ObfuscationManifest, cfg ServerObfuscationConfig) ServerObfuscationConfig {
	if obf.H1 == nil || obf.H2 == nil || obf.H3 == nil || obf.H4 == nil {
		headers := GenerateHeaderRanges()
		cfg.H1 = resolveHeader(obf.H1, headers[0])
		cfg.H2 = resolveHeader(obf.H2, headers[1])
		cfg.H3 = resolveHeader(obf.H3, headers[2])
		cfg.H4 = resolveHeader(obf.H4, headers[3])
		return cfg
	}
	cfg.H1 = *obf.H1
	cfg.H2 = *obf.H2
	cfg.H3 = *obf.H3
	cfg.H4 = *obf.H4
	return cfg
}

// fillMissingJunk generates junk parameters for any nil values.
// Retries until Jc is non-zero, since GenerateJunkParamsWithForbidden
// can produce Jc=0 (rand.Int [0,11)).
func fillMissingJunk(obf ObfuscationManifest, cfg ServerObfuscationConfig) (ServerObfuscationConfig, error) {
	if obf.Jc != nil && obf.Jmin != nil && obf.Jmax != nil {
		cfg.Jc = *obf.Jc
		cfg.Jmin = *obf.Jmin
		cfg.Jmax = *obf.Jmax
		return cfg, nil
	}
	forbidden := PaddedSizes(cfg.S1, cfg.S2, cfg.S3, cfg.S4)
	jc := resolveInt(obf.Jc)
	jmin := resolveInt(obf.Jmin)
	jmax := resolveInt(obf.Jmax)
	for range 100 {
		junk, err := GenerateJunkParamsWithForbidden(forbidden)
		if err != nil {
			return ServerObfuscationConfig{}, err
		}
		cfg.Jc = pickNonZero(jc, junk.Jc)
		cfg.Jmin = pickNonZero(jmin, junk.Jmin)
		cfg.Jmax = pickNonZero(jmax, junk.Jmax)
		if cfg.Jc != 0 {
			return cfg, nil
		}
	}
	return cfg, nil
}

// resolveInt returns the dereferenced value or 0 if nil.
func resolveInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// resolveHeader returns the explicit header or the generated fallback.
func resolveHeader(explicit *HeaderRange, fallback HeaderRange) HeaderRange {
	if explicit == nil {
		return fallback
	}
	return *explicit
}

// pickNonZero returns current if non-zero, otherwise generated.
func pickNonZero(current, generated int) int {
	if current == 0 && generated != 0 {
		return generated
	}
	return current
}

// resolvePeerCredentials generates or reuses peer credentials based on persisted state.
// For each peer in the manifest, it either generates fresh keys or reuses existing
// credentials from PersistedCredentials (when available and fullReset is false).
//
// Server peers get only a keypair (no PresharedKey). Client peers get a keypair
// plus a unique PresharedKey for each client-to-server connection.
//
// The returned map contains credentials for all peers in the manifest, with
// PublicKey always derived from PrivateKey to ensure consistency.
func resolvePeerCredentials(
	peers map[string]PeerManifest,
	persisted *PersistedCredentials,
	serverName string,
	fullReset bool,
) map[string]PeerCredentials {
	result := make(map[string]PeerCredentials, len(peers))

	// Process server peer
	serverCreds := PeerCredentials{}
	if !fullReset && persisted.Server.PrivateKey != "" {
		// Reuse persisted server credentials
		serverCreds.PrivateKey = persisted.Server.PrivateKey
		serverCreds.PublicKey = persisted.Server.PublicKey
	} else {
		// Generate fresh server keypair
		priv, pub := GenerateKeyPair()
		serverCreds.PrivateKey = priv
		serverCreds.PublicKey = pub
	}
	// Server never gets PresharedKey
	result[serverName] = serverCreds

	// Process client peers
	for name := range peers {
		if name == serverName {
			continue // server already handled
		}

		clientCreds := PeerCredentials{}
		persistedClient, hasPersisted := persisted.Peers[name]

		if !fullReset && hasPersisted && persistedClient.PrivateKey != "" {
			// Reuse persisted client PrivateKey and PresharedKey
			clientCreds.PrivateKey = persistedClient.PrivateKey
			clientCreds.PresharedKey = persistedClient.PresharedKey
		} else {
			// Generate fresh client credentials
			priv, _ := GenerateKeyPair()
			clientCreds.PrivateKey = priv
			clientCreds.PresharedKey = GeneratePSK()
		}
		// ALWAYS derive PublicKey from PrivateKey to ensure consistency
		clientCreds.PublicKey = DerivePublicKey(clientCreds.PrivateKey)

		result[name] = clientCreds
	}

	return result
}

// buildServerConfig constructs a server configuration from manifest and credentials.
// Generates PostUp/PostDown iptables rules when MainIface is set.
// Sorts client peers by name for deterministic output.
func buildServerConfig(
	manifest Manifest,
	serverName string,
	obf ServerObfuscationConfig,
	creds map[string]PeerCredentials,
) ([]byte, error) {
	serverPeer, ok := manifest.Peers[serverName]
	if !ok {
		return nil, fmt.Errorf("server peer %s not found in manifest", serverName)
	}

	serverCreds, ok := creds[serverName]
	if !ok {
		return nil, fmt.Errorf("credentials for server peer %s not found", serverName)
	}

	// Extract network address from server address (e.g., "10.0.0.1/24" → "10.0.0.0/24")
	subnet := ExtractSubnet(serverPeer.Address)

	// Build InterfaceConfig
	iface := InterfaceConfig{
		Address:        serverPeer.Address,
		ListenPort:     serverPeer.ListenPort,
		PrivateKey:     serverCreds.PrivateKey,
		PublicKey:      serverCreds.PublicKey,
		MTU:            manifest.Network.MTU,
		TunName:        serverPeer.TunName,
		MainIface:      serverPeer.MainIface,
		ClientToClient: false, // TODO: support client-to-client routing
	}

	// Set defaults
	if iface.MTU == 0 {
		iface.MTU = 1280
	}
	if iface.TunName == "" {
		iface.TunName = "awg0"
	}

	// Generate iptables rules if MainIface is set
	if serverPeer.MainIface != "" {
		postUp4 := GeneratePostUp(iface.TunName, serverPeer.MainIface, subnet, iface.ClientToClient)
		postDown4 := GeneratePostDown(iface.TunName, serverPeer.MainIface, subnet, iface.ClientToClient)
		postUp6 := GeneratePostUp6(iface.TunName, serverPeer.MainIface, subnet, iface.ClientToClient)
		postDown6 := GeneratePostDown6(iface.TunName, serverPeer.MainIface, subnet, iface.ClientToClient)

		// Concatenate IPv4 and IPv6 rules with newline
		iface.PostUp = postUp4 + "\n" + postUp6
		iface.PostDown = postDown4 + "\n" + postDown6
	}

	// Build ServerConfig
	cfg := ServerConfig{
		Interface:   iface,
		Obfuscation: obf,
	}

	// Add client peers (sorted by name for deterministic output)
	var peerNames []string
	for name := range manifest.Peers {
		if name != serverName {
			peerNames = append(peerNames, name)
		}
	}
	sort.Strings(peerNames)

	for _, peerName := range peerNames {
		peer := manifest.Peers[peerName]
		peerCreds, ok := creds[peerName]
		if !ok {
			return nil, fmt.Errorf("credentials for peer %s not found", peerName)
		}

		// Determine protocol (default to "quic")
		protocol := peer.Protocol
		if protocol == "" {
			protocol = ProtocolQUIC
		}

		// Generate I-packets for this client
		i1, i2, i3, i4, i5 := GenerateCPS(protocol, iface.MTU, obf.S1, obf.Jc)

		peerCfg := PeerConfig{
			Name:         peerName,
			PublicKey:    peerCreds.PublicKey,
			PresharedKey: peerCreds.PresharedKey,
			AllowedIPs:   peer.Address,
			ClientObfuscation: &ClientObfuscationConfig{
				I1:                      i1,
				I2:                      i2,
				I3:                      i3,
				I4:                      i4,
				I5:                      i5,
				ServerObfuscationConfig: obf,
			},
		}

		cfg.Peers = append(cfg.Peers, peerCfg)
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := WriteServerConfig(&buf, cfg); err != nil {
		return nil, fmt.Errorf("write server config: %w", err)
	}

	return buf.Bytes(), nil
}

// buildClientConfig constructs a client configuration from manifest and credentials.
func buildClientConfig(
	manifest Manifest,
	peerName string,
	serverName string,
	obf ServerObfuscationConfig,
	creds map[string]PeerCredentials,
) ([]byte, error) {
	clientPeer, ok := manifest.Peers[peerName]
	if !ok {
		return nil, fmt.Errorf("client peer %s not found in manifest", peerName)
	}

	serverPeer, ok := manifest.Peers[serverName]
	if !ok {
		return nil, fmt.Errorf("server peer %s not found in manifest", serverName)
	}

	clientCreds, ok := creds[peerName]
	if !ok {
		return nil, fmt.Errorf("credentials for client peer %s not found", peerName)
	}

	serverCreds, ok := creds[serverName]
	if !ok {
		return nil, fmt.Errorf("credentials for server peer %s not found", serverName)
	}

	// Determine protocol (default to "quic")
	protocol := clientPeer.Protocol
	if protocol == "" {
		protocol = ProtocolQUIC
	}

	// Generate I-packets for this client
	i1, i2, i3, i4, i5 := GenerateCPS(protocol, manifest.Network.MTU, obf.S1, obf.Jc)

	// Build ClientInterfaceConfig
	mtu := manifest.Network.MTU
	if mtu == 0 {
		mtu = 1280
	}

	clientObf := ClientObfuscationConfig{
		I1:                      i1,
		I2:                      i2,
		I3:                      i3,
		I4:                      i4,
		I5:                      i5,
		ServerObfuscationConfig: obf,
	}

	iface := ClientInterfaceConfig{
		PrivateKey:  clientCreds.PrivateKey,
		Address:     clientPeer.Address,
		DNS:         strings.Join(manifest.Network.DNS, ", "),
		MTU:         mtu,
		Obfuscation: clientObf,
	}

	// Build ClientPeerConfig (server side)
	peerCfg := ClientPeerConfig{
		PublicKey:           serverCreds.PublicKey,
		PresharedKey:        clientCreds.PresharedKey,
		Endpoint:            serverPeer.Endpoint,
		AllowedIPs:          "0.0.0.0/0, ::/0",
		PersistentKeepalive: 0,
	}

	if clientPeer.Keepalive != nil {
		peerCfg.PersistentKeepalive = *clientPeer.Keepalive
	}

	cfg := ClientConfig{
		Interface: iface,
		Peer:      peerCfg,
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := WriteClientConfig(&buf, cfg); err != nil {
		return nil, fmt.Errorf("write client config: %w", err)
	}

	return buf.Bytes(), nil
}

// Generate orchestrates the full config generation pipeline.
// It loads existing credentials, resolves obfuscation, builds all configs,
// and optionally writes them to disk.
//
// The function uses a two-pass approach: all configs are computed in memory
// first, then written to disk. This ensures atomicity — if any config build
// fails, no files are written.
//
//nolint:gocognit // high cognitive complexity is expected for orchestrator function
func Generate(manifest Manifest, opts GenerateOptions) (GenerateResult, error) {
	var result GenerateResult

	// Step 1: Identify server peer
	serverName, serverCount := manifest.ServerPeer()
	if serverCount != 1 {
		return result, fmt.Errorf("exactly one server peer required, found %d", serverCount)
	}
	result.ServerPeer = serverName

	// Step 2: Resolve obfuscation
	obf, err := resolveObfuscation(manifest.Obfuscation)
	if err != nil {
		return result, fmt.Errorf("resolve obfuscation: %w", err)
	}

	// Step 3: Load or create credentials
	var persisted *PersistedCredentials
	if opts.OutputDir != "" {
		persisted, err = LoadCredentials(opts.OutputDir, serverName)
		if err != nil && !os.IsNotExist(err) {
			return result, fmt.Errorf("load credentials: %w", err)
		}
		// If LoadCredentials failed with IsNotExist, treat as first run
		if err != nil && os.IsNotExist(err) {
			persisted = EmptyCredentials()
		}
	} else {
		persisted = EmptyCredentials()
	}

	// Step 4: Resolve peer credentials
	creds := resolvePeerCredentials(manifest.Peers, persisted, serverName, opts.FullReset)

	// Step 5: Build server config
	serverBytes, err := buildServerConfig(manifest, serverName, obf, creds)
	if err != nil {
		return result, fmt.Errorf("build server config: %w", err)
	}

	// Step 6: Build client configs (sorted by name, filtered by PeerFilter)
	var clientPeerNames []string
	for name := range manifest.Peers {
		if name != serverName {
			clientPeerNames = append(clientPeerNames, name)
		}
	}
	sort.Strings(clientPeerNames)

	// Apply PeerFilter if non-empty
	var filteredClients []string
	if len(opts.PeerFilter) > 0 {
		filterSet := make(map[string]struct{}, len(opts.PeerFilter))
		for _, p := range opts.PeerFilter {
			filterSet[p] = struct{}{}
		}
		for _, name := range clientPeerNames {
			if _, ok := filterSet[name]; ok {
				filteredClients = append(filteredClients, name)
			}
		}
	} else {
		filteredClients = clientPeerNames
	}

	// Step 7: Collect all FileOutput
	result.Files = append(result.Files, FileOutput{
		RelPath: serverName + "/" + outputConfigName,
		Content: serverBytes,
	})

	for _, peerName := range filteredClients {
		clientBytes, err := buildClientConfig(manifest, peerName, serverName, obf, creds)
		if err != nil {
			return result, fmt.Errorf("build client config for %s: %w", peerName, err)
		}

		result.Files = append(result.Files, FileOutput{
			RelPath: peerName + "/" + outputConfigName,
			Content: clientBytes,
		})
		if opts.VPNLinks {
			appendVPNLink(&result, peerName, clientBytes, manifest, serverName)
		}
	}

	// Populate ClientPeers
	result.ClientPeers = filteredClients

	// Step 8: Write files to disk if not dry run and output dir is set
	if !opts.DryRun && opts.OutputDir != "" {
		for _, file := range result.Files {
			fullPath := filepath.Join(opts.OutputDir, file.RelPath)
			dir := filepath.Dir(fullPath)

			// Create directory if it doesn't exist
			if err := os.MkdirAll(dir, 0750); err != nil {
				return result, fmt.Errorf("create directory %s: %w", dir, err)
			}

			// Write file
			if err := os.WriteFile(fullPath, file.Content, 0600); err != nil {
				return result, fmt.Errorf("write file %s: %w", fullPath, err)
			}
		}
	}

	return result, nil
}
