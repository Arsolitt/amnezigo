# Fix Known Gotchas Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 4 documented known gotchas in AGENTS.md.

**Architecture:** Four independent fixes across different files. Gotcha #2 is the largest (touches types, parser, writer, manager, and CLI init). Each task is self-contained and independently testable.

**Tech Stack:** Go, Cobra CLI, crypto/rand

---

### Task 1: Fix "random" protocol deterministic selection (Gotcha #3)

**Files:**
- Modify: `protocols.go:1-26`
- Create: `protocols_test.go`

- [ ] **Step 1: Write the failing test**

Create `protocols_test.go`:

```go
package amnezigo

import (
	"testing"
)

func TestGetTemplate_NamedProtocols(t *testing.T) {
	tests := []struct {
		protocol string
		wantNil  bool
	}{
		{"quic", false},
		{"dns", false},
		{"dtls", false},
		{"stun", false},
	}
	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			tmpl := getTemplate(tt.protocol)
			if tt.wantNil && tmpl.I1 == nil {
				t.Error("expected non-nil template")
			}
			if !tt.wantNil && tmpl.I1 == nil {
				t.Error("expected non-nil I1")
			}
		})
	}
}

func TestGetTemplate_RandomIsNotDeterministic(t *testing.T) {
	seen := make(map[int]bool)
	for i := 0; i < 20; i++ {
		tmpl := getTemplate("random")
		if tmpl.I1 == nil {
			t.Fatal("random template returned nil I1")
		}
		if len(tmpl.I1) > 0 {
			seen[len(tmpl.I1)] = true
		}
	}
	if len(seen) == 1 {
		t.Error("random protocol always returns same template, expected variety")
	}
}

func TestGetTemplate_UnknownFallsBackToRandom(t *testing.T) {
	tmpl := getTemplate("unknown_protocol")
	if tmpl.I1 == nil {
		t.Error("unknown protocol should fall back to random selection, got nil I1")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestGetTemplate_RandomIsNotDeterministic -v ./...`
Expected: FAIL — `len(seen) == 1` because "random" always picks DTLS.

- [ ] **Step 3: Write minimal implementation**

Replace `protocols.go` entirely:

```go
package amnezigo

import (
	"crypto/rand"
	"math/big"
)

// getTemplate returns the I1I5Template for the specified protocol.
// Valid protocols: "quic", "dns", "dtls", "stun", "random" (default).
func getTemplate(protocol string) I1I5Template {
	switch protocol {
	case "quic":
		return QUICTemplate()
	case "dns":
		return DNSTemplate()
	case "dtls":
		return DTLSTemplate()
	case "stun":
		return STUNTemplate()
	default:
		protocols := []func() I1I5Template{
			QUICTemplate,
			DNSTemplate,
			DTLSTemplate,
			STUNTemplate,
		}
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(protocols))))
		return protocols[n.Int64()]()
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run TestGetTemplate -v ./...`
Expected: All PASS

- [ ] **Step 5: Run linter**

Run: `golangci-lint run --fix`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add protocols.go protocols_test.go
git commit -m "fix: use crypto/rand for random protocol selection instead of deterministic modulo"
```

---

### Task 2: Fix GenerateConfig point ranges (Gotcha #4)

**Files:**
- Modify: `generator.go:123-150`
- Modify: `generator_test.go:8-34` (remove TestGenerateHeaders)
- Modify: `generator_test.go:83-134` (update TestGenerateConfig)

- [ ] **Step 1: Update TestGenerateConfig to require true ranges**

In `generator_test.go`, update `TestGenerateConfig` — replace the TODO comment and point-range assertions:

Replace lines 91-104:
```go
	// All header ranges must be valid
	// TODO: Update to require Min < Max once GenerateConfig uses GenerateHeaderRanges
	if cfg.H1.Min == 0 || cfg.H1.Max == 0 {
		t.Error("H1 must have non-zero Min and Max")
	}
	if cfg.H2.Min == 0 || cfg.H2.Max == 0 {
		t.Error("H2 must have non-zero Min and Max")
	}
	if cfg.H3.Min == 0 || cfg.H3.Max == 0 {
		t.Error("H3 must have non-zero Min and Max")
	}
	if cfg.H4.Min == 0 || cfg.H4.Max == 0 {
		t.Error("H4 must have non-zero Min and Max")
	}
