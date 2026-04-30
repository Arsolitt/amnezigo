package amnezigo

import (
	"encoding/json"
	"strings"
	"testing"
)

// testServerConfig returns a minimal ServerConfig for analysis tests.
// The obfuscation parameters are chosen so that multiple heuristics can be
// exercised by overriding specific fields.
func testServerConfig() ServerConfig {
	return ServerConfig{
		Interface: InterfaceConfig{
			MTU:        1280,
			ListenPort: 51820,
		},
		Peers: []PeerConfig{
			{Name: "laptop"},
		},
		Obfuscation: ServerObfuscationConfig{
			Jc:   3,
			Jmin: 500,
			Jmax: 900,
			S1:   10,
			S2:   20,
			S3:   30,
			S4:   8,
			H1:   HeaderRange{Min: 100000000, Max: 200000000},
			H2:   HeaderRange{Min: 300000000, Max: 400000000},
			H3:   HeaderRange{Min: 500000000, Max: 600000000},
			H4:   HeaderRange{Min: 700000000, Max: 800000000},
		},
	}
}

func defaultOpts() AnalyzeOptions {
	return AnalyzeOptions{
		Protocol: "random",
	}
}

// --- JSON serialisation stability ---

func TestAnalysisReport_JSONRoundTrip(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, defaultOpts())

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded AnalysisReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Re-marshal and compare lengths as a basic stability check.
	data2, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-Marshal failed: %v", err)
	}
	if len(data) != len(data2) {
		t.Errorf("JSON round-trip changed size: %d -> %d", len(data), len(data2))
	}
}

func TestAnalysisReport_JSONContainsExpectedKeys(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, defaultOpts())

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	jsonStr := string(data)
	requiredKeys := []string{
		"peers", "findings", "ordering", "sample_note",
		"config", "handshake", "headers", "junk",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, `"`+key+`"`) {
			t.Errorf("JSON output missing required key %q", key)
		}
	}
}

// --- Heuristic unit tests ---

// TestRISK001_JunkRangeContainsRawWGSize triggers when junk range includes a
// raw WG size (e.g. 148).
func TestRISK001_JunkRangeContainsRawWGSize(t *testing.T) {
	cfg := testServerConfig()
	// Junk range [100..200] includes wgInitiationSize=148.
	cfg.Obfuscation.Jmin = 100
	cfg.Obfuscation.Jmax = 200
	report := Analyze(cfg, defaultOpts())
	assertFindingPresent(t, report.Findings, "RISK001")
}

func TestRISK001_SafeRange_NoFinding(t *testing.T) {
	cfg := testServerConfig()
	// Junk range [500..900] excludes all raw WG sizes.
	cfg.Obfuscation.Jmin = 500
	cfg.Obfuscation.Jmax = 900
	report := Analyze(cfg, defaultOpts())
	assertFindingAbsent(t, report.Findings, "RISK001")
}

// TestRISK002_NarrowIPacketCluster triggers when I-packet sizes span less
// than iPacketClusterMinWidth.
func TestRISK002_NarrowIPacketCluster(t *testing.T) {
	// We can't directly control I-packet sizes via config since they're
	// generated, but we can test the checkRISK002 heuristic directly.
	report := AnalysisReport{
		Peers: []PeerProfile{
			{
				Name: "test",
				Snapshot: PeerSnapshot{
					I1: 100, I2: 105, I3: 110, I4: 115, I5: 118,
				},
			},
		},
	}
	findings := checkRISK002(nil, report)
	assertFindingPresent(t, findings, "RISK002")
}

func TestRISK002_WideCluster_NoFinding(t *testing.T) {
	report := AnalysisReport{
		Peers: []PeerProfile{
			{
				Name: "test",
				Snapshot: PeerSnapshot{
					I1: 100, I2: 120, I3: 140, I4: 160, I5: 180,
				},
			},
		},
	}
	findings := checkRISK002(nil, report)
	assertFindingAbsent(t, findings, "RISK002")
}

// TestRISK003_SmallS4Padding triggers when S4 < s4MinPadding.
func TestRISK003_SmallS4Padding(t *testing.T) {
	cfg := testServerConfig()
	cfg.Obfuscation.S4 = 2
	report := Analyze(cfg, defaultOpts())
	assertFindingPresent(t, report.Findings, "RISK003")
}

