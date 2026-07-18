package amnezigo

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	outputVPNLinkName = "amnezigo.vpn"
	vpnLinkScheme     = "vpn://"
	vpnContainerAWG   = "amnezia-awg"

	qCompressHeaderSize = 4 // big-endian uint32 uncompressed length
)

// vpnEnvelope is the outer JSON container understood by AmneziaVPN's
// vpn:// import handler (containersModel / importController).
type vpnEnvelope struct {
	HostName         string         `json:"hostName"`
	Description      string         `json:"description,omitempty"`
	DefaultContainer string         `json:"defaultContainer"`
	DNS1             string         `json:"dns1,omitempty"`
	DNS2             string         `json:"dns2,omitempty"`
	Containers       []vpnContainer `json:"containers"`
}

// vpnContainer wraps a single protocol container. For AWG 2.0 the container
// type is "amnezia-awg".
type vpnContainer struct {
	Container string       `json:"container"`
	Awg       vpnAwgConfig `json:"awg"`
}

// vpnAwgConfig holds the AWG-specific fields the import handler reads.
// LastConfig is a JSON-encoded vpnLastConfig (double serialization).
type vpnAwgConfig struct {
	LastConfig         string `json:"last_config"`
	Port               string `json:"port"`
	TransportProto     string `json:"transport_proto"`
	IsThirdPartyConfig bool   `json:"isThirdPartyConfig"`
}

// vpnLastConfig is the inner JSON consumed by the AmneziaVPN connect path.
// The app's Wireguard.kt:parseConfig reads structured fields (client_priv_key,
// server_pub_key, allowed_ips, Jc, I1, etc.) — NOT the raw INI. The Config
// field preserves the verbatim INI for display purposes only.
//
// Field names match the JSON keys expected by configWireguard() and
// configExtensionParameters() in the AmneziaVPN Android source:
//   - client_priv_key, client_ip, server_pub_key, psk_key (base64 strings)
//   - allowed_ips (JSON array, not comma-separated string)
//   - isObfuscationEnabled (bool; must be true for AWG extension)
//   - Jc/Jmin/Jmax/S1-S4/H1-H4/I1-I5 (string values, INI key names)
//
// I1-I5 carry CPS tag strings (e.g. "<r 2><b 0x0100>") verbatim — the
// amneziawg-go UAPI handler fully parses CPS tags at connect time.
type vpnLastConfig struct {
	Config               string   `json:"config"`
	HostName             string   `json:"hostName"`
	ClientPrivKey        string   `json:"client_priv_key"`
	ClientIP             string   `json:"client_ip"`
	ServerPubKey         string   `json:"server_pub_key"`
	PSKKey               string   `json:"psk_key,omitempty"`
	AllowedIPs           []string `json:"allowed_ips"`
	MTU                  string   `json:"mtu,omitempty"`
	PersistentKeepAlive  string   `json:"persistent_keep_alive,omitempty"`
	Jc                   string   `json:"Jc,omitempty"`
	Jmin                 string   `json:"Jmin,omitempty"`
	Jmax                 string   `json:"Jmax,omitempty"`
	S1                   string   `json:"S1,omitempty"`
	S2                   string   `json:"S2,omitempty"`
	S3                   string   `json:"S3,omitempty"`
	S4                   string   `json:"S4,omitempty"`
	H1                   string   `json:"H1,omitempty"`
	H2                   string   `json:"H2,omitempty"`
	H3                   string   `json:"H3,omitempty"`
	H4                   string   `json:"H4,omitempty"`
	I1                   string   `json:"I1,omitempty"`
	I2                   string   `json:"I2,omitempty"`
	I3                   string   `json:"I3,omitempty"`
	I4                   string   `json:"I4,omitempty"`
	I5                   string   `json:"I5,omitempty"`
	Port                 int      `json:"port"`
	IsObfuscationEnabled bool     `json:"isObfuscationEnabled"`
}

