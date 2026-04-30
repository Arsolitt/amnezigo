package amnezigo

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetTemplate_NamedProtocols(t *testing.T) {
	tests := []struct {
		protocol string
		wantNil  bool
	}{
		{"quic", false},
		{"dns", false},
		{"dtls", false},
		{"stun", false},
		{"sip", false},
	}
	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			tmpl := getTemplate(tt.protocol)
			if tt.wantNil && tmpl.I1 == nil {
				t.Error("expected non-nil template")
			}
			if !tt.wantNil && tmpl.I1 == nil {
				t.Error("expected non-nil I1")
			}
		})
	}
}

func TestGetTemplate_RandomIsNotDeterministic(t *testing.T) {
	seen := make(map[int]bool)
	for range 20 {
		tmpl := getTemplate("random")
		if tmpl.I1 == nil {
			t.Fatal("random template returned nil I1")
		}
		if len(tmpl.I1) > 0 {
			seen[len(tmpl.I1)] = true
		}
	}
	if len(seen) == 1 {
		t.Error("random protocol always returns same template, expected variety")
	}
}

func TestGetTemplate_UnknownFallsBackToRandom(t *testing.T) {
	tmpl := getTemplate("unknown_protocol")
	if tmpl.I1 == nil {
		t.Error("unknown protocol should fall back to random selection, got nil I1")
	}
}

// TestQUICTemplate_ChainsDcidViaDTag pins the I1→I2 DCID-reuse design
// established in P1.1. A future contributor swapping <d> back to <random 8>
// would silently regress mimicry quality (every I-packet looking unrelated).
// The test inspects the rendered template, not its tag-spec list, to ensure
// mapTagType + BuildCPSTag are wired correctly.
func TestQUICTemplate_ChainsDcidViaDTag(t *testing.T) {
	tmpl := QUICTemplate()
	i2 := buildCPSFromTemplate(tmpl.I2)
	if !strings.Contains(i2, "<d>") {
		t.Errorf("QUIC I2 = %q, expected to contain <d> (DCID passthrough from I1)", i2)
	}
	// I1 must still produce a fresh random DCID so <d> has something to copy.
	i1 := buildCPSFromTemplate(tmpl.I1)
	if !strings.Contains(i1, "<r 8>") {
		t.Errorf("QUIC I1 = %q, expected <r 8> as the DCID source for I2's <d>", i1)
	}
}

// existingTemplatePrefixes is the canonical list of leading-byte prefixes
// already in use by shipped templates. New templates MUST NOT begin with any
// of these so that --protocol random produces distinguishable shapes.
//
// MAINTENANCE CONTRACT: every template PR that introduces a new fixed leading
// byte sequence appends it here in the same diff. Reviewers reject template
// PRs that add a new prefix without updating this slice.
var existingTemplatePrefixes = [][]byte{
	{0xC0},             // QUIC long-header byte 0
	{0x16},             // DTLS handshake content type
	{0x00, 0x01},       // STUN binding request message type
	[]byte("OPTIONS "), // SIP OPTIONS method literal
	// future templates append here
}

// assertTemplateAvoidsExistingPrefixes builds I1's CPS via buildCPSFromTemplate
// and asserts the rendered byte stream does not start with any prefix in
// existingTemplatePrefixes. Skips the template's own prefix (ownPrefix) so
// it does not flag itself.
func assertTemplateAvoidsExistingPrefixes(t *testing.T, tmpl I1I5Template, ownPrefix []byte) {
	t.Helper()
	cps := buildCPSFromTemplate(tmpl.I1)
	for _, prefix := range existingTemplatePrefixes {
		if bytes.Equal(prefix, ownPrefix) {
			continue // skip the template's own prefix
		}
		if bytes.HasPrefix([]byte(cps), prefix) {
			t.Errorf("template I1 begins with already-used prefix %x", prefix)
		}
	}
}
