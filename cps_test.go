package amnezigo

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildCPSTagBytes(t *testing.T) {
	tag := BuildCPSTag("b", "0xc00000000108")
	expected := "<b 0xc00000000108>"
	if tag != expected {
		t.Errorf("BuildCPSTag(\"b\", \"0xc00000000108\") = %q, want %q", tag, expected)
	}
}

func TestBuildCPSTagRandom(t *testing.T) {
	tag := BuildCPSTag("r", "20")
	expectedPrefix := "<r 20>"
	if !strings.HasPrefix(tag, expectedPrefix) {
		t.Errorf("BuildCPSTag(\"r\", \"20\") = %q, want prefix %q", tag, expectedPrefix)
	}
}

func TestBuildCPSTagRandomChars(t *testing.T) {
	tag := BuildCPSTag("rc", "8")
	expectedPrefix := "<rc 8>"
	if !strings.HasPrefix(tag, expectedPrefix) {
		t.Errorf("BuildCPSTag(\"rc\", \"8\") = %q, want prefix %q", tag, expectedPrefix)
	}
}

// TestRcAlphabetMatchesReference locks the <rc> tag's alphabet contract to
// [a-zA-Z] (52 ASCII letters, lowercase first then uppercase, alphabetical),
// matching amneziawg-go device/obf_randchars.go. amnezigo only emits the
// "<rc N>" tag literal; the receiver fills random bytes at packet emission
// time. This test guards against alphabet drift (e.g., the alphanumeric form
// incorrectly cited in third-party articles).
//
// For mixed letter+digit fields, use tag concatenation: "<rc 4><rd 2>"
// produces 4 letters then 2 digits. Do not introduce a separate
// alphanumeric tag.
func TestRcAlphabetMatchesReference(t *testing.T) {
	// Reference alphabet from amneziawg-go device/obf_randchars.go:
	// 52 ASCII letters, lowercase first, then uppercase, sorted alphabetically.
	// Source of truth — DO NOT modify without verifying upstream first.
	const reference = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	if cpsRcAlphabet != reference {
		t.Errorf("cpsRcAlphabet drift: got %q, want %q", cpsRcAlphabet, reference)
	}

	t.Run("length is 52", func(t *testing.T) {
		if got := len(cpsRcAlphabet); got != 52 {
			t.Errorf("cpsRcAlphabet length = %d, want 52 (26 lowercase + 26 uppercase)", got)
		}
	})

	t.Run("no digits", func(t *testing.T) {
		for _, r := range cpsRcAlphabet {
			if r >= '0' && r <= '9' {
				t.Errorf(
					"cpsRcAlphabet contains digit %q — must be letters only "+
						"per amneziawg-go/device/obf_randchars.go", r)
			}
		}
	})

	t.Run("only ASCII letters", func(t *testing.T) {
		for _, r := range cpsRcAlphabet {
			isLower := r >= 'a' && r <= 'z'
			isUpper := r >= 'A' && r <= 'Z'
			if !isLower && !isUpper {
				t.Errorf("cpsRcAlphabet contains non-letter %q — must be [a-zA-Z]", r)
			}
		}
	})
}

func TestBuildCPSTagRandomDigits(t *testing.T) {
	tag := BuildCPSTag("rd", "4")
	expectedPrefix := "<rd 4>"
	if !strings.HasPrefix(tag, expectedPrefix) {
		t.Errorf("BuildCPSTag(\"rd\", \"4\") = %q, want prefix %q", tag, expectedPrefix)
	}
}

func TestBuildCPSTagCounter(t *testing.T) {
	tag := BuildCPSTag("c", "")
	expected := "<c>"
	if tag != expected {
		t.Errorf("BuildCPSTag(\"c\", \"\") = %q, want %q", tag, expected)
	}
}

