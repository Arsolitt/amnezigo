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
