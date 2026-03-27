package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	tabPadding     = 3
	separatorWidth = 76
)

var (
	cfgFile string
)

// NewRootCmd creates the root command for the CLI.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "amnezigo",
		Short: "AmneziaWG v2.0 Configuration Generator for star topology",
		Long:  `Generate AmneziaWG v2.0 configurations for star topology networks.`,
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(NewClientCommand())
	rootCmd.AddCommand(NewEdgeCommand())

	return rootCmd
}

// Execute runs the CLI application.
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