func TestRISK003_AdequateS4_NoFinding(t *testing.T) {
	cfg := testServerConfig()
	cfg.Obfuscation.S4 = 10
	report := Analyze(cfg, defaultOpts())
	assertFindingAbsent(t, report.Findings, "RISK003")
}

// TestRISK004_ClosePaddedSizes triggers when two padded sizes are within
// paddedSizeMinDiff.
func TestRISK004_ClosePaddedSizes(t *testing.T) {
	obf := ServerObfuscationConfig{
		// S1+148 = 148, S2+92 = 150 => diff=2 < 5.
		S1: 0, S2: 58, S3: 30, S4: 10,
	}
	padded := paddedSizes(obf.S1, obf.S2, obf.S3, obf.S4)
	findings := checkRISK004(nil, padded)
	assertFindingPresent(t, findings, "RISK004")
}

func TestRISK004_FarApart_NoFinding(t *testing.T) {
	obf := ServerObfuscationConfig{
		S1: 10, S2: 20, S3: 30, S4: 8,
	}
	padded := paddedSizes(obf.S1, obf.S2, obf.S3, obf.S4)
	findings := checkRISK004(nil, padded)
	assertFindingAbsent(t, findings, "RISK004")
}

// TestRISK005_PaddedNearRawWG triggers when a padded size is near a raw WG size.
func TestRISK005_PaddedNearRawWG(t *testing.T) {
	// S3+64 = 66, wgCookieReplySize=64 => diff=2 < 5 => info.
	obf := ServerObfuscationConfig{
		S1: 10, S2: 20, S3: 2, S4: 8,
	}
	padded := paddedSizes(obf.S1, obf.S2, obf.S3, obf.S4)
	findings := checkRISK005(nil, padded)
	assertFindingPresent(t, findings, "RISK005")
}

func TestRISK005_FarFromRawWG_NoFinding(t *testing.T) {
	obf := ServerObfuscationConfig{
		S1: 20, S2: 30, S3: 40, S4: 10,
	}
	padded := paddedSizes(obf.S1, obf.S2, obf.S3, obf.S4)
	findings := checkRISK005(nil, padded)
	assertFindingAbsent(t, findings, "RISK005")
}

// TestRISK006_NarrowJunkRange triggers when junk range width < junkMinWidth.
func TestRISK006_NarrowJunkRange(t *testing.T) {
	findings := checkRISK006(nil, 10)
	assertFindingPresent(t, findings, "RISK006")
}

func TestRISK006_WideRange_NoFinding(t *testing.T) {
	findings := checkRISK006(nil, 100)
	assertFindingAbsent(t, findings, "RISK006")
}

func TestRISK006_ZeroWidth_NoFinding(t *testing.T) {
	// Width=0 means junk is disabled, not "narrow".
	findings := checkRISK006(nil, 0)
	assertFindingAbsent(t, findings, "RISK006")
}

// TestRISK007_NarrowHeaderRange triggers when H-range width < headerMinWidth.
func TestRISK007_NarrowHeaderRange(t *testing.T) {
	headers := HeaderProfile{
		H1: HeaderRangeInfo{Min: 100, Max: 200, Width: 101},
		H2: HeaderRangeInfo{Min: 300, Max: 400, Width: 101},
		H3: HeaderRangeInfo{Min: 500, Max: 600, Width: 101},
		H4: HeaderRangeInfo{Min: 700, Max: 800, Width: 101},
	}
	findings := checkRISK007(nil, headers)
	assertFindingPresent(t, findings, "RISK007")
}

func TestRISK007_WideRanges_NoFinding(t *testing.T) {
	headers := HeaderProfile{
		H1: HeaderRangeInfo{Min: 100000000, Max: 200000000, Width: 100000001},
		H2: HeaderRangeInfo{Min: 300000000, Max: 400000000, Width: 100000001},
		H3: HeaderRangeInfo{Min: 500000000, Max: 600000000, Width: 100000001},
		H4: HeaderRangeInfo{Min: 700000000, Max: 800000000, Width: 100000001},
	}
	findings := checkRISK007(nil, headers)
	assertFindingAbsent(t, findings, "RISK007")
}

// TestRISK008_NoPeers triggers when peer count is zero.
func TestRISK008_NoPeers(t *testing.T) {
	findings := checkRISK008(nil, 0)
	assertFindingPresent(t, findings, "RISK008")
}

