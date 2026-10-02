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

// resolveObfuscation merges explicit manifest values with randomly generated
// ones for the protocol generation selected by obf.AWGVersion (an empty value
// selects DefaultAWGVersion). Explicit values are preserved, nil fields fall
// back to the documented per-version defaults, and the persisted
// header-protection key is reused unless fullReset is set.
func resolveObfuscation(
	obf ObfuscationManifest,
	persisted *PersistedCredentials,
	fullReset bool,
) (ServerObfuscationConfig, error) {
	version, err := ParseAWGVersion(obf.AWGVersion)
	if err != nil {
		return ServerObfuscationConfig{}, err
	}
	if err := checkObfuscationVersionGates(obf, version); err != nil {
		return ServerObfuscationConfig{}, err
	}
	headerProtection, randomTrailers, disableCookies := resolveObfuscationFlags(obf, version)

	result := ServerObfuscationConfig{
		Version: version,
		S1:      resolveInt(obf.S1),
		S2:      resolveInt(obf.S2),
		S3:      resolveInt(obf.S3),
		S4:      resolveInt(obf.S4),
	}

	minS := 0
	if headerProtection {
		minS = headerProtectionNonceSize
	}
	uniform := randomTrailers && obf.S1 == nil && obf.S2 == nil && obf.S3 == nil && obf.S4 == nil
	result = fillMissingSPrefixes(result, minS, uniform)
	if err := checkSPrefixFloor(result, headerProtection); err != nil {
		return ServerObfuscationConfig{}, err
	}

	result = fillMissingHeaders(obf, result, headerProtection)

	if headerProtection {
		if !fullReset && persisted.HeaderProtectionKey != "" {
			result.HeaderProtectionKey = persisted.HeaderProtectionKey
		} else {
			result.HeaderProtectionKey = GenerateHeaderProtectionKey()
		}
	}

	if version >= AWG30 {
		if err := applyTransportRanges(obf, &result); err != nil {
			return ServerObfuscationConfig{}, err
		}
	}
	result.RandomTrailers = randomTrailers
	result.DisableCookies = disableCookies

	junkResult, err := fillMissingJunk(obf, result)
	if err != nil {
		return ServerObfuscationConfig{}, err
	}
	result = junkResult

	return result, nil
}

// checkObfuscationVersionGates rejects 3.x manifest fields under a lower target
// version. Engine builds reject unknown INI keys ("Line unrecognized"), so
// silently accepting a field the target cannot express would hide a
// configuration mistake.
func checkObfuscationVersionGates(obf ObfuscationManifest, version AWGVersion) error {
	if version < AWG30 {
		switch {
		case obf.HeaderProtection != nil:
			return requireVersionGate("header_protection", version, AWG30)
		case obf.ContentPadding != nil:
			return requireVersionGate("content_padding", version, AWG30)
		case obf.RekeyAfterTime != nil:
			return requireVersionGate("rekey_after_time", version, AWG30)
		case obf.RekeyTimeout != nil:
			return requireVersionGate("rekey_timeout", version, AWG30)
		case obf.RejectAfterTime != nil:
			return requireVersionGate("reject_after_time", version, AWG30)
		case obf.KeepaliveTimeout != nil:
			return requireVersionGate("keepalive_timeout", version, AWG30)
		case obf.MaxHandshakeAttempts != nil:
			return requireVersionGate("max_handshake_attempts", version, AWG30)
		}
	}
	if version < AWG31 {
		switch {
		case obf.RandomTrailers != nil:
			return requireVersionGate("random_trailers", version, AWG31)
		case obf.DisableCookies != nil:
			return requireVersionGate("disable_cookies", version, AWG31)
		}
	}
	return nil
}

// requireVersionGate builds the error for a manifest field the selected AWG
// version does not understand. field is the manifest JSON name.
func requireVersionGate(field string, have, minVersion AWGVersion) error {
	return fmt.Errorf("obfuscation.%s requires awg_version %s or later (got %q)", field, minVersion, have)
}

