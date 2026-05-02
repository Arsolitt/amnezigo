package cli

import "testing"

func TestNewGenerateCommand(t *testing.T) {
	cmd := NewGenerateCommand()
	if cmd.Use != "generate" {
		t.Errorf("expected use 'generate', got %q", cmd.Use)
	}

	flags := []string{"project", "output", "full-reset", "dry-run", "peer", "jpath"}
	for _, name := range flags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("missing flag: %s", name)
		}
	}
}
