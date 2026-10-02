package amnezigo

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
)

const (
	s4RangeMax         = 33
	sPrefixRangeMax    = 65
	jcRangeMax         = 11
	junkMinValue       = 64
	junkRangeSize      = 961 // 1024 - 64 + 1
	headerMinValue     = uint32(5)
	headerMaxValue     = uint32(2147483647)
	headerMinRange     = uint32(10000000)
	headerMaxAttempts  = 1000
	sMaxAttempts       = 1000 // retry budget for the six-pair S-prefix check
	junkMaxAttempts    = 1000 // retry budget for collision-aware junk-range selection
	iPacketMaxAttempts = 1000 // retry budget for collision-aware I-packet generation

	// headerProtectionNonceSize is the smallest S-prefix accepted by the
	// engine when AWG 3.x header protection is active: the header cipher nonce
	// is the first S bytes of the packet (mirrors device.HeaderCipherNonceSize
	// in amneziawg-go).
	headerProtectionNonceSize = 12
)

// GenerateSPrefixes generates S1-S4 size prefixes such that the four AWG-padded
// handshake sizes are pairwise distinct. minS is the inclusive lower bound for
// every value: 0 for AWG 2.0, headerProtectionNonceSize when AWG 3.x header
// protection is active (the engine rejects S < 12 in that mode).
func GenerateSPrefixes(minS int) SPrefixes {
	for range sMaxAttempts {
		s := SPrefixes{
			S1: randIntInRange(minS, sPrefixRangeMax),
			S2: randIntInRange(minS, sPrefixRangeMax),
			S3: randIntInRange(minS, sPrefixRangeMax),
			S4: randIntInRange(minS, s4RangeMax),
		}
		if pairsDistinct(s) {
			return s
		}
	}
	panic("failed to generate non-colliding S-prefixes after sMaxAttempts attempts")
}

// GenerateSPrefixesWithS1 generates S2-S4 such that the six pairwise S-padded
// sizes are distinct, given a caller-supplied S1. minS is the inclusive lower
// bound for the generated values (0 for the legacy AWG 2.0 path,
// headerProtectionNonceSize under AWG 3.x header protection). Used by
// GenerateConfig where S1 comes from user input rather than being randomly
// chosen.
func GenerateSPrefixesWithS1(minS, fixedS1 int) SPrefixes {
	for range sMaxAttempts {
		s := SPrefixes{
			S1: fixedS1,
			S2: randIntInRange(minS, sPrefixRangeMax),
			S3: randIntInRange(minS, sPrefixRangeMax),
			S4: randIntInRange(minS, s4RangeMax),
		}
		if pairsDistinct(s) {
			return s
		}
	}
	panic("failed to generate non-colliding S-prefixes for fixed S1 after sMaxAttempts attempts")
}

// GenerateUniformSPrefixes draws one value in [minS, s4RangeMax) and returns it
// as S1=S2=S3=S4. AWG 3.1 random_trailers makes the receiver classify the
// packet type by size, so the reference recommends equal S values to avoid
// misclassification. The four padded sizes (148+S, 92+S, 64+S, 32+S) stay
// pairwise distinct for any S, so no retry loop is needed.
func GenerateUniformSPrefixes(minS int) SPrefixes {
	s := randIntInRange(minS, s4RangeMax)
	return SPrefixes{S1: s, S2: s, S3: s, S4: s}
}

// pairsDistinct returns true iff the four AWG-padded sizes are pairwise distinct.
func pairsDistinct(s SPrefixes) bool {
	padded := PaddedSizes(s.S1, s.S2, s.S3, s.S4)
	for i := range 4 {
		for j := i + 1; j < 4; j++ {
			if padded[i] == padded[j] {
				return false
			}
		}
	}
	return true
}

// randIntInRange returns a uniform random int in [minV, maxExclusive).
func randIntInRange(minV, maxExclusive int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(maxExclusive-minV)))
	return int(n.Int64()) + minV
}

