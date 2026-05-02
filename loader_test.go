package amnezigo

import (
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