```

With:
```go
	// All header ranges must be true ranges (Min < Max)
	for i, hr := range []HeaderRange{cfg.H1, cfg.H2, cfg.H3, cfg.H4} {
		if hr.Min >= hr.Max {
			t.Errorf("H%d must have Min < Max, got Min=%d Max=%d", i+1, hr.Min, hr.Max)
		}
		if hr.Min < 5 {
			t.Errorf("H%d Min must be >= 5, got %d", i+1, hr.Min)
		}
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestGenerateConfig -v ./...`
Expected: FAIL — H1.Min == H1.Max (point ranges).

- [ ] **Step 3: Remove GenerateHeaders function and TestGenerateHeaders**

Delete the `GenerateHeaders` function (lines 27-53) from `generator.go`.
Delete the `TestGenerateHeaders` function (lines 8-34) from `generator_test.go`.
Delete the `Headers` struct from `types.go` (lines 87-90).

- [ ] **Step 4: Update GenerateConfig to use GenerateHeaderRanges**

In `generator.go`, replace the `GenerateConfig` function:

```go
// GenerateConfig combines all obfuscation parameters into a config.
func GenerateConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig {
	h := GenerateHeaderRanges()
	s := GenerateSPrefixes()
	j := GenerateJunkParams()
	cps := generateCPSConfig(protocol, mtu, s1)

	return ClientObfuscationConfig{
		ServerObfuscationConfig: ServerObfuscationConfig{
			Jc:   jc,
			Jmin: j.Jmin,
			Jmax: j.Jmax,
			S1:   s1,
			S2:   s.S2,
			S3:   s.S3,
			S4:   s.S4,
			H1:   h[0],
			H2:   h[1],
			H3:   h[2],
			H4:   h[3],
		},
		I1: cps.I1,
		I2: cps.I2,
		I3: cps.I3,
		I4: cps.I4,
		I5: cps.I5,
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -run "TestGenerateConfig|TestGenerateHeaderRanges" -v ./...`
Expected: All PASS

- [ ] **Step 6: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 7: Run linter**

Run: `golangci-lint run --fix`
Expected: No errors

- [ ] **Step 8: Commit**

```bash
git add generator.go generator_test.go types.go
git commit -m "fix: use GenerateHeaderRanges in GenerateConfig instead of point ranges"
```

---

### Task 3: Add DNS and PersistentKeepalive to InterfaceConfig (Gotcha #2)

**Files:**
- Modify: `types.go:18-31`
- Modify: `parser.go:86-106`
- Modify: `writer.go:10-26`
- Modify: `manager.go:191-239`
- Modify: `manager_test.go:312-359`

- [ ] **Step 1: Add fields to InterfaceConfig**

In `types.go`, add two fields to `InterfaceConfig`:

```go
// InterfaceConfig represents the [Interface] section of a WireGuard config.
type InterfaceConfig struct {
	PrivateKey            string
	PublicKey             string
	Address               string
	PostUp                string
	PostDown              string
	MainIface             string
	TunName               string
	EndpointV4            string
	EndpointV6            string
	ListenPort            int
	MTU                   int
	ClientToClient        bool
	DNS                   string
	PersistentKeepalive   int
}
```

- [ ] **Step 2: Write the failing test for DNS/keepalive roundtrip**

Add to `manager_test.go`:

```go
func TestBuildPeerConfig_CustomDNSAndKeepalive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey:          "serverpriv=",
			PublicKey:           "serverpub=",
			Address:             "10.8.0.1/24",
			ListenPort:          12345,
			MTU:                 1280,
			DNS:                 "9.9.9.9, 1.0.0.1",
			PersistentKeepalive: 15,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{
				Name:         "testpeer",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey: "psk=",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clientCfg, err := mgr.ExportPeer("testpeer", "quic", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("ExportPeer failed: %v", err)
	}

	if clientCfg.Interface.DNS != "9.9.9.9, 1.0.0.1" {
		t.Errorf("expected DNS '9.9.9.9, 1.0.0.1', got '%s'", clientCfg.Interface.DNS)
	}
	if clientCfg.Peer.PersistentKeepalive != 15 {
		t.Errorf("expected PersistentKeepalive 15, got %d", clientCfg.Peer.PersistentKeepalive)
	}
}

func TestBuildPeerConfig_DefaultDNSAndKeepalive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "serverpriv=",
			PublicKey:  "serverpub=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345,
			MTU:        1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{
				Name:         "testpeer",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey: "psk=",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clientCfg, err := mgr.ExportPeer("testpeer", "quic", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("ExportPeer failed: %v", err)
	}

	if clientCfg.Interface.DNS != "1.1.1.1, 8.8.8.8" {
		t.Errorf("expected default DNS '1.1.1.1, 8.8.8.8', got '%s'", clientCfg.Interface.DNS)
	}
	if clientCfg.Peer.PersistentKeepalive != 25 {
		t.Errorf("expected default PersistentKeepalive 25, got %d", clientCfg.Peer.PersistentKeepalive)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -run TestBuildPeerConfig_CustomDNSAndKeepalive -v ./...`
Expected: FAIL — DNS is hardcoded to "1.1.1.1, 8.8.8.8".

- [ ] **Step 4: Add DNS and PersistentKeepalive to parser**

In `parser.go`, in the `[Interface]` section regular fields switch (after line 105, the `MTU` case), add:

```go
			case "DNS":
				cfg.Interface.DNS = value
			case "PersistentKeepalive":
				if ka, err := strconv.Atoi(value); err == nil {
					cfg.Interface.PersistentKeepalive = ka
				}
```

- [ ] **Step 5: Add DNS and PersistentKeepalive to writer**

In `writer.go`, in `WriteServerConfig`, after the `MTU` line (line 18), add:

```go
	if cfg.Interface.DNS != "" {
		fmt.Fprintf(w, "DNS = %s\n", cfg.Interface.DNS)
	}
	if cfg.Interface.PersistentKeepalive != 0 {
		fmt.Fprintf(w, "PersistentKeepalive = %d\n", cfg.Interface.PersistentKeepalive)
	}
```

- [ ] **Step 6: Update BuildPeerConfig to use stored values with defaults**

In `manager.go`, update `BuildPeerConfig`. Replace the hardcoded values:

```go
	dns := serverCfg.Interface.DNS
	if dns == "" {
		dns = "1.1.1.1, 8.8.8.8"
	}

	keepalive := serverCfg.Interface.PersistentKeepalive
	if keepalive == 0 {
		keepalive = defaultPersistentKeepalive
	}
```

Then use these variables in the ClientConfig construction:

```go
	peerConfig := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey:  peer.PrivateKey,
			Address:     peerIP + "/32",
			DNS:         dns,
			MTU:         serverCfg.Interface.MTU,
			Obfuscation: obfConfig,
		},
		Peer: ClientPeerConfig{
			PublicKey:           serverPublicKey,
			PresharedKey:        peer.PresharedKey,
			Endpoint:            endpoint,
			AllowedIPs:          allowedIPs,
			PersistentKeepalive: keepalive,
		},
	}
```

- [ ] **Step 7: Update init CLI to save DNS and keepalive**

In `internal/cli/init.go`, add fields to the `serverCfg` construction (around line 128):

```go
	DNS:                 initDNS,
	PersistentKeepalive: initKeepalive,
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test -run TestBuildPeerConfig -v ./...`
Expected: All PASS

- [ ] **Step 9: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 10: Run linter**

Run: `golangci-lint run --fix`
Expected: No errors

- [ ] **Step 11: Commit**

```bash
git add types.go parser.go writer.go manager.go manager_test.go internal/cli/init.go
git commit -m "fix: persist DNS and PersistentKeepalive from init flags to config"
```

---

### Task 4: Add --endpoint flag to export command (Gotcha #1)

**Files:**
- Modify: `internal/cli/export.go:20-56`

- [ ] **Step 1: Write the failing test**

This is a CLI command change. The logic change is in `runExport`. Add a unit test for the override behavior by testing `resolveExportEndpoint` usage. Since `runExport` calls `resolveExportEndpoint`, we test by verifying that when `--endpoint` is provided, it bypasses auto-resolution.

Add to the top of `internal/cli/export.go` a package-level var for the endpoint flag:

```go
var peerEndpoint string
```

In `NewExportCommand()`, add the flag:

```go
cmd.Flags().StringVar(&peerEndpoint, "endpoint", "", "Override endpoint (skip auto-detection)")
```

In `runExport`, replace the `endpoint` resolution:

```go
func runExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	var endpoint string
	if peerEndpoint != "" {
		endpoint = peerEndpoint
	} else {
		endpoint = resolveExportEndpoint(serverCfg)
	}

	peersToExport, err := selectPeersToExport(serverCfg.Peers, args)
	if err != nil {
		return err
	}

	return writePeerConfigs(mgr, peersToExport, endpoint)
}
```

- [ ] **Step 2: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 3: Run linter**

Run: `golangci-lint run --fix`
Expected: No errors

- [ ] **Step 4: Build to verify compilation**

Run: `go build -o build/amnezigo ./cmd/amnezigo/`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add internal/cli/export.go
git commit -m "feat: add --endpoint flag to export command for manual override"
```

---

### Task 5: Update AGENTS.md Known Gotchas

**Files:**
- Modify: `AGENTS.md` — remove the "Known Gotchas" section (all 4 are now fixed)

- [ ] **Step 1: Remove fixed gotchas from AGENTS.md**

In `AGENTS.md`, remove the entire `### Known Gotchas` section and its subsections (`### CLI Behavior`, `### Obfuscation Generation`, `### CPS Generation`).

- [ ] **Step 2: Commit**

```bash
git add AGENTS.md
git commit -m "docs: remove fixed known gotchas from AGENTS.md"
```
