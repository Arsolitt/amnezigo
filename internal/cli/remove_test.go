package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveCommand(t *testing.T) {
	// Test 1: Removing existing client
	t.Run("remove existing client", func(t *testing.T) {
		// Create temporary directory
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Create initial config with server interface and two peers
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
H1 = 100
H2 = 200
H3 = 300
H4 = 400
I1 = abc
I2 = def
I3 = ghi
I4 = jkl
I5 = mno

[Peer]
#_Name = client-to-remove
#_Role = client
PublicKey = clientpublickey1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = clientprivatekey1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2026-03-17T10:00:00Z

[Peer]
#_Name = another-client
#_Role = client
PublicKey = anotherpublickey1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.3/32
#_PrivateKey = anotherprivatekey1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2026-03-17T11:00:00Z
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		// Execute remove command by calling runClientRemove directly
		// to avoid Cobra framework issues with global state
		// Set cfgFile explicitly since PersistentPreRunE won't run
		cfgFile = configPath
		if err := runClientRemove(nil, []string{"client-to-remove"}); err != nil {
			t.Fatalf("remove command failed: %v", err)
		}

		// Read back config and verify peer was removed
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		configStr := string(content)

		// Verify the removed peer is not in config
		if strings.Contains(configStr, "client-to-remove") {
			t.Error("expected removed client to not be in config")
		}

		// Verify the other peer is still in config
		if !strings.Contains(configStr, "another-client") {
			t.Error("expected another-client to still be in config")
		}

		// Verify we still have one [Peer] section
		peerCount := strings.Count(configStr, "[Peer]")
		if peerCount != 1 {
			t.Errorf("expected 1 peer section, got %d", peerCount)
		}
	})

	// Test 2: Error when client not found
	t.Run("error when client not found", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Create config with one peer
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
H1 = 100
H2 = 200
H3 = 300
H4 = 400
I1 = abc
I2 = def
I3 = ghi
I4 = jkl
I5 = mno

[Peer]
#_Name = existing-client
#_Role = client
PublicKey = existingpublickey1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = existingprivatekey1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2026-03-17T10:00:00Z
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		// Execute remove command by calling runClientRemove directly
		cfgFile = configPath
		err := runClientRemove(nil, []string{"nonexistent-client"})
		if err == nil {
			t.Error("expected error for nonexistent client, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected error about client not found, got: %v", err)
		}
	})
}
