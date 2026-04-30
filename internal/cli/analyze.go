package cli

import (
	"fmt"
	"math/rand/v2"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

// randReader wraps math/rand/v2.Rand to satisfy io.Reader for seeded analysis.
type randReader struct {
	rng *rand.Rand
}

func (r *randReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(r.rng.UintN(256)) //nolint:mnd,gosec // byte range; overflow impossible (max 255)
	}
	return len(p), nil
}

// NewAnalyzeCommand creates the `analyze` subcommand. It loads a server config,
// runs heuristic analysis, and prints a human-readable or JSON report. The
// command always exits 0 on success — findings are informational, not errors.
func NewAnalyzeCommand() *cobra.Command {
	var (
		protocol string
		peerName string
		output   string
		samples  int
		seed     uint64
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze obfuscation config for potential weaknesses",
		Long: `Run heuristic analysis on the server configuration.

Reports potential weaknesses (RISK001-RISK009) and profiles handshake
sizes, junk parameters, header ranges, and I-packet distributions.

The command never fails on findings — all output is informational.
Use --seed N for reproducible output (0 = crypto/rand).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAnalyze(cmd, protocol, peerName, output, samples, seed)
		},
	}

	cmd.Flags().StringVar(&protocol, "protocol", "random",
		"obfuscation protocol: random, quic, dns, dtls, stun")
	cmd.Flags().StringVar(&peerName, "peer", "",
		"analyze only this peer (empty = all)")
	cmd.Flags().StringVar(&output, "output", outputFormatText,
		"output format: text, json")
	cmd.Flags().IntVar(&samples, "samples", 0,
		"number of samples for distribution analysis (0 = snapshot only)")
	cmd.Flags().Uint64Var(&seed, "seed", 0,
		"PRNG seed for reproducible output (0 = crypto/rand)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf",
		"server config file path")

	return cmd
}

func runAnalyze(cmd *cobra.Command, protocol, peerName, output string, samples int, seed uint64) error {
	if output != outputFormatText && output != outputFormatJSON {
		return fmt.Errorf("invalid output format %q: must be text or json", output)
	}

	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	opts := amnezigo.AnalyzeOptions{
		Protocol: protocol,
		PeerName: peerName,
		Samples:  samples,
	}

	if seed != 0 {
		//nolint:gosec // deterministic seed is intentional for --seed reproducibility
		opts.Rand = &randReader{rng: rand.New(rand.NewPCG(seed, seed))}
	}

	report := amnezigo.Analyze(serverCfg, opts)

	switch output {
	case outputFormatJSON:
		jsonStr, fmtErr := amnezigo.FormatJSON(report)
		if fmtErr != nil {
			return fmtErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), jsonStr)
	default:
		fmt.Fprint(cmd.OutOrStdout(), amnezigo.FormatText(report))
	}

	return nil
}
