package amnezigo

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// sampleClientINI is a minimal AWG 2.0 client config with all obfuscation
// field types that EncodeVPNLink must preserve.
const sampleClientINI = `[Interface]
PrivateKey = oN+f...exampleKey...=
Address = 10.0.0.2/32
DNS = 1.1.1.1
MTU = 1280
Jc = 5
Jmin = 250
Jmax = 750
S1 = 30
S2 = 35
S3 = 20
S4 = 12
H1 = 100-200
H2 = 150-250
H3 = 200-300
H4 = 250-350
I1 = <b 0x12><r 16>
I2 = <b 0x34><r 16>
I3 = <b 0x56><r 16>
I4 = <b 0x78><r 16>

[Peer]
PublicKey = SERVER_PUB_KEY_EXAMPLE
PresharedKey = PSK_EXAMPLE_VALUE_HERE
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`

// decodeVPNLink reverses EncodeVPNLink: strips vpn://, Base64URL-decodes,
// strips the 4-byte qCompress header, zlib-inflates, returns the envelope
// JSON bytes. Mirrors AmneziaVPN's decode path.
func decodeVPNLink(t *testing.T, link string) []byte {
	t.Helper()

	if !strings.HasPrefix(link, "vpn://") {
		t.Fatalf("link does not start with vpn://: %q", link[:min(len(link), 20)])
	}

	b64 := strings.TrimPrefix(link, "vpn://")
	compressed, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64url decode: %v", err)
	}

	if len(compressed) < qCompressHeaderSize {
		t.Fatalf("compressed payload too short: %d bytes", len(compressed))
	}

	expectedLen := binary.BigEndian.Uint32(compressed[:qCompressHeaderSize])
	zr, err := zlib.NewReader(bytes.NewReader(compressed[qCompressHeaderSize:]))
	if err != nil {
		t.Fatalf("zlib reader: %v", err)
	}
	defer zr.Close()

	var inflated bytes.Buffer
	if _, err := inflated.ReadFrom(zr); err != nil {
		t.Fatalf("zlib inflate: %v", err)
	}

	if uint32(inflated.Len()) != expectedLen {
		t.Fatalf("uncompressed length mismatch: header says %d, got %d", expectedLen, inflated.Len())
	}

	return inflated.Bytes()
}

// envelopeMap is a test helper that decodes a vpn:// link into a generic map
// for field assertions.
func envelopeMap(t *testing.T, link string) map[string]any {
	t.Helper()

	raw := decodeVPNLink(t, link)

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal envelope JSON: %v\nraw: %s", err, raw)
	}

	return m
}

// lastConfigMap extracts and JSON-decodes containers[0].awg.last_config
// from the envelope.
func lastConfigMap(t *testing.T, env map[string]any) map[string]any {
	t.Helper()

	containers, ok := env["containers"].([]any)
	if !ok || len(containers) == 0 {
		t.Fatalf("no containers in envelope: %v", env)
	}

	container, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatal("containers[0] is not an object")
	}

	awg, ok := container["awg"].(map[string]any)
	if !ok {
		t.Fatal("containers[0].awg is not an object")
	}

	lastConfigStr, ok := awg["last_config"].(string)
	if !ok {
		t.Fatal("containers[0].awg.last_config is not a string")
	}

	var lc map[string]any
	if err := json.Unmarshal([]byte(lastConfigStr), &lc); err != nil {
		t.Fatalf("unmarshal last_config JSON: %v", err)
	}

	return lc
}

func TestEncodeVPNLink_DecodesToValidJSON(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		[]string{"1.1.1.1", "1.0.0.1"},
	)

	env := envelopeMap(t, link)

	if env["hostName"] != "vpn.example.com" {
		t.Errorf("hostName = %v, want vpn.example.com", env["hostName"])
	}
	if env["defaultContainer"] != vpnContainerAWG {
		t.Errorf("defaultContainer = %v, want %s", env["defaultContainer"], vpnContainerAWG)
	}

	containers, _ := env["containers"].([]any)
	container, _ := containers[0].(map[string]any)
	if container["container"] != vpnContainerAWG {
		t.Errorf("container type = %v, want %s", container["container"], vpnContainerAWG)
	}

	awg, _ := container["awg"].(map[string]any)
	if awg["isThirdPartyConfig"] != true {
		t.Errorf("isThirdPartyConfig = %v, want true", awg["isThirdPartyConfig"])
	}
	if awg["port"] != "51820" {
		t.Errorf("port = %v, want \"51820\"", awg["port"])
	}
	if awg["transport_proto"] != "udp" {
		t.Errorf("transport_proto = %v, want udp", awg["transport_proto"])
	}

	if env["dns1"] != "1.1.1.1" {
		t.Errorf("dns1 = %v, want 1.1.1.1", env["dns1"])
	}
	if env["dns2"] != "1.0.0.1" {
		t.Errorf("dns2 = %v, want 1.0.0.1", env["dns2"])
	}

	lc := lastConfigMap(t, env)

	configStr, ok := lc["config"].(string)
	if !ok {
		t.Fatal("last_config.config is not a string")
	}
	if !strings.Contains(configStr, "[Interface]") {
		t.Error("last_config.config missing [Interface] section")
	}
	if !strings.Contains(configStr, "S3 =") {
		t.Error("last_config.config missing S3 =")
	}
}

func TestEncodeVPNLink_NoBase64Padding(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
	)

	encoded := strings.TrimPrefix(link, vpnLinkScheme)
	if strings.Contains(encoded, "=") {
		t.Errorf("encoded link contains '=' padding: %q", encoded[:min(len(encoded), 40)])
	}
}

