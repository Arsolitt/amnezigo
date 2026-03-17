# Simplify AllowedIPs and Add PresharedKey Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace complex AllowedIPs calculation with `0.0.0.0/0, ::/0` and add PresharedKey to server-side peer blocks.

**Architecture:** Add PresharedKey field to PeerConfig struct, update writer/parser to handle it, simplify export to use hardcoded AllowedIPs, remove unused allowedips.go.

**Tech Stack:** Go 1.x, spf13/cobra

---

## Task 1: Add PresharedKey to PeerConfig struct

**Files:**
- Modify: `internal/config/types.go:24-30`
- Test: `internal/config/types_test.go`

**Step 1: Update PeerConfig struct**

Add `PresharedKey string` field to PeerConfig:

```go
type PeerConfig struct {
	Name         string
	PrivateKey   string
	PublicKey    string
	PresharedKey string
	AllowedIPs   string
	CreatedAt    time.Time
}
```

**Step 2: Run tests to verify no breakage**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/config/types.go
git commit -m "feat(config): add PresharedKey field to PeerConfig"
```

---

## Task 2: Update parser to read PresharedKey from peer blocks

**Files:**
- Modify: `internal/config/parser.go:140-155`
- Test: `internal/config/parser_test.go`

**Step 1: Add test for parsing PresharedKey**

In `parser_test.go`, add to an existing test config or create new test:

```go
func TestParsePeerPresharedKey(t *testing.T) {
	config := `[Interface]
PrivateKey = server_priv
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
PublicKey = peer_pub
PresharedKey = testpsk123
AllowedIPs = 10.8.0.2/32
`
	cfg, err := Parse(strings.NewReader(config))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if cfg.Peers[0].PresharedKey != "testpsk123" {
		t.Errorf("Expected PresharedKey 'testpsk123', got '%s'", cfg.Peers[0].PresharedKey)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -run TestParsePeerPresharedKey -v`
Expected: FAIL

**Step 3: Add PresharedKey parsing case**

In `parser.go`, add case in the peer parsing switch (around line 147):

```go
case "PresharedKey":
	currentPeer.PresharedKey = value
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -run TestParsePeerPresharedKey -v`
Expected: PASS

**Step 5: Run all parser tests**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/config/parser.go internal/config/parser_test.go
git commit -m "feat(config): parse PresharedKey from peer blocks"
```

---

## Task 3: Update writer to output PresharedKey in server peer blocks

**Files:**
- Modify: `internal/config/writer.go:59-73`
- Test: `internal/config/writer_test.go`

**Step 1: Add test for writing PresharedKey in server config**

In `writer_test.go`:

```go
func TestWriteServerConfigWithPresharedKey(t *testing.T) {
	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "server_priv",
			Address:    "10.8.0.1/24",
			ListenPort: 55424,
			MTU:        1420,
		},
		Peers: []PeerConfig{
			{
				PublicKey:    "peer_pub",
				PresharedKey: "testpsk123",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	var buf bytes.Buffer
	err := WriteServerConfig(&buf, cfg)
	if err != nil {
		t.Fatalf("WriteServerConfig failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "PresharedKey = testpsk123") {
		t.Error("Output should contain PresharedKey in peer block")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/... -run TestWriteServerConfigWithPresharedKey -v`
Expected: FAIL

**Step 3: Add PresharedKey output in WriteServerConfig**

In `writer.go`, after line 68 (`PublicKey`), add:

```go
if peer.PresharedKey != "" {
	fmt.Fprintf(w, "PresharedKey = %s\n", peer.PresharedKey)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config/... -run TestWriteServerConfigWithPresharedKey -v`
Expected: PASS

**Step 5: Run all writer tests**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/config/writer.go internal/config/writer_test.go
git commit -m "feat(config): write PresharedKey in server peer blocks"
```

---

## Task 4: Update add command to generate and store PresharedKey

**Files:**
- Modify: `internal/cli/add.go:90-100`
- Test: `internal/cli/add_test.go`

**Step 1: Update test to expect PresharedKey in server config**

Check existing tests in `add_test.go` - they verify server config content. Add assertion for PresharedKey presence.

**Step 2: Run tests to see current state**

Run: `go test ./internal/cli/... -run TestAdd -v`
Expected: May pass or fail depending on current test expectations

**Step 3: Generate PSK and store in PeerConfig**

In `add.go`, around line 95 where PeerConfig is created:

```go
import "github.com/Arsolitt/amnezigo/internal/crypto"

// In the function where peer is created:
psk := crypto.GeneratePSK()
peer := PeerConfig{
	Name:         name,
	PrivateKey:   clientPrivateKey,
	PublicKey:    clientPublicKey,
	PresharedKey: psk,
	AllowedIPs:   clientIP + "/32",
	CreatedAt:    time.Now(),
}
```

Note: Check if `crypto.GeneratePSK()` exists. If not, it needs to be added (32 bytes base64).

**Step 4: Run tests**

Run: `go test ./internal/cli/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/add.go internal/cli/add_test.go
git commit -m "feat(cli): generate and store PresharedKey when adding peer"
```

---

## Task 5: Simplify AllowedIPs in export command

**Files:**
- Modify: `internal/cli/export.go:114-136`
- Test: `internal/cli/export_test.go`

**Step 1: Update test to expect simplified AllowedIPs**

In `export_test.go`, find the test that verifies AllowedIPs (around line 139-142). Change expected value:

```go
// Change from:
expectedAllowedIPs := network.CalculateAllowedIPs("10.8.0.0/24")
// To:
expectedAllowedIPs := "0.0.0.0/0, ::/0"
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/... -run TestExport -v`
Expected: FAIL

**Step 3: Simplify AllowedIPs in export.go**

In `export.go`, replace lines 114-115:

```go
// Remove:
// allowedIPs := network.CalculateAllowedIPs(subnet)

// Replace with:
allowedIPs := "0.0.0.0/0, ::/0"
```

Also remove the import of `"github.com/Arsolitt/amnezigo/internal/network"` if no longer needed.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/... -run TestExport -v`
Expected: PASS

**Step 5: Run all CLI tests**

Run: `go test ./internal/cli/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cli/export.go internal/cli/export_test.go
git commit -m "feat(cli): simplify AllowedIPs to 0.0.0.0/0, ::/0"
```

---

## Task 6: Remove unused allowedips.go and tests

**Files:**
- Delete: `internal/network/allowedips.go`
- Delete: `internal/network/allowedips_test.go`

**Step 1: Verify no other usage**

Run: `grep -r "CalculateAllowedIPs" --include="*.go" .`
Expected: Only in allowedips.go itself (or nowhere)

**Step 2: Delete the files**

```bash
rm internal/network/allowedips.go
rm internal/network/allowedips_test.go
```

**Step 3: Run all tests**

Run: `go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor: remove unused AllowedIPs calculator"
```

---

## Task 7: Final verification

**Step 1: Run all tests**

Run: `go test ./...`
Expected: All PASS

**Step 2: Run linter/type check**

Run: `go vet ./...`
Expected: No issues

**Step 3: Build binary**

Run: `go build .`
Expected: Success

**Step 4: Final commit (if any fixes needed)**

```bash
git status
# If clean, no action needed
```

---

## Summary

After completion:
- Server config peer blocks contain `PresharedKey = <psk>`
- Client config uses `AllowedIPs = 0.0.0.0/0, ::/0`
- `allowedips.go` removed
- All tests pass
