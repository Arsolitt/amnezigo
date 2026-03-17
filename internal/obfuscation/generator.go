package obfuscation

import (
	"crypto/rand"
	"math/big"

	"github.com/Arsolitt/amnezigo/internal/config"
)

// Headers represents H1-H4 obfuscation headers
type Headers struct {
	H1, H2, H3, H4 uint32
}

// SPrefixes represents S1-S4 obfuscation size prefixes
type SPrefixes struct {
	S1, S2, S3, S4 int
}

// JunkParams represents Jc, Jmin, Jmax obfuscation junk parameters
type JunkParams struct {
	Jc, Jmin, Jmax int
}

// GenerateHeaders generates 4 non-overlapping non-zero headers
func GenerateHeaders() Headers {
	// Divide uint32 space into 4 non-overlapping regions
	// Region 1: 0x00000001 - 0x3FFFFFFF
	// Region 2: 0x40000000 - 0x7FFFFFFF
	// Region 3: 0x80000000 - 0xBFFFFFFF
	// Region 4: 0xC0000000 - 0xFFFFFFFF

	h1, _ := rand.Int(rand.Reader, big.NewInt(0x3FFFFFFE))
	h1Val := uint32(h1.Uint64()) + 1 // Ensure non-zero

	h2, _ := rand.Int(rand.Reader, big.NewInt(0x3FFFFFFF))
	h2Val := uint32(h2.Uint64()) + 0x40000000

	h3, _ := rand.Int(rand.Reader, big.NewInt(0x3FFFFFFF))
	h3Val := uint32(h3.Uint64()) + 0x80000000

	h4, _ := rand.Int(rand.Reader, big.NewInt(0x3FFFFFFF))
	h4Val := uint32(h4.Uint64()) + 0xC0000000

	return Headers{
		H1: h1Val,
		H2: h2Val,
		H3: h3Val,
		H4: h4Val,
	}
}

// GenerateSPrefixes generates S1-S4 size prefixes with constraints
func GenerateSPrefixes() SPrefixes {
	var s1, s2, s3, s4 int

	// S4: 0-32 range
	s4Int, _ := rand.Int(rand.Reader, big.NewInt(33))
	s4 = int(s4Int.Int64())

	// S1, S2, S3: 0-64 range
	// S1+56 must NOT equal S2 (to avoid Init/Response size collision)
	for {
		s1Int, _ := rand.Int(rand.Reader, big.NewInt(65))
		s1 = int(s1Int.Int64())

		s2Int, _ := rand.Int(rand.Reader, big.NewInt(65))
		s2 = int(s2Int.Int64())

		if s1+56 != s2 {
			break
		}
	}

	s3Int, _ := rand.Int(rand.Reader, big.NewInt(65))
	s3 = int(s3Int.Int64())

	return SPrefixes{
		S1: s1,
		S2: s2,
		S3: s3,
		S4: s4,
	}
}

// GenerateJunkParams generates Jc, Jmin, Jmax junk parameters
func GenerateJunkParams() JunkParams {
	// Jc: 0-10 range
	jcInt, _ := rand.Int(rand.Reader, big.NewInt(11))
	jc := int(jcInt.Int64())

	// Jmin, Jmax: 64-1024 range, with Jmin < Jmax
	jminInt, _ := rand.Int(rand.Reader, big.NewInt(961)) // 1024 - 64 + 1
	jmin := int(jminInt.Int64()) + 64

	jmaxInt, _ := rand.Int(rand.Reader, big.NewInt(961)) // 1024 - 64 + 1
	jmax := int(jmaxInt.Int64()) + 64

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
func GenerateCPS(protocol string) (string, string, string, string, string) {
	cpsConfig := generateCPSConfig(protocol)
	return cpsConfig.I1, cpsConfig.I2, cpsConfig.I3, cpsConfig.I4, cpsConfig.I5
}

// GenerateConfig combines all obfuscation parameters into a config
func GenerateConfig(protocol string) config.ObfuscationConfig {
	h := GenerateHeaders()
	s := GenerateSPrefixes()
	j := GenerateJunkParams()
	i1, i2, i3, i4, i5 := GenerateCPS(protocol)

	return config.ObfuscationConfig{
		Jc:   j.Jc,
		Jmin: j.Jmin,
		Jmax: j.Jmax,
		S1:   s.S1,
		S2:   s.S2,
		S3:   s.S3,
		S4:   s.S4,
		H1:   h.H1,
		H2:   h.H2,
		H3:   h.H3,
		H4:   h.H4,
		I1:   i1,
		I2:   i2,
		I3:   i3,
		I4:   i4,
		I5:   i5,
	}
}
