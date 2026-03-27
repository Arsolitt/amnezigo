package amnezigo

import (
	"strings"
	"testing"
)

func TestGenerateHeaders(t *testing.T) {
	h := GenerateHeaders()

	// All headers must be non-zero
	if h.H1 == 0 {
		t.Error("H1 must be non-zero")
	}
	if h.H2 == 0 {
		t.Error("H2 must be non-zero")
	}
	if h.H3 == 0 {
		t.Error("H3 must be non-zero")
	}
	if h.H4 == 0 {
		t.Error("H4 must be non-zero")
	}

	// All headers must be different (no overlap)
	headers := []uint32{h.H1, h.H2, h.H3, h.H4}
	for i := range headers {
		for j := i + 1; j < len(headers); j++ {
			if headers[i] == headers[j] {
				t.Errorf("Headers must not overlap: H%d (%d) == H%d (%d)", i+1, headers[i], j+1, headers[j])
			}
		}
	}
}

func TestGenerateSPrefixes(t *testing.T) {
	s := GenerateSPrefixes()

	// S1, S2, S3: 0-64 range
	if s.S1 < 0 || s.S1 > 64 {
		t.Errorf("S1 must be in range 0-64, got %d", s.S1)
	}
	if s.S2 < 0 || s.S2 > 64 {
		t.Errorf("S2 must be in range 0-64, got %d", s.S2)
	}
	if s.S3 < 0 || s.S3 > 64 {
		t.Errorf("S3 must be in range 0-64, got %d", s.S3)
	}

	// S4: 0-32 range
	if s.S4 < 0 || s.S4 > 32 {
		t.Errorf("S4 must be in range 0-32, got %d", s.S4)
	}

	// Constraint: S1 + 56 must NOT equal S2 (to avoid Init/Response size collision)
	if s.S1+56 == s.S2 {
		t.Errorf("S1+56 must not equal S2 to avoid Init/Response size collision: S1=%d, S2=%d", s.S1, s.S2)
	}
}

func TestGenerateJunkParams(t *testing.T) {
	j := GenerateJunkParams()

	// Jc: 0-10 range
	if j.Jc < 0 || j.Jc > 10 {
		t.Errorf("Jc must be in range 0-10, got %d", j.Jc)
	}

	// Jmin, Jmax: 64-1024 range
	if j.Jmin < 64 || j.Jmin > 1024 {
		t.Errorf("Jmin must be in range 64-1024, got %d", j.Jmin)
	}
	if j.Jmax < 64 || j.Jmax > 1024 {
		t.Errorf("Jmax must be in range 64-1024, got %d", j.Jmax)
	}

	// Jmin must be less than Jmax
	if j.Jmin >= j.Jmax {
		t.Errorf("Jmin must be less than Jmax: Jmin=%d, Jmax=%d", j.Jmin, j.Jmax)
	}
}

func TestGenerateConfig(t *testing.T) {
	cfg := GenerateConfig("random", 1280, 32, 5)

	// Valid Jc range from 0-10
	if cfg.Jc < 0 || cfg.Jc > 10 {
		t.Errorf("Jc must be in range 0-10, got %d", cfg.Jc)
	}

	// All header ranges must be valid
	// TODO: Update to require Min < Max once GenerateConfig uses GenerateHeaderRanges
	if cfg.H1.Min == 0 || cfg.H1.Max == 0 {
		t.Error("H1 must have non-zero Min and Max")
	}
	if cfg.H2.Min == 0 || cfg.H2.Max == 0 {
		t.Error("H2 must have non-zero Min and Max")
	}
	if cfg.H3.Min == 0 || cfg.H3.Max == 0 {
		t.Error("H3 must have non-zero Min and Max")
	}
	if cfg.H4.Min == 0 || cfg.H4.Max == 0 {
		t.Error("H4 must have non-zero Min and Max")
	}

	// S1-S3: 0-64, S4: 0-32
	if cfg.S1 < 0 || cfg.S1 > 64 {
		t.Errorf("S1 must be in range 0-64, got %d", cfg.S1)
	}
	if cfg.S2 < 0 || cfg.S2 > 64 {
		t.Errorf("S2 must be in range 0-64, got %d", cfg.S2)
	}
	if cfg.S3 < 0 || cfg.S3 > 64 {
		t.Errorf("S3 must be in range 0-64, got %d", cfg.S3)
	}
	if cfg.S4 < 0 || cfg.S4 > 32 {
		t.Errorf("S4 must be in range 0-32, got %d", cfg.S4)
	}

	// S1+56 != S2
	if cfg.S1+56 == cfg.S2 {
		t.Errorf("S1+56 must not equal S2: S1=%d, S2=%d", cfg.S1, cfg.S2)
	}

	// Jmin < Jmax
	if cfg.Jmin >= cfg.Jmax {
		t.Errorf("Jmin must be less than Jmax: Jmin=%d, Jmax=%d", cfg.Jmin, cfg.Jmax)
	}

	// I1 should now be populated (real implementation)
	if cfg.I1 == "" {
		t.Error("I1 should not be empty (real implementation)")
	}
}

