package amnezigo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_ValidJSON(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "valid")
	m, err := LoadManifest(dir, nil)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error: %v", dir, err)
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if m.Network.MTU != 1280 {
		t.Errorf("Network.MTU = %d, want 1280", m.Network.MTU)
	}
	if len(m.Peers) == 0 {
		t.Fatal("Peers map is empty, expected at least 1 peer")
	}
}

func TestLoadManifest_ValidJsonnet(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "valid-jsonnet")
	m, err := LoadManifest(dir, nil)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error: %v", dir, err)
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if _, ok := m.Peers["laptop"]; !ok {
		t.Error("expected 'laptop' peer in manifest")
	}
}

func TestLoadManifest_JsonnetWithLibImport(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "valid-jsonnet-with-lib")
	// lib/ is relative to manifest dir — default jpath
	m, err := LoadManifest(dir, nil)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error: %v", dir, err)
	}
	// The imported network.libsonnet sets MTU to 1420
	if m.Network.MTU != 1420 {
		t.Errorf("Network.MTU = %d, want 1420 (from imported lib)", m.Network.MTU)
	}
}

func TestLoadManifest_JsonnetPrecedence(t *testing.T) {
	// When both .amnezigo.jsonnet and amnezigo.json exist,
	// jsonnet must take precedence.
	dir := filepath.Join("testdata", "loader", "precedence")
	m, err := LoadManifest(dir, nil)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error: %v", dir, err)
	}
	server, ok := m.Peers["server"]
	if !ok {
		t.Fatal("missing 'server' peer")
	}
	if !strings.Contains(server.Endpoint, "from-jsonnet") {
		t.Errorf("Endpoint = %q, want to contain 'from-jsonnet' (jsonnet takes precedence)", server.Endpoint)
	}
}

func TestLoadManifest_JSONOnly(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "json-only")
	m, err := LoadManifest(dir, nil)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error: %v", dir, err)
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
}

func TestLoadManifest_EmptyDir(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "empty-dir")
	_, err := LoadManifest(dir, nil)
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
	if !strings.Contains(err.Error(), "no manifest file found") {
		t.Errorf("error = %q, want to contain 'no manifest file found'", err.Error())
	}
}

func TestLoadManifest_NonexistentDir(t *testing.T) {
	_, err := LoadManifest("/nonexistent/path", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent directory, got nil")
	}
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "invalid-json")
	_, err := LoadManifest(dir, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	// Error should mention the file path for user orientation
	if !strings.Contains(err.Error(), "amnezigo.json") {
		t.Errorf("error = %q, want to contain file name", err.Error())
	}
}

func TestLoadManifest_MissingVersion(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "invalid-version")
	_, err := LoadManifest(dir, nil)
	if err == nil {
		t.Fatal("expected error for missing/zero version, got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error = %q, want to mention 'version'", err.Error())
	}
}

func TestLoadManifest_UnsupportedVersion(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "unsupported-version")
	_, err := LoadManifest(dir, nil)
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %q, want to contain 'unsupported'", err.Error())
	}
}

func TestLoadManifest_JsonnetImportError(t *testing.T) {
	dir := filepath.Join("testdata", "loader", "jsonnet-import-error")
	_, err := LoadManifest(dir, nil)
	if err == nil {
		t.Fatal("expected error for Jsonnet import failure, got nil")
	}
	// go-jsonnet error includes the file path and import target
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error = %q, want to mention the missing import", err.Error())
	}
}

