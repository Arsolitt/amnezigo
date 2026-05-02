package amnezigo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet"
)

const (
	manifestJSON    = "amnezigo.json"
	manifestJsonnet = ".amnezigo.jsonnet"

	// currentManifestVersion is the only schema version this loader supports.
	// Bump (and add migration logic) when the manifest schema evolves.
	currentManifestVersion = 1
)

// LoadManifest discovers and loads a manifest from dir. It checks for
// .amnezigo.jsonnet first (Jsonnet takes precedence), then amnezigo.json.
// If jpathDirs is nil, it defaults to [dir/lib] for Jsonnet library resolution.
// Returns an error if no manifest file is found or loading fails.
func LoadManifest(dir string, jpathDirs []string) (Manifest, error) {
	jsonnetPath := filepath.Join(dir, manifestJsonnet)
	jsonPath := filepath.Join(dir, manifestJSON)

	_, errJsonnet := os.Stat(jsonnetPath)
	_, errJSON := os.Stat(jsonPath)

	switch {
	case errJsonnet == nil:
		return loadFromJsonnet(jsonnetPath, resolveJpath(dir, jpathDirs))
	case errJSON == nil:
		return loadFromJSON(jsonPath)
	default:
		return Manifest{}, fmt.Errorf(
			"no manifest file found in %s (expected %s or %s)",
			dir, manifestJsonnet, manifestJSON)
	}
}

// LoadManifestFromFile loads a manifest from an explicit file path. File
// extension determines evaluation: .jsonnet → Jsonnet VM, otherwise → plain
// JSON. If jpathDirs is nil, it defaults to [parentDir/lib].
func LoadManifestFromFile(path string, jpathDirs []string) (Manifest, error) {
	dir := filepath.Dir(path)

	if strings.HasSuffix(path, ".jsonnet") {
		return loadFromJsonnet(path, resolveJpath(dir, jpathDirs))
	}

	return loadFromJSON(path)
}

// resolveJpath returns jpathDirs if non-empty, or the default [dir/lib].
func resolveJpath(dir string, jpathDirs []string) []string {
	if len(jpathDirs) > 0 {
		return jpathDirs
	}

	return []string{filepath.Join(dir, "lib")}
}

// loadFromJSON reads and parses a JSON manifest file.
func loadFromJSON(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", path, err)
	}

	if err := validateManifestVersion(path, m.Version); err != nil {
		return Manifest{}, err
	}

	return m, nil
}

// loadFromJsonnet evaluates a Jsonnet file and parses the resulting JSON.
func loadFromJsonnet(path string, jpathDirs []string) (Manifest, error) {
	vm := createJsonnetVM(jpathDirs)

	output, err := vm.EvaluateFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("evaluate jsonnet %s: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		return Manifest{}, fmt.Errorf("parse jsonnet output from %s: %w", path, err)
	}

	if err := validateManifestVersion(path, m.Version); err != nil {
		return Manifest{}, err
	}

	return m, nil
}

// createJsonnetVM creates a Jsonnet VM with the given library search paths.
func createJsonnetVM(jpathDirs []string) *jsonnet.VM {
	vm := jsonnet.MakeVM()
	if len(jpathDirs) > 0 {
		vm.Importer(&jsonnet.FileImporter{
			JPaths: jpathDirs,
		})
	}

	return vm
}

// validateManifestVersion checks that the manifest version is supported.
func validateManifestVersion(path string, version int) error {
	if version == 0 {
		return fmt.Errorf("%s: missing or zero version field", path)
	}

	if version != currentManifestVersion {
		return fmt.Errorf(
			"%s: unsupported schema version %d (expected %d)",
			path, version, currentManifestVersion)
	}

	return nil
}
