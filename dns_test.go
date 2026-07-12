package amnezigo

import (
	"strings"
	"testing"
)

func TestDNSTemplate_AllIntervalsNonEmpty_I1ToI4(t *testing.T) {
	tmpl := DNSTemplate()
	for i, intvl := range [][]TagSpec{tmpl.I1, tmpl.I2, tmpl.I3, tmpl.I4} {
		if len(intvl) == 0 {
			t.Errorf("I%d is empty; DNS template requires I1-I4 populated", i+1)
		}
	}
}

func TestDNSTemplate_I5Empty(t *testing.T) {
	tmpl := DNSTemplate()
	if len(tmpl.I5) != 0 {
		t.Errorf("I5 must be empty for named templates, got %d tags", len(tmpl.I5))
	}
}

func TestDNSTemplate_NoForbiddenTags(t *testing.T) {
	allowed := map[string]struct{}{
		"bytes": {}, "random": {}, "random_chars": {}, "random_digits": {}, "timestamp": {},
	}
	tmpl := DNSTemplate()
	for i, intvl := range [][]TagSpec{tmpl.I1, tmpl.I2, tmpl.I3, tmpl.I4, tmpl.I5} {
		for j, spec := range intvl {
			if _, ok := allowed[spec.Type]; !ok {
				t.Errorf("I%d[%d] uses forbidden tag type %q", i+1, j, spec.Type)
			}
		}
	}
}

func TestDNSTemplate_NoCounterLiteral(t *testing.T) {
	// Scans the rendered CPS string for the literal substring "<c>".
	// mapTagType already maps Type:"counter" to "" silently, so a field-level
	// check guards a non-existent attack surface. The string-level scan catches
	// both the Type:"counter" path AND any future regression where someone
	// hand-codes BuildCPSTag("c", ...) or smuggles a literal <c> into a bytes value.
	tmpl := DNSTemplate()
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

func TestDNSTemplate_FitsMTU(t *testing.T) {
	const mtu = 1280
	const s1 = 64
	maxI := calculateMaxISize(mtu, s1) // 1018
	tmpl := DNSTemplate()
	intervals := map[string][]TagSpec{"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		n := calculateCPSLength(cps)
		if n >= maxI {
			t.Errorf("%s is %d bytes, exceeds maxISize=%d for MTU=%d S1=%d", name, n, maxI, mtu, s1)
		}
	}
}

func TestDNSTemplate_ByteBudgetUnderCeiling(t *testing.T) {
	const ceiling = 700
	tmpl := DNSTemplate()
	intervals := map[string][]TagSpec{"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		n := calculateCPSLength(cps)
		if n > ceiling {
			t.Errorf("%s is %d bytes, exceeds informal ceiling %d", name, n, ceiling)
		}
	}
}

func TestDNSTemplate_AtMostOneTimestampPerInterval(t *testing.T) {
	tmpl := DNSTemplate()
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

func TestDNSTemplate_AvoidsExistingPrefixes(t *testing.T) {
	// DNS I1 starts with <r 2> (random transaction ID) — no fixed prefix.
	// The shared helper consults the centralized existingTemplatePrefixes slice
	// so this test stays correct as future templates extend the list.
	assertTemplateAvoidsExistingPrefixes(t, DNSTemplate(), nil)
}

// TestDNSTemplate_RcAndRdTagsSurviveMapTagType is a regression test for a bug
// where dns.go used Type:"rc"/Type:"rd" (short names) that mapTagType did not
// recognize, causing <rc> and <rd> tags to be silently dropped during CPS
// generation. This produced I-packets that were too short and had malformed DNS
// structure (label lengths without label content), causing AmneziaVPN error
// 1000 on connect while sing-box tolerated the shorter packets.
//
// After the fix, dns.go uses Type:"random_chars"/Type:"random_digits" (the full
// names that mapTagType expects, matching sip.go's convention). This test
// verifies that every random_chars/random_digits tag in the template produces
// the corresponding <rc>/<rd> tag in the rendered CPS string.
func TestDNSTemplate_RcAndRdTagsSurviveMapTagType(t *testing.T) {
	tmpl := DNSTemplate()
	intervals := map[string][]TagSpec{
		"I1": tmpl.I1, "I2": tmpl.I2, "I3": tmpl.I3, "I4": tmpl.I4,
	}
	for name, intvl := range intervals {
		cps := buildCPSFromTemplate(intvl)
		for _, tag := range intvl {
			switch tag.Type {
			case "random_chars":
				expected := "<rc " + tag.Value + ">"
				if !strings.Contains(cps, expected) {
					t.Errorf("%s: CPS %q missing <rc> tag %q (mapTagType dropped it)", name, cps, expected)
				}
			case "random_digits":
				expected := "<rd " + tag.Value + ">"
				if !strings.Contains(cps, expected) {
					t.Errorf("%s: CPS %q missing <rd> tag %q (mapTagType dropped it)", name, cps, expected)
				}
			}
		}
	}
}
