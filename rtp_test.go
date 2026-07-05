package amnezigo

import (
	"strings"
	"testing"
)

func TestRTPTemplate_AllIntervalsNonEmpty_I1ToI4(t *testing.T) {
	tmpl := RTPTemplate()
	for i, intvl := range [][]TagSpec{tmpl.I1, tmpl.I2, tmpl.I3, tmpl.I4} {
		if len(intvl) == 0 {
			t.Errorf("I%d is empty; RTP template requires I1-I4 populated", i+1)
		}
	}
}

func TestRTPTemplate_I5Empty(t *testing.T) {
	tmpl := RTPTemplate()
	if len(tmpl.I5) != 0 {
		t.Errorf("I5 must be empty for named templates, got %d tags", len(tmpl.I5))
	}
}

func TestRTPTemplate_NoForbiddenTags(t *testing.T) {
	allowed := map[string]struct{}{
		"bytes": {}, "random": {}, "random_chars": {}, "random_digits": {}, "timestamp": {},
	}
	tmpl := RTPTemplate()
	for i, intvl := range [][]TagSpec{tmpl.I1, tmpl.I2, tmpl.I3, tmpl.I4, tmpl.I5} {
		for j, spec := range intvl {
			if _, ok := allowed[spec.Type]; !ok {
				t.Errorf("I%d[%d] uses forbidden tag type %q", i+1, j, spec.Type)
			}
		}
	}
}

func TestRTPTemplate_NoCounterLiteral(t *testing.T) {
	// Scans the rendered CPS string for the literal substring "<c>".
	tmpl := RTPTemplate()
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

func TestRTPTemplate_FitsMTU(t *testing.T) {
	const mtu = 1280
	const s1 = 64
	maxI := calculateMaxISize(mtu, s1) // 1018
	tmpl := RTPTemplate()
	intervals := map[string][]TagSpec{"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		n := calculateCPSLength(cps)
		if n >= maxI {
			t.Errorf("%s is %d bytes, exceeds maxISize=%d for MTU=%d S1=%d", name, n, maxI, mtu, s1)
		}
	}
}

func TestRTPTemplate_ByteBudgetUnderCeiling(t *testing.T) {
	const ceiling = 700
	tmpl := RTPTemplate()
	intervals := map[string][]TagSpec{"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		n := calculateCPSLength(cps)
		if n > ceiling {
			t.Errorf("%s is %d bytes, exceeds informal ceiling %d", name, n, ceiling)
		}
	}
}

func TestRTPTemplate_AtMostOneTimestampPerInterval(t *testing.T) {
	tmpl := RTPTemplate()
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

func TestRTPTemplate_AvoidsExistingPrefixes(t *testing.T) {
	// RTP I1 starts with 0x80 (V=2, P=0, X=0, CC=0).
	// Distinct from QUIC (c0..), DTLS (16..), STUN (00 01..), SIP ("OPTIONS ").
	// DNS has no fixed prefix (random transaction ID).
	assertTemplateAvoidsExistingPrefixes(t, RTPTemplate(), []byte{0x80})
}
