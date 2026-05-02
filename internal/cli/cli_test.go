package cli

import (
	"slices"
	"testing"
)

func TestRootCmd_RegisteredCommands(t *testing.T) {
	rootCmd := NewRootCmd()
	var names []string
	for _, cmd := range rootCmd.Commands() {
		names = append(names, cmd.Name())
	}

	for _, exp := range []string{"analyze", "generate", "validate"} {
		if !slices.Contains(names, exp) {
			t.Errorf("missing command: %s", exp)
		}
	}

	for _, leg := range []string{"init", "add", "edit", "remove", "export", "list"} {
		if slices.Contains(names, leg) {
			t.Errorf("legacy command still registered: %s", leg)
		}
	}
}