func TestRISK008_HasPeers_NoFinding(t *testing.T) {
	findings := checkRISK008(nil, 2)
	assertFindingAbsent(t, findings, "RISK008")
}

// TestRISK009_VanillaWGShape triggers when all obfuscation params are zero.
func TestRISK009_VanillaWGShape(t *testing.T) {
	obf := ServerObfuscationConfig{
		S1: 0, S2: 0, S3: 0, S4: 0,
		Jc: 0, Jmin: 0, Jmax: 0,
	}
	findings := checkRISK009(nil, obf)
	assertFindingPresent(t, findings, "RISK009")
}

func TestRISK009_NonZeroParams_NoFinding(t *testing.T) {
	obf := ServerObfuscationConfig{
		S1: 10, S2: 20, S3: 30, S4: 8,
		Jc: 3, Jmin: 500, Jmax: 900,
	}
	findings := checkRISK009(nil, obf)
	assertFindingAbsent(t, findings, "RISK009")
}

// --- Severity validation ---

func TestAllFindingsUseWarningOrInfo(t *testing.T) {
	// Build a config that triggers as many findings as possible.
	cfg := ServerConfig{
		Interface: InterfaceConfig{MTU: 1280, ListenPort: 51820},
		Peers:     []PeerConfig{},
		Obfuscation: ServerObfuscationConfig{
			S1: 0, S2: 0, S3: 0, S4: 0,
			Jc: 0, Jmin: 0, Jmax: 0,
			H1: HeaderRange{Min: 5, Max: 10},
			H2: HeaderRange{Min: 20, Max: 30},
			H3: HeaderRange{Min: 40, Max: 50},
			H4: HeaderRange{Min: 60, Max: 70},
		},
	}
	report := Analyze(cfg, defaultOpts())
	for _, f := range report.Findings {
		if f.Severity != SeverityWarning && f.Severity != SeverityInfo {
			t.Errorf("finding %s has severity %q — only warning/info allowed",
				f.RuleID, f.Severity)
		}
	}
}

// --- Orchestrator tests ---

func TestAnalyze_PopulatesConfigInfo(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, AnalyzeOptions{Protocol: "quic"})

	if report.Config.MTU != 1280 {
		t.Errorf("MTU = %d, want 1280", report.Config.MTU)
	}
	if report.Config.ListenPort != 51820 {
		t.Errorf("ListenPort = %d, want 51820", report.Config.ListenPort)
	}
	if report.Config.PeerCount != 1 {
		t.Errorf("PeerCount = %d, want 1", report.Config.PeerCount)
	}
	if report.Config.Protocol != "quic" {
		t.Errorf("Protocol = %q, want quic", report.Config.Protocol)
	}
}

func TestAnalyze_DefaultProtocol(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, AnalyzeOptions{})
	if report.Config.Protocol != defaultAnalysisProtocol {
		t.Errorf("Protocol = %q, want %q", report.Config.Protocol, defaultAnalysisProtocol)
	}
}

func TestAnalyze_PeerFilter(t *testing.T) {
	cfg := testServerConfig()
	cfg.Peers = append(cfg.Peers, PeerConfig{Name: "phone"})
	report := Analyze(cfg, AnalyzeOptions{Protocol: "random", PeerName: "laptop"})
	if len(report.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(report.Peers))
	}
	if report.Peers[0].Name != "laptop" {
		t.Errorf("expected peer laptop, got %q", report.Peers[0].Name)
	}
}

func TestAnalyze_PeerFilter_NoMatch(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, AnalyzeOptions{Protocol: "random", PeerName: "nonexistent"})
	if len(report.Peers) != 0 {
		t.Errorf("expected 0 peers, got %d", len(report.Peers))
	}
}

func TestAnalyze_SnapshotMode(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, AnalyzeOptions{Protocol: "random", Samples: 0})
	if len(report.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(report.Peers))
	}
	peer := report.Peers[0]
	if peer.Distrib != nil {
		t.Error("expected nil distribution in snapshot mode")
	}
	// Snapshot must have positive I-packet sizes.
	if peer.Snapshot.I1 <= 0 || peer.Snapshot.I2 <= 0 ||
		peer.Snapshot.I3 <= 0 || peer.Snapshot.I4 <= 0 ||
		peer.Snapshot.I5 <= 0 {
		t.Errorf("snapshot I-packet sizes must be positive: %+v", peer.Snapshot)
	}
}

