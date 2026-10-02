package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo/internal/buildinfo"
)

// NewVersionCommand returns a fresh `version` subcommand. It prints the build
// stamp injected at compile time (see internal/buildinfo), which lets
// `amnezigo version` identify the exact release a binary came from.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the amnezigo version and commit",
		Long:  `Print the version and the commit the binary was built from.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "amnezigo %s (%s)\n", buildinfo.Version, buildinfo.Commit)
			return nil
		},
	}

	return cmd
}
