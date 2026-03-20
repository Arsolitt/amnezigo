package config

import "time"

type HeaderRange struct {
	Min, Max uint32
}

type ServerConfig struct {
	Interface   InterfaceConfig
	Peers       []PeerConfig
	Obfuscation ServerObfuscationConfig
}

type InterfaceConfig struct {
	PrivateKey     string
	PublicKey      string
	Address        string
	ListenPort     int
	MTU            int
	PostUp         string
	PostDown       string
	MainIface      string
	TunName        string
	EndpointV4     string
	EndpointV6     string
	ClientToClient bool
}

type PeerConfig struct {
	Name              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
}

type ServerObfuscationConfig struct {
	Jc, Jmin, Jmax int
	S1, S2, S3, S4 int
	H1, H2, H3, H4 HeaderRange
}

type ClientObfuscationConfig struct {
	ServerObfuscationConfig
	I1, I2, I3, I4, I5 string
}

type ClientConfig struct {
	Interface ClientInterfaceConfig
	Peer      ClientPeerConfig
}

type ClientInterfaceConfig struct {
	PrivateKey  string
	Address     string
	DNS         string
	MTU         int
	Obfuscation ClientObfuscationConfig
}

type ClientPeerConfig struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          string
	PersistentKeepalive int
}
