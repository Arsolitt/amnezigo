package amnezigo

import "fmt"

// Preset represents a named bundle of obfuscation parameters tuned for a
// specific network environment. Each preset produces a valid server
// obfuscation config that passes ValidatePacketSizes.
type Preset struct {
	Name            string
	Description     string
	DefaultProtocol string
	H1, H2, H3, H4  HeaderRange
	MTU             int
	S1, S2, S3, S4  int
	Jc, Jmin, Jmax  int
}

// ToServerObfuscation converts a Preset into a ServerObfuscationConfig.
func (p Preset) ToServerObfuscation() ServerObfuscationConfig {
	return ServerObfuscationConfig{
		Jc:   p.Jc,
		Jmin: p.Jmin,
		Jmax: p.Jmax,
		S1:   p.S1,
		S2:   p.S2,
		S3:   p.S3,
		S4:   p.S4,
		H1:   p.H1,
		H2:   p.H2,
		H3:   p.H3,
		H4:   p.H4,
	}
}

// presetRegistry is the built-in preset library. Each preset's S-prefixes are
// chosen so that the four AWG-padded handshake sizes (S1+148, S2+92, S3+64,
// S4+32) are pairwise distinct, the junk range [Jmin..Jmax] excludes all
// padded and raw WG sizes, and H1-H4 ranges are non-overlapping and above
// the WG type-id window [1..4].
//
// Padded sizes per preset (for quick reference during review):
//
//	lan-conservative:  S1+148=158, S2+92=102, S3+64=79, S4+32=37  → all distinct
//	home-balanced:     S1+148=178, S2+92=127, S3+64=84, S4+32=44  → all distinct
//	mobile-aggressive: S1+148=208, S2+92=152, S3+64=114, S4+32=56 → all distinct
//	test-minimal:      S1+148=153, S2+92=99,  S3+64=71, S4+32=39  → all distinct
//
// Junk ranges must exclude ALL padded sizes AND raw WG constants (148, 92, 64, 32).
//
//nolint:mnd // preset configuration data — numeric literals are the domain values themselves
var presetRegistry = []Preset{
	{
		Name:            "lan-conservative",
		Description:     "Small S values, narrow junk range. Designed for corporate LANs with minimal DPI where low overhead is preferred over deep obfuscation.",
		MTU:             1280,
		S1:              10,
		S2:              10,
		S3:              15,
		S4:              5,
		Jc:              3,
		Jmin:            160,
		Jmax:            240,
		H1:              HeaderRange{Min: 10, Max: 1000000},
		H2:              HeaderRange{Min: 2000000, Max: 100000000},
		H3:              HeaderRange{Min: 200000000, Max: 500000000},
		H4:              HeaderRange{Min: 700000000, Max: 2000000000},
		DefaultProtocol: "random",
	},
	{
		Name:            "home-balanced",
		Description:     "Moderate S values and junk range. Good default for home internet connections where some DPI may exist but is not aggressive.",
		MTU:             1280,
		S1:              30,
		S2:              35,
		S3:              20,
		S4:              12,
		Jc:              5,
		Jmin:            250,
		Jmax:            750,
		H1:              HeaderRange{Min: 100, Max: 5000000},
		H2:              HeaderRange{Min: 10000000, Max: 200000000},
		H3:              HeaderRange{Min: 400000000, Max: 800000000},
		H4:              HeaderRange{Min: 1000000000, Max: 2100000000},
		DefaultProtocol: "quic",
	},
	{
		Name:            "mobile-aggressive",
		Description:     "Large S values, wide junk range, high junk count. Maximum entropy for carrier networks with heavy DPI inspection (MTS, Beeline, etc.).",
		MTU:             1280,
		S1:              60,
		S2:              60,
		S3:              50,
		S4:              24,
		Jc:              8,
		Jmin:            500,
		Jmax:            1000,
		H1:              HeaderRange{Min: 50, Max: 10000000},
		H2:              HeaderRange{Min: 50000000, Max: 500000000},
		H3:              HeaderRange{Min: 700000000, Max: 1200000000},
		H4:              HeaderRange{Min: 1500000000, Max: 2147000000},
		DefaultProtocol: "dns",
	},
	{
		Name:            "test-minimal",
		Description:     "Smallest valid parameter set for integration testing and CI. Not intended for production use.",
		MTU:             1280,
		S1:              5,
		S2:              7,
		S3:              7,
		S4:              7,
		Jc:              1,
		Jmin:            200,
		Jmax:            250,
		H1:              HeaderRange{Min: 5, Max: 10000},
		H2:              HeaderRange{Min: 20000, Max: 50000},
		H3:              HeaderRange{Min: 100000, Max: 500000},
		H4:              HeaderRange{Min: 1000000, Max: 5000000},
		DefaultProtocol: "random",
	},
}

// GetPreset returns the preset with the given name, or an error if not found.
func GetPreset(name string) (Preset, error) {
	for _, p := range presetRegistry {
		if p.Name == name {
			return p, nil
		}
	}
	available := make([]string, 0, len(presetRegistry))
	for _, p := range presetRegistry {
		available = append(available, p.Name)
	}
	return Preset{}, fmt.Errorf("unknown preset %q; available presets: %v", name, available)
}

// ListPresets returns a copy of all built-in presets.
func ListPresets() []Preset {
	result := make([]Preset, len(presetRegistry))
	copy(result, presetRegistry)
	return result
}
