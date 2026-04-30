package amnezigo

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

// Analysis heuristic thresholds.
const (
	// defaultAnalysisProtocol is the default protocol used when none is specified.
	defaultAnalysisProtocol = "random"

	// iPacketClusterMinWidth is the minimum acceptable spread of I-packet sizes.
	iPacketClusterMinWidth = 20
	// s4MinPadding is the minimum acceptable S4 transport padding.
	s4MinPadding = 8
	// paddedSizeMinDiff is the minimum acceptable difference between padded sizes.
	paddedSizeMinDiff = 5
	// junkMinWidth is the minimum acceptable junk range width.
	junkMinWidth = 32
	// headerMinWidth is the minimum acceptable H-range width.
	headerMinWidth = 1_000_000
	// meanRoundingFactor is the multiplier for rounding mean to 2 decimal places.
	meanRoundingFactor = 100
)

// AnalysisReport is the top-level output of Analyze().
type AnalysisReport struct {
	Peers      []PeerProfile    `json:"peers"`
	Findings   []Finding        `json:"findings"`
	Ordering   OrderingDesc     `json:"ordering"`
	SampleNote string           `json:"sample_note"`
	Config     ConfigInfo       `json:"config"`
	Handshake  HandshakeProfile `json:"handshake"`
	Headers    HeaderProfile    `json:"headers"`
	Junk       JunkProfile      `json:"junk"`
}

// ConfigInfo holds basic config metadata.
type ConfigInfo struct {
	Protocol   string `json:"protocol"`
	MTU        int    `json:"mtu"`
	ListenPort int    `json:"listen_port"`
	PeerCount  int    `json:"peer_count"`
}

// HandshakeProfile describes the four padded handshake sizes.
type HandshakeProfile struct {
	Init      PaddedSize `json:"init"`
	Response  PaddedSize `json:"response"`
	Cookie    PaddedSize `json:"cookie"`
	Transport PaddedSize `json:"transport"`
}

// PaddedSize shows S-prefix, raw WG size, and padded total.
type PaddedSize struct {
	SPrefix int `json:"s_prefix"`
	RawSize int `json:"raw_size"`
	Padded  int `json:"padded"`
}

// JunkProfile describes junk packet parameters.
type JunkProfile struct {
	Jc    int `json:"jc"`
	Jmin  int `json:"jmin"`
	Jmax  int `json:"jmax"`
	Width int `json:"width"`
}

// HeaderProfile describes the four H-ranges.
type HeaderProfile struct {
	H1 HeaderRangeInfo `json:"h1"`
	H2 HeaderRangeInfo `json:"h2"`
	H3 HeaderRangeInfo `json:"h3"`
	H4 HeaderRangeInfo `json:"h4"`
}

// HeaderRangeInfo describes a single header range with width.
type HeaderRangeInfo struct {
	Min   uint32 `json:"min"`
	Max   uint32 `json:"max"`
	Width uint32 `json:"width"`
}

// PeerProfile holds I-packet analysis for one peer.
type PeerProfile struct {
	Distrib  *PeerDistrib `json:"distribution,omitempty"`
	Name     string       `json:"name"`
	Snapshot PeerSnapshot `json:"snapshot"`
}

// PeerSnapshot is a single-generation sample of I-packet sizes.
type PeerSnapshot struct {
	I1 int `json:"i1"`
	I2 int `json:"i2"`
	I3 int `json:"i3"`
	I4 int `json:"i4"`
	I5 int `json:"i5"`
}

// PeerDistrib holds statistics from N samples of I-packet generation.
type PeerDistrib struct {
	I1      Stats `json:"i1"`
	I2      Stats `json:"i2"`
	I3      Stats `json:"i3"`
	I4      Stats `json:"i4"`
	I5      Stats `json:"i5"`
	Samples int   `json:"samples"`
}

// Stats holds basic statistical measures for a value set.
type Stats struct {
	Mean   float64 `json:"mean"`
	Min    int     `json:"min"`
	Max    int     `json:"max"`
	Median int     `json:"median"`
}

// OrderingDesc describes the packet ordering on the wire per handshake.
type OrderingDesc struct {
	Steps []string `json:"steps"`
}

// AnalyzeOptions configures the analysis.
type AnalyzeOptions struct {
	// Rand is the randomness source for CPS generation.
	// When nil, the default crypto/rand is used.
	Rand io.Reader
	// Protocol selects the obfuscation protocol template.
	Protocol string
	// PeerName filters to a specific peer (empty = all peers).
	PeerName string
	// Samples is the number of samples for distribution analysis (0 = snapshot only).
	Samples int
}