// resolveObfuscationFlags derives the effective AWG 3.x flags. Each flag
// defaults to true for every version that understands it and can be turned off
// by an explicit manifest value.
func resolveObfuscationFlags(obf ObfuscationManifest, version AWGVersion) (bool, bool, bool) {
	headerProtection := version >= AWG30
	if obf.HeaderProtection != nil {
		headerProtection = *obf.HeaderProtection
	}
	randomTrailers := version >= AWG31
	if obf.RandomTrailers != nil {
		randomTrailers = *obf.RandomTrailers
	}
	disableCookies := version >= AWG31
	if obf.DisableCookies != nil {
		disableCookies = *obf.DisableCookies
	}
	return headerProtection, randomTrailers, disableCookies
}

// checkSPrefixFloor rejects S-prefix values below the header-protection floor.
// The ChaCha20 header cipher nonce is the first S bytes of the packet, so the
// engine refuses S < 12 whenever header protection is on.
func checkSPrefixFloor(cfg ServerObfuscationConfig, headerProtection bool) error {
	if !headerProtection {
		return nil
	}
	floor := headerProtectionNonceSize
	for i, s := range [4]int{cfg.S1, cfg.S2, cfg.S3, cfg.S4} {
		if s < floor {
			return fmt.Errorf("header protection requires S1-S4 >= %d (got S%d=%d)", floor, i+1, s)
		}
	}
	return nil
}

