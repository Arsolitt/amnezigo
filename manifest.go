package amnezigo

// NetworkConfig holds network-level settings shared by all peers.
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