// GenerateJunkParams generates Jc, Jmin, Jmax junk parameters without
// collision avoidance. Preserved for backward compatibility; new callers
// should prefer GenerateJunkParamsWithForbidden.
func GenerateJunkParams() JunkParams {
	// Jc: 0-10 range
	jcInt, _ := rand.Int(rand.Reader, big.NewInt(jcRangeMax))
	jc := int(jcInt.Int64())

	// Jmin, Jmax: 64-1024 range, with Jmin < Jmax
	jminInt, _ := rand.Int(rand.Reader, big.NewInt(junkRangeSize))
	jmin := int(jminInt.Int64()) + junkMinValue

	jmaxInt, _ := rand.Int(rand.Reader, big.NewInt(junkRangeSize))
	jmax := int(jmaxInt.Int64()) + junkMinValue

	// Ensure Jmin < Jmax
	if jmin >= jmax {
		jmin, jmax = jmax, jmin
	}
	if jmin == jmax {
		jmax = jmin + 1
	}

	return JunkParams{
		Jc:   jc,
		Jmin: jmin,
		Jmax: jmax,
	}
}

// GenerateJunkParamsWithForbidden generates Jc, Jmin, Jmax such that the
// resulting junk range [Jmin..Jmax] excludes every size in forbiddenSizes
// (typically the four AWG-padded handshake sizes) AND excludes the four raw
// WireGuard message constants (148, 92, 64, 32). The raw constants are
// invariant and always checked.
//
// Returns an error if the retry budget is exhausted.
func GenerateJunkParamsWithForbidden(forbiddenSizes [4]int) (JunkParams, error) {
	jcInt, _ := rand.Int(rand.Reader, big.NewInt(jcRangeMax))
	jc := int(jcInt.Int64())

	for range junkMaxAttempts {
		jminInt, _ := rand.Int(rand.Reader, big.NewInt(junkRangeSize))
		jmin := int(jminInt.Int64()) + junkMinValue
		jmaxInt, _ := rand.Int(rand.Reader, big.NewInt(junkRangeSize))
		jmax := int(jmaxInt.Int64()) + junkMinValue

		if jmin > jmax {
			jmin, jmax = jmax, jmin
		}
		if jmin == jmax {
			jmax = jmin + 1
		}
		if junkRangeOK(forbiddenSizes, jmin, jmax) {
			return JunkParams{Jc: jc, Jmin: jmin, Jmax: jmax}, nil
		}
	}
	return JunkParams{}, fmt.Errorf("failed to generate non-colliding junk range after %d attempts", junkMaxAttempts)
}

// junkRangeOK returns true iff [jmin..jmax] (inclusive) excludes every
// forbidden size AND the four raw WG constants. Zero entries in
// forbiddenSizes are treated as unset (the junk range starts at 64, so 0
// never matters in practice).
func junkRangeOK(forbiddenSizes [4]int, jmin, jmax int) bool {
	rawWG := [...]int{WGInitiationSize, WGResponseSize, WGCookieReplySize, WGTransportSize}
	for _, f := range forbiddenSizes {
		if f == 0 {
			continue
		}
		if f >= jmin && f <= jmax {
			return false
		}
	}
	for _, f := range rawWG {
		if f >= jmin && f <= jmax {
			return false
		}
	}
	return true
}

// GenerateCPS generates I1-I5 custom packet strings based on protocol template
// or random mode with MTU constraints. Public API; uses an empty forbidden set
// (callers wanting collision avoidance should use GenerateConfig).
func GenerateCPS(protocol string, mtu, s1, _ int) (string, string, string, string, string) {
	cpsConfig := generateCPSConfig(protocol, mtu, s1, [4]int{})
	return cpsConfig.I1, cpsConfig.I2, cpsConfig.I3, cpsConfig.I4, cpsConfig.I5
}

// GenerateConfig combines all obfuscation parameters into a config that
// satisfies the AWG 2.0 size-classification invariant: the four S-padded
// handshake sizes are pairwise distinct, no I-packet length collides with a
// padded size, and the junk range avoids both padded and raw WG sizes.
func GenerateConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig {
	h := GenerateHeaderRanges()
	s := GenerateSPrefixesWithS1(0, s1)
	forbidden := PaddedSizes(s.S1, s.S2, s.S3, s.S4)
	j, err := GenerateJunkParamsWithForbidden(forbidden)
	if err != nil {
		panic(fmt.Sprintf("GenerateConfig: junk-range generation failed: %v", err))
	}
	cps := generateCPSConfig(protocol, mtu, s1, forbidden)

	cfg := ClientObfuscationConfig{
		ServerObfuscationConfig: ServerObfuscationConfig{
			Jc:   jc,
			Jmin: j.Jmin,
			Jmax: j.Jmax,
			S1:   s.S1,
			S2:   s.S2,
			S3:   s.S3,
			S4:   s.S4,
			H1:   h[0],
			H2:   h[1],
			H3:   h[2],
			H4:   h[3],
		},
		I1: cps.I1,
		I2: cps.I2,
		I3: cps.I3,
		I4: cps.I4,
		I5: cps.I5,
	}

	// Defensive post-condition. Must hold by construction; panic on violation.
	iSizes := []int{
		calculateCPSLength(cfg.I1), calculateCPSLength(cfg.I2),
		calculateCPSLength(cfg.I3), calculateCPSLength(cfg.I4),
		calculateCPSLength(cfg.I5),
	}
	if vErr := ValidatePacketSizes(cfg.S1, cfg.S2, cfg.S3, cfg.S4,
		iSizes, cfg.Jmin, cfg.Jmax); vErr != nil {
		panic(fmt.Sprintf("GenerateConfig: produced invalid sizes: %v", vErr))
	}
	return cfg
}

