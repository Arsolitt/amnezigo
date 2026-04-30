package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Arsolitt/amnezigo"
)

// writeTestConfig writes a minimal valid server config to path and returns it.
func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()

	serverPriv, _ := amnezigo.GenerateKeyPair()
	clientPriv, clientPub := amnezigo.GenerateKeyPair()

	configPath := filepath.Join(dir, "awg0.conf")
	content := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 10.8.0.1/24
ListenPort = 51820
MTU = 1280
Jc = 3
Jmin = 500
Jmax = 900
S1 = 10
S2 = 20
S3 = 30
S4 = 8
H1 = 100000000
H2 = 300000000
H3 = 500000000
H4 = 700000000
#_EndpointV4 = 1.2.3.4:51820

[Peer]
#_Name = laptop
#_PrivateKey = %s
PublicKey = %s
PresharedKey = testpsk123
AllowedIPs = 10.8.0.2/32
#_GenKeyTime = 2024-03-17T12:00:00Z
`, serverPriv, clientPriv, clientPub)

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	return configPath
}

func TestAnalyzeCommand_TextOutput(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", configPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "AmneziaWG Config Analysis") {
		t.Error("expected text output header")
	}
	if !strings.Contains(output, "Handshake Sizes") {
		t.Error("expected Handshake Sizes section")
	}
}

func TestAnalyzeCommand_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", configPath, "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze command failed: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Error("expected JSON output to start with '{'")
	}
	if !strings.Contains(output, `"peers"`) {
		t.Error("JSON output missing 'peers' key")
	}
}

func TestAnalyzeCommand_PeerFilter(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", configPath, "--peer", "laptop"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "laptop") {
		t.Error("expected output to contain filtered peer name")
	}
}

func TestAnalyzeCommand_ProtocolFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", configPath, "--protocol", "quic"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze command with --protocol quic failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "quic") {
		t.Error("expected output to mention protocol quic")
	}
}

func TestAnalyzeCommand_SamplesFlag(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", configPath, "--samples", "10"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze command with --samples failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Distribution") {
		t.Error("expected output to contain Distribution section when samples > 0")
	}
}

func TestAnalyzeCommand_SeedFlag_Accepted(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"--config", configPath,
		"--output", "json",
		"--seed", "42",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("analyze with --seed failed: %v", err)
	}

	output := buf.String()
	if !strings.HasPrefix(strings.TrimSpace(output), "{") {
		t.Error("expected JSON output")
	}
}

func TestAnalyzeCommand_InvalidConfig(t *testing.T) {
	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config", "/nonexistent/path/awg0.conf"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid config path")
	}
}

func TestAnalyzeCommand_InvalidOutput(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := writeTestConfig(t, tmpDir)

	cmd := NewAnalyzeCommand()
	buf := &strings.Builder{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--config", configPath, "--output", "xml"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid output format")
	}
}
