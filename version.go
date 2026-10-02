package amnezigo

import "fmt"

// AWGVersion identifies the AmneziaWG protocol generation targeted by
// generated configs. Each generation is a superset of the previous one, and
// each engine build rejects INI keys it does not know ("Line unrecognized" in
// amneziawg-tools/src/config.c), so a key MUST never be emitted for a lower
// target version.
type AWGVersion int

const (
	// AWG20 is AmneziaWG 2.0: Jc/Jmin/Jmax, S1-S4, H1-H4 ranges, I1-I5.
	AWG20 AWGVersion = iota + 1
	// AWG30 adds HeaderProtectionKey, ContentPaddingAddition and the uint16
	// timer/padding ranges (RekeyAfterTime, RekeyTimeout, RejectAfterTime,
	// KeepaliveTimeout, MaxHandshakeAttempts).
	AWG30
	// AWG31 adds RandomTrailers and DisableCookies.
	AWG31
)

// DefaultAWGVersion is emitted when a manifest leaves obfuscation.awg_version
// unset.
const DefaultAWGVersion = AWG31

// ParseAWGVersion parses a manifest obfuscation.awg_version value. An empty
// string (unset) returns DefaultAWGVersion.
func ParseAWGVersion(s string) (AWGVersion, error) {
	switch s {
	case "":
		return DefaultAWGVersion, nil
	case "2.0":
		return AWG20, nil
	case "3.0":
		return AWG30, nil
	case "3.1":
		return AWG31, nil
	default:
		return 0, fmt.Errorf("unsupported awg_version %q (expected \"2.0\", \"3.0\", or \"3.1\")", s)
	}
}

// String returns the dotted version label ("2.0", "3.0", "3.1").
func (v AWGVersion) String() string {
	switch v {
	case AWG20:
		return "2.0"
	case AWG30:
		return "3.0"
	case AWG31:
		return "3.1"
	default:
		return fmt.Sprintf("AWGVersion(%d)", int(v))
	}
}