func TestBuildCPSTagTimestamp(t *testing.T) {
	tag := BuildCPSTag("t", "")
	expected := "<t>"
	if tag != expected {
		t.Errorf("BuildCPSTag(\"t\", \"\") = %q, want %q", tag, expected)
	}
}

func TestBuildCPS(t *testing.T) {
	tags := []string{
		BuildCPSTag("b", "0xc00000000108"),
		BuildCPSTag("r", "8"),
		BuildCPSTag("c", ""),
		BuildCPSTag("t", ""),
		BuildCPSTag("r", "50"),
	}
	cps := BuildCPS(tags)

	// Verify all expected tags are present
	expectedParts := []string{
		"<b 0xc00000000108>",
		"<r 8>",
		"<c>",
		"<t>",
		"<r 50>",
	}

	for _, part := range expectedParts {
		if !strings.Contains(cps, part) {
			t.Errorf("BuildCPS result %q does not contain expected part %q", cps, part)
		}
	}
}

func TestBuildCPSMultipleTags(t *testing.T) {
	tags := []string{
		BuildCPSTag("rc", "10"),
		BuildCPSTag("rd", "5"),
		BuildCPSTag("b", "0x00"),
	}
	cps := BuildCPS(tags)

	// Should contain all tags in order
	expected := "<rc 10><rd 5><b 0x00>"
	if cps != expected {
		t.Errorf("BuildCPS() = %q, want %q", cps, expected)
	}
}

func TestBuildCPSEmpty(t *testing.T) {
	cps := BuildCPS([]string{})
	if cps != "" {
		t.Errorf("BuildCPS([]string{}) = %q, want empty string", cps)
	}
}

func TestMapTagType(t *testing.T) {
	tests := []struct {
		name     string
		tagType  string
		expected string
	}{
		{"bytes maps to b", "bytes", "b"},
		{"random maps to r", "random", "r"},
		{"random_chars maps to rc", "random_chars", "rc"},
		{"random_digits maps to rd", "random_digits", "rd"},
		{"counter maps to c", "counter", "c"},
		{"timestamp maps to t", "timestamp", "t"},
		{"unknown type returns empty", "unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTagType(tt.tagType)
			if result != tt.expected {
				t.Errorf("mapTagType(%q) = %q, want %q", tt.tagType, result, tt.expected)
			}
		})
	}
}

func TestCalculateMaxISize(t *testing.T) {
	tests := []struct {
		mtu, s1  int
		expected int
	}{
		{1280, 32, 1050}, // 1280 - 49 - 149 - 32 = 1050
		{1420, 64, 1158}, // 1420 - 49 - 149 - 64 = 1158
		{1280, 0, 1082},  // 1280 - 49 - 149 - 0 = 1082
	}
	for _, tt := range tests {
		result := calculateMaxISize(tt.mtu, tt.s1)
		if result != tt.expected {
			t.Errorf("calculateMaxISize(%d, %d) = %d, want %d",
				tt.mtu, tt.s1, result, tt.expected)
		}
	}
}

func TestCalculateCPSLength(t *testing.T) {
	tests := []struct {
		name     string
		cps      string
		expected int
	}{
		{"bytes_counter_timestamp", "<b 0xdeadbeef><c><t>", 16}, // 4 + 8 + 4
		{"random_types", "<r 10><rc 5><rd 3>", 18},              // 10 + 5 + 3
		{"single_byte_and_counter", "<b 0xff><c>", 9},           // 1 + 8
		{"empty", "", 0},
		{"only_counter", "<c>", 8},           // 8
		{"only_timestamp", "<t>", 4},         // 4
		{"mixed", "<b 0x11><c><r 5><t>", 18}, // 1 + 8 + 5 + 4
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCPSLength(tt.cps)
			if result != tt.expected {
				t.Errorf("calculateCPSLength(%q) = %d, want %d", tt.cps, result, tt.expected)
			}
		})
	}
}