func TestLoadManifestFromFile_ExplicitJSON(t *testing.T) {
	path := filepath.Join("testdata", "loader", "valid", "amnezigo.json")
	m, err := LoadManifestFromFile(path, nil)
	if err != nil {
		t.Fatalf("LoadManifestFromFile(%q) error: %v", path, err)
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
}

func TestLoadManifestFromFile_ExplicitJsonnet(t *testing.T) {
	path := filepath.Join("testdata", "loader", "valid-jsonnet", ".amnezigo.jsonnet")
	m, err := LoadManifestFromFile(path, nil)
	if err != nil {
		t.Fatalf("LoadManifestFromFile(%q) error: %v", path, err)
	}
	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
}

func TestLoadManifestFromFile_NonexistentFile(t *testing.T) {
	_, err := LoadManifestFromFile("/nonexistent/file.json", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestLoadManifest_CustomJpath(t *testing.T) {
	// Use valid-jsonnet-with-lib but pass jpath explicitly
	// pointing to the lib/ directory
	dir := filepath.Join("testdata", "loader", "valid-jsonnet-with-lib")
	libDir := filepath.Join(dir, "lib")
	m, err := LoadManifest(dir, []string{libDir})
	if err != nil {
		t.Fatalf("LoadManifest with custom jpath error: %v", err)
	}
	if m.Network.MTU != 1420 {
		t.Errorf("Network.MTU = %d, want 1420", m.Network.MTU)
	}
}

// TestLoadManifest_DefaultJpathIncludesLib verifies that the default jpath
// includes lib/ relative to the manifest directory even when jpathDirs is nil.
// This is the convention from cheburbox and P3.3 depends on it.
func TestLoadManifest_DefaultJpathIncludesLib(t *testing.T) {
	// The valid-jsonnet-with-lib fixture imports from lib/ without explicit jpath
	dir := filepath.Join("testdata", "loader", "valid-jsonnet-with-lib")
	m, err := LoadManifest(dir, nil)
	if err != nil {
		t.Fatalf("LoadManifest with default jpath error: %v", err)
	}
	if m.Network.MTU != 1420 {
		t.Errorf("expected lib import to work with default jpath, got MTU = %d", m.Network.MTU)
	}
}

func TestLoadManifest_PermissionDenied(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, manifestJSON)
	if err := os.WriteFile(path, []byte(`{"version":1}`), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadManifest(tmp, nil)
	if err == nil {
		t.Fatal("expected error for unreadable file, got nil")
	}
}

func TestLoadManifestFromFile_JsonnetExtensionDetection(t *testing.T) {
	// A .jsonnet file must be evaluated through the Jsonnet VM,
	// not parsed as raw JSON.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "custom.jsonnet")
	content := `{
    version: 1,
    network: { mtu: 1500 },
    obfuscation: {},
    peers: {},
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := LoadManifestFromFile(path, nil)
	if err != nil {
		t.Fatalf("LoadManifestFromFile error: %v", err)
	}
	if m.Network.MTU != 1500 {
		t.Errorf("MTU = %d, want 1500", m.Network.MTU)
	}
}

func TestLoadManifest_LargeManifest_ManyPeers(t *testing.T) {
	// Verify the loader handles manifests with many peers without issues.
	tmp := t.TempDir()
	// Generate a manifest with 100 peers via Jsonnet
	content := `
local peers = {
    ['peer-%03d' % i]: { address: '10.0.%d.%d/32' % [i / 256, i % 256] }
    for i in std.range(1, 100)
};
{
    version: 1,
    network: { mtu: 1280 },
    obfuscation: {},
    peers: { server: { address: '10.0.0.1/24', endpoint: 'vpn:51820', listen_port: 51820 } } + peers,
}
`
	path := filepath.Join(tmp, manifestJsonnet)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := LoadManifest(tmp, nil)
	if err != nil {
		t.Fatalf("LoadManifest error: %v", err)
	}
	// 100 generated peers + 1 server
	if len(m.Peers) != 101 {
		t.Errorf("len(Peers) = %d, want 101", len(m.Peers))
	}
}

func TestLoadManifest_JsonnetSyntaxError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, manifestJsonnet)
	if err := os.WriteFile(path, []byte(`{ version: 1, broken`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadManifest(tmp, nil)
	if err == nil {
		t.Fatal("expected error for Jsonnet syntax error, got nil")
	}
	// go-jsonnet errors include file:line
	if !strings.Contains(err.Error(), manifestJsonnet) {
		t.Errorf("error = %q, want to mention the Jsonnet file", err.Error())
	}
}

func TestLoadManifest_EmptyJSONFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, manifestJSON)
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadManifest(tmp, nil)
	if err == nil {
		t.Fatal("expected error for empty JSON (version=0), got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error = %q, want to mention 'version'", err.Error())
	}
}

func TestLoadManifestFromFile_JSONExtension(t *testing.T) {
	// .json file must be parsed as plain JSON, not through Jsonnet VM.
	// Jsonnet syntax (trailing commas, unquoted keys) must cause an error.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "manifest.json")
	if err := os.WriteFile(path, []byte(`{version: 1}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadManifestFromFile(path, nil)
	if err == nil {
		t.Fatal("expected JSON parse error for Jsonnet-style syntax in .json file")
	}
}
