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

// vpnLastConfig is the inner JSON whose Config field carries the verbatim
// AWG 2.0 INI — the authoritative tunnel config the mobile engine reads
// at connect time.
type vpnLastConfig struct {
	Config   string `json:"config"`
	HostName string `json:"hostName,omitempty"`
	Port     int    `json:"port,omitempty"`
}

// EncodeVPNLink wraps a client AWG INI config into an AmneziaVPN vpn:// import
// link. The link is importable by the AmneziaVPN app (not the standalone
// AmneziaWG app). The verbatim INI is carried in last_config.config — the
// authoritative source the tunnel reads. Outer envelope fields (hostName, dns)
// are metadata for the app's server list.
func EncodeVPNLink(clientINI []byte, endpoint string, listenPort int, dns []string) string {
	host, port := resolveEndpoint(endpoint, listenPort)

	lastCfgJSON, err := json.Marshal(vpnLastConfig{
		Config:   string(clientINI),
		HostName: host,
		Port:     port,
	})
	if err != nil {
		// vpnLastConfig has only string/int fields — json.Marshal is infallible.
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
	}
	if len(dns) > 0 {
		envelope.DNS1 = dns[0]
	}
	if len(dns) > 1 {
		envelope.DNS2 = dns[1]
	}

	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		// vpnEnvelope has only string/bool/slice fields — json.Marshal is infallible.
		panic(fmt.Sprintf("vpnlink: marshal envelope: %v", err))
	}

	compressed := qCompress(envelopeJSON)
	encoded := base64.RawURLEncoding.EncodeToString(compressed)
	return vpnLinkScheme + encoded
}

// appendVPNLink builds a vpn:// link for a client peer and appends it as a
// FileOutput. The endpoint and listenPort come from the server peer's manifest
// entry so every client link points at the same server.
func appendVPNLink(result *GenerateResult, peerName string, clientBytes []byte, manifest Manifest, serverName string) {
	serverPeer := manifest.Peers[serverName]
	link := EncodeVPNLink(
		clientBytes,
		serverPeer.Endpoint,
		serverPeer.ListenPort,
		manifest.Network.DNS,
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