// TestCalculateCPSLengthMatchesAccountedSize locks calculateCPSLength against
// hand-derived byte counts from the AmneziaWG 2.0 spec. Source of truth:
// amneziawg-go/device/obf_timestamp.go (<t> = uint32 BigEndian = 4 bytes).
func TestCalculateCPSLengthMatchesAccountedSize(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     int
	}{
		{"timestamp_only", "<t>", 4},
		{"single_byte", "<b 0xff>", 1},
		{"byte_plus_timestamp", "<b 0xff><t>", 5}, // acceptance criterion
		{"random_plus_timestamp", "<r 10><t>", 14},
		{"long_bytes_plus_timestamp", "<b 0xdeadbeef><t>", 8},
		{"random_charset_digit_plus_timestamp", "<rc 5><rd 3><t>", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCPSLength(tt.template)
			if got != tt.want {
				t.Errorf("calculateCPSLength(%q) = %d, want %d", tt.template, got, tt.want)
			}
		})
	}
}

func TestGenerateRandomTags(t *testing.T) {
	// Valid tag types
	validTypes := map[string]bool{
		"b":  true,
		"r":  true,
		"rc": true,
		"rd": true,
		"t":  true,
	}

	// Test count is within bounds
	t.Run("count within bounds", func(t *testing.T) {
		tags := generateRandomTags()
		if len(tags) < 3 || len(tags) > 6 {
			t.Errorf("generateRandomTags() returned %d tags, want between 3 and 6", len(tags))
		}
	})

	// Test all tag types are valid
	t.Run("all tag types valid", func(t *testing.T) {
		tags := generateRandomTags()
		for _, tag := range tags {
			if !validTypes[tag.Type] {
				t.Errorf("invalid tag type %q", tag.Type)
			}
		}
	})

	// Test randomness by running multiple times
	t.Run("randomness", func(t *testing.T) {
		results := make([]string, 10)
		for i := range 10 {
			tags := generateRandomTags()
			var sb strings.Builder
			for _, tag := range tags {
				sb.WriteString(tag.Type)
				sb.WriteString("|")
			}
			results[i] = sb.String()
		}

		// At least 50% of runs should be different
		unique := make(map[string]bool)
		for _, r := range results {
			unique[r] = true
		}
		if len(unique) < 5 {
			t.Errorf("insufficient randomness: only %d unique results out of 10", len(unique))
		}
	})

	// Test tag type "b" has hex value
	t.Run("type b has hex value", func(t *testing.T) {
		tags := generateRandomTags()
		found := false
		for _, tag := range tags {
			if tag.Type == "b" {
				if !strings.HasPrefix(tag.Value, "0x") {
					t.Errorf("type 'b' tag has value %q, want hex with 0x prefix", tag.Value)
				}
				// Value should be 8-32 hex characters (4-16 bytes)
				hexLen := len(tag.Value) - 2 // remove "0x"
				if hexLen < 8 || hexLen > 32 {
					t.Errorf("type 'b' tag hex length %d, want 8-32", hexLen)
				}
				found = true
				break
			}
		}
		if !found {
			t.Skip("no 'b' type tag generated, cannot verify hex value")
		}
	})

	// Test tag type "r"/"rc"/"rd" have numeric value
	t.Run("random types have numeric value", func(t *testing.T) {
		tags := generateRandomTags()
		found := false
		for _, tag := range tags {
			if tag.Type == "r" || tag.Type == "rc" || tag.Type == "rd" {
				if tag.Value == "" {
					t.Errorf("type %q tag has empty value, want numeric string", tag.Type)
				}
				// Parse as number to verify it's valid
				var num int
				_, err := fmt.Sscanf(tag.Value, "%d", &num)
				if err != nil {
					t.Errorf("type %q tag has value %q, want numeric: %v", tag.Type, tag.Value, err)
				}
				if num < 5 || num > 40 {
					t.Errorf("type %q tag numeric value %d, want 5-40", tag.Type, num)
				}
				found = true
				break
			}
		}
		if !found {
			t.Skip("no 'r'/'rc'/'rd' type tag generated, cannot verify numeric value")
		}
	})

	// Test tag type "t" has empty value
	t.Run("type t has empty value", func(t *testing.T) {
		tags := generateRandomTags()
		foundT := false
		for _, tag := range tags {
			if tag.Type == "t" {
				if tag.Value != "" {
					t.Errorf("type 't' tag has value %q, want empty string", tag.Value)
				}
				foundT = true
			}
		}
		if !foundT {
			t.Skip("no 't' type tag generated, cannot verify empty value")
		}
	})
}

