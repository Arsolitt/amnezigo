package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

// NewGenerateCommand creates the `generate` subcommand. It reads a manifest
// (amnezigo.json or .amnezigo.jsonnet) and produces per-peer WireGuard configs.
func NewGenerateCommand() *cobra.Command {
	var (
		projectDir string
		outputDir  string
		fullReset  bool
		dryRun     bool
		peers      []string
		jpathDirs  []string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate AmneziaWG configs from manifest",
		Long: `Reads amnezigo.json or .amnezigo.jsonnet manifest and generates
per-peer WireGuard configs.

By default, credentials are reused from a previous run. Use --full-reset to
regenerate all keys. Use --dry-run to compute configs without writing files.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGenerate(cmd, projectDir, outputDir, fullReset, dryRun, peers, jpathDirs)
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", "", "project directory containing manifest (default: current dir)")
	cmd.Flags().StringVar(&outputDir, "output", "", "output directory for generated configs (default: project/output)")
	cmd.Flags().BoolVar(&fullReset, "full-reset", false, "regenerate all credentials")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "compute configs without writing files")
	cmd.Flags().StringSliceVar(&peers, "peer", nil, "generate only for specific peers (can be repeated)")
	cmd.Flags().StringSliceVar(&jpathDirs, "jpath", nil, "jsonnet library search paths")

	return cmd
}

func runGenerate(
	cmd *cobra.Command,
	projectDir, outputDir string,
	fullReset, dryRun bool,
	peers, jpathDirs []string,
) error {
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}

	if outputDir == "" {
		outputDir = filepath.Join(projectDir, "output")
	}

	manifest, err := amnezigo.LoadManifest(projectDir, jpathDirs)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	opts := amnezigo.GenerateOptions{
		ProjectDir: projectDir,
		OutputDir:  outputDir,
		FullReset:  fullReset,
		DryRun:     dryRun,
		PeerFilter: peers,
		JpathDirs:  jpathDirs,
	}

	result, err := amnezigo.Generate(manifest, opts)
	if err != nil {
		return fmt.Errorf("generating configs: %w", err)
	}

	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run — no files written")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Generated %d config(s):\n", len(result.Files))
	for _, f := range result.Files {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s (%d bytes)\n", f.RelPath, len(f.Content))
	}

	if len(result.Findings) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nWarnings: %d\n", len(result.Findings))
		for _, f := range result.Findings {
			fmt.Fprintln(cmd.OutOrStdout(), f.OneLine())
		}
	}

	return nil
}