func TestAnalyze_DistributionMode(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, AnalyzeOptions{Protocol: "random", Samples: 50})
	if len(report.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(report.Peers))
	}
	peer := report.Peers[0]
	if peer.Distrib == nil {
		t.Fatal("expected non-nil distribution")
	}
	if peer.Distrib.Samples != 50 {
		t.Errorf("Samples = %d, want 50", peer.Distrib.Samples)
	}
	// Stats should have min <= median <= max.
	checkStats(t, "i1", peer.Distrib.I1)
	checkStats(t, "i2", peer.Distrib.I2)
	checkStats(t, "i3", peer.Distrib.I3)
	checkStats(t, "i4", peer.Distrib.I4)
	checkStats(t, "i5", peer.Distrib.I5)
}

// --- Handshake / Junk / Header profile tests ---

func TestBuildHandshakeProfile(t *testing.T) {
	obf := ServerObfuscationConfig{S1: 10, S2: 20, S3: 30, S4: 5}
	hp := buildHandshakeProfile(obf)

	if hp.Init.Padded != 10+wgInitiationSize {
		t.Errorf("Init.Padded = %d, want %d", hp.Init.Padded, 10+wgInitiationSize)
	}
	if hp.Response.Padded != 20+wgResponseSize {
		t.Errorf("Response.Padded = %d, want %d", hp.Response.Padded, 20+wgResponseSize)
	}
	if hp.Cookie.Padded != 30+wgCookieReplySize {
		t.Errorf("Cookie.Padded = %d, want %d", hp.Cookie.Padded, 30+wgCookieReplySize)
	}
	if hp.Transport.Padded != 5+wgTransportSize {
		t.Errorf("Transport.Padded = %d, want %d", hp.Transport.Padded, 5+wgTransportSize)
	}
}

func TestBuildJunkProfile(t *testing.T) {
	obf := ServerObfuscationConfig{Jc: 5, Jmin: 100, Jmax: 200}
	jp := buildJunkProfile(obf)

	if jp.Width != 101 {
		t.Errorf("Width = %d, want 101", jp.Width)
	}
	if jp.Jc != 5 {
		t.Errorf("Jc = %d, want 5", jp.Jc)
	}
}

func TestBuildJunkProfile_InvalidRange(t *testing.T) {
	obf := ServerObfuscationConfig{Jc: 0, Jmin: 200, Jmax: 100}
	jp := buildJunkProfile(obf)
	if jp.Width != 0 {
		t.Errorf("Width = %d, want 0 for invalid range", jp.Width)
	}
}

func TestBuildHeaderProfile(t *testing.T) {
	obf := ServerObfuscationConfig{
		H1: HeaderRange{Min: 100, Max: 200},
		H2: HeaderRange{Min: 300, Max: 400},
		H3: HeaderRange{Min: 500, Max: 600},
		H4: HeaderRange{Min: 700, Max: 800},
	}
	hp := buildHeaderProfile(obf)

	if hp.H1.Width != 101 {
		t.Errorf("H1.Width = %d, want 101", hp.H1.Width)
	}
	if hp.H2.Width != 101 {
		t.Errorf("H2.Width = %d, want 101", hp.H2.Width)
	}
}

func TestBuildOrdering_WithJunk(t *testing.T) {
	od := buildOrdering(3)
	if len(od.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(od.Steps))
	}
	if !strings.Contains(od.Steps[1], "junk") {
		t.Errorf("step 2 should mention junk: %q", od.Steps[1])
	}
}

func TestBuildOrdering_NoJunk(t *testing.T) {
	od := buildOrdering(0)
	if len(od.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(od.Steps))
	}
}

// --- Formatter tests ---

func TestFormatText_ContainsExpectedSections(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, defaultOpts())
	text := FormatText(report)

	sections := []string{
		"AmneziaWG Config Analysis",
		"Handshake Sizes",
		"Junk Packets",
		"Header Ranges",
		"Wire Ordering",
		"Note:",
	}
	for _, s := range sections {
		if !strings.Contains(text, s) {
			t.Errorf("text output missing section %q", s)
		}
	}
}

func TestFormatText_ContainsPeerInfo(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, defaultOpts())
	text := FormatText(report)

	if !strings.Contains(text, "laptop") {
		t.Error("text output should contain peer name")
	}
	if !strings.Contains(text, "I-Packets") {
		t.Error("text output should contain I-Packets section")
	}
}

