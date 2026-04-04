package amnezigo

import (
	"testing"
)

func TestGetTemplate_NamedProtocols(t *testing.T) {
	tests := []struct {
		protocol string
		wantNil  bool
	}{
		{"quic", false},
		{"dns", false},
		{"dtls", false},
		{"stun", false},
	}
	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			tmpl := getTemplate(tt.protocol)
			if tt.wantNil && tmpl.I1 == nil {
				t.Error("expected non-nil template")
			}
			if !tt.wantNil && tmpl.I1 == nil {
				t.Error("expected non-nil I1")
			}
		})
	}
}

func TestGetTemplate_RandomIsNotDeterministic(t *testing.T) {
	seen := make(map[int]bool)
	for range 20 {
		tmpl := getTemplate("random")
		if tmpl.I1 == nil {
			t.Fatal("random template returned nil I1")
		}
		if len(tmpl.I1) > 0 {
			seen[len(tmpl.I1)] = true
		}
	}
	if len(seen) == 1 {
		t.Error("random protocol always returns same template, expected variety")
	}
}

func TestGetTemplate_UnknownFallsBackToRandom(t *testing.T) {
	tmpl := getTemplate("unknown_protocol")
	if tmpl.I1 == nil {
		t.Error("unknown protocol should fall back to random selection, got nil I1")
	}
}
