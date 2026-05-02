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