func TestGenerateSimpleCPS(t *testing.T) {
	cps := generateSimpleCPS(1280, 32, [4]int{})

	maxI := calculateMaxISize(1280, 32)

	if cps.I1 == "" || cps.I2 == "" || cps.I3 == "" || cps.I4 == "" || cps.I5 == "" {
		t.Error("All I1-I5 should be non-empty")
	}

	for _, i := range []string{cps.I1, cps.I2, cps.I3, cps.I4, cps.I5} {
		if calculateCPSLength(i) >= maxI {
			t.Errorf("CPS %q exceeds maxISize %d", i, maxI)
		}
	}
}

func TestGenerateSimpleCPSFallback(t *testing.T) {
	// Force fallback by setting impossible constraints. With maxSize=0, every
	// generation attempt fails the size check. Empty forbidden set keeps
	// scope to MTU-only assertion.
	result := generateSimpleI(0, [4]int{})

	if result == "" {
		t.Error("generateSimpleI fallback should return non-empty string, got empty string")
	}

	if result != "<t>" {
		t.Errorf("generateSimpleI fallback should return '<t>', got %q", result)
	}
}

func TestGenerateSimpleCPSTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		mtu, s1 int
	}{
		{"standard", 1280, 32},
		{"small_mtu", 500, 10},
		{"large_s1", 1280, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cps := generateSimpleCPS(tt.mtu, tt.s1, [4]int{})
			maxI := calculateMaxISize(tt.mtu, tt.s1)

			for _, i := range []string{cps.I1, cps.I2, cps.I3, cps.I4, cps.I5} {
				if i == "" {
					t.Error("CPS should not be empty")
				}
				if calculateCPSLength(i) >= maxI {
					t.Errorf("CPS %q exceeds maxISize %d", i, maxI)
				}
			}
		})
	}
}

func TestGenerateCPSConfig_Random(t *testing.T) {
	cps := generateCPSConfig("random", 1280, 32, [4]int{})
	maxI := calculateMaxISize(1280, 32)

	for _, i := range []string{cps.I1, cps.I2, cps.I3, cps.I4, cps.I5} {
		if calculateCPSLength(i) >= maxI {
			t.Errorf("CPS %q exceeds maxISize %d", i, maxI)
		}
	}
}

func TestGenerateCPSConfig_Protocol(t *testing.T) {
	for _, protocol := range []string{"quic", "dns", "dtls", "stun"} {
		t.Run(protocol, func(t *testing.T) {
			cps := generateCPSConfig(protocol, 1280, 32, [4]int{})
			maxI := calculateMaxISize(1280, 32)

			for _, i := range []string{cps.I1, cps.I2, cps.I3, cps.I4, cps.I5} {
				if calculateCPSLength(i) >= maxI {
					t.Errorf("%s: CPS %q exceeds maxISize %d", protocol, i, maxI)
				}
			}
		})
	}
}

func TestGenerateRandomTagsUniqueConstraint(t *testing.T) {
	// Run many times to catch random duplicates
	for i := range 10000 {
		tags := generateRandomTags()

		countT := 0
		for _, tag := range tags {
			if tag.Type == "t" {
				countT++
			}
		}

		if countT > 1 {
			t.Errorf("iteration %d: found %d 't' tags, expected at most 1", i, countT)
		}
	}
}