func TestEncodeVPNLink_NoSSHCredentials(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
	)

	raw := decodeVPNLink(t, link)

	// The SSH credential keys must not appear anywhere in the JSON payload.
	if bytes.Contains(raw, []byte(`"userName"`)) {
		t.Error("envelope JSON contains SSH userName field")
	}
	if bytes.Contains(raw, []byte(`"password"`)) {
		t.Error("envelope JSON contains SSH password field")
	}
}

func TestEncodeVPNLink_PreservesAWG2Fields(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
	)

	lc := lastConfigMap(t, envelopeMap(t, link))

	configStr, ok := lc["config"].(string)
	if !ok {
		t.Fatal("last_config.config is not a string")
	}

	checks := []struct {
		name string
		want string
	}{
		{"S3", "S3 ="},
		{"S4", "S4 ="},
		{"H1 range", "H1 = 100-200"},
	}
	for _, c := range checks {
		if !strings.Contains(configStr, c.want) {
			t.Errorf("last_config.config missing %s: want %q", c.name, c.want)
		}
	}

	// At least one I-packet interval must be present.
	hasInterval := strings.Contains(configStr, "I1 =") ||
		strings.Contains(configStr, "I2 =") ||
		strings.Contains(configStr, "I3 =") ||
		strings.Contains(configStr, "I4 =")
	if !hasInterval {
		t.Error("last_config.config has no I1-I4 interval")
	}
}

func TestEncodeVPNLink_EndpointWithoutPort(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com",
		55424,
		nil,
	)

	env := envelopeMap(t, link)

	if env["hostName"] != "vpn.example.com" {
		t.Errorf("hostName = %v, want vpn.example.com", env["hostName"])
	}

	containers, _ := env["containers"].([]any)
	container, _ := containers[0].(map[string]any)
	awg, _ := container["awg"].(map[string]any)
	if awg["port"] != "55424" {
		t.Errorf("port = %v, want \"55424\"", awg["port"])
	}

	lc := lastConfigMap(t, env)
	if lc["hostName"] != "vpn.example.com" {
		t.Errorf("last_config.hostName = %v, want vpn.example.com", lc["hostName"])
	}
	if port, _ := lc["port"].(float64); port != 55424 {
		t.Errorf("last_config.port = %v, want 55424", lc["port"])
	}
}

func TestEncodeVPNLink_DNSOmit(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
	)

	env := envelopeMap(t, link)

	if _, ok := env["dns1"]; ok {
		t.Error("dns1 present with empty DNS slice, should be omitted")
	}
	if _, ok := env["dns2"]; ok {
		t.Error("dns2 present with empty DNS slice, should be omitted")
	}

	// Single DNS → dns1 set, dns2 omitted.
	link1 := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		[]string{"8.8.8.8"},
	)
	env1 := envelopeMap(t, link1)
	if env1["dns1"] != "8.8.8.8" {
		t.Errorf("dns1 = %v, want 8.8.8.8", env1["dns1"])
	}
	if _, ok := env1["dns2"]; ok {
		t.Error("dns2 present with single-element DNS, should be omitted")
	}
}

func TestGenerate_VPNLinks(t *testing.T) {
	manifest, err := LoadManifest(filepath.Join("testdata", "loader", "valid"), nil)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	result, err := Generate(manifest, GenerateOptions{
		ProjectDir: filepath.Join("testdata", "loader", "valid"),
		DryRun:     true,
		VPNLinks:   true,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	serverName := result.ServerPeer
	if serverName == "" {
		t.Fatal("no server peer in result")
	}

	var vpnFiles, confFiles int
	for _, f := range result.Files {
		if strings.HasSuffix(f.RelPath, outputVPNLinkName) {
			vpnFiles++
			// .vpn files must only appear under client peer dirs, never the server.
			if strings.HasPrefix(f.RelPath, serverName+"/") {
				t.Errorf("server peer has a .vpn file: %s", f.RelPath)
			}
			if !bytes.HasPrefix(f.Content, []byte(vpnLinkScheme)) {
				t.Errorf("vpn file %s does not start with %s", f.RelPath, vpnLinkScheme)
			}
		}
		if strings.HasSuffix(f.RelPath, outputConfigName) {
			confFiles++
		}
	}

	if vpnFiles == 0 {
		t.Error("no .vpn files generated, expected one per client peer")
	}

	// Every client peer should have both a .conf and a .vpn.
	for _, peerName := range result.ClientPeers {
		hasConf := false
		hasVPN := false
		for _, f := range result.Files {
			if f.RelPath == peerName+"/"+outputConfigName {
				hasConf = true
			}
			if f.RelPath == peerName+"/"+outputVPNLinkName {
				hasVPN = true
			}
		}
		if !hasConf {
			t.Errorf("client %s missing .conf file", peerName)
		}
		if !hasVPN {
			t.Errorf("client %s missing .vpn file", peerName)
		}
	}

	// Sanity: server gets a .conf but no .vpn.
	serverConf := serverName + "/" + outputConfigName
	foundServerConf := false
	for _, f := range result.Files {
		if f.RelPath == serverConf {
			foundServerConf = true
		}
	}
	if !foundServerConf {
		t.Error("server .conf file missing")
	}

	// Verify one vpn:// link round-trips to valid JSON.
	for _, f := range result.Files {
		if strings.HasSuffix(f.RelPath, outputVPNLinkName) {
			_ = envelopeMap(t, string(f.Content))
		}
	}
}