// Analyze produces an AnalysisReport for the given server config.
// I-packets are freshly generated from the config parameters, not read from disk.
func Analyze(cfg ServerConfig, opts AnalyzeOptions) AnalysisReport {
	if opts.Protocol == "" {
		opts.Protocol = defaultAnalysisProtocol
	}

	report := AnalysisReport{
		Config: ConfigInfo{
			MTU:        cfg.Interface.MTU,
			ListenPort: cfg.Interface.ListenPort,
			PeerCount:  len(cfg.Peers),
			Protocol:   opts.Protocol,
		},
		Handshake: buildHandshakeProfile(cfg.Obfuscation),
		Junk:      buildJunkProfile(cfg.Obfuscation),
		Headers:   buildHeaderProfile(cfg.Obfuscation),
		Ordering:  buildOrdering(cfg.Obfuscation.Jc),
		SampleNote: "I-packet sizes are freshly generated from config parameters " +
			"and may differ on each run.",
	}

	// Select peers to analyze.
	peers := cfg.Peers
	if opts.PeerName != "" {
		peers = filterPeers(cfg.Peers, opts.PeerName)
	}

	for _, peer := range peers {
		pp := analyzePeer(peer, cfg, opts)
		report.Peers = append(report.Peers, pp)
	}

	report.Findings = runHeuristics(cfg.Obfuscation, report)

	return report
}

// filterPeers returns only peers matching the given name.
func filterPeers(peers []PeerConfig, name string) []PeerConfig {
	var result []PeerConfig
	for _, p := range peers {
		if p.Name == name {
			result = append(result, p)
		}
	}
	return result
}

func buildHandshakeProfile(obf ServerObfuscationConfig) HandshakeProfile {
	return HandshakeProfile{
		Init: PaddedSize{
			SPrefix: obf.S1,
			RawSize: wgInitiationSize,
			Padded:  obf.S1 + wgInitiationSize,
		},
		Response: PaddedSize{
			SPrefix: obf.S2,
			RawSize: wgResponseSize,
			Padded:  obf.S2 + wgResponseSize,
		},
		Cookie: PaddedSize{
			SPrefix: obf.S3,
			RawSize: wgCookieReplySize,
			Padded:  obf.S3 + wgCookieReplySize,
		},
		Transport: PaddedSize{
			SPrefix: obf.S4,
			RawSize: wgTransportSize,
			Padded:  obf.S4 + wgTransportSize,
		},
	}
}

func buildJunkProfile(obf ServerObfuscationConfig) JunkProfile {
	width := 0
	if obf.Jmax >= obf.Jmin {
		width = obf.Jmax - obf.Jmin + 1
	}
	return JunkProfile{
		Jc:    obf.Jc,
		Jmin:  obf.Jmin,
		Jmax:  obf.Jmax,
		Width: width,
	}
}

func buildHeaderProfile(obf ServerObfuscationConfig) HeaderProfile {
	return HeaderProfile{
		H1: headerRangeInfo(obf.H1),
		H2: headerRangeInfo(obf.H2),
		H3: headerRangeInfo(obf.H3),
		H4: headerRangeInfo(obf.H4),
	}
}

func headerRangeInfo(r HeaderRange) HeaderRangeInfo {
	width := uint32(0)
	if r.Max >= r.Min {
		width = r.Max - r.Min + 1
	}
	return HeaderRangeInfo{
		Min:   r.Min,
		Max:   r.Max,
		Width: width,
	}
}

func buildOrdering(jc int) OrderingDesc {
	steps := []string{
		"i1 -> i2 -> i3 -> i4 -> i5",
	}
	if jc > 0 {
		steps = append(steps, fmt.Sprintf("junk x %d", jc))
	}
	steps = append(steps, "Handshake Init")
	return OrderingDesc{Steps: steps}
}

func analyzePeer(peer PeerConfig, cfg ServerConfig, opts AnalyzeOptions) PeerProfile {
	pp := PeerProfile{Name: peer.Name}

	// Generate a snapshot (single sample).
	snap := generateSnapshot(cfg, opts.Protocol)
	pp.Snapshot = snap

	// Distribution analysis (if requested).
	if opts.Samples > 0 {
		pp.Distrib = generateDistribution(cfg, opts.Protocol, opts.Samples)
	}

	return pp
}

