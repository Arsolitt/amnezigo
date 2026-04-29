package amnezigo

import (
	"strings"
	"testing"
)

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

// TestGenerateSPrefixes_SixPairsDistinct asserts that the six pairwise
// S-padded sizes are distinct across many runs. Catches regressions of the
// pre-roadmap behavior where only the S1+56 != S2 pair was checked.
func TestGenerateSPrefixes_SixPairsDistinct(t *testing.T) {
	const iterations = 1000
	for i := range iterations {
		s := GenerateSPrefixes()
		padded := [4]int{s.S1 + 148, s.S2 + 92, s.S3 + 64, s.S4 + 32}
		labels := [4]string{"S1+148", "S2+92", "S3+64", "S4+32"}
		for a := range 4 {
			for b := a + 1; b < 4; b++ {
				if padded[a] == padded[b] {
					t.Fatalf("iteration %d: collision %s == %s == %d (S=%v)",
						i, labels[a], labels[b], padded[a], s)
				}
			}
		}
	}
}

// TestGenerateSPrefixesWithS1_RespectsFixedS1 verifies the new helper
// generates S2/S3/S4 such that all six pairs are distinct against the
// caller-supplied S1.
func TestGenerateSPrefixesWithS1_RespectsFixedS1(t *testing.T) {
	const iterations = 1000
	for fixedS1 := range 65 { // exhaustive over the legal user S1 range [0..64]
		for i := range iterations / 65 {
			s := GenerateSPrefixesWithS1(fixedS1)
			if s.S1 != fixedS1 {
				t.Fatalf("S1 must equal fixed value: want %d, got %d", fixedS1, s.S1)
			}
			padded := [4]int{s.S1 + 148, s.S2 + 92, s.S3 + 64, s.S4 + 32}
			for a := range 4 {
				for b := a + 1; b < 4; b++ {
					if padded[a] == padded[b] {
						t.Fatalf("S1=%d iter=%d: pair %d/%d collision (S=%v)",
							fixedS1, i, a, b, s)
					}
				}
			}
		}
	}
}

// TestGenerateJunkParamsWithForbidden_ExcludesPaddedAndRawWGSizes asserts the
// generated [Jmin..Jmax] range excludes every forbidden size plus the four
// raw WG message sizes (148, 92, 64, 32).
func TestGenerateJunkParamsWithForbidden_ExcludesPaddedAndRawWGSizes(t *testing.T) {
	const iterations = 1000
	for i := range iterations {
		s := GenerateSPrefixes()
		forbidden := [4]int{s.S1 + 148, s.S2 + 92, s.S3 + 64, s.S4 + 32}
		j, err := GenerateJunkParamsWithForbidden(forbidden)
		if err != nil {
			t.Fatalf("iter %d: forbidden=%v: %v", i, forbidden, err)
		}
		all := append([]int{}, forbidden[:]...)
		all = append(all, 148, 92, 64, 32)
		for _, f := range all {
			if f >= j.Jmin && f <= j.Jmax {
				t.Errorf("iter %d: [Jmin=%d..Jmax=%d] contains forbidden %d (S=%v)",
					i, j.Jmin, j.Jmax, f, s)
			}
		}
	}
}

// TestGenerateConfig_NoCollisionsProperty is the property test mandated by the
// roadmap: 1000 random configs must all satisfy ValidatePacketSizes.
// Skipped under -short to keep PR CI fast.
func TestGenerateConfig_NoCollisionsProperty(t *testing.T) {
	if testing.Short() {
		t.Skip("property test skipped under -short")
	}
	const iterations = 1000
	for i := range iterations {
		cfg := GenerateConfig("random", 1280, 32, 5)
		iSizes := []int{
			calculateCPSLength(cfg.I1), calculateCPSLength(cfg.I2),
			calculateCPSLength(cfg.I3), calculateCPSLength(cfg.I4),
			calculateCPSLength(cfg.I5),
		}
		if err := ValidatePacketSizes(cfg.S1, cfg.S2, cfg.S3, cfg.S4,
			iSizes, cfg.Jmin, cfg.Jmax); err != nil {
			t.Fatalf("iter %d: %v\n  S=%d,%d,%d,%d J=[%d..%d] I=%v",
				i, err, cfg.S1, cfg.S2, cfg.S3, cfg.S4, cfg.Jmin, cfg.Jmax, iSizes)
		}
	}
}

// TestGenerateConfig_NoCollisionsProperty_AllProtocols runs the property test
// across each protocol template at 200 iterations each (1000 total).
func TestGenerateConfig_NoCollisionsProperty_AllProtocols(t *testing.T) {
	if testing.Short() {
		t.Skip("property test skipped under -short")
	}
	const perProto = 200
	for _, protocol := range []string{"random", "quic", "dns", "dtls", "stun"} {
		t.Run(protocol, func(t *testing.T) {
			for i := range perProto {
				cfg := GenerateConfig(protocol, 1280, 32, 5)
				iSizes := []int{
					calculateCPSLength(cfg.I1), calculateCPSLength(cfg.I2),
					calculateCPSLength(cfg.I3), calculateCPSLength(cfg.I4),
					calculateCPSLength(cfg.I5),
				}
				if err := ValidatePacketSizes(cfg.S1, cfg.S2, cfg.S3, cfg.S4,
					iSizes, cfg.Jmin, cfg.Jmax); err != nil {
					t.Fatalf("%s iter %d: %v\n  S=%d,%d,%d,%d J=[%d..%d] I=%v",
						protocol, i, err, cfg.S1, cfg.S2, cfg.S3, cfg.S4,
						cfg.Jmin, cfg.Jmax, iSizes)
				}
			}
		})
	}
}

func TestGenerateConfig(t *testing.T) {
	cfg := GenerateConfig("random", 1280, 32, 5)

	// Valid Jc range from 0-10
	if cfg.Jc < 0 || cfg.Jc > 10 {
		t.Errorf("Jc must be in range 0-10, got %d", cfg.Jc)
	}

	// All header ranges must be true ranges (Min < Max)
	for i, hr := range []HeaderRange{cfg.H1, cfg.H2, cfg.H3, cfg.H4} {
		if hr.Min >= hr.Max {
			t.Errorf("H%d must have Min < Max, got Min=%d Max=%d", i+1, hr.Min, hr.Max)
		}
		if hr.Min < 5 {
			t.Errorf("H%d Min must be >= 5, got %d", i+1, hr.Min)
		}
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
		// Forbidden values: standard WG type-ids must never appear in any H range.
		for _, fid := range []uint32{1, 2, 3, 4} {
			if fid >= r.Min && fid <= r.Max {
				t.Errorf("H%d range [%d-%d] contains forbidden WG type-id %d", i+1, r.Min, r.Max, fid)
			}
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

// TestGenerateHeaderRanges_NeverIncludesWGTypeIDs is the property test mandated
// by roadmap P0.4: 10 000 random GenerateHeaderRanges invocations must never
// produce a range that includes any of the standard WireGuard message type-ids
// (1, 2, 3, 4). Skipped under -short to keep PR CI fast.
func TestGenerateHeaderRanges_NeverIncludesWGTypeIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("property test skipped under -short")
	}
	const iterations = 10000
	forbidden := []uint32{1, 2, 3, 4}
	for i := range iterations {
		ranges := GenerateHeaderRanges()
		for k, r := range ranges {
			for _, fid := range forbidden {
				if fid >= r.Min && fid <= r.Max {
					t.Fatalf("iteration %d: H%d range [%d-%d] contains forbidden WG type-id %d",
						i, k+1, r.Min, r.Max, fid)
				}
			}
		}
	}
}
