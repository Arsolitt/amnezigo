package amnezigo

import (
	"strings"
	"testing"
)

// TestParseAWGVersion covers the accepted manifest values and the unset default.
func TestParseAWGVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  AWGVersion
	}{
		{"unset defaults to 3.1", "", AWG31},
		{"explicit 2.0", "2.0", AWG20},
		{"explicit 3.0", "3.0", AWG30},
		{"explicit 3.1", "3.1", AWG31},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAWGVersion(tt.input)
			if err != nil {
				t.Fatalf("ParseAWGVersion(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseAWGVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseAWGVersion_RejectsUnknown verifies unsupported values are rejected
// with a message naming the accepted set.
func TestParseAWGVersion_RejectsUnknown(t *testing.T) {
	for _, input := range []string{"2", "4.0", "3.10", "v3.1", "latest"} {
		t.Run(input, func(t *testing.T) {
			_, err := ParseAWGVersion(input)
			if err == nil {
				t.Fatalf("ParseAWGVersion(%q) expected error, got nil", input)
			}
			if !strings.Contains(err.Error(), "unsupported awg_version") {
				t.Errorf("error = %q, want to contain %q", err.Error(), "unsupported awg_version")
			}
			if !strings.Contains(err.Error(), `"2.0", "3.0", or "3.1"`) {
				t.Errorf("error = %q, want to list the accepted versions", err.Error())
			}
		})
	}
}

// TestAWGVersion_String verifies the dotted labels and the debug fallback.
func TestAWGVersion_String(t *testing.T) {
	tests := []struct {
		v    AWGVersion
		want string
	}{
		{AWG20, "2.0"},
		{AWG30, "3.0"},
		{AWG31, "3.1"},
		{0, "AWGVersion(0)"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("AWGVersion(%d).String() = %q, want %q", int(tt.v), got, tt.want)
		}
	}
}

// TestAWGVersion_ZeroValueIsLegacy pins the invariant that a zero-value version
// sorts below every real version, so hand-built legacy ServerObfuscationConfig
// values keep emitting exactly the 2.0 key set.
func TestAWGVersion_ZeroValueIsLegacy(t *testing.T) {
	if AWG20 <= 0 {
		t.Fatalf("AWG20 = %d, want > 0 so the zero value stays below it", int(AWG20))
	}
	if DefaultAWGVersion != AWG31 {
		t.Errorf("DefaultAWGVersion = %v, want 3.1", DefaultAWGVersion)
	}
}
