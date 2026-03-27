package obfuscation

import (
	"crypto/rand"
	"math/big"
	"sort"

	"github.com/Arsolitt/amnezigo/internal/config"
)

const (
	headerRegion1Max  = 0x3FFFFFFE
	headerRegion23Max = 0x3FFFFFFF
	s4RangeMax        = 33
	sPrefixRangeMax   = 65
	jcRangeMax        = 11
	junkMinValue      = 64
	junkRangeSize     = 961 // 1024 - 64 + 1
	headerMinValue    = uint32(5)
	headerMaxValue    = uint32(2147483647)
	headerMinRange    = uint32(10000000)
	headerMaxAttempts = 1000

	headerRegion2Start = uint32(0x40000000)
	headerRegion3Start = uint32(0x80000000)
	headerRegion4Start = uint32(0xC0000000)
)

// HeaderRange represents a min-max range for obfuscation.
type HeaderRange struct {
	Min, Max uint32
}

// Headers represents H1-H4 obfuscation headers.
type Headers struct {
	H1, H2, H3, H4 uint32
}

// SPrefixes represents S1-S4 obfuscation size prefixes.
type SPrefixes struct {
	S1, S2, S3, S4 int
}

// JunkParams represents Jc, Jmin, Jmax obfuscation junk parameters.
type JunkParams struct {
	Jc, Jmin, Jmax int
}

// GenerateHeaders generates 4 non-overlapping non-zero headers.
func GenerateHeaders() Headers {
	// Divide uint32 space into 4 non-overlapping regions
	// Region 1: 0x00000001 - 0x3FFFFFFF
	// Region 2: 0x40000000 - 0x7FFFFFFF
	// Region 3: 0x80000000 - 0xBFFFFFFF
	// Region 4: 0xC0000000 - 0xFFFFFFFF

	h1, _ := rand.Int(rand.Reader, big.NewInt(headerRegion1Max))
	h1Val := uint32(h1.Uint64()) + 1 //nolint:gosec // G115: values bounded by crypto/rand Int ranges

	h2, _ := rand.Int(rand.Reader, big.NewInt(headerRegion23Max))
	h2Val := uint32(h2.Uint64()) + headerRegion2Start //nolint:gosec // G115: bounded by crypto/rand range

	h3, _ := rand.Int(rand.Reader, big.NewInt(headerRegion23Max))
	h3Val := uint32(h3.Uint64()) + headerRegion3Start //nolint:gosec // G115: bounded by crypto/rand range

	h4, _ := rand.Int(rand.Reader, big.NewInt(headerRegion23Max))
	h4Val := uint32(h4.Uint64()) + headerRegion4Start //nolint:gosec // G115: bounded by crypto/rand range

	return Headers{
		H1: h1Val,
		H2: h2Val,
		H3: h3Val,
		H4: h4Val,
	}
}

// GenerateSPrefixes generates S1-S4 size prefixes with constraints.
func GenerateSPrefixes() SPrefixes {
	var s1, s2, s3, s4 int

	// S4: 0-32 range
	s4Int, _ := rand.Int(rand.Reader, big.NewInt(s4RangeMax))
	s4 = int(s4Int.Int64())

	// S1, S2, S3: 0-64 range
	// S1+56 must NOT equal S2 (to avoid Init/Response size collision)
	for {
		s1Int, _ := rand.Int(rand.Reader, big.NewInt(sPrefixRangeMax))
		s1 = int(s1Int.Int64())

		s2Int, _ := rand.Int(rand.Reader, big.NewInt(sPrefixRangeMax))
		s2 = int(s2Int.Int64())

		if s1+56 != s2 {
			break
		}
	}

	s3Int, _ := rand.Int(rand.Reader, big.NewInt(sPrefixRangeMax))
	s3 = int(s3Int.Int64())

	return SPrefixes{
		S1: s1,
		S2: s2,
		S3: s3,
		S4: s4,
	}
}

// GenerateJunkParams generates Jc, Jmin, Jmax junk parameters.
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

// GenerateCPS generates I1-I5 custom packet strings based on protocol template
// or random mode with MTU constraints.
func GenerateCPS(protocol string, mtu, s1, _ int) (string, string, string, string, string) {
	cpsConfig := generateCPSConfig(protocol, mtu, s1)
	return cpsConfig.I1, cpsConfig.I2, cpsConfig.I3, cpsConfig.I4, cpsConfig.I5
}

// GenerateConfig combines all obfuscation parameters into a config.
func GenerateConfig(protocol string, mtu, s1, jc int) config.ClientObfuscationConfig {
	h := GenerateHeaders()
	s := GenerateSPrefixes()
	j := GenerateJunkParams()
	cps := generateCPSConfig(protocol, mtu, s1)

	return config.ClientObfuscationConfig{
		ServerObfuscationConfig: config.ServerObfuscationConfig{
			Jc:   jc,
			Jmin: j.Jmin,
			Jmax: j.Jmax,
			S1:   s1,
			S2:   s.S2,
			S3:   s.S3,
			S4:   s.S4,
			H1:   config.HeaderRange{Min: h.H1, Max: h.H1}, // TODO: Use ranges instead
			H2:   config.HeaderRange{Min: h.H2, Max: h.H2}, // TODO: Use ranges instead
			H3:   config.HeaderRange{Min: h.H3, Max: h.H3}, // TODO: Use ranges instead
			H4:   config.HeaderRange{Min: h.H4, Max: h.H4}, // TODO: Use ranges instead
		},
		I1: cps.I1,
		I2: cps.I2,
		I3: cps.I3,
		I4: cps.I4,
		I5: cps.I5,
	}
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

		// Check for overlaps
		valid := true
		for i := range 3 {
			if sortedRanges[i].Max >= sortedRanges[i+1].Min {
				valid = false
				break
			}
		}

		if valid {
			// Return sorted ranges
			for i := range 4 {
				ranges[i] = sortedRanges[i]
			}
			return ranges
		}
	}

	panic("failed to generate non-overlapping header ranges after 1000 attempts")
}

// GenerateServerConfig generates server obfuscation config (without I1-I5).
func GenerateServerConfig(_, s1, jc int) config.ServerObfuscationConfig {
	h := GenerateHeaderRanges()
	s := GenerateSPrefixes()
	j := GenerateJunkParams()

	return config.ServerObfuscationConfig{
		Jc:   jc,
		Jmin: j.Jmin,
		Jmax: j.Jmax,
		S1:   s1,
		S2:   s.S2,
		S3:   s.S3,
		S4:   s.S4,
		H1:   config.HeaderRange{Min: h[0].Min, Max: h[0].Max},
		H2:   config.HeaderRange{Min: h[1].Min, Max: h[1].Max},
		H3:   config.HeaderRange{Min: h[2].Min, Max: h[2].Max},
		H4:   config.HeaderRange{Min: h[3].Min, Max: h[3].Max},
	}
}
