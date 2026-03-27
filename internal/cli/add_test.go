package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Arsolitt/amnezigo"
)

func TestAddCommand(t *testing.T) {
	// Reset global state before each subtest
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	// Test 1: Adding client with auto-assigned IP
	t.Run("add client with auto-assigned IP", func(t *testing.T) {
		// Reset global state to default
		cfgFile = ""
		clientIPAddr = ""

		// Create temporary directory
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Create initial config with server interface
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
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		// Execute add command
		cmd := NewClientAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "client1"})
		// Ensure global state is reset
		cfgFile = configPath
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		// Read back config and verify peer was added
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		configStr := string(content)

		// Verify peer section was added
		if !strings.Contains(configStr, "[Peer]") {
			t.Error("expected [Peer] section in config")
		}

		// Verify #_Name
		if !strings.Contains(configStr, `#_Name = client1`) {
			t.Error("expected #_Name field with client1")
		}

		// Verify #_PrivateKey is present
		if !strings.Contains(configStr, "#_PrivateKey = ") {
			t.Error("expected #_PrivateKey field")
		}

		// Verify AllowedIPs with auto-assigned IP (should be 10.8.0.2)
		if !strings.Contains(configStr, `AllowedIPs = 10.8.0.2/32`) {
			t.Error("expected AllowedIPs with auto-assigned IP 10.8.0.2/32")
		}
	})

	// Test 2: Adding client with specific IP
	t.Run("add client with specific IP", func(t *testing.T) {
		cfgFile = ""
		clientIPAddr = ""
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
H1 = 100
H2 = 200
H3 = 300
H4 = 400
I1 = abc
I2 = def
I3 = ghi
I4 = jkl
I5 = mno
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewClientAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "--ipaddr", "10.8.0.50", "client2"})
		cfgFile = configPath
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		// Verify specific IP was used
		if !strings.Contains(string(content), `AllowedIPs = 10.8.0.50/32`) {
			t.Error("expected AllowedIPs with specified IP 10.8.0.50/32")
		}
	})

	// Test 3: Duplicate name rejection
	t.Run("reject duplicate client name", func(t *testing.T) {
		cfgFile = ""
		clientIPAddr = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Create config with an existing peer
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
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewClientAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "existing-client"})
		cfgFile = configPath

		err := cmd.Execute()
		if err == nil {
			t.Error("expected error for duplicate client name, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected error about existing client, got: %v", err)
		}
	})

	// Test 4: IP allocation from subnet
	t.Run("IP allocation skips .0 and .1", func(t *testing.T) {
		cfgFile = ""
		clientIPAddr = ""
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
H1 = 100
H2 = 200
H3 = 300
H4 = 400
I1 = abc
I2 = def
I3 = ghi
I4 = jkl
I5 = mno
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewClientAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "client-auto"})
		cfgFile = configPath
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		// First auto-assigned IP should be .2 (skipping .0 and .1)
		configStr := string(content)
		if !strings.Contains(configStr, `AllowedIPs = 10.8.0.2/32`) {
			t.Error("expected first auto-assigned IP to be 10.8.0.2/32, skipping .0 and .1")
		}

		// Second client should get .3
		cmd2 := NewClientAddCommand()
		cmd2.SetArgs([]string{"--config", configPath, "client-auto2"})
		cfgFile = configPath
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("add command failed for second client: %v", err)
		}

		content, err = os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		configStr = string(content)
		if !strings.Contains(configStr, `AllowedIPs = 10.8.0.3/32`) {
			t.Error("expected second auto-assigned IP to be 10.8.0.3/32")
		}
	})

	// Test 5: IP allocation with different subnet
	t.Run("IP allocation with different subnet", func(t *testing.T) {
		cfgFile = ""
		clientIPAddr = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Use 192.168.100.x subnet instead of 10.8.0.x
		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 192.168.100.1/24
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
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewClientAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "client-subnet1"})
		cfgFile = configPath
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add command failed: %v", err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		configStr := string(content)
		if !strings.Contains(configStr, `AllowedIPs = 192.168.100.2/32`) {
			t.Error("expected first auto-assigned IP to be 192.168.100.2/32")
		}

		// Second client should get .3
		cmd2 := NewClientAddCommand()
		cmd2.SetArgs([]string{"--config", configPath, "client-subnet2"})
		cfgFile = configPath
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("add command failed for second client: %v", err)
		}

		content, err = os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		configStr = string(content)
		if !strings.Contains(configStr, `AllowedIPs = 192.168.100.3/32`) {
			t.Error("expected second auto-assigned IP to be 192.168.100.3/32")
		}
	})
}

