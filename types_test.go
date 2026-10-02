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

// TestU16Range_StringAndIsZero covers the engine's INI notation ("N" when the
// bounds are equal, "Min-Max" otherwise) and the zero-value unset semantics.
func TestU16Range_StringAndIsZero(t *testing.T) {
	tests := []struct {
		name     string
		r        U16Range
		want     string
		wantZero bool
	}{
		{"zero value is unset", U16Range{}, "0", true},
		{"equal bounds collapse to N", U16Range{Min: 12, Max: 12}, "12", false},
		{"distinct bounds use Min-Max", U16Range{Min: 2, Max: 10}, "2-10", false},
		{"zero min with non-zero max is not unset", U16Range{Min: 0, Max: 5}, "0-5", false},
		{"max uint16", U16Range{Min: 65535, Max: 65535}, "65535", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
			if got := tt.r.IsZero(); got != tt.wantZero {
				t.Errorf("IsZero() = %v, want %v", got, tt.wantZero)
			}
		})
	}
}

// TestU16Range_JSONShape pins the manifest field names and the integer width.
func TestU16Range_JSONShape(t *testing.T) {
	b, err := json.Marshal(U16Range{Min: 2, Max: 10})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"min":2`) {
		t.Errorf("expected lowercase min key in %q", got)
	}
	if !strings.Contains(got, `"max":10`) {
		t.Errorf("expected lowercase max key in %q", got)
	}

	var decoded U16Range
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded != (U16Range{Min: 2, Max: 10}) {
		t.Errorf("round-trip mismatch: got %+v", decoded)
	}
}
