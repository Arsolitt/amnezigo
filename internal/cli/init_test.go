package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommand_RequiredFlags(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		args        []string
		expectError bool
	}{
		{
			name:        "missing required --ipaddr flag",
			args:        []string{"init"},
			expectError: true,
			errorMsg:    "required",
		},
		{
			name:        "with --ipaddr flag",
			args:        []string{"init", "--ipaddr", "10.8.0.1/24"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for config
			_ = t.TempDir()

			// Set up root command with custom config path
			rootCmd := NewRootCmd()
			rootCmd.SetArgs(tt.args)

			// Capture output
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			// Execute
			err := rootCmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorMsg != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("error message should contain %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestInitCommand_CreatesConfigFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "awg0.conf")

	// Set up root command
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "--ipaddr", "10.8.0.1/24", "--config", configPath})

	// Execute
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute init command: %v", err)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file was not created at %s", configPath)
	}

	// Read and verify config file content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	contentStr := string(content)

	// Verify required sections exist
	requiredSections := []string{
		"[Interface]",
		"PrivateKey",
		"Address",
		"ListenPort",
		"Jc",
		"Jmin",
		"Jmax",
		"S1",
		"S2",
		"S3",
		"S4",
		"H1",
		"H2",
		"H3",
		"H4",
		// I1-I5 are client-only fields, not in server config
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("config file missing required section/field: %s", section)
		}
	}

	// Verify address matches what we provided
	if !strings.Contains(contentStr, "Address = 10.8.0.1/24") {
		t.Errorf("config address doesn't match: expected 'Address = 10.8.0.1/24'")
	}

	// Verify port is in valid range (should be auto-generated)
	if !strings.Contains(contentStr, "ListenPort = ") {
		t.Errorf("config missing ListenPort")
	}
}

func TestInitCommand_AutoPortGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "awg0.conf")

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"init", "--ipaddr", "10.8.0.1/24", "--config", configPath})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute init command: %v", err)
	}

	// Read config and extract port
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	// Parse port from config
	var port int
	_, err = fmt.Sscanf(string(content), "ListenPort = %d", &port)
	if err != nil {
		// Try scanning each line
		lines := strings.SplitSeq(string(content), "\n")
		for line := range lines {
			if strings.Contains(line, "ListenPort =") {
				_, err = fmt.Sscanf(line, "ListenPort = %d", &port)
				if err == nil {
					break
				}
			}
		}
	}

	if err != nil {
		t.Fatalf("failed to parse port from config: %v", err)
	}

	// Verify port is in valid range
	if port < 10000 || port > 65535 {
		t.Errorf("port %d is outside valid range 10000-65535", port)
	}
}

func TestInitCommand_WithPreset(t *testing.T) {
	t.Cleanup(func() { initPreset = "" })

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "awg0.conf")

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",
		"--ipaddr", "10.8.0.1/24",
		"--config", configPath,
		"--preset", "home-balanced",
		"--iface", "eth0",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute init with preset: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	contentStr := string(content)

	// home-balanced preset has S1=30, S2=35, S3=20, S4=12, Jc=5, Jmin=250, Jmax=750.
	expectedFields := []string{
		"S1 = 30",
		"S2 = 35",
		"S3 = 20",
		"S4 = 12",
		"Jc = 5",
		"Jmin = 250",
		"Jmax = 750",
	}
	for _, field := range expectedFields {
		if !strings.Contains(contentStr, field) {
			t.Errorf("config missing preset field %q", field)
		}
	}
}

func TestInitCommand_WithInvalidPreset(t *testing.T) {
	t.Cleanup(func() { initPreset = "" })

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "awg0.conf")

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",
		"--ipaddr", "10.8.0.1/24",
		"--config", configPath,
		"--preset", "nonexistent",
		"--iface", "eth0",
	})

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid preset, got nil")
	}

	if !strings.Contains(err.Error(), "invalid preset") {
		t.Errorf("error should mention invalid preset, got: %v", err)
	}
}

func TestInitCommand_WithOptionalFlags(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "awg0.conf")

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",
		"--ipaddr", "10.8.0.1/24",
		"--config", configPath,
		"--port", "55424",
		"--mtu", "1420",
		"--dns", "1.1.1.1,8.8.8.8",
		"--keepalive", "30",
		"--iface", "eth0",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("failed to execute init command: %v", err)
	}

	// Verify config file was created
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	contentStr := string(content)

	// Verify optional flags were applied
	if !strings.Contains(contentStr, "ListenPort = 55424") {
		t.Errorf("config should contain specified port 55424")
	}

	if !strings.Contains(contentStr, "MTU = 1420") {
		t.Errorf("config should contain specified MTU 1420")
	}

	// Verify PostUp contains the specified interface
	if !strings.Contains(contentStr, "-i eth0") {
		t.Errorf("PostUp rules should contain interface eth0")
	}

	if !strings.Contains(contentStr, "-o eth0") {
		t.Errorf("PostUp rules should contain interface eth0")
	}
}
