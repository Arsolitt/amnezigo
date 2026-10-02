package amnezigo

import (
	"strconv"
	"time"
)

// HeaderRange represents a min-max range for obfuscation headers.
type HeaderRange struct {
	Min uint32 `json:"min"`
	Max uint32 `json:"max"`
}

// U16Range is an inclusive uint16 range for the AWG 3.x range parameters
// (ContentPaddingAddition and the five timer parameters). It serializes as
// "N" when Min == Max and "Min-Max" otherwise, matching
// u16_range_to_string in amneziawg-tools/src/type.c.
//
// The zero value (Min == 0 && Max == 0) means unset/disabled: the key is
// omitted from the emitted config and the engine keeps its default.
type U16Range struct {
	Min uint16 `json:"min"`
	Max uint16 `json:"max"`
}

// IsZero reports whether the range is the unset/disabled zero value.
func (r U16Range) IsZero() bool {
	return r.Min == 0 && r.Max == 0
}

// String renders the range in the engine's INI notation: "N" when the bounds
// are equal, "Min-Max" otherwise.
func (r U16Range) String() string {
	if r.Min == r.Max {
		return strconv.FormatUint(uint64(r.Min), 10)
	}
	return strconv.FormatUint(uint64(r.Min), 10) + "-" + strconv.FormatUint(uint64(r.Max), 10)
}

// ServerConfig represents the full WireGuard server configuration.
type ServerConfig struct {
	Peers       []PeerConfig
	Interface   InterfaceConfig
	Obfuscation ServerObfuscationConfig
}

// InterfaceConfig represents the [Interface] section of a WireGuard config.
type InterfaceConfig struct {
	TunName             string
	DNS                 string
	Address             string
	PostUp              string
	PostDown            string
	MainIface           string
	EndpointV6          string
	PrivateKey          string
	PublicKey           string
	EndpointV4          string
	MTU                 int
	ListenPort          int
	PersistentKeepalive int
	ClientToClient      bool
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

	// AWG 3.x device-level transport protection (see version.go). The zero
	// Version disables emission of every field below, which keeps legacy
	// hand-built configs emitting exactly the 2.0 key set.
	Version              AWGVersion
	HeaderProtectionKey  string // base64 32-byte key; "" = header protection disabled
	ContentPadding       U16Range
	RekeyAfterTime       U16Range
	RekeyTimeout         U16Range
	RejectAfterTime      U16Range
	KeepaliveTimeout     U16Range
	MaxHandshakeAttempts U16Range
	RandomTrailers       bool
	DisableCookies       bool
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
	Type  string // "b", "r", "rc", "rd", "t", "d"
	Value string // hex for "b", number for "r"/"rc"/"rd", empty for "t"/"d"
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