// TestFindNextAvailableIP tests the IP allocation logic.
func TestFindNextAvailableIP(t *testing.T) {
	tests := []struct {
		name          string
		serverAddress string
		expectedIP    string
		existingIPs   []string
		expectError   bool
	}{
		{
			name:          "first available IP in /24 subnet",
			serverAddress: "10.8.0.1/24",
			existingIPs:   []string{},
			expectedIP:    "10.8.0.2",
			expectError:   false,
		},
		{
			name:          "skip .0, .1 and used IPs",
			serverAddress: "10.8.0.1/24",
			existingIPs:   []string{"10.8.0.2", "10.8.0.3"},
			expectedIP:    "10.8.0.4",
			expectError:   false,
		},
		{
			name:          "skip server IP .1",
			serverAddress: "10.8.0.1/24",
			existingIPs:   []string{},
			expectedIP:    "10.8.0.2",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := amnezigo.FindNextAvailableIP(tt.serverAddress, tt.existingIPs)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ip != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

// TestCreatedAtTimestamp tests that CreatedAt field is set.
func TestCreatedAtTimestamp(t *testing.T) {
	cfgFile = ""
	clientIPAddr = ""
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
H1 = 100
H2 = 200
H3 = 300
H4 = 400
I1 = abc
I2 = def
I3 = ghi
I4 = jkl
I5 = mno
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture time range (allow some margin)
	timeRangeStart := time.Now().Add(-time.Second)

	cmd := NewClientAddCommand()
	cmd.SetArgs([]string{"--config", configPath, "client-timestamp"})
	cfgFile = configPath
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	timeRangeEnd := time.Now().Add(time.Second)

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	configStr := string(content)
	if !strings.Contains(configStr, "#_GenKeyTime = ") {
		t.Error("expected #_GenKeyTime field in config")
	}

	// Extract and parse the timestamp
	lines := strings.Split(configStr, "\n")
	foundTimestamp := false
	for _, line := range lines {
		if strings.Contains(line, "#_GenKeyTime = ") {
			parts := strings.Split(line, " = ")
			if len(parts) == 2 {
				timestampStr := strings.TrimSpace(parts[1])
				createdAt, err := time.Parse(time.RFC3339, timestampStr)
				if err != nil {
					t.Fatalf("failed to parse timestamp: %v", err)
				}

				// Verify timestamp is within reasonable range (allow for timing variations)
				if createdAt.Before(timeRangeStart) {
					t.Errorf("CreatedAt timestamp %v is before time range start %v", createdAt, timeRangeStart)
				}
				if createdAt.After(timeRangeEnd) {
					t.Errorf("CreatedAt timestamp %v is after time range end %v", createdAt, timeRangeEnd)
				}
				foundTimestamp = true
			}
		}
	}

	if !foundTimestamp {
		t.Error("did not find #_GenKeyTime field in config")
	}
}

// TestPresharedKey tests that PresharedKey is generated and stored when adding a peer.
func TestPresharedKey(t *testing.T) {
	cfgFile = ""
	clientIPAddr = ""
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
H1 = 100
H2 = 200
H3 = 300
H4 = 400
I1 = abc
I2 = def
I3 = ghi
I4 = jkl
I5 = mno
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := NewClientAddCommand()
	cmd.SetArgs([]string{"--config", configPath, "client-psk"})
	cfgFile = configPath
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	configStr := string(content)
	if !strings.Contains(configStr, "PresharedKey = ") {
		t.Error("expected PresharedKey field in config")
	}

	// Verify PSK is a valid base64 string (44 characters for 32 bytes)
	lines := strings.Split(configStr, "\n")
	foundPSK := false
	for _, line := range lines {
		if strings.Contains(line, "PresharedKey = ") {
			parts := strings.Split(line, " = ")
			if len(parts) == 2 {
				pskStr := strings.TrimSpace(parts[1])
				if len(pskStr) != 44 {
					t.Errorf("expected PSK to be 44 characters, got %d", len(pskStr))
				}
				foundPSK = true
			}
		}
	}

	if !foundPSK {
		t.Error("did not find #_PresharedKey field in config")
	}
}
