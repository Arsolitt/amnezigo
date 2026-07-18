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
		"",
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
		"",
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
		"",
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
		"",
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
		"",
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
		"",
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
		"",
	)
	env1 := envelopeMap(t, link1)
	if env1["dns1"] != "8.8.8.8" {
		t.Errorf("dns1 = %v, want 8.8.8.8", env1["dns1"])
	}
	if _, ok := env1["dns2"]; ok {
		t.Error("dns2 present with single-element DNS, should be omitted")
	}
}

func TestEncodeVPNLink_DescriptionInEnvelope(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
		"My Server",
	)

	env := envelopeMap(t, link)

	if env["description"] != "My Server" {
		t.Errorf("description = %v, want %q", env["description"], "My Server")
	}
}

func TestEncodeVPNLink_DescriptionOmittedWhenEmpty(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
		"",
	)

	env := envelopeMap(t, link)

	// omitempty must drop the key entirely (not just empty string) so the
	// AmneziaVPN app falls back to hostName for the displayed server name.
	if _, ok := env["description"]; ok {
		t.Error("description present with empty string, should be omitted (omitempty)")
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

func TestGenerate_VPNLinks_DisplayName(t *testing.T) {
	manifest, err := LoadManifest(filepath.Join("testdata", "loader", "valid"), nil)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// Set a display name on the client peer before generating. Peers is a map,
	// so the in-place mutation is visible to Generate even though the manifest
	// is passed by value.
	const wantDescription = "Alice's phone"
	phonePeer := manifest.Peers["phone"]
	phonePeer.DisplayName = wantDescription
	manifest.Peers["phone"] = phonePeer

	result, err := Generate(manifest, GenerateOptions{
		ProjectDir: filepath.Join("testdata", "loader", "valid"),
		DryRun:     true,
		VPNLinks:   true,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Find the phone peer's .vpn output and decode the envelope.
	var phoneVPN *FileOutput
	for i := range result.Files {
		if result.Files[i].RelPath == "phone/"+outputVPNLinkName {
			phoneVPN = &result.Files[i]
			break
		}
	}
	if phoneVPN == nil {
		t.Fatal("phone peer .vpn file not found in output")
	}

	env := envelopeMap(t, string(phoneVPN.Content))
	if env["description"] != wantDescription {
		t.Errorf("description = %v, want %q", env["description"], wantDescription)
	}
}

// TestEncodeVPNLink_StructuredLastConfig verifies that last_config contains
// the structured JSON fields the AmneziaVPN connect path reads via
// configWireguard() — not just the raw INI in "config". Without these fields,
// the app throws JSONException on getString("client_priv_key") → error 1000.
func TestEncodeVPNLink_StructuredLastConfig(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
		"",
	)

	lc := lastConfigMap(t, envelopeMap(t, link))

	// Required structured fields — configWireguard reads these via getString/getInt.
	requiredFields := []struct {
		key string
		val string
	}{
		{"client_priv_key", "oN+f...exampleKey...="},
		{"client_ip", "10.0.0.2/32"},
		{"server_pub_key", "SERVER_PUB_KEY_EXAMPLE"},
		{"hostName", "vpn.example.com"},
	}
	for _, f := range requiredFields {
		got, ok := lc[f.key].(string)
		if !ok {
			t.Errorf("last_config.%s is not a string, got %T", f.key, lc[f.key])
			continue
		}
		if got != f.val {
			t.Errorf("last_config.%s = %q, want %q", f.key, got, f.val)
		}
	}

	// Port must be a JSON number (int), not a string.
	if port, ok := lc["port"].(float64); !ok || port != 51820 {
		t.Errorf("last_config.port = %v, want 51820 (number)", lc["port"])
	}

	// allowed_ips must be a JSON array, not a comma-separated string.
	allowedIPs, ok := lc["allowed_ips"].([]any)
	if !ok {
		t.Fatalf("last_config.allowed_ips is not a JSON array, got %T", lc["allowed_ips"])
	}
	if len(allowedIPs) == 0 {
		t.Fatal("last_config.allowed_ips is empty")
	}
	if allowedIPs[0] != "0.0.0.0/0" {
		t.Errorf("last_config.allowed_ips[0] = %v, want 0.0.0.0/0", allowedIPs[0])
	}

	// isObfuscationEnabled must be true for AWG configs.
	if obf, ok := lc["isObfuscationEnabled"].(bool); !ok || !obf {
		t.Errorf("last_config.isObfuscationEnabled = %v, want true", lc["isObfuscationEnabled"])
	}

	// AWG obfuscation fields must be present as strings.
	obfFields := []string{"Jc", "Jmin", "Jmax", "S1", "S2", "S3", "S4", "H1", "H2", "H3", "H4"}
	for _, f := range obfFields {
		if _, ok := lc[f].(string); !ok {
			t.Errorf("last_config.%s is not a string, got %T", f, lc[f])
		}
	}

	// H1 range format must be preserved.
	if h1, _ := lc["H1"].(string); h1 != "100-200" {
		t.Errorf("last_config.H1 = %q, want 100-200", h1)
	}

	// MTU must be present (JSON key is lowercase "mtu", matching Android's configData.optStringOrNull("mtu")).
	if mtu := lc["mtu"]; mtu == nil {
		t.Error("last_config.mtu is missing")
	} else if mtu.(string) != "1280" {
		t.Errorf("last_config.mtu = %v, want 1280", mtu)
	}

	// PSK must be present.
	if psk, _ := lc["psk_key"].(string); psk != "PSK_EXAMPLE_VALUE_HERE" {
		t.Errorf("last_config.psk_key = %q, want PSK_EXAMPLE_VALUE_HERE", psk)
	}

	// PersistentKeepalive must be present.
	if ka, _ := lc["persistent_keep_alive"].(string); ka != "25" {
		t.Errorf("last_config.persistent_keep_alive = %q, want 25", ka)
	}
}

// TestEncodeVPNLink_I1I5CPSStringsPreserved verifies that I1-I5 values in
// last_config carry the original CPS tag strings verbatim (not expanded to
// hex). The amneziawg-go UAPI handler fully parses CPS tags at connect time,
// so expansion would freeze random bytes and reduce obfuscation quality.
func TestEncodeVPNLink_I1I5CPSStringsPreserved(t *testing.T) {
	link := EncodeVPNLink(
		[]byte(sampleClientINI),
		"vpn.example.com:51820",
		51820,
		nil,
		"",
	)

	lc := lastConfigMap(t, envelopeMap(t, link))

	// I1-I4 must contain CPS tag notation (starting with "<").
	for _, key := range []string{"I1", "I2", "I3", "I4"} {
		val, ok := lc[key].(string)
		if !ok {
			t.Errorf("last_config.%s is not a string, got %T", key, lc[key])
			continue
		}
		if !strings.HasPrefix(val, "<") {
			t.Errorf("last_config.%s = %q, want CPS tag string starting with '<'", key, val)
		}
		if !strings.Contains(val, "<b 0x") {
			t.Errorf("last_config.%s = %q, missing <b 0x...> tag", key, val)
		}
	}

	// I5 should be absent (empty in sampleClientINI — omitted via omitempty tag).
	if _, ok := lc["I5"]; ok {
		t.Error("last_config.I5 should be absent (empty in sample INI)")
	}
}

// TestEncodeVPNLink_AllowedIPsJSONArray verifies that allowed_ips is a JSON
// array even when the INI has multiple comma-separated values. The app's
// configWireguard reads it via getJSONArray — a string would throw JSONException.
func TestEncodeVPNLink_AllowedIPsJSONArray(t *testing.T) {
	// Config with multiple AllowedIPs entries.
	multiIPINI := strings.Replace(sampleClientINI, "AllowedIPs = 0.0.0.0/0",
		"AllowedIPs = 0.0.0.0/0, ::/0", 1)

	link := EncodeVPNLink([]byte(multiIPINI), "vpn.example.com:51820", 51820, nil, "")
	lc := lastConfigMap(t, envelopeMap(t, link))

	allowedIPs, ok := lc["allowed_ips"].([]any)
	if !ok {
		t.Fatalf("last_config.allowed_ips is not a JSON array, got %T", lc["allowed_ips"])
	}
	if len(allowedIPs) != 2 {
		t.Fatalf("last_config.allowed_ips has %d entries, want 2", len(allowedIPs))
	}
	if allowedIPs[0] != "0.0.0.0/0" {
		t.Errorf("allowed_ips[0] = %v, want 0.0.0.0/0", allowedIPs[0])
	}
	if allowedIPs[1] != "::/0" {
		t.Errorf("allowed_ips[1] = %v, want ::/0", allowedIPs[1])
	}
}
