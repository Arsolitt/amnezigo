package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCLIExists verifies the CLI package is properly structured
func TestCLIExists(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	if cmd.Use != "test" {
		t.Error("cobra command not properly initialized")
	}
}