// GenerateHeaderRanges generates 4 non-overlapping H1-H4 ranges.
func GenerateHeaderRanges() [4]HeaderRange {
	for range headerMaxAttempts {
		ranges := [4]HeaderRange{}

		// Generate 4 random ranges
		for i := range 4 {
			// Calculate available space after reserving minRange for remaining ranges
			minRangeVal := big.NewInt(int64(headerMaxValue - headerMinValue - headerMinRange*3))
			if minRangeVal.Int64() <= 0 {
				minRangeVal = big.NewInt(1)
			}

			minRand, _ := rand.Int(rand.Reader, minRangeVal)
			//nolint:gosec // G115: bounded by crypto/rand range
			ranges[i].Min = headerMinValue + uint32(minRand.Uint64())

			// Calculate available space for Max
			maxRangeVal := big.NewInt(int64(headerMaxValue - ranges[i].Min - headerMinRange))
			if maxRangeVal.Int64() < int64(headerMinRange) {
				maxRangeVal = big.NewInt(int64(headerMinRange))
			}

			maxRand, _ := rand.Int(rand.Reader, maxRangeVal)
			//nolint:gosec // G115: bounded by crypto/rand range
			ranges[i].Max = min(ranges[i].Min+headerMinRange+uint32(maxRand.Uint64()), headerMaxValue)
		}

		// Sort ranges by Min value
		sortedRanges := ranges[:]
		sort.Slice(sortedRanges, func(i, j int) bool {
			return sortedRanges[i].Min < sortedRanges[j].Min
		})

		if headerRangesValid(sortedRanges) {
			// Return sorted ranges
			for i := range 4 {
				ranges[i] = sortedRanges[i]
			}
			return ranges
		}
	}

	panic("failed to generate non-overlapping header ranges after 1000 attempts")
}

// headerRangesValid reports whether the four sorted header ranges are
// pairwise non-overlapping AND each individually passes validateHeaderRange
// (i.e. excludes the WG message type-ids 1..4). The forbidden-id check is
// defence-in-depth: with headerMinValue = 5, generated Min is structurally
// above the [1..4] window, so the second loop should rarely fire.
func headerRangesValid(sortedRanges []HeaderRange) bool {
	for i := range 3 {
		if sortedRanges[i].Max >= sortedRanges[i+1].Min {
			return false
		}
	}
	for i := range 4 {
		if ValidateHeaderRange(sortedRanges[i]) != nil {
			return false
		}
	}
	return true
}

// GenerateServerConfig generates server obfuscation config (without I1-I5),
// satisfying the AWG 2.0 size-classification invariant for handshake and junk
// sizes.
func GenerateServerConfig(_, s1, jc int) ServerObfuscationConfig {
	h := GenerateHeaderRanges()
	s := GenerateSPrefixesWithS1(0, s1)
	forbidden := PaddedSizes(s.S1, s.S2, s.S3, s.S4)
	j, err := GenerateJunkParamsWithForbidden(forbidden)
	if err != nil {
		panic(fmt.Sprintf("GenerateServerConfig: junk-range generation failed: %v", err))
	}

	return ServerObfuscationConfig{
		Jc:   jc,
		Jmin: j.Jmin,
		Jmax: j.Jmax,
		S1:   s.S1,
		S2:   s.S2,
		S3:   s.S3,
		S4:   s.S4,
		H1:   HeaderRange{Min: h[0].Min, Max: h[0].Max},
		H2:   HeaderRange{Min: h[1].Min, Max: h[1].Max},
		H3:   HeaderRange{Min: h[2].Min, Max: h[2].Max},
		H4:   HeaderRange{Min: h[3].Min, Max: h[3].Max},
	}
}
