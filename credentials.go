package amnezigo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const outputConfigName = "awg0.conf"

// PeerCredentials holds the cryptographic material for a single peer.
type PeerCredentials struct {
	PrivateKey   string // base64-encoded X25519 private key
	PublicKey    string // base64-encoded X25519 public key (derived from PrivateKey)
	PresharedKey string // base64-encoded 32-byte PSK (per peer-to-server connection)
}

// PersistedCredentials holds all credentials extracted from existing output
// configs. Used by the generate pipeline to reuse keys between runs.
type PersistedCredentials struct {
	// Peers maps peer name → credentials. Only peers found in output are present.
	Peers map[string]PeerCredentials
	// Server holds the server peer's keypair. Zero-value on first run.
	Server PeerCredentials
}

// EmptyCredentials returns a PersistedCredentials with an initialized map and
// zero-value server credentials. Used on first run or when --full-reset is set.
func EmptyCredentials() *PersistedCredentials {
	return &PersistedCredentials{
		Peers: make(map[string]PeerCredentials),
	}
}

// LoadCredentials reads existing output configs from outputDir and extracts
// all persisted credentials. Returns empty credentials (not an error) if the
// output directory or server config does not exist (first-run path).
//
// The serverPeerName identifies which subdirectory contains the server config.
// Peer credentials are extracted from the server config's [Peer] sections,
// which include #_Name and #_PrivateKey metadata comments written by
// WriteServerConfig. When the server config is missing, the function falls
// back to reading individual client configs from peer subdirectories.
func LoadCredentials(outputDir, serverPeerName string) (*PersistedCredentials, error) {
	serverConfigPath := filepath.Join(outputDir, serverPeerName, outputConfigName)

	serverCfg, err := LoadServerConfig(serverConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load server config for credentials: %w", err)
	}

	creds := EmptyCredentials()

	if err == nil {
		// Primary path: server config exists — extract all credentials from it.
		creds.Server = PeerCredentials{
			PrivateKey: serverCfg.Interface.PrivateKey,
			PublicKey:  serverCfg.Interface.PublicKey,
		}
		for _, peer := range serverCfg.Peers {
			if peer.Name == "" {
				continue // unnamed peers cannot be matched to manifest entries
			}
			creds.Peers[peer.Name] = PeerCredentials{
				PrivateKey:   peer.PrivateKey,
				PublicKey:    peer.PublicKey,
				PresharedKey: peer.PresharedKey,
			}
		}
		return creds, nil
	}

	// Fallback path: server config missing — scan subdirectories for client configs.
	entries, dirErr := os.ReadDir(outputDir)
	if os.IsNotExist(dirErr) {
		return creds, nil // first run — no output dir at all
	}
	if dirErr != nil {
		return nil, fmt.Errorf("read output dir: %w", dirErr)
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == serverPeerName {
			continue
		}
		clientPath := filepath.Join(outputDir, entry.Name(), outputConfigName)
		peerCreds, clientErr := extractClientCredentials(clientPath)
		if clientErr != nil {
			continue // skip unreadable client configs
		}
		if peerCreds.PrivateKey != "" {
			creds.Peers[entry.Name()] = peerCreds
		}
	}

	return creds, nil
}

// extractClientCredentials reads a client config file and extracts PrivateKey
// from [Interface] and PresharedKey from [Peer]. This is a lightweight INI
// scanner — not a full ParseClientConfig. PublicKey is derived from PrivateKey
// via DerivePublicKey when PrivateKey is present.
func extractClientCredentials(configPath string) (PeerCredentials, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return PeerCredentials{}, err
	}
	defer f.Close()

	var creds PeerCredentials
	var currentSection string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#_")) {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line
			continue
		}
		parts := strings.SplitN(line, "=", maxSplitParts)
		if len(parts) != maxSplitParts {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch {
		case currentSection == sectionInterface && key == keyPrivateKey:
			creds.PrivateKey = value
		case currentSection == sectionPeer && key == keyPresharedKey:
			creds.PresharedKey = value
		}
	}
	if err := scanner.Err(); err != nil {
		return PeerCredentials{}, fmt.Errorf("scan %s: %w", configPath, err)
	}

	// Derive PublicKey from PrivateKey if available.
	// DerivePublicKey panics on invalid base64 or wrong key length, which can
	// happen if a config was manually edited or written by a different tool.
	// Recover from the panic and leave PublicKey empty in that case; the
	// pipeline will re-derive or regenerate it.
	if creds.PrivateKey != "" {
		creds.PublicKey = tryDerivePublicKey(creds.PrivateKey)
	}

	return creds, nil
}

// tryDerivePublicKey derives a public key from a base64-encoded private key,
// returning an empty string instead of panicking if the key is invalid.
func tryDerivePublicKey(privateKey string) string {
	result := ""
	func() {
		defer func() {
			if recover() != nil {
				result = ""
			}
		}()
		result = DerivePublicKey(privateKey)
	}()
	return result
}
