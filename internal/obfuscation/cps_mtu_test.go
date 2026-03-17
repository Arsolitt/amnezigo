package obfuscation

import (
	"testing"

	"github.com/Arsolitt/amnezigo/internal/obfuscation/protocols"
)

// TestBuildAndValidateCPSValidation tests the validation behavior of buildAndValidateCPS
func TestBuildAndValidateCPSValidation(t *testing.T) {
	tests := []struct {
		name      string
		tags      []protocols.TagSpec
		maxSize   int
		wantEmpty bool
	}{
		{
			name: "progressive_reduction_to_fit",
			tags: []protocols.TagSpec{
				{Type: "random", Value: "50"},  // 50 bytes
				{Type: "random", Value: "30"},  // 30 bytes
				{Type: "counter", Value: ""},   // 4 bytes
				{Type: "timestamp", Value: ""}, // 4 bytes
			},
			maxSize:   60, // Should fit after removing timestamp
			wantEmpty: false,
		},
		{
			name: "exact_fit",
			tags: []protocols.TagSpec{
				{Type: "random", Value: "20"},  // 20 bytes
				{Type: "counter", Value: ""},   // 4 bytes
				{Type: "timestamp", Value: ""}, // 4 bytes
			},
			maxSize:   29, // Just enough for 28 bytes
			wantEmpty: false,
		},
		{
			name: "single_tag_fits",
			tags: []protocols.TagSpec{
				{Type: "random", Value: "5"}, // 5 bytes
			},
			maxSize:   6, // Should fit
			wantEmpty: false,
		},
		{
			name: "all_tags_too_large_returns_empty",
			tags: []protocols.TagSpec{
				{Type: "bytes", Value: "0xdeadbeefdeadbeefdeadbeefdeadbeef"}, // 16 bytes
				{Type: "random", Value: "100"},                               // 100 bytes
				{Type: "counter", Value: ""},                                 // 4 bytes
				{Type: "timestamp", Value: ""},                               // 4 bytes
			},
			maxSize:   10, // Too small for even minimal tags
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAndValidateCPS(tt.tags, tt.maxSize)

			// Check if result is empty as expected
			if tt.wantEmpty && result != "" {
				t.Errorf("Expected empty result for maxSize=%d, got %q", tt.maxSize, result)
			}
			if !tt.wantEmpty && result == "" {
				t.Errorf("Expected non-empty result for maxSize=%d, got empty string", tt.maxSize)
			}

			// For non-empty results, verify they fit within constraints
			if result != "" {
				if calculateCPSLength(result) >= tt.maxSize {
					t.Errorf("CPS %q has length %d, exceeds maxSize %d", result, calculateCPSLength(result), tt.maxSize)
				}
			}
		})
	}
}

// TestGenerateProtocolCPSMTUConstraints tests that protocol CPS generation respects MTU constraints
func TestGenerateProtocolCPSMTUConstraints(t *testing.T) {
	tests := []struct {
		name string
		mtu  int
		s1   int
		jc   int
	}{
		{"standard", 1280, 32, 5},
		{"small_mtu", 500, 10, 3},
		{"tiny_mtu", 200, 32, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, protocol := range []string{"quic", "dns", "dtls", "stun"} {
				t.Run(protocol, func(t *testing.T) {
					cps := generateProtocolCPS(protocol, tt.mtu, tt.s1, tt.jc)
					maxI := calculateMaxISize(tt.mtu, tt.s1, tt.jc)

					// Validate size constraints (only when maxI > 0 and interval is non-empty)
					// Note: I5 is intentionally empty for all protocol templates
					if maxI > 0 {
						for i, interval := range []string{cps.I1, cps.I2, cps.I3, cps.I4, cps.I5} {
							if interval != "" && calculateCPSLength(interval) >= maxI {
								t.Errorf("%s: I%d = %q has length %d, exceeds maxISize %d",
									protocol, i+1, interval, calculateCPSLength(interval), maxI)
							}
						}
					}
				})
			}
		})
	}
}

// TestGenerateProtocolCPSZeroMaxI tests edge case where maxI is zero
func TestGenerateProtocolCPSZeroMaxI(t *testing.T) {
	for _, protocol := range []string{"quic", "dns", "dtls", "stun"} {
		t.Run(protocol, func(t *testing.T) {
			// MTU so small that maxI becomes 0
			mtu := 100
			s1 := 32
			jc := 5

			cps := generateProtocolCPS(protocol, mtu, s1, jc)

			// All intervals should be non-empty and fallback to minimal
			for i, interval := range []string{cps.I1, cps.I2, cps.I3, cps.I4, cps.I5} {
				if interval == "" {
					t.Errorf("%s: I%d should not be empty even with zero maxI", protocol, i+1)
				}
				// With maxI=0, everything should fallback to <c>
				if interval != "<c>" {
					t.Errorf("%s: With maxI=0, I%d should be '<c>', got %q", protocol, i+1, interval)
				}
			}
		})
	}
}
