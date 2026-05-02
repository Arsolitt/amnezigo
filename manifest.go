package amnezigo

// NetworkConfig holds network-level settings shared by all peers.
type NetworkConfig struct {
	DNS []string `json:"dns,omitempty"`
	MTU int      `json:"mtu,omitempty"`
}
