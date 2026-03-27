package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Arsolitt/amnezigo"
)

func TestEdgeAddCommand(t *testing.T) {
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	t.Run("add edge with auto-assigned IP", func(t *testing.T) {
		cfgFile = ""
		clientIPAddr = ""
		edgeIPAddr = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 10.8.0.1/24
ListenPort = 12345
MTU = 1280
Jc = 5
Jmin = 20
Jmax = 30
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400

[Peer]
#_Role = client
#_Name = existing
PublicKey = existingpub1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewEdgeAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "moscow"})
		cfgFile = configPath
		if err := cmd.Execute(); err != nil {
			t.Fatalf("edge add failed: %v", err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		configStr := string(content)

		if !strings.Contains(configStr, `#_Name = moscow`) {
			t.Error("expected edge name in config")
		}
		if !strings.Contains(configStr, `#_Role = edge`) {
			t.Error("expected #_Role = edge")
		}
		if !strings.Contains(configStr, `AllowedIPs = 10.8.0.3/32`) {
			t.Error("expected auto-assigned IP 10.8.0.3/32 (skipping .2 used by client)")
		}
	})
}

func TestEdgeExportCommand(t *testing.T) {
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	t.Run("export edge config", func(t *testing.T) {
		cfgFile = ""
		clientProtocol = ""
		edgeProtocol = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		serverPriv, _ := amnezigo.GenerateKeyPair()
		edgePriv, edgePub := amnezigo.GenerateKeyPair()

		initialConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
Jc = 3
Jmin = 64
Jmax = 64
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400
#_EndpointV4 = 1.2.3.4:55424

[Peer]
#_Role = edge
#_Name = moscow
#_PrivateKey = %s
PublicKey = %s
PresharedKey = edgepsk123
AllowedIPs = 10.8.0.3/32
#_GenKeyTime = 2024-03-17T12:00:00Z
`, serverPriv, edgePriv, edgePub)
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		t.Chdir(tmpDir)

		cmd := NewEdgeExportCommand()
		cmd.SetArgs([]string{"--config", configPath, "moscow"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("edge export failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(tmpDir, "moscow.conf"))
		if err != nil {
			t.Fatalf("failed to read edge config: %v", err)
		}
		configStr := string(content)

		if !strings.Contains(configStr, "PrivateKey = "+edgePriv) {
			t.Error("expected edge private key in config")
		}
		if !strings.Contains(configStr, "AllowedIPs = 10.8.0.1/32") {
			t.Error("expected hub IP as AllowedIPs")
		}
		if strings.Contains(configStr, "DNS =") {
			t.Error("edge config should not have DNS")
		}
	})
}

func TestEdgeRemoveCommand(t *testing.T) {
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	t.Run("remove existing edge", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 10.8.0.1/24
ListenPort = 12345
MTU = 1280
Jc = 5
Jmin = 20
Jmax = 30
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400

[Peer]
#_Role = edge
#_Name = moscow
PublicKey = pub1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = priv1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2026-03-17T10:00:00Z

[Peer]
#_Role = edge
#_Name = berlin
PublicKey = pub22345678901234567890123456789012345678901234567890123456789=
AllowedIPs = 10.8.0.3/32
#_PrivateKey = priv2234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2026-03-17T11:00:00Z
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cfgFile = configPath
		if err := runEdgeRemove(nil, []string{"moscow"}); err != nil {
			t.Fatalf("edge remove failed: %v", err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		configStr := string(content)

		if strings.Contains(configStr, "moscow") {
			t.Error("expected removed edge to not be in config")
		}
		if !strings.Contains(configStr, "berlin") {
			t.Error("expected berlin to still be in config")
		}
	})

	t.Run("error when edge not found", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 10.8.0.1/24
ListenPort = 12345
MTU = 1280
Jc = 5
Jmin = 20
Jmax = 30
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cfgFile = configPath
		err := runEdgeRemove(nil, []string{"nonexistent"})
		if err == nil {
			t.Error("expected error for nonexistent edge, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected error about edge not found, got: %v", err)
		}
	})
}

func TestEdgeListCommand(t *testing.T) {
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	t.Run("list edges", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 10.8.0.1/24
ListenPort = 12345
MTU = 1280
Jc = 5
Jmin = 20
Jmax = 30
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400

[Peer]
#_Role = edge
#_Name = moscow
PublicKey = pub1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = priv1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2024-01-15T10:30:00Z

[Peer]
#_Role = edge
#_Name = berlin
PublicKey = pub22345678901234567890123456789012345678901234567890123456789=
AllowedIPs = 10.8.0.3/32
#_PrivateKey = priv2234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2024-01-16T14:22:00Z
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewEdgeListCommand()
		cmd.SetArgs([]string{"--config", configPath})
		cfgFile = configPath

		output := &strings.Builder{}
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		w.Close()
		os.Stdout = oldStdout

		var buf [4096]byte
		n, _ := r.Read(buf[:])
		output.Write(buf[:n])

		if err != nil {
			t.Fatalf("edge list failed: %v", err)
		}

		outputStr := output.String()
		if !strings.Contains(outputStr, "moscow") {
			t.Error("expected 'moscow' in output")
		}
		if !strings.Contains(outputStr, "berlin") {
			t.Error("expected 'berlin' in output")
		}
	})

	t.Run("list with no edges shows message", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 10.8.0.1/24
ListenPort = 12345
MTU = 1280
Jc = 5
Jmin = 20
Jmax = 30
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewEdgeListCommand()
		cmd.SetArgs([]string{"--config", configPath})
		cfgFile = configPath

		output := &strings.Builder{}
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		w.Close()
		os.Stdout = oldStdout

		var buf [4096]byte
		n, _ := r.Read(buf[:])
		output.Write(buf[:n])

		if err != nil {
			t.Fatalf("edge list failed: %v", err)
		}

		if !strings.Contains(output.String(), "No edges configured") {
			t.Error("expected 'No edges configured' in output")
		}
	})
}
