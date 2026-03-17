package obfuscation

import (
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
