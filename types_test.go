package amnezigo

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestConfigExists verifies the config package is properly structured
// This is a placeholder test that will be expanded with actual config tests.
func TestConfigExists(t *testing.T) {
	// This test verifies the package compiles
	// More specific tests will be added as config features are implemented
	t.Log("Config package initialized successfully")
}

// TestPeerConfigPresharedKey verifies that PeerConfig has a PresharedKey field.
func TestPeerConfigPresharedKey(t *testing.T) {
	peer := PeerConfig{
		PresharedKey: "preshared-key-123",
	}

	if peer.PresharedKey != "preshared-key-123" {
		t.Errorf("Expected PresharedKey to be 'preshared-key-123', got '%s'", peer.PresharedKey)
	}
}

func TestHeaderRange_JSONRoundTrip(t *testing.T) {
	hr := HeaderRange{Min: 100, Max: 5000000}
	b, err := json.Marshal(hr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"min":100`) {
		t.Errorf("expected lowercase min key in %q", got)
	}
	if !strings.Contains(got, `"max":5000000`) {
		t.Errorf("expected lowercase max key in %q", got)
	}

	var decoded HeaderRange
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != hr {
		t.Errorf("round-trip mismatch: got %+v, want %+v", decoded, hr)
	}
}