func generateSnapshot(cfg ServerConfig, protocol string) PeerSnapshot {
	i1, i2, i3, i4, i5 := GenerateCPS(protocol, cfg.Interface.MTU, cfg.Obfuscation.S1, 0)
	return PeerSnapshot{
		I1: calculateCPSLength(i1),
		I2: calculateCPSLength(i2),
		I3: calculateCPSLength(i3),
		I4: calculateCPSLength(i4),
		I5: calculateCPSLength(i5),
	}
}

func generateDistribution(cfg ServerConfig, protocol string, n int) *PeerDistrib {
	const (
		iPacketCount = 5
		idxI2        = 2
		idxI3        = 3
		idxI4        = 4
	)

	samples := make([][iPacketCount]int, n)
	for i := range n {
		i1, i2, i3, i4, i5 := GenerateCPS(protocol, cfg.Interface.MTU, cfg.Obfuscation.S1, 0)
		samples[i] = [iPacketCount]int{
			calculateCPSLength(i1),
			calculateCPSLength(i2),
			calculateCPSLength(i3),
			calculateCPSLength(i4),
			calculateCPSLength(i5),
		}
	}

	return &PeerDistrib{
		Samples: n,
		I1:      computeStats(samples, 0),
		I2:      computeStats(samples, 1),
		I3:      computeStats(samples, idxI2),
		I4:      computeStats(samples, idxI3),
		I5:      computeStats(samples, idxI4),
	}
}

func computeStats(samples [][5]int, idx int) Stats {
	if len(samples) == 0 {
		return Stats{}
	}

	values := make([]int, len(samples))
	sum := 0
	for i, s := range samples {
		values[i] = s[idx]
		sum += s[idx]
	}

	sort.Ints(values)

	return Stats{
		Min:    values[0],
		Max:    values[len(values)-1],
		Mean:   math.Round(float64(sum)/float64(len(values))*meanRoundingFactor) / meanRoundingFactor,
		Median: medianOf(values),
	}
}

// medianOf returns the median of a sorted, non-empty slice.
//
//nolint:mnd // division by 2 is inherent to median calculation
func medianOf(sorted []int) int {
	mid := len(sorted) / 2
	if len(sorted)%2 != 0 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// runHeuristics applies all RISK001-RISK009 checks.
func runHeuristics(obf ServerObfuscationConfig, report AnalysisReport) []Finding {
	var findings []Finding

	padded := paddedSizes(obf.S1, obf.S2, obf.S3, obf.S4)

	findings = checkRISK001(findings, obf)
	findings = checkRISK002(findings, report)
	findings = checkRISK003(findings, obf)
	findings = checkRISK004(findings, padded)
	findings = checkRISK005(findings, padded)
	findings = checkRISK006(findings, report.Junk.Width)
	findings = checkRISK007(findings, report.Headers)
	findings = checkRISK008(findings, report.Config.PeerCount)
	findings = checkRISK009(findings, obf)

	return findings
}

// checkRISK001 checks if junk range contains raw WG sizes.
func checkRISK001(findings []Finding, obf ServerObfuscationConfig) []Finding {
	rawWG := [4]int{wgInitiationSize, wgResponseSize, wgCookieReplySize, wgTransportSize}
	for _, wgSize := range rawWG {
		if wgSize >= obf.Jmin && wgSize <= obf.Jmax {
			findings = append(findings, Finding{
				Code:     "RISK001",
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"junk range [%d..%d] contains raw WG size %d — junk packets may be misclassified",
					obf.Jmin, obf.Jmax, wgSize),
			})
		}
	}
	return findings
}

// checkRISK002 checks if I-packet cluster width is too narrow.
func checkRISK002(findings []Finding, report AnalysisReport) []Finding {
	for _, peer := range report.Peers {
		sizes := []int{
			peer.Snapshot.I1, peer.Snapshot.I2, peer.Snapshot.I3,
			peer.Snapshot.I4, peer.Snapshot.I5,
		}
		iMin, iMax := sizes[0], sizes[0]
		for _, s := range sizes[1:] {
			if s < iMin {
				iMin = s
			}
			if s > iMax {
				iMax = s
			}
		}
		if iMax-iMin < iPacketClusterMinWidth {
			findings = append(findings, Finding{
				Code:     "RISK002",
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"peer %q I-packet sizes span only %d B (%d..%d) — narrow cluster is easier to fingerprint",
					peer.Name, iMax-iMin, iMin, iMax),
			})
		}
	}
	return findings
}

