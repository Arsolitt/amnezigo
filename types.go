package amnezigo

import "time"

// HeaderRange represents a min-max range for obfuscation headers.
type HeaderRange struct {
	Min, Max uint32
}

// ServerConfig represents the full WireGuard server configuration.
type ServerConfig struct {
	Peers       []PeerConfig
	Interface   InterfaceConfig
	Obfuscation ServerObfuscationConfig
}

// InterfaceConfig represents the [Interface] section of a WireGuard config.
type InterfaceConfig struct {
	PrivateKey     string
	PublicKey      string
	Address        string
	PostUp         string
	PostDown       string
	MainIface      string
	TunName        string
	EndpointV4     string
	EndpointV6     string
	ListenPort     int
	MTU            int
	ClientToClient bool
}

// PeerConfig represents a [Peer] section of a WireGuard server config.
type PeerConfig struct {
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
	Name              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
}

// ServerObfuscationConfig represents server-side obfuscation parameters.
type ServerObfuscationConfig struct {
	Jc, Jmin, Jmax int
	S1, S2, S3, S4 int
	H1, H2, H3, H4 HeaderRange
}

// ClientObfuscationConfig represents client-side obfuscation parameters,
// extending ServerObfuscationConfig with I1-I5 custom packet strings.
type ClientObfuscationConfig struct {
	I1 string
	I2 string
	I3 string
	I4 string
	I5 string

	ServerObfuscationConfig //nolint:embeddedstructfieldcheck // ordering conflicts with govet fieldalignment
}

// ClientConfig represents the full WireGuard client configuration.
type ClientConfig struct {
	Peer      ClientPeerConfig
	Interface ClientInterfaceConfig
}

// ClientInterfaceConfig represents the [Interface] section of a client config.
type ClientInterfaceConfig struct {
	PrivateKey  string
	Address     string
	DNS         string
	Obfuscation ClientObfuscationConfig
	MTU         int
}

// ClientPeerConfig represents the [Peer] section of a client config.
type ClientPeerConfig struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          string
	PersistentKeepalive int
}

// Headers represents H1-H4 obfuscation headers.
type Headers struct {
	H1, H2, H3, H4 uint32
}

// SPrefixes represents S1-S4 obfuscation size prefixes.
type SPrefixes struct {
	S1, S2, S3, S4 int
}

// JunkParams represents Jc, Jmin, Jmax obfuscation junk parameters.
type JunkParams struct {
	Jc, Jmin, Jmax int
}

// simpleTag represents a CPS tag with type and value.
type simpleTag struct {
	Type  string // "b", "r", "rc", "rd", "t", "c"
	Value string // hex for "b", number for "r"/"rc"/"rd", empty for "t"/"c"
}

// CPSConfig holds the five intervals (I1-I5) of custom packet strings.
type CPSConfig struct {
	I1, I2, I3, I4, I5 string
}

// TagSpec defines a single tag with type and value.
type TagSpec struct {
	Type  string
	Value string
}

// I1I5Template contains the five intervals (I1-I5) for a protocol template.
type I1I5Template struct {
	I1, I2, I3, I4, I5 []TagSpec
}
