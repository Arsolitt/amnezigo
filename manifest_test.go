package amnezigo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNetworkConfig_JSONRoundTrip(t *testing.T) {
	nc := NetworkConfig{
		MTU: 1280,
		DNS: []string{"1.1.1.1", "8.8.8.8"},
	}
	b, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded NetworkConfig
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.MTU != nc.MTU {
		t.Errorf("MTU = %d, want %d", decoded.MTU, nc.MTU)
	}
	if len(decoded.DNS) != len(nc.DNS) {
		t.Fatalf("DNS len = %d, want %d", len(decoded.DNS), len(nc.DNS))
	}
	for i := range nc.DNS {
		if decoded.DNS[i] != nc.DNS[i] {
			t.Errorf("DNS[%d] = %q, want %q", i, decoded.DNS[i], nc.DNS[i])
		}
	}
}

func TestNetworkConfig_OmitsEmptyFields(t *testing.T) {
	nc := NetworkConfig{}
	b, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, `"mtu"`) {
		t.Errorf("empty MTU (0) should be omitted, got %q", got)
	}
	if strings.Contains(got, `"dns"`) {
		t.Errorf("nil DNS should be omitted, got %q", got)
	}
}

func TestObfuscationManifest_ExplicitParams(t *testing.T) {
	input := `{
		"s1": 30, "s2": 35, "s3": 20, "s4": 12,
		"h1": {"min": 100, "max": 5000000},
		"jc": 5, "jmin": 250, "jmax": 750
	}`
	var om ObfuscationManifest
	if err := json.Unmarshal([]byte(input), &om); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if om.S1 == nil || *om.S1 != 30 {
		t.Errorf("S1 = %v, want 30", om.S1)
	}
	if om.H1 == nil || om.H1.Min != 100 || om.H1.Max != 5000000 {
		t.Errorf("H1 = %v, want {100, 5000000}", om.H1)
	}
	if om.Jc == nil || *om.Jc != 5 {
		t.Errorf("Jc = %v, want 5", om.Jc)
	}
}

func TestObfuscationManifest_ZeroValueDistinctFromAbsent(t *testing.T) {
	withZero := `{"s1": 0}`
	var om ObfuscationManifest
	if err := json.Unmarshal([]byte(withZero), &om); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if om.S1 == nil {
		t.Fatal("S1 = nil after explicit 0; pointer must distinguish 0 from absent")
	}
	if *om.S1 != 0 {
		t.Errorf("S1 = %d, want 0", *om.S1)
	}

	withoutS1 := `{}`
	var om2 ObfuscationManifest
	if err := json.Unmarshal([]byte(withoutS1), &om2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if om2.S1 != nil {
		t.Errorf("S1 should be nil when absent, got %v", om2.S1)
	}
}

func TestObfuscationManifest_OmitsNilFieldsOnMarshal(t *testing.T) {
	om := ObfuscationManifest{Protocol: "quic"}
	b, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, `"s1"`) {
		t.Errorf("nil S1 should be omitted, got %q", got)
	}
	if strings.Contains(got, `"h1"`) {
		t.Errorf("nil H1 should be omitted, got %q", got)
	}
	if !strings.Contains(got, `"protocol":"quic"`) {
		t.Errorf("protocol should be present, got %q", got)
	}
}