// checkRISK003 checks if S4 transport padding is too small.
func checkRISK003(findings []Finding, obf ServerObfuscationConfig) []Finding {
	if obf.S4 < s4MinPadding {
		findings = append(findings, Finding{
			Code:     "RISK003",
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"S4 = %d is small — transport padding < %d B makes keepalive packets easily distinguishable",
				obf.S4, s4MinPadding),
		})
	}
	return findings
}

// checkRISK004 checks if padded sizes are too close to each other.
func checkRISK004(findings []Finding, padded [4]int) []Finding {
	pairLabels := [4]string{"Init", "Response", "Cookie", "Transport"}
	for i := range 4 {
		for j := i + 1; j < 4; j++ {
			diff := padded[i] - padded[j]
			if diff < 0 {
				diff = -diff
			}
			if diff > 0 && diff < paddedSizeMinDiff {
				findings = append(findings, Finding{
					Code:     "RISK004",
					Severity: SeverityWarning,
					Message: fmt.Sprintf(
						"padded %s (%d) and %s (%d) differ by only %d B — close sizes weaken classification",
						pairLabels[i], padded[i], pairLabels[j], padded[j], diff),
				})
			}
		}
	}
	return findings
}

// checkRISK005 checks if padded sizes land near raw WG sizes.
func checkRISK005(findings []Finding, padded [4]int) []Finding {
	rawWG := [4]int{wgInitiationSize, wgResponseSize, wgCookieReplySize, wgTransportSize}
	pairLabels := [4]string{"Init", "Response", "Cookie", "Transport"}
	for i, p := range padded {
		for _, raw := range rawWG {
			diff := p - raw
			if diff < 0 {
				diff = -diff
			}
			if diff > 0 && diff < paddedSizeMinDiff {
				findings = append(findings, Finding{
					Code:     "RISK005",
					Severity: SeverityInfo,
					Message: fmt.Sprintf(
						"padded %s (%d) is within ±%d B of raw WG size %d — may confuse naive DPI",
						pairLabels[i], p, paddedSizeMinDiff-1, raw),
				})
			}
		}
	}
	return findings
}

// checkRISK006 checks if junk range width is too narrow.
func checkRISK006(findings []Finding, junkWidth int) []Finding {
	if junkWidth > 0 && junkWidth < junkMinWidth {
		findings = append(findings, Finding{
			Code:     "RISK006",
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"junk range width is %d B — narrow range makes junk packets predictable",
				junkWidth),
		})
	}
	return findings
}

// checkRISK007 checks if H-range widths are too narrow.
func checkRISK007(findings []Finding, headers HeaderProfile) []Finding {
	hRanges := [4]struct {
		name string
		info HeaderRangeInfo
	}{
		{"H1", headers.H1},
		{"H2", headers.H2},
		{"H3", headers.H3},
		{"H4", headers.H4},
	}
	for _, hr := range hRanges {
		if hr.info.Width > 0 && hr.info.Width < headerMinWidth {
			findings = append(findings, Finding{
				Code:     "RISK007",
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"%s range width is %d (< 1M) — narrow header range reduces entropy",
					hr.name, hr.info.Width),
			})
		}
	}
	return findings
}

// checkRISK008 checks if no peers are defined.
func checkRISK008(findings []Finding, peerCount int) []Finding {
	if peerCount == 0 {
		findings = append(findings, Finding{
			Code:     "RISK008",
			Severity: SeverityInfo,
			Message:  "no peers defined — I-packet analysis skipped",
		})
	}
	return findings
}

// checkRISK009 checks if config has vanilla WG shape.
func checkRISK009(findings []Finding, obf ServerObfuscationConfig) []Finding {
	if obf.S1 == 0 && obf.S2 == 0 && obf.S3 == 0 && obf.S4 == 0 &&
		obf.Jc == 0 && obf.Jmin == 0 && obf.Jmax == 0 {
		findings = append(findings, Finding{
			Code:     "RISK009",
			Severity: SeverityWarning,
			Message:  "all S-prefixes and junk parameters are zero — config behaves like vanilla WireGuard",
		})
	}
	return findings
}

