package config

import "time"

type ServerConfig struct {
	Interface   InterfaceConfig
	Peers       []PeerConfig
	PSK         string
	Obfuscation ObfuscationConfig
}

type InterfaceConfig struct {
	PrivateKey string
	PublicKey  string
	Address    string
	ListenPort int
	MTU        int
	PostUp     string
	PostDown   string
	MainIface  string
	TunName    string
}

type PeerConfig struct {
	Name       string
	PrivateKey string
	PublicKey  string
	AllowedIPs string
	CreatedAt  time.Time
}

type ObfuscationConfig struct {
	Jc, Jmin, Jmax     int
	S1, S2, S3, S4     int
	H1, H2, H3, H4     uint32
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
	Obfuscation ObfuscationConfig
}

type ClientPeerConfig struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          string
	PersistentKeepalive int
}
