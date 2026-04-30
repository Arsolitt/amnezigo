package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	validateOutputFormat string
	validateStrict       bool
	validateQuiet        bool
)

// exitFn is the OS exit function; tests override it via t.Cleanup
// to capture the exit code without aborting the test binary.
var exitFn = os.Exit

// NewValidateCommand returns a fresh `validate` subcommand. Reads a server
// config, runs every validation rule, prints findings in the requested
// format, and exits non-zero on errors (or warnings under --strict).
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <config>",
		Short: "Validate an AmneziaWG server config against AWG 2.0 invariants",
		Long: `Validate runs every check the generator enforces (size collisions,
header ranges, required fields, deprecated tags) against an existing config.

Exit code:
  0 — no errors (warnings/info may still be printed)
  1 — at least one error (or any warning when --strict is set)

Examples:
  amnezigo validate /etc/amnezia/awg0.conf
  amnezigo validate awg0.conf --output json
  amnezigo validate awg0.conf --strict --quiet
`,
		Args: cobra.ExactArgs(1),
		RunE: runValidate,
	}
	cmd.Flags().StringVar(&validateOutputFormat, "output", outputFormatText, "Output format: text|json")
	cmd.Flags().BoolVar(&validateStrict, "strict", false, "Treat warnings as errors for exit code")
	cmd.Flags().BoolVar(&validateQuiet, "quiet", false, "Suppress summary line; print findings only")
	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	path := args[0]
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	cfg, warnings, parseErr := amnezigo.ParseServerConfigWithOptions(
		f, amnezigo.ParseOptions{Strict: true})

	var findings []amnezigo.Finding

	// Convert pre-parse warnings (unknown keys, raw <c>) into findings.
	for _, w := range warnings {
		findings = append(findings, amnezigo.Finding{
			Severity: amnezigo.SeverityWarning,
			Code:     w.Code,
			Location: amnezigo.Location{File: path, Line: w.Line, Key: w.Key},
			Message:  w.Message,
		})
	}

	// If parse aborted (structural error), emit one fatal finding and stop.
	if parseErr != nil {
		findings = append(findings, amnezigo.Finding{
			Severity: amnezigo.SeverityError,
			Code:     "PSE001",
			Location: amnezigo.Location{File: path},
			Message:  parseErr.Error(),
		})
	} else {
		// Run the full validation orchestrator.
		for _, ff := range amnezigo.ValidateServerConfig(&cfg) {
			ff.Location.File = path
			findings = append(findings, ff)
		}
	}

	return emitFindings(cmd.OutOrStdout(), path, findings)
}

type validateSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
}

func emitFindings(w io.Writer, path string, findings []amnezigo.Finding) error {
	s := summarizeFindings(findings)
	switch validateOutputFormat {
	case outputFormatText:
		printText(w, path, findings, s)
	case outputFormatJSON:
		if err := printJSON(w, path, findings, s); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown --output format %q (want: text|json)", validateOutputFormat)
	}

	failed := s.Errors > 0
	if validateStrict && s.Warnings > 0 {
		failed = true
	}
	if failed {
		exitFn(1)
	}
	return nil
}

func summarizeFindings(findings []amnezigo.Finding) validateSummary {
	var s validateSummary
	for _, f := range findings {
		switch f.Severity {
		case amnezigo.SeverityError:
			s.Errors++
		case amnezigo.SeverityWarning:
			s.Warnings++
		case amnezigo.SeverityInfo:
			s.Info++
		}
	}
	return s
}

func printText(w io.Writer, path string, findings []amnezigo.Finding, s validateSummary) {
	for _, f := range findings {
		fmt.Fprintln(w, f.OneLine())
		if f.Detail != "" {
			for l := range strings.SplitSeq(f.Detail, "\n") {
				fmt.Fprintln(w, "  "+l)
			}
		}
	}
	if !validateQuiet {
		marker := "✓"
		if s.Errors > 0 {
			marker = "✗"
		}
		fmt.Fprintf(w, "%s %s: %d errors, %d warnings, %d info\n",
			marker, path, s.Errors, s.Warnings, s.Info)
	}
}

func printJSON(w io.Writer, path string, findings []amnezigo.Finding, s validateSummary) error {
	type doc struct {
		File     string             `json:"file"`
		Findings []amnezigo.Finding `json:"findings"`
		Summary  validateSummary    `json:"summary"`
	}
	if findings == nil {
		findings = []amnezigo.Finding{} // never null in JSON
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc{File: path, Summary: s, Findings: findings})
}
