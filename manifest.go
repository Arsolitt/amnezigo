package amnezigo

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
