package amnezigo

import (
	"fmt"
	"sort"
)

// NetworkConfig holds network-level settings shared by all peers.
// These are global because AWG/WG MTU is per-interface (not per-peer),
// and DNS applies uniformly to all client configs.
type NetworkConfig struct {
	DNS []string `json:"dns,omitempty"`
	MTU int      `json:"mtu,omitempty"`
}

// ObfuscationManifest holds the obfuscation profile for the network.
// All values are explicit — presets are resolved at the jsonnet level
// before producing JSON. The manifest only carries resolved parameters.
//
// Parameters use pointer types to distinguish "user set to 0"
// from "user did not set this field".
type ObfuscationManifest struct {
	S1       *int         `json:"s1,omitempty"`
	S2       *int         `json:"s2,omitempty"`
	S3       *int         `json:"s3,omitempty"`
	S4       *int         `json:"s4,omitempty"`
	H1       *HeaderRange `json:"h1,omitempty"`
	H2       *HeaderRange `json:"h2,omitempty"`
	H3       *HeaderRange `json:"h3,omitempty"`
	H4       *HeaderRange `json:"h4,omitempty"`
	Jc       *int         `json:"jc,omitempty"`
	Jmin     *int         `json:"jmin,omitempty"`
	Jmax     *int         `json:"jmax,omitempty"`
	Protocol string       `json:"protocol,omitempty"`
}

// PeerManifest declares a single network peer in the manifest.
// The peer's name comes from the map key in Manifest.Peers, not from
// a field in this struct.
//
// A peer is a "server peer" when both Endpoint and ListenPort are set.
// All other peers are "client peers". At most one server peer is valid
// per manifest (enforced by validation, not by types).
type PeerManifest struct {
	Keepalive  *int   `json:"keepalive,omitempty"`
	Address    string `json:"address"`
	TunName    string `json:"tun_name,omitempty"`
	MainIface  string `json:"main_iface,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	ListenPort int    `json:"listen_port,omitempty"`
}

// IsServer reports whether this peer has the structural markers of a
// server peer: a non-empty Endpoint AND a non-zero ListenPort.
func (p *PeerManifest) IsServer() bool {
	return p.Endpoint != "" && p.ListenPort != 0
}

// Manifest is the top-level user-facing configuration format for amnezigo.
// It declares the full network topology: global settings, obfuscation
// profile, and all peers in a flat map. The manifest replaces the old
// imperative init/add/edit/remove/export flow with a single declarative
// file that drives `amnezigo generate`.
//
// Version must be 1. The version field exists for forward compatibility;
// future schema changes bump the version and may include migration logic.
//
// Peer names are map keys in the Peers field. The name is used as the
// output directory name and for human reference.
type Manifest struct {
	Peers       map[string]PeerManifest `json:"peers"`
	Obfuscation ObfuscationManifest     `json:"obfuscation"`
	Network     NetworkConfig           `json:"network"`
	Version     int                     `json:"version"`
}

// ServerPeer returns the name and count of server peers in the manifest.
// A valid manifest has exactly one server peer (count == 1). If count is
// 0 or > 1, validation should reject the manifest.
//
// When count > 1, the returned name is one of the server peers (arbitrary
// due to map iteration order) — callers should check count first.
func (m *Manifest) ServerPeer() (string, int) {
	var name string
	var count int
	for n, p := range m.Peers {
		if p.IsServer() {
			name = n
			count++
		}
	}
	return name, count
}

// ServerPeerName returns the name of the sole server peer. It panics if
// the manifest does not contain exactly one server peer. Use this in
// code paths that have already validated the manifest (e.g., the generate
// pipeline). Use ServerPeer() for validation code that needs the count.
func (m *Manifest) ServerPeerName() string {
	name, count := m.ServerPeer()
	if count != 1 {
		panic(fmt.Sprintf("ServerPeerName: expected exactly 1 server peer, found %d", count))
	}
	return name
}

// PeerNames returns all peer names sorted alphabetically. Use this when
// deterministic iteration order is needed (output directory creation,
// config file generation, testing).
func (m *Manifest) PeerNames() []string {
	names := make([]string, 0, len(m.Peers))
	for n := range m.Peers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