func TestFormatText_DistributionMode(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, AnalyzeOptions{Protocol: "random", Samples: 10})
	text := FormatText(report)

	if !strings.Contains(text, "Distribution") {
		t.Error("text output should contain Distribution section in distribution mode")
	}
	if !strings.Contains(text, "samples") {
		t.Error("text output should mention sample count")
	}
}

func TestFormatText_FindingsSection(t *testing.T) {
	cfg := testServerConfig()
	cfg.Obfuscation.S4 = 2 // Trigger RISK003.
	report := Analyze(cfg, defaultOpts())
	text := FormatText(report)

	if !strings.Contains(text, "Findings") {
		t.Error("text output should contain Findings section")
	}
	if !strings.Contains(text, "RISK003") {
		t.Error("text output should contain RISK003 finding")
	}
}

func TestFormatJSON_Valid(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, defaultOpts())

	jsonStr, err := FormatJSON(report)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	// Must be valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("FormatJSON produced invalid JSON: %v", err)
	}
}

func TestFormatJSON_Indented(t *testing.T) {
	cfg := testServerConfig()
	report := Analyze(cfg, defaultOpts())

	jsonStr, err := FormatJSON(report)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	// Indented JSON should contain newlines and spaces.
	if !strings.Contains(jsonStr, "\n") {
		t.Error("FormatJSON should produce indented output")
	}
}

// --- Stats helpers ---

func TestComputeStats_SingleSample(t *testing.T) {
	samples := [][5]int{{10, 20, 30, 40, 50}}
	s := computeStats(samples, 0)
	if s.Min != 10 || s.Max != 10 || s.Median != 10 {
		t.Errorf("single sample stats wrong: %+v", s)
	}
}

func TestComputeStats_MultipleSamples(t *testing.T) {
	samples := [][5]int{
		{10, 0, 0, 0, 0},
		{20, 0, 0, 0, 0},
		{30, 0, 0, 0, 0},
	}
	s := computeStats(samples, 0)
	if s.Min != 10 {
		t.Errorf("Min = %d, want 10", s.Min)
	}
	if s.Max != 30 {
		t.Errorf("Max = %d, want 30", s.Max)
	}
	if s.Median != 20 {
		t.Errorf("Median = %d, want 20", s.Median)
	}
	if s.Mean != 20.0 {
		t.Errorf("Mean = %f, want 20.0", s.Mean)
	}
}

func TestComputeStats_Empty(t *testing.T) {
	var samples [][5]int
	s := computeStats(samples, 0)
	if s.Min != 0 || s.Max != 0 || s.Mean != 0 || s.Median != 0 {
		t.Errorf("empty samples should produce zero stats: %+v", s)
	}
}

func TestMedianOf_Odd(t *testing.T) {
	m := medianOf([]int{1, 2, 3, 4, 5})
	if m != 3 {
		t.Errorf("median of odd slice = %d, want 3", m)
	}
}

func TestMedianOf_Even(t *testing.T) {
	m := medianOf([]int{1, 2, 3, 4})
	if m != 2 {
		t.Errorf("median of even slice = %d, want 2 (integer average of 2 and 3)", m)
	}
}

// --- Test helpers ---

func assertFindingPresent(t *testing.T, findings []Finding, ruleID string) {
	t.Helper()
	for _, f := range findings {
		if f.RuleID == ruleID {
			return
		}
	}
	t.Errorf("expected finding %s to be present", ruleID)
}

func assertFindingAbsent(t *testing.T, findings []Finding, ruleID string) {
	t.Helper()
	for _, f := range findings {
		if f.RuleID == ruleID {
			t.Errorf("expected finding %s to be absent, but found: %s", ruleID, f.Message)
			return
		}
	}
}

func checkStats(t *testing.T, name string, s Stats) {
	t.Helper()
	if s.Min > s.Median {
		t.Errorf("%s: Min (%d) > Median (%d)", name, s.Min, s.Median)
	}
	if s.Median > s.Max {
		t.Errorf("%s: Median (%d) > Max (%d)", name, s.Median, s.Max)
	}
	if s.Mean < float64(s.Min) || s.Mean > float64(s.Max) {
		t.Errorf("%s: Mean (%.2f) outside [Min=%d, Max=%d]", name, s.Mean, s.Min, s.Max)
	}
}