// fillMissingSPrefixes generates S-prefixes for any zero values, drawing every
// generated value at or above minS (the header-protection floor). Retries until
// all zero fields get non-zero values, since GenerateSPrefixes can produce 0
// when minS is 0. When uniform is set, one value is drawn for all four fields,
// as the AWG 3.1 random_trailers recommendation requires equal S values.
func fillMissingSPrefixes(cfg ServerObfuscationConfig, minS int, uniform bool) ServerObfuscationConfig {
	if cfg.S1 != 0 && cfg.S2 != 0 && cfg.S3 != 0 && cfg.S4 != 0 {
		return cfg
	}
	for range 100 {
		var p SPrefixes
		if uniform {
			p = GenerateUniformSPrefixes(minS)
		} else {
			p = GenerateSPrefixes(minS)
		}
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

// fillMissingHeaders generates header ranges for any nil values. With header
// protection active and every H field unset, the H1..H4 = 1..4 ranges are used:
// the 4-byte message type is encrypted, so the old WireGuard type-id avoidance
// no longer applies.
func fillMissingHeaders(
	obf ObfuscationManifest,
	cfg ServerObfuscationConfig,
	headerProtection bool,
) ServerObfuscationConfig {
	if headerProtection && obf.H1 == nil && obf.H2 == nil && obf.H3 == nil && obf.H4 == nil {
		headers := standardHeaderRanges()
		cfg.H1, cfg.H2, cfg.H3, cfg.H4 = headers[0], headers[1], headers[2], headers[3]
		return cfg
	}
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

// standardHeaderRanges returns the H1..H4 = 1..4 ranges recommended for AWG
// 3.x header protection. The 4-byte message type is encrypted by the header
// cipher, so the WireGuard type-ids 1..4 are no longer observable and cannot
// be matched by a DPI parser.
//
//nolint:mnd // protocol values — 1..4 are the WireGuard message type-ids.
func standardHeaderRanges() [4]HeaderRange {
	return [4]HeaderRange{{Min: 1, Max: 1}, {Min: 2, Max: 2}, {Min: 3, Max: 3}, {Min: 4, Max: 4}}
}

// Default AWG 3.x transport-protection ranges: jitter around the WireGuard
// protocol constants (120 s rekey, 5 s rekey timeout, 180 s reject, 10 s
// keepalive, 18 handshake attempts). The 3.1 reference warns against pushing
// these parameters to extremes.
//
//nolint:mnd // protocol defaults — the numeric literals are the domain values.
var (
	defaultContentPadding       = U16Range{Min: 2, Max: 10}
	defaultRekeyAfterTime       = U16Range{Min: 120, Max: 180}
	defaultRekeyTimeout         = U16Range{Min: 5, Max: 8}
	defaultRejectAfterTime      = U16Range{Min: 180, Max: 240}
	defaultKeepaliveTimeout     = U16Range{Min: 8, Max: 12}
	defaultMaxHandshakeAttempts = U16Range{Min: 16, Max: 20}
)

// applyTransportRanges resolves the six AWG 3.x range parameters into cfg.
// A nil manifest field selects the version default; U16Range{0, 0} disables the
// key (the writer omits it).
func applyTransportRanges(obf ObfuscationManifest, cfg *ServerObfuscationConfig) error {
	contentPadding, err := resolveU16Range("content_padding", obf.ContentPadding, defaultContentPadding)
	if err != nil {
		return err
	}
	rekeyAfterTime, err := resolveU16Range("rekey_after_time", obf.RekeyAfterTime, defaultRekeyAfterTime)
	if err != nil {
		return err
	}
	rekeyTimeout, err := resolveU16Range("rekey_timeout", obf.RekeyTimeout, defaultRekeyTimeout)
	if err != nil {
		return err
	}
	rejectAfterTime, err := resolveU16Range("reject_after_time", obf.RejectAfterTime, defaultRejectAfterTime)
	if err != nil {
		return err
	}
	keepaliveTimeout, err := resolveU16Range("keepalive_timeout", obf.KeepaliveTimeout, defaultKeepaliveTimeout)
	if err != nil {
		return err
	}
	maxHandshakeAttempts, err := resolveU16Range(
		"max_handshake_attempts",
		obf.MaxHandshakeAttempts,
		defaultMaxHandshakeAttempts,
	)
	if err != nil {
		return err
	}
	cfg.ContentPadding = contentPadding
	cfg.RekeyAfterTime = rekeyAfterTime
	cfg.RekeyTimeout = rekeyTimeout
	cfg.RejectAfterTime = rejectAfterTime
	cfg.KeepaliveTimeout = keepaliveTimeout
	cfg.MaxHandshakeAttempts = maxHandshakeAttempts
	return nil
}

// resolveU16Range resolves one AWG 3.x range parameter. A nil explicit value
// falls back to the version default; an explicit value is validated: {0, 0}
// disables the key (the writer omits it), mixing one zero bound with a
// non-zero one is an error, and Max must not be below Min.
func resolveU16Range(field string, explicit *U16Range, fallback U16Range) (U16Range, error) {
	if explicit == nil {
		return fallback, nil
	}
	r := *explicit
	if (r.Min == 0) != (r.Max == 0) {
		return U16Range{}, fmt.Errorf(
			"obfuscation.%s: bounds must both be zero or both non-zero (got %d-%d)", field, r.Min, r.Max)
	}
	if r.Max < r.Min {
		return U16Range{}, fmt.Errorf("obfuscation.%s: max (%d) is below min (%d)", field, r.Max, r.Min)
	}
	return r, nil
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

// loadPersistedCredentials returns the credentials persisted in outputDir, or
// empty credentials when outputDir is unset or holds no server config yet
// (first-run path). Only IO/parse failures other than "does not exist" are
// returned as errors.
func loadPersistedCredentials(outputDir, serverName string) (*PersistedCredentials, error) {
	if outputDir == "" {
		return EmptyCredentials(), nil
	}
	persisted, err := LoadCredentials(outputDir, serverName)
	switch {
	case err == nil:
		return persisted, nil
	case os.IsNotExist(err):
		// First run: neither the output dir nor a server config exists yet.
		return EmptyCredentials(), nil
	default:
		return nil, fmt.Errorf("load credentials: %w", err)
	}
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

	// Step 2: Load or create credentials. The persisted header-protection key
	// is an input to obfuscation resolution, so this runs before Step 3.
	persisted, err := loadPersistedCredentials(opts.OutputDir, serverName)
	if err != nil {
		return result, err
	}

	// Step 3: Resolve obfuscation
	obf, err := resolveObfuscation(manifest.Obfuscation, persisted, opts.FullReset)
	if err != nil {
		return result, fmt.Errorf("resolve obfuscation: %w", err)
	}

	// Step 4: Resolve peer credentials
	creds := resolvePeerCredentials(manifest.Peers, persisted, serverName, opts.FullReset)

	// Step 5: Build server config
	serverBytes, err := buildServerConfig(manifest, serverName, obf, creds)
	if err != nil {
		return result, fmt.Errorf("build server config: %w", err)
	}

	// Step 6: Validate the generated server config so AWG 3.x findings (such
	// as TRL001) reach the CLI.
	if parsed, perr := ParseServerConfig(bytes.NewReader(serverBytes)); perr == nil {
		result.Findings = append(result.Findings, ValidateServerConfig(&parsed)...)
	}

	// Step 7: Build client configs (sorted by name, filtered by PeerFilter)
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

	// Step 8: Collect all FileOutput
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

	// Step 9: Write files to disk if not dry run and output dir is set
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
