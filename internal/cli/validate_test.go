package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Arsolitt/amnezigo"
)

// writeKnownGoodConfig generates and writes a valid server config to path.
func writeKnownGoodConfig(t *testing.T, path string) {
	t.Helper()
	obf := amnezigo.GenerateServerConfig(1280, 32, 5)
	cfg := amnezigo.ServerConfig{
		Interface: amnezigo.InterfaceConfig{
			PrivateKey: "aaa", PublicKey: "bbb",
			Address: "10.0.0.1/24", ListenPort: 51820, MTU: 1280,
		},
		Obfuscation: obf,
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	if err := amnezigo.WriteServerConfig(f, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// writeCollidingConfig writes a config with S1=0, S2=56 causing S-prefix collision.
func writeCollidingConfig(t *testing.T, path string) {
	t.Helper()
	obf := amnezigo.GenerateServerConfig(1280, 32, 5)
	obf.S1 = 0
	obf.S2 = 56 // 0+148 == 56+92 → collision
	cfg := amnezigo.ServerConfig{
		Interface: amnezigo.InterfaceConfig{
			PrivateKey: "aaa", PublicKey: "bbb",
			Address: "10.0.0.1/24", ListenPort: 51820, MTU: 1280,
		},
		Obfuscation: obf,
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	if err := amnezigo.WriteServerConfig(f, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// appendLine appends a line to a file.
func appendLine(t *testing.T, path, line string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatalf("append to %s: %v", path, err)
	}
}

func TestValidateCommand_CleanConfig(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "awg0.conf")
	writeKnownGoodConfig(t, path)

	var stdout, stderr bytes.Buffer
	cmd := NewValidateCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("validate failed on clean config: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "0 errors") {
		t.Errorf("expected '0 errors' summary, got: %q", stdout.String())
	}
}

func TestValidateCommand_FailsOnSPrefixCollision(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "awg0.conf")
	writeCollidingConfig(t, path)

	var capturedCode int
	origExit := exitFn
	t.Cleanup(func() { exitFn = origExit })
	exitFn = func(code int) { capturedCode = code }

	cmd := NewValidateCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}
	if capturedCode != 1 {
		t.Errorf("expected exit code 1 on S-collision, got %d", capturedCode)
	}
	if !strings.Contains(stdout.String(), "PSC001") {
		t.Errorf("PSC001 not surfaced; got: %q", stdout.String())
	}
}

func TestValidateCommand_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "awg0.conf")
	writeKnownGoodConfig(t, path)

	cmd := NewValidateCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{path, "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("json validate failed: %v", err)
	}
	var doc struct {
		File     string             `json:"file"`
		Findings []amnezigo.Finding `json:"findings"`
		Summary  struct {
			Errors   int `json:"errors"`
			Warnings int `json:"warnings"`
			Info     int `json:"info"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout.String())
	}
	if doc.File != path {
		t.Errorf("file=%q, want %q", doc.File, path)
	}
	if doc.Summary.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", doc.Summary.Errors)
	}
}

func TestValidateCommand_StrictPromotesWarnings(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "awg0.conf")
	writeKnownGoodConfig(t, path)
	appendLine(t, path, "WeirdKey = whatever")

	var capturedCode int
	origExit := exitFn
	t.Cleanup(func() { exitFn = origExit })
	exitFn = func(code int) { capturedCode = code }

	cmd := NewValidateCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{path, "--strict"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}
	if capturedCode != 1 {
		t.Errorf("expected exit code 1 under --strict on warning, got %d", capturedCode)
	}
}

func TestValidateCommand_WarningsOnlyDoNotFail(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "awg0.conf")
	writeKnownGoodConfig(t, path)
	appendLine(t, path, "WeirdKey = whatever")

	cmd := NewValidateCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{path})

	// Should succeed (exit 0) — warnings don't fail without --strict.
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error on warning-only config: %v", err)
	}
	if !strings.Contains(stdout.String(), "KEY001") {
		t.Errorf("KEY001 not surfaced; got: %q", stdout.String())
	}
}

func TestValidateCommand_QuietSuppressesSummary(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "awg0.conf")
	writeKnownGoodConfig(t, path)

	cmd := NewValidateCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{path, "--quiet"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	// Quiet suppresses the summary line, so output should be empty for a clean config.
	if strings.TrimSpace(stdout.String()) != "" {
		t.Errorf("expected empty output with --quiet on clean config, got: %q", stdout.String())
	}
}
