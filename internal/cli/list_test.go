package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {
	// Reset global state before each subtest
	oldCfgFile := cfgFile
	defer func() {
		cfgFile = oldCfgFile
	}()

	// Test 1: List multiple clients
	t.Run("list multiple clients in table format", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Create config with multiple clients
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
#_Name = client1
PublicKey = publickey1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = privatekey1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2024-01-15T10:30:00Z

[Peer]
#_Name = client2
PublicKey = publickey2345678901234567890123456789012345678901234567890123456789=
AllowedIPs = 10.8.0.3/32
#_PrivateKey = privatekey2345678901234567890123456789012345678901234567890123456789=
#_GenKeyTime = 2024-01-16T14:22:00Z
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		// Capture output
		var buf bytes.Buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := NewListCommand()
		cmd.SetArgs([]string{"--config", configPath})
		cfgFile = configPath

		err := cmd.Execute()
		w.Close()
		os.Stdout = oldStdout
		io.Copy(&buf, r)
		outputStr := buf.String()

		if err != nil {
			t.Fatalf("list command failed: %v", err)
		}

		// Verify header
		if !strings.Contains(outputStr, "NAME") {
			t.Error("expected 'NAME' in output")
		}
		if !strings.Contains(outputStr, "IP") {
			t.Error("expected 'IP' in output")
		}
		if !strings.Contains(outputStr, "CREATED") {
			t.Error("expected 'CREATED' in output")
		}

		// Verify clients are listed
		if !strings.Contains(outputStr, "client1") {
			t.Error("expected 'client1' in output")
		}
		if !strings.Contains(outputStr, "client2") {
			t.Error("expected 'client2' in output")
		}

		// Verify IP addresses
		if !strings.Contains(outputStr, "10.8.0.2/32") {
			t.Error("expected '10.8.0.2/32' in output")
		}
		if !strings.Contains(outputStr, "10.8.0.3/32") {
			t.Error("expected '10.8.0.3/32' in output")
		}

		// Verify timestamps
		if !strings.Contains(outputStr, "2024-01-15") {
			t.Error("expected '2024-01-15' in output")
		}
		if !strings.Contains(outputStr, "2024-01-16") {
			t.Error("expected '2024-01-16' in output")
		}
	})

	// Test 2: List with no clients
	t.Run("list with no clients shows message", func(t *testing.T) {
		cfgFile = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		// Create config with no clients
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

		// Capture output
		var buf bytes.Buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := NewListCommand()
		cmd.SetArgs([]string{"--config", configPath})
		cfgFile = configPath

		err := cmd.Execute()
		w.Close()
		os.Stdout = oldStdout
		io.Copy(&buf, r)
		outputStr := buf.String()

		if err != nil {
			t.Fatalf("list command failed: %v", err)
		}

		// Verify no clients message
		if !strings.Contains(outputStr, "No peers configured") {
			t.Error("expected 'No peers configured' in output")
		}

		// Verify header is not shown when no clients
		if strings.Contains(outputStr, "NAME") {
			t.Error("did not expect header when no clients")
		}
	})

	// Test 3: Output format with separator
	t.Run("output format includes separator line", func(t *testing.T) {
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
#_Name = test-client
PublicKey = publickey1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = privatekey1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2024-01-15T10:30:00Z
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		// Capture output
		var buf bytes.Buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := NewListCommand()
		cmd.SetArgs([]string{"--config", configPath})
		cfgFile = configPath

		err := cmd.Execute()
		w.Close()
		os.Stdout = oldStdout
		io.Copy(&buf, r)
		outputStr := buf.String()

		if err != nil {
			t.Fatalf("list command failed: %v", err)
		}

		// Verify separator line (should contain dashes)
		if !strings.Contains(outputStr, "---") {
			t.Error("expected separator line with dashes in output")
		}

		// Verify the format is columnar (name, ip, created should be on the same line or in table format)
		lines := strings.Split(strings.TrimSpace(outputStr), "\n")
		if len(lines) < 2 {
			t.Errorf("expected at least 2 lines (header + separator), got %d", len(lines))
		}
	})
}

// TestListTimestampFormat tests that timestamps are formatted correctly.
func TestListTimestampFormat(t *testing.T) {
	// Reset global state
	oldCfgFile := cfgFile
	defer func() {
		cfgFile = oldCfgFile
	}()

	cfgFile = ""
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "awg0.conf")

	// Create config with a client
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
#_Name = timestamp-client
PublicKey = publickey1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
#_PrivateKey = privatekey1234567890123456789012345678901234567890123456789012345678=
#_GenKeyTime = 2024-01-15T10:30:45Z
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewListCommand()
	cmd.SetArgs([]string{"--config", configPath})
	cfgFile = configPath

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)
	outputStr := buf.String()

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Verify timestamp format (should be like "2024-01-15 10:30")
	if !strings.Contains(outputStr, "2024-01-15 10:30") {
		t.Errorf("expected timestamp in format 'YYYY-MM-DD HH:MM', got: %s", outputStr)
	}

	// Verify the minute part is preserved
	if !strings.Contains(outputStr, "10:30") {
		t.Errorf("expected timestamp to preserve minutes, got: %s", outputStr)
	}
}