// FormatText produces a human-readable text report.
func FormatText(report AnalysisReport) string {
	var b strings.Builder

	b.WriteString("=== AmneziaWG Config Analysis ===\n\n")

	// Config info.
	fmt.Fprintf(&b, "MTU: %d | Port: %d | Peers: %d | Protocol: %s\n\n",
		report.Config.MTU, report.Config.ListenPort,
		report.Config.PeerCount, report.Config.Protocol)

	// Handshake sizes.
	b.WriteString("--- Handshake Sizes ---\n")
	fmt.Fprintf(&b, "  Init:      S1=%d + %d = %d bytes\n",
		report.Handshake.Init.SPrefix, report.Handshake.Init.RawSize, report.Handshake.Init.Padded)
	fmt.Fprintf(&b, "  Response:  S2=%d + %d = %d bytes\n",
		report.Handshake.Response.SPrefix, report.Handshake.Response.RawSize, report.Handshake.Response.Padded)
	fmt.Fprintf(&b, "  Cookie:    S3=%d + %d = %d bytes\n",
		report.Handshake.Cookie.SPrefix, report.Handshake.Cookie.RawSize, report.Handshake.Cookie.Padded)
	fmt.Fprintf(&b, "  Transport: S4=%d + %d = %d bytes\n\n",
		report.Handshake.Transport.SPrefix, report.Handshake.Transport.RawSize, report.Handshake.Transport.Padded)

	// Junk profile.
	b.WriteString("--- Junk Packets ---\n")
	fmt.Fprintf(&b, "  Count: %d (Jc) | Range: [%d..%d] | Width: %d B\n\n",
		report.Junk.Jc, report.Junk.Jmin, report.Junk.Jmax, report.Junk.Width)

	// Header ranges.
	b.WriteString("--- Header Ranges ---\n")
	fmt.Fprintf(&b, "  H1: [%d..%d] (width %d)\n",
		report.Headers.H1.Min, report.Headers.H1.Max, report.Headers.H1.Width)
	fmt.Fprintf(&b, "  H2: [%d..%d] (width %d)\n",
		report.Headers.H2.Min, report.Headers.H2.Max, report.Headers.H2.Width)
	fmt.Fprintf(&b, "  H3: [%d..%d] (width %d)\n",
		report.Headers.H3.Min, report.Headers.H3.Max, report.Headers.H3.Width)
	fmt.Fprintf(&b, "  H4: [%d..%d] (width %d)\n\n",
		report.Headers.H4.Min, report.Headers.H4.Max, report.Headers.H4.Width)

	// Peers.
	formatPeers(&b, report.Peers)

	// Ordering.
	b.WriteString("--- Wire Ordering ---\n")
	for i, step := range report.Ordering.Steps {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, step)
	}
	b.WriteString("\n")

	// Findings.
	if len(report.Findings) > 0 {
		b.WriteString("--- Findings ---\n")
		for _, f := range report.Findings {
			fmt.Fprintf(&b, "  [%s] %s: %s\n", f.Severity, f.Code, f.Message)
		}
		b.WriteString("\n")
	}

	// Disclaimer.
	b.WriteString("Note: " + report.SampleNote + "\n")

	return b.String()
}

func formatPeers(b *strings.Builder, peers []PeerProfile) {
	if len(peers) == 0 {
		return
	}
	b.WriteString("--- I-Packets (per peer) ---\n")
	for _, peer := range peers {
		fmt.Fprintf(b, "  Peer %q:\n", peer.Name)
		fmt.Fprintf(b, "    i1=%d  i2=%d  i3=%d  i4=%d  i5=%d bytes\n",
			peer.Snapshot.I1, peer.Snapshot.I2, peer.Snapshot.I3,
			peer.Snapshot.I4, peer.Snapshot.I5)
		if peer.Distrib != nil {
			fmt.Fprintf(b, "    Distribution (%d samples):\n", peer.Distrib.Samples)
			writeStatLine(b, "i1", peer.Distrib.I1)
			writeStatLine(b, "i2", peer.Distrib.I2)
			writeStatLine(b, "i3", peer.Distrib.I3)
			writeStatLine(b, "i4", peer.Distrib.I4)
			writeStatLine(b, "i5", peer.Distrib.I5)
		}
	}
	b.WriteString("\n")
}

func writeStatLine(b *strings.Builder, name string, s Stats) {
	fmt.Fprintf(b, "      %s: min=%d max=%d mean=%.1f median=%d\n",
		name, s.Min, s.Max, s.Mean, s.Median)
}

// FormatJSON produces a JSON representation of the report.
func FormatJSON(report AnalysisReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal analysis report: %w", err)
	}
	return string(data), nil
}