func TestGenerateCPSWithProtocol(t *testing.T) {
	// Test QUIC protocol
	i1, _, _, _, i5 := GenerateCPS("quic", 1280, 32, 5)

	// Verify I1 is not empty
	if i1 == "" {
		t.Error("I1 should not be empty for QUIC protocol")
	}

	// Verify I5 is empty (QUIC template has I5 empty)
	if i5 != "" {
		t.Error("I5 should be empty for QUIC protocol (template has empty I5)")
	}

	// Verify CPS tags are present in I1
	expectedTags := []string{"<b", "<r", "<t>"}
	for _, tag := range expectedTags {
		if !strings.Contains(i1, tag) {
			t.Errorf("I1 should contain tag %s for QUIC protocol", tag)
		}
	}

	// Test different protocols
	protocols := []string{"quic", "dns", "dtls", "stun"}
	for _, protocol := range protocols {
		i1, _, _, _, _ := GenerateCPS(protocol, 1280, 32, 5)

		// All protocols should generate non-empty I1
		if i1 == "" {
			t.Errorf("I1 should not be empty for %s protocol", protocol)
		}
	}
}

func TestGenerateConfig_WithMTU(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		mtu      int
		s1       int
		jc       int
	}{
		{
			name:     "random_protocol",
			protocol: "random",
			mtu:      1280,
			s1:       32,
			jc:       5,
		},
		{
			name:     "quic_protocol",
			protocol: "quic",
			mtu:      1280,
			s1:       64,
			jc:       3,
		},
		{
			name:     "dns_protocol",
			protocol: "dns",
			mtu:      1280,
			s1:       64,
			jc:       4,
		},
		{
			name:     "dtls_protocol",
			protocol: "dtls",
			mtu:      1280,
			s1:       64,
			jc:       3,
		},
		{
			name:     "stun_protocol",
			protocol: "stun",
			mtu:      1280,
			s1:       64,
			jc:       4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GenerateConfig(tt.protocol, tt.mtu, tt.s1, tt.jc)

			// Verify I1-I4 are generated (I5 may be empty for some protocols)
			if cfg.I1 == "" {
				t.Error("I1 should not be empty")
			}
			if cfg.I2 == "" {
				t.Error("I2 should not be empty")
			}
			if cfg.I3 == "" {
				t.Error("I3 should not be empty")
			}
			if cfg.I4 == "" {
				t.Error("I4 should not be empty")
			}

			// Verify CPS length constraints
			maxI := calculateMaxISize(tt.mtu, tt.s1)
			intervals := []string{cfg.I1, cfg.I2, cfg.I3, cfg.I4, cfg.I5}
			for i, interval := range intervals {
				if interval != "" && calculateCPSLength(interval) >= maxI {
					t.Errorf("I%d has length %d, exceeds maxI %d: %q",
						i+1, calculateCPSLength(interval), maxI, interval)
				}
			}
		})
	}
}

func TestGenerateHeaderRanges(t *testing.T) {
	ranges := GenerateHeaderRanges()

	// Must return 4 ranges
	if len(ranges) != 4 {
		t.Fatalf("expected 4 ranges, got %d", len(ranges))
	}

	// Each range must be at least 10,000,000 in size
	minSize := uint32(10000000)
	for i, r := range ranges {
		size := r.Max - r.Min
		if size < minSize {
			t.Errorf("H%d range too small: %d (min %d)", i+1, size, minSize)
		}
		if r.Min < 5 {
			t.Errorf("H%d Min below 5: %d", i+1, r.Min)
		}
		if r.Max > 2147483647 {
			t.Errorf("H%d Max above 2147483647: %d", i+1, r.Max)
		}
		if r.Min >= r.Max {
			t.Errorf("H%d Min >= Max: %d >= %d", i+1, r.Min, r.Max)
		}
	}

	// Check non-overlapping
	for i := range 4 {
		for j := i + 1; j < 4; j++ {
			if ranges[i].Max >= ranges[j].Min && ranges[i].Min <= ranges[j].Max {
				t.Errorf("H%d and H%d overlap: [%d-%d] vs [%d-%d]",
					i+1, j+1, ranges[i].Min, ranges[i].Max, ranges[j].Min, ranges[j].Max)
			}
		}
	}
}