// EncodeVPNLink wraps a client AWG INI config into an AmneziaVPN vpn:// import
// link. The link is importable by the AmneziaVPN app (not the standalone
// AmneziaWG app).
//
// The last_config JSON carries both the verbatim INI (in "config", for display)
// and structured fields (client_priv_key, server_pub_key, allowed_ips, Jc, I1,
// etc.) that the app's connect path reads via configWireguard(). Without these
// structured fields, the app throws JSONException → error 1000.
//
// I1-I5 CPS tag strings are preserved verbatim — amneziawg-go's UAPI handler
// fully parses CPS tags (<r>, <rc>, <rd>, <b>, <t>, <d>) at connect time.
func EncodeVPNLink(clientINI []byte, endpoint string, listenPort int, dns []string, description string) string {
	host, port := resolveEndpoint(endpoint, listenPort)
	kv := parseINIKeyValue(string(clientINI))

	// Parse AllowedIPs into a JSON array (the app expects an array, not a string).
	var allowedIPs []string
	if val, ok := kv["AllowedIPs"]; ok {
		for ip := range strings.SplitSeq(val, ",") {
			allowedIPs = append(allowedIPs, strings.TrimSpace(ip))
		}
	}

	// AWG obfuscation is enabled when Jc is present in the INI.
	isObfEnabled := kv["Jc"] != ""

	lastCfg := vpnLastConfig{
		Config:               string(clientINI),
		HostName:             host,
		Port:                 port,
		ClientPrivKey:        kv["PrivateKey"],
		ClientIP:             kv["Address"],
		ServerPubKey:         kv["PublicKey"],
		AllowedIPs:           allowedIPs,
		IsObfuscationEnabled: isObfEnabled,
	}

	// Optional fields — only included when present in the INI.
	if val, ok := kv["PresharedKey"]; ok {
		lastCfg.PSKKey = val
	}
	if val, ok := kv["MTU"]; ok {
		lastCfg.MTU = val
	}
	if val, ok := kv["PersistentKeepalive"]; ok {
		lastCfg.PersistentKeepAlive = val
	}

	// AWG obfuscation fields — CPS tag strings preserved verbatim.
	if isObfEnabled {
		lastCfg.Jc = kv["Jc"]
		lastCfg.Jmin = kv["Jmin"]
		lastCfg.Jmax = kv["Jmax"]
		lastCfg.S1 = kv["S1"]
		lastCfg.S2 = kv["S2"]
		lastCfg.S3 = kv["S3"]
		lastCfg.S4 = kv["S4"]
		lastCfg.H1 = kv["H1"]
		lastCfg.H2 = kv["H2"]
		lastCfg.H3 = kv["H3"]
		lastCfg.H4 = kv["H4"]
		lastCfg.I1 = kv["I1"]
		lastCfg.I2 = kv["I2"]
		lastCfg.I3 = kv["I3"]
		lastCfg.I4 = kv["I4"]
		lastCfg.I5 = kv["I5"]
	}

	lastCfgJSON, err := json.Marshal(lastCfg)
	if err != nil {
		panic(fmt.Sprintf("vpnlink: marshal last_config: %v", err))
	}

	envelope := vpnEnvelope{
		HostName: host,
		Containers: []vpnContainer{{
			Container: vpnContainerAWG,
			Awg: vpnAwgConfig{
				LastConfig:         string(lastCfgJSON),
				IsThirdPartyConfig: true,
				Port:               strconv.Itoa(port),
				TransportProto:     "udp",
			},
		}},
		DefaultContainer: vpnContainerAWG,
		Description:      description,
	}
	if len(dns) > 0 {
		envelope.DNS1 = dns[0]
	}
	if len(dns) > 1 {
		envelope.DNS2 = dns[1]
	}

	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		panic(fmt.Sprintf("vpnlink: marshal envelope: %v", err))
	}

	compressed := qCompress(envelopeJSON)
	encoded := base64.RawURLEncoding.EncodeToString(compressed)
	return vpnLinkScheme + encoded
}

// parseINIKeyValue extracts key-value pairs from an INI config string.
// Section headers ([Interface], [Peer]) and comment lines (#) are skipped.
// This mirrors the parseConfigData() function in AmneziaVPN's Wireguard.kt,
// which splits on "=" and trims whitespace.
func parseINIKeyValue(ini string) map[string]string {
	result := make(map[string]string)
	for line := range strings.SplitSeq(ini, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "#") {
			continue
		}
		if key, value, found := strings.Cut(line, "="); found {
			result[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return result
}

// appendVPNLink builds a vpn:// link for a client peer and appends it as a
// FileOutput. The endpoint and listenPort come from the server peer's manifest
// entry so every client link points at the same server.
func appendVPNLink(result *GenerateResult, peerName string, clientBytes []byte, manifest Manifest, serverName string) {
	serverPeer := manifest.Peers[serverName]
	clientPeer := manifest.Peers[peerName]
	link := EncodeVPNLink(
		clientBytes,
		serverPeer.Endpoint,
		serverPeer.ListenPort,
		manifest.Network.DNS,
		clientPeer.DisplayName,
	)
	result.Files = append(result.Files, FileOutput{
		RelPath: peerName + "/" + outputVPNLinkName,
		Content: []byte(link),
	})
}

// resolveEndpoint splits an endpoint into host and port. If the endpoint lacks
// a port (net.SplitHostPort fails), the listenPort fallback is used instead.
func resolveEndpoint(endpoint string, listenPort int) (string, int) {
	if h, p, err := net.SplitHostPort(endpoint); err == nil {
		if parsedPort, err := strconv.Atoi(p); err == nil {
			return h, parsedPort
		}
	}

	return endpoint, listenPort
}

// qCompress prepends a 4-byte big-endian uncompressed-length header then
// zlib-compresses the payload, matching Qt's qCompress format. The 4-byte
// header lets the Qt decompressor pre-allocate the output buffer.
func qCompress(data []byte) []byte {
	var buf bytes.Buffer

	// 4-byte big-endian uncompressed length prefix.
	var header [qCompressHeaderSize]byte
	//nolint:gosec // G115: envelope JSON is memory-bounded, always < 4GiB
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	buf.Write(header[:])

	w := zlib.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()

	return buf.Bytes()
}
