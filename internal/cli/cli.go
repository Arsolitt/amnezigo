package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(editCmd)

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
