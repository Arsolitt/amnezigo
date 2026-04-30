package amnezigo

import (
	"strings"
	"testing"
)

func TestSIPTemplate_AllIntervalsNonEmpty_I1ToI4(t *testing.T) {
	tmpl := SIPTemplate()
	for i, intvl := range [][]TagSpec{tmpl.I1, tmpl.I2, tmpl.I3, tmpl.I4} {
		if len(intvl) == 0 {
			t.Errorf("I%d is empty; SIP template requires I1-I4 populated", i+1)
		}
	}
}

func TestSIPTemplate_I5Empty(t *testing.T) {
	tmpl := SIPTemplate()
	if len(tmpl.I5) != 0 {
		t.Errorf("I5 must be empty for named templates, got %d tags", len(tmpl.I5))
	}
}

func TestSIPTemplate_NoForbiddenTags(t *testing.T) {
	allowed := map[string]struct{}{
		"bytes": {}, "random": {}, "random_chars": {}, "random_digits": {}, "timestamp": {},
	}
	tmpl := SIPTemplate()
	for i, intvl := range [][]TagSpec{tmpl.I1, tmpl.I2, tmpl.I3, tmpl.I4, tmpl.I5} {
		for j, spec := range intvl {
			if _, ok := allowed[spec.Type]; !ok {
				t.Errorf("I%d[%d] uses forbidden tag type %q", i+1, j, spec.Type)
			}
		}
	}
}

func TestSIPTemplate_NoCounterLiteral(t *testing.T) {
	// Scans the rendered CPS string for the literal substring "<c>".
	// mapTagType already maps Type:"counter" to "" silently, so a field-level
	// check guards a non-existent attack surface. The string-level scan catches
	// both the Type:"counter" path AND any future regression where someone
	// hand-codes BuildCPSTag("c", ...) or smuggles a literal <c> into a bytes value.
	tmpl := SIPTemplate()
	intervals := map[string][]TagSpec{
		"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4, "I5": tmpl.I5,
	}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		if strings.Contains(cps, "<c>") {
			t.Errorf("%s contains forbidden <c> tag in rendered CPS: %q", name, cps)
		}
	}
}

func TestSIPTemplate_FitsMTU(t *testing.T) {
	const mtu = 1280
	const s1 = 64
	maxI := calculateMaxISize(mtu, s1) // 1018
	tmpl := SIPTemplate()
	intervals := map[string][]TagSpec{"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		n := calculateCPSLength(cps)
		if n >= maxI {
			t.Errorf("%s is %d bytes, exceeds maxISize=%d for MTU=%d S1=%d", name, n, maxI, mtu, s1)
		}
	}
}

func TestSIPTemplate_ByteBudgetUnderCeiling(t *testing.T) {
	const ceiling = 700
	tmpl := SIPTemplate()
	intervals := map[string][]TagSpec{"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		n := calculateCPSLength(cps)
		if n > ceiling {
			t.Errorf("%s is %d bytes, exceeds informal ceiling %d", name, n, ceiling)
		}
	}
}

func TestSIPTemplate_AtMostOneTimestampPerInterval(t *testing.T) {
	tmpl := SIPTemplate()
	intervals := map[string][]TagSpec{
		"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4, "I5": tmpl.I5,
	}
	for name, intvl := range intervals {
		count := 0
		for _, spec := range intvl {
			if spec.Type == "timestamp" {
				count++
			}
		}
		if count > 1 {
			t.Errorf("%s has %d <t> tags; at most one allowed", name, count)
		}
	}
}

func TestSIPTemplate_AvoidsExistingPrefixes(t *testing.T) {
	// SIP I1 starts with "OPTIONS " — ASCII 0x4F 0x50 0x54 0x49 0x4F 0x4E 0x53 0x20.
	// Distinct from QUIC (c0..), DTLS (16..), STUN (00 01..). DNS has no fixed prefix.
	// The shared helper consults the centralized existingTemplatePrefixes slice
	// so this test stays correct as future templates extend the list.
	assertTemplateAvoidsExistingPrefixes(t, SIPTemplate(), []byte("OPTIONS "))
}
