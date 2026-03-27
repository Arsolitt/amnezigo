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

type PeerConfig struct {
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
	Name              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
}

type ServerObfuscationConfig struct {
	Jc, Jmin, Jmax int
	S1, S2, S3, S4 int
	H1, H2, H3, H4 HeaderRange
}

type ClientObfuscationConfig struct {
	I1 string
	I2 string
	I3 string
	I4 string
	I5 string
	ServerObfuscationConfig
}

type ClientConfig struct {
	Interface ClientInterfaceConfig
	Peer      ClientPeerConfig
}

type ClientInterfaceConfig struct {
	Obfuscation ClientObfuscationConfig
	PrivateKey  string
	Address     string
	DNS         string
	MTU         int
}

type ClientPeerConfig struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          string
	PersistentKeepalive int
}
