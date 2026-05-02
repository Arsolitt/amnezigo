package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	// outputFormatText is the text output format identifier.
	outputFormatText = "text"
	// outputFormatJSON is the JSON output format identifier.
	outputFormatJSON = "json"
)

var (
	cfgFile string
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "amnezigo",
		Short: "AmneziaWG v2.0 Configuration Generator",
		Long:  `Declarative AmneziaWG v2.0 configuration generator.`,
	}

	rootCmd.AddCommand(NewGenerateCommand())
	rootCmd.AddCommand(NewValidateCommand())
	rootCmd.AddCommand(NewAnalyzeCommand())

	return rootCmd
}

func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
