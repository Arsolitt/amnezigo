package amnezigo

import "testing"

func TestGenerateOptions_ZeroValue(t *testing.T) {
	var opts GenerateOptions
	if opts.ProjectDir != "" {
		t.Error("expected empty ProjectDir")
	}
	if opts.FullReset {
		t.Error("expected FullReset false")
	}
	if opts.PeerFilter != nil {
		t.Error("expected nil PeerFilter")
	}
}

func TestFileOutput_ZeroValue(t *testing.T) {
	var f FileOutput
	if f.RelPath != "" {
		t.Error("expected empty RelPath")
	}
	if f.Content != nil {
		t.Error("expected nil Content")
	}
}

func TestResolveObfuscation_Empty(t *testing.T) {
	obf := ObfuscationManifest{}
	result, err := resolveObfuscation(obf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All S fields should be non-zero
	if result.S1 == 0 {
		t.Error("expected S1 != 0")
	}
	if result.S2 == 0 {
		t.Error("expected S2 != 0")
	}
	if result.S3 == 0 {
		t.Error("expected S3 != 0")
	}
	if result.S4 == 0 {
		t.Error("expected S4 != 0")
	}

	// All J fields should be non-zero
	if result.Jc == 0 {
		t.Error("expected Jc != 0")
	}
	if result.Jmin == 0 {
		t.Error("expected Jmin != 0")
	}
	if result.Jmax == 0 {
		t.Error("expected Jmax != 0")
	}

	// All H fields should be non-zero (check if not HeaderRange{0, 0})
	if result.H1.Min == 0 && result.H1.Max == 0 {
		t.Error("expected H1 to be non-zero")
	}
	if result.H2.Min == 0 && result.H2.Max == 0 {
		t.Error("expected H2 to be non-zero")
	}
	if result.H3.Min == 0 && result.H3.Max == 0 {
		t.Error("expected H3 to be non-zero")
	}
	if result.H4.Min == 0 && result.H4.Max == 0 {
		t.Error("expected H4 to be non-zero")
	}
}

func TestResolveObfuscation_ExplicitValues(t *testing.T) {
	s1, s2, s3, s4 := 50, 100, 150, 200
	obf := ObfuscationManifest{
		S1: &s1,
		S2: &s2,
		S3: &s3,
		S4: &s4,
	}
	result, err := resolveObfuscation(obf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicit S values should be preserved
	if result.S1 != 50 {
		t.Errorf("expected S1 = 50, got %d", result.S1)
	}
	if result.S2 != 100 {
		t.Errorf("expected S2 = 100, got %d", result.S2)
	}
	if result.S3 != 150 {
		t.Errorf("expected S3 = 150, got %d", result.S3)
	}
	if result.S4 != 200 {
		t.Errorf("expected S4 = 200, got %d", result.S4)
	}

	// J fields should be generated
	if result.Jc == 0 {
		t.Error("expected Jc != 0")
	}
	if result.Jmin == 0 {
		t.Error("expected Jmin != 0")
	}
	if result.Jmax == 0 {
		t.Error("expected Jmax != 0")
	}

	// H fields should be generated
	if result.H1.Min == 0 && result.H1.Max == 0 {
		t.Error("expected H1 to be non-zero")
	}
	if result.H2.Min == 0 && result.H2.Max == 0 {
		t.Error("expected H2 to be non-zero")
	}
	if result.H3.Min == 0 && result.H3.Max == 0 {
		t.Error("expected H3 to be non-zero")
	}
	if result.H4.Min == 0 && result.H4.Max == 0 {
		t.Error("expected H4 to be non-zero")
	}
}

func TestResolveObfuscation_PartialExplicit(t *testing.T) {
	s1 := 50
	obf := ObfuscationManifest{
		S1: &s1,
	}
	result, err := resolveObfuscation(obf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// S1 should be preserved
	if result.S1 != 50 {
		t.Errorf("expected S1 = 50, got %d", result.S1)
	}

	// Other S values should be non-zero
	if result.S2 == 0 {
		t.Error("expected S2 != 0")
	}
	if result.S3 == 0 {
		t.Error("expected S3 != 0")
	}
	if result.S4 == 0 {
		t.Error("expected S4 != 0")
	}

	// All S values should be distinct
	sValues := []int{result.S1, result.S2, result.S3, result.S4}
	for i := range sValues {
		for j := i + 1; j < len(sValues); j++ {
			if sValues[i] == sValues[j] {
				t.Errorf("S values should be distinct: S[%d]=%d equals S[%d]=%d", i, sValues[i], j, sValues[j])
			}
		}
	}
}
