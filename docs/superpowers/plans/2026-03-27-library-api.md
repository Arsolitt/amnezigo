# Library API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor amnezigo from a CLI-only tool into a Go library with a flat root package API, while keeping the CLI as a thin wrapper.

**Architecture:** Move all business logic from `internal/config`, `internal/crypto`, `internal/obfuscation`, `internal/network` into the root `package amnezigo`. Add a high-level `Manager` type. Move CLI entry point to `cmd/amnezigo/`. Eliminate duplicate `HeaderRange` type.

**Tech Stack:** Go 1.26.1, existing deps (cobra, golang.org/x/crypto)

**Spec:** `docs/superpowers/specs/2026-03-27-library-api-design.md`

---

### Task 1: Move main.go to cmd/amnezigo/

**Files:**
- Create: `cmd/amnezigo/main.go`
- Delete: `main.go`

- [ ] **Step 1: Create cmd/amnezigo/main.go**

```go
package main

import "github.com/Arsolitt/amnezigo/internal/cli"

func main() {
	cli.Execute()
}
```

- [ ] **Step 2: Delete root main.go**

```bash
rm main.go
```

- [ ] **Step 3: Verify build still works**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Run tests to confirm nothing broke**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "refactor: move CLI entry point to cmd/amnezigo/"
```

---

### Task 2: Create root types.go

Consolidate types from `internal/config/types.go` and remove duplicate types from `internal/obfuscation/generator.go`. All types go into root `package amnezigo`.

**Files:**
- Create: `types.go`
- Reference: `internal/config/types.go` (source)
- Reference: `internal/obfuscation/generator.go:30-47` (Headers, SPrefixes, JunkParams — duplicates to remove)

- [ ] **Step 1: Write types.go**

```go
package amnezigo

import "time"

type HeaderRange struct {
	Min, Max uint32
}

type ServerConfig struct {
	Peers       []PeerConfig
	Interface   InterfaceConfig
	Obfuscation ServerObfuscationConfig
}

type InterfaceConfig struct {
	PrivateKey     string
	PublicKey      string
	Address        string
	PostUp         string
	PostDown       string
	MainIface      string
	TunName        string
	EndpointV4     string
	EndpointV6     string
	ListenPort     int
	MTU            int
	ClientToClient bool
}

type PeerConfig struct {
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
	Name              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
}

type ServerObfuscationConfig struct {
	Jc, Jmin, Jmax int
	S1, S2, S3, S4 int
	H1, H2, H3, H4 HeaderRange
}

type ClientObfuscationConfig struct {
	I1 string
	I2 string
	I3 string
	I4 string
	I5 string
	ServerObfuscationConfig
}

type ClientConfig struct {
	Peer      ClientPeerConfig
	Interface ClientInterfaceConfig
}

type ClientInterfaceConfig struct {
	PrivateKey  string
	Address     string
	DNS         string
	Obfuscation ClientObfuscationConfig
	MTU         int
}

type ClientPeerConfig struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          string
	PersistentKeepalive int
}

type Headers struct {
	H1, H2, H3, H4 uint32
}

type SPrefixes struct {
	S1, S2, S3, S4 int
}

type JunkParams struct {
	Jc, Jmin, Jmax int
}

type simpleTag struct {
	Type  string
	Value string
}

type CPSConfig struct {
	I1, I2, I3, I4, I5 string
}

type TagSpec struct {
	Type  string
	Value string
}

type I1I5Template struct {
	I1, I2, I3, I4, I5 []TagSpec
}
```

Note: `simpleTag`, `CPSConfig`, `TagSpec`, `I1I5Template` are unexported — they are internal implementation details of CPS generation.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors (no imports of these types yet from root)

- [ ] **Step 3: Commit**

```bash
git add types.go && git commit -m "refactor: add root package types (consolidated from config + obfuscation)"
```

---

### Task 3: Create root parser.go

Move `internal/config/parser.go` to root, changing `package config` → `package amnezigo` and removing all `config.` prefixes from type references.

**Files:**
- Create: `parser.go`
- Source: `internal/config/parser.go`

- [ ] **Step 1: Write parser.go**

Copy `internal/config/parser.go` content with these transformations:
1. `package config` → `package amnezigo`
2. Remove all `config.` prefixes (e.g., `config.ServerConfig` → `ServerConfig`, `config.HeaderRange` → `HeaderRange`, `config.PeerConfig` → `PeerConfig`)
3. `parseHeaderRange` stays unexported — no changes needed

The file should contain:
- `ParseServerConfig(r io.Reader) (ServerConfig, error)`
- `parseHeaderRange(value string) HeaderRange`

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add parser.go && git commit -m "refactor: add parser to root package"
```

---

### Task 4: Create root writer.go

Move `internal/config/writer.go` to root, same transformations as parser.

**Files:**
- Create: `writer.go`
- Source: `internal/config/writer.go`

- [ ] **Step 1: Write writer.go**

Copy `internal/config/writer.go` content with these transformations:
1. `package config` → `package amnezigo`
2. Remove all `config.` prefixes from type references

The file should contain:
- `WriteServerConfig(w io.Writer, cfg ServerConfig) error`
- `WriteClientConfig(w io.Writer, cfg ClientConfig) error`

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add writer.go && git commit -m "refactor: add writer to root package"
```

---

### Task 5: Create root keys.go

Move `internal/crypto/keys.go` to root.

**Files:**
- Create: `keys.go`
- Source: `internal/crypto/keys.go`

- [ ] **Step 1: Write keys.go**

Copy `internal/crypto/keys.go` with transformation:
1. `package crypto` → `package amnezigo`
2. Remove package comment (root package will have its own)
3. Imports: keep only `"crypto/rand"`, `"encoding/base64"`, `"golang.org/x/crypto/curve25519"`

The file should contain:
- `GenerateKeyPair() (string, string)`
- `DerivePublicKey(privateKey string) string`
- `GeneratePSK() string`

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add keys.go && git commit -m "refactor: add crypto keys to root package"
```

---

### Task 6: Create root protocol template files

Move protocol templates from `internal/obfuscation/protocols/` to root. These are `package protocols` → `package amnezigo`. Functions stay unexported.

**Files:**
- Create: `protocols.go`, `dns.go`, `quic.go`, `dtls.go`, `stun.go`
- Source: `internal/obfuscation/protocols/*.go`

- [ ] **Step 1: Write protocols.go**

Copy `internal/obfuscation/protocols/protocols.go` with:
1. `package protocols` → `package amnezigo`
2. Remove `TagSpec` and `I1I5Template` type definitions (already in `types.go`)

The file should contain only:
- `getTemplate(protocol string) I1I5Template` (renamed from `GetTemplate` to unexported since it's internal)

- [ ] **Step 2: Write dns.go, quic.go, dtls.go, stun.go**

Copy each file with `package protocols` → `package amnezigo`. Remove `I1I5Template` and `TagSpec` references since they're in `types.go` (same package). No other changes needed — all functions are already unexported.

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add protocols.go dns.go quic.go dtls.go stun.go && git commit -m "refactor: add protocol templates to root package"
```

---

### Task 7: Create root generator.go

Move `internal/obfuscation/generator.go` to root, removing duplicate types.

**Files:**
- Create: `generator.go`
- Source: `internal/obfuscation/generator.go`

- [ ] **Step 1: Write generator.go**

Copy `internal/obfuscation/generator.go` with these transformations:
1. `package obfuscation` → `package amnezigo`
2. Remove `HeaderRange` type definition (already in `types.go`)
3. Remove `Headers` type definition (already in `types.go`)
4. Remove `SPrefixes` type definition (already in `types.go`)
5. Remove `JunkParams` type definition (already in `types.go`)
6. Remove import of `"github.com/Arsolitt/amnezigo/internal/config"` (no longer needed — types are in same package)
7. Remove `config.` prefix from all type references

The file should contain:
- All constants (headerRegion*, s4RangeMax, etc.)
- `GenerateHeaders() Headers`
- `GenerateSPrefixes() SPrefixes`
- `GenerateJunkParams() JunkParams`
- `GenerateHeaderRanges() [4]HeaderRange`
- `GenerateServerConfig(_, s1, jc int) ServerObfuscationConfig`
- `GenerateClientConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig`
- `GenerateCPS(protocol string, mtu, s1, jc int) (string, string, string, string, string)`

Note: `GenerateClientConfig` name collides with the obfuscation concept. It generates a `ClientObfuscationConfig` — keep the name as-is since it's clear from the return type.

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add generator.go && git commit -m "refactor: add obfuscation generator to root package"
```

---

### Task 8: Create root cps.go

Move `internal/obfuscation/cps.go` to root.

**Files:**
- Create: `cps.go`
- Source: `internal/obfuscation/cps.go`

- [ ] **Step 1: Write cps.go**

Copy `internal/obfuscation/cps.go` with these transformations:
1. `package obfuscation` → `package amnezigo`
2. Remove `simpleTag` type definition (already in `types.go`)
3. Remove `CPSConfig` type definition (already in `types.go`)
4. Remove import of `"github.com/Arsolitt/amnezigo/internal/obfuscation/protocols"` (types now in same package)
5. Change `protocols.GetTemplate` → `getTemplate` (now unexported, same package)
6. Change `protocols.TagSpec` → `TagSpec` (same package)

The file should contain:
- All constants (reserve, handshakeSize, etc.)
- `calculateMaxISize(mtu, s1 int) int` (unexported)
- `BuildCPSTag(tagType, value string) string` (exported — useful for library users)
- `BuildCPS(tags []string) string` (exported)
- `generateCPSConfig(protocol string, mtu, s1 int) CPSConfig` (unexported)
- All other unexported CPS functions

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cps.go && git commit -m "refactor: add CPS generation to root package"
```

---

### Task 9: Create root iptables.go

Move `internal/network/iptables.go` to root.

**Files:**
- Create: `iptables.go`
- Source: `internal/network/iptables.go`

- [ ] **Step 1: Write iptables.go**

Copy `internal/network/iptables.go` with:
1. `package network` → `package amnezigo`
2. No other changes needed (no type references to update)

The file should contain:
- `GeneratePostUp(tunName, mainIface, subnet string, clientToClient bool) string`
- `GeneratePostDown(tunName, mainIface, subnet string, clientToClient bool) string`

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add iptables.go && git commit -m "refactor: add iptables generation to root package"
```

---

### Task 10: Create root helpers.go

Extract utility functions from CLI that are useful as library functions.

**Files:**
- Create: `helpers.go`
- Source: `internal/cli/add.go:166-205` (findNextAvailableIP)
- Source: `internal/cli/init.go:188-232` (isValidIPAddr, extractSubnet, detectMainInterface)
- Source: `internal/cli/init.go:203-212` (generateRandomPort)

- [ ] **Step 1: Write helpers.go**

```go
package amnezigo

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net"
	"strconv"
	"strings"
)

const (
	minPort  = 10000
	portRange = 55536
)

func IsValidIPAddr(ipaddr string) bool {
	ip, _, err := net.ParseCIDR(ipaddr)
	return err == nil && ip != nil
}

func ExtractSubnet(ipaddr string) string {
	_, ipnet, err := net.ParseCIDR(ipaddr)
	if err != nil {
		return ipaddr
	}
	ones, _ := ipnet.Mask.Size()
	return ipnet.IP.String() + "/" + strconv.Itoa(ones)
}

func GenerateRandomPort() (int, error) {
	maxPort := big.NewInt(portRange)
	n, err := rand.Int(rand.Reader, maxPort)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + minPort, nil
}

func DetectMainInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				return iface.Name
			}
		}
	}

	return ""
}

func FindNextAvailableIP(serverAddress string, existingIPs []string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(serverAddress)
	if err != nil {
		return "", err
	}

	existing := make(map[string]bool)
	for _, ipStr := range existingIPs {
		existing[ipStr] = true
	}

	for i := 2; i <= 254; i++ {
		ipBytes := ip.To4()
		if ipBytes == nil {
			return "", errors.New("not an IPv4 address")
		}

		ipBytes[3] = byte(i)
		candidateIP := ipBytes.String()

		if existing[candidateIP] {
			continue
		}

		if !ipnet.Contains(net.ParseIP(candidateIP)) {
			continue
		}

		return candidateIP, nil
	}

	return "", nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add helpers.go && git commit -m "refactor: extract utility functions to root package"
```

---

### Task 11: Create root manager.go (new code)

This is the only truly new code — the high-level Manager type.

**Files:**
- Create: `manager.go`
- Create: `manager_test.go`

- [ ] **Step 1: Write manager_test.go first (TDD)**

```go
package amnezigo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager("test.conf")
	if mgr.ConfigPath != "test.conf" {
		t.Errorf("expected ConfigPath 'test.conf', got '%s'", mgr.ConfigPath)
	}
}

func TestManagerLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "testpriv=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345,
			MTU:        1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc:   5,
			Jmin: 100,
			Jmax: 200,
			S1:   10,
			S2:   20,
			S3:   30,
			S4:   5,
			H1:   HeaderRange{Min: 100, Max: 200},
			H2:   HeaderRange{Min: 300, Max: 400},
			H3:   HeaderRange{Min: 500, Max: 600},
			H4:   HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	if err := mgr.Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Interface.PrivateKey != cfg.Interface.PrivateKey {
		t.Errorf("PrivateKey mismatch")
	}
	if loaded.Interface.Address != cfg.Interface.Address {
		t.Errorf("Address mismatch")
	}
	if loaded.Interface.ListenPort != cfg.Interface.ListenPort {
		t.Errorf("ListenPort mismatch")
	}
	if loaded.Obfuscation.Jc != cfg.Obfuscation.Jc {
		t.Errorf("Jc mismatch")
	}
}

func TestManagerAddClient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	initialCfg := ServerConfig{
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
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	if err := mgr.Save(initialCfg); err != nil {
		t.Fatalf("initial Save failed: %v", err)
	}

	peer, err := mgr.AddClient("testclient", "")
	if err != nil {
		t.Fatalf("AddClient failed: %v", err)
	}

	if peer.Name != "testclient" {
		t.Errorf("expected name 'testclient', got '%s'", peer.Name)
	}
	if !strings.HasSuffix(peer.AllowedIPs, "/32") {
		t.Errorf("expected AllowedIPs to end with /32, got '%s'", peer.AllowedIPs)
	}
	if peer.PublicKey == "" {
		t.Error("expected non-empty PublicKey")
	}
	if peer.PresharedKey == "" {
		t.Error("expected non-empty PresharedKey")
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(loaded.Peers))
	}
	if loaded.Peers[0].Name != "testclient" {
		t.Errorf("expected peer name 'testclient', got '%s'", loaded.Peers[0].Name)
	}
}

func TestManagerAddClientDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)
	_, _ = mgr.AddClient("dup", "")

	_, err := mgr.AddClient("dup", "")
	if err == nil {
		t.Fatal("expected error for duplicate client name")
	}
}

func TestManagerAddClientWithIP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	peer, err := mgr.AddClient("manual", "10.8.0.50")
	if err != nil {
		t.Fatalf("AddClient with IP failed: %v", err)
	}

	if peer.AllowedIPs != "10.8.0.50/32" {
		t.Errorf("expected AllowedIPs '10.8.0.50/32', got '%s'", peer.AllowedIPs)
	}
}

func TestManagerRemoveClient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{Name: "keep", PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "remove", PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	err := mgr.RemoveClient("remove")
	if err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	loaded, _ := mgr.Load()
	if len(loaded.Peers) != 1 {
		t.Fatalf("expected 1 peer after removal, got %d", len(loaded.Peers))
	}
	if loaded.Peers[0].Name != "keep" {
		t.Errorf("expected remaining peer 'keep', got '%s'", loaded.Peers[0].Name)
	}
}

func TestManagerRemoveClientNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	err := mgr.RemoveClient("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent client")
	}
}

func TestManagerFindClient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{Name: "target", PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	peer, err := mgr.FindClient("target")
	if err != nil {
		t.Fatalf("FindClient failed: %v", err)
	}
	if peer.Name != "target" {
		t.Errorf("expected name 'target', got '%s'", peer.Name)
	}
}

func TestManagerListClients(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{
			{Name: "a", PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "b", PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clients := mgr.ListClients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
}

func TestManagerExportClient(t *testing.T) {
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
				Name:         "exportme",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey:  "psk=",
				AllowedIPs:   "10.8.0.2/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	clientCfg, err := mgr.ExportClient("exportme", "random", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("ExportClient failed: %v", err)
	}

	if clientCfg.Peer.Endpoint != "1.2.3.4:12345" {
		t.Errorf("expected endpoint '1.2.3.4:12345', got '%s'", clientCfg.Peer.Endpoint)
	}
	if clientCfg.Peer.PublicKey != "serverpub=" {
		t.Errorf("expected server public key, got '%s'", clientCfg.Peer.PublicKey)
	}
	if clientCfg.Interface.PrivateKey != "clientpriv=" {
		t.Errorf("expected client private key, got '%s'", clientCfg.Interface.PrivateKey)
	}
}

func TestLoadServerConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	if err := SaveServerConfig(path, cfg); err != nil {
		t.Fatalf("SaveServerConfig failed: %v", err)
	}

	loaded, err := LoadServerConfig(path)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	if loaded.Interface.PrivateKey != cfg.Interface.PrivateKey {
		t.Error("PrivateKey mismatch")
	}
}

func TestSaveServerConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Peers: []PeerConfig{},
	}

	err := SaveServerConfig(path, cfg)
	if err != nil {
		t.Fatalf("SaveServerConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[Interface]") {
		t.Error("expected [Interface] section")
	}
	if !strings.Contains(content, "PrivateKey = priv=") {
		t.Error("expected PrivateKey in output")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestManager -v .`
Expected: compilation errors (Manager type not yet defined)

- [ ] **Step 3: Write manager.go**

```go
package amnezigo

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Manager struct {
	ConfigPath string
}

func NewManager(configPath string) *Manager {
	return &Manager{ConfigPath: configPath}
}

func (m *Manager) Load() (ServerConfig, error) {
	return LoadServerConfig(m.ConfigPath)
}

func (m *Manager) Save(cfg ServerConfig) error {
	return SaveServerConfig(m.ConfigPath, cfg)
}

func (m *Manager) AddClient(name, ip string) (PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return PeerConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	for _, peer := range serverCfg.Peers {
		if peer.Name == name {
			return PeerConfig{}, fmt.Errorf("client with name '%s' already exists", name)
		}
	}

	clientIP := ip
	if clientIP == "" {
		_, ipnet, err := parseCIDR(serverCfg.Interface.Address)
		if err != nil {
			return PeerConfig{}, fmt.Errorf("invalid server address: %w", err)
		}

		existingIPs := make([]string, 0, len(serverCfg.Peers))
		for _, peer := range serverCfg.Peers {
			if before, ok := strings.CutSuffix(peer.AllowedIPs, "/32"); ok {
				peerIP := parseIP(before)
				if peerIP != nil && ipnet.Contains(peerIP) {
					existingIPs = append(existingIPs, before)
				}
			}
		}

		clientIP, err = FindNextAvailableIP(serverCfg.Interface.Address, existingIPs)
		if err != nil {
			return PeerConfig{}, fmt.Errorf("failed to assign IP address: %w", err)
		}
	}

	privateKey, publicKey := GenerateKeyPair()
	psk := GeneratePSK()

	newPeer := PeerConfig{
		Name:        name,
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		PresharedKey: psk,
		AllowedIPs:  clientIP + "/32",
		CreatedAt:   time.Now(),
	}

	serverCfg.Peers = append(serverCfg.Peers, newPeer)

	if err := m.Save(serverCfg); err != nil {
		return PeerConfig{}, fmt.Errorf("failed to save server config: %w", err)
	}

	return newPeer, nil
}

func (m *Manager) RemoveClient(name string) error {
	serverCfg, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	peerIndex := -1
	for i, peer := range serverCfg.Peers {
		if peer.Name == name {
			peerIndex = i
			break
		}
	}

	if peerIndex == -1 {
		return fmt.Errorf("client '%s' not found", name)
	}

	serverCfg.Peers = append(serverCfg.Peers[:peerIndex], serverCfg.Peers[peerIndex+1:]...)

	if err := m.Save(serverCfg); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	return nil
}

func (m *Manager) FindClient(name string) (*PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	for i := range serverCfg.Peers {
		if serverCfg.Peers[i].Name == name {
			return &serverCfg.Peers[i], nil
		}
	}

	return nil, fmt.Errorf("client '%s' not found", name)
}

func (m *Manager) ListClients() []PeerConfig {
	serverCfg, err := m.Load()
	if err != nil {
		return nil
	}
	return serverCfg.Peers
}

func (m *Manager) ExportClient(name, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	var client PeerConfig
	found := false
	for _, peer := range serverCfg.Peers {
		if peer.Name == name {
			client = peer
			found = true
			break
		}
	}
	if !found {
		return ClientConfig{}, fmt.Errorf("client '%s' not found", name)
	}

	return m.BuildClientConfig(client, protocol, endpoint)
}

func (m *Manager) BuildClientConfig(peer PeerConfig, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	clientIP := strings.TrimSuffix(peer.AllowedIPs, "/32")
	allowedIPs := "0.0.0.0/0, ::/0"

	serverPublicKey := serverCfg.Interface.PublicKey
	if serverPublicKey == "" {
		serverPublicKey = DerivePublicKey(serverCfg.Interface.PrivateKey)
	}

	i1, i2, i3, i4, i5 := GenerateCPS(
		protocol,
		serverCfg.Interface.MTU,
		serverCfg.Obfuscation.S1,
		serverCfg.Obfuscation.Jc,
	)

	obfConfig := ClientObfuscationConfig{
		ServerObfuscationConfig: serverCfg.Obfuscation,
		I1:                      i1,
		I2:                      i2,
		I3:                      i3,
		I4:                      i4,
		I5:                      i5,
	}

	clientConfig := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey:  peer.PrivateKey,
			Address:     clientIP + "/32",
			DNS:         "1.1.1.1, 8.8.8.8",
			MTU:         serverCfg.Interface.MTU,
			Obfuscation: obfConfig,
		},
		Peer: ClientPeerConfig{
			PublicKey:           serverPublicKey,
			PresharedKey:        peer.PresharedKey,
			Endpoint:            endpoint,
			AllowedIPs:          allowedIPs,
			PersistentKeepalive: 25,
		},
	}

	return clientConfig, nil
}

func LoadServerConfig(path string) (ServerConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return ServerConfig{}, err
	}
	defer file.Close()

	return ParseServerConfig(file)
}

func SaveServerConfig(path string, cfg ServerConfig) error {
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if err := WriteServerConfig(file, cfg); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	file.Close()

	return os.Rename(tmpPath, path)
}
```

Important: The `AddClient` method must use `net.ParseCIDR` and `net.ParseIP` directly (not `parseCIDR`/`parseIP` shorthand). Add `"net"` and `"strings"` to the imports block:

```go
import (
    "fmt"
    "net"
    "os"
    "strings"
    "time"
)
```

And in `AddClient`, replace `parseCIDR(...)` with `net.ParseCIDR(...)` and `parseIP(before)` with `net.ParseIP(before)`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run "TestManager|TestLoadServerConfig|TestSaveServerConfig" -v .`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add manager.go manager_test.go && git commit -m "feat: add Manager type and convenience file I/O functions"
```

---

### Task 12: Update internal/cli to use root package

Rewrite all CLI command files to import `amnezigo` instead of `internal/config`, `internal/crypto`, `internal/obfuscation`, `internal/network`. Remove all extracted helper functions from CLI.

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/add.go`
- Modify: `internal/cli/remove.go`
- Modify: `internal/cli/edit.go`
- Modify: `internal/cli/list.go`
- Modify: `internal/cli/export.go`
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Update internal/cli/cli.go**

Replace all imports of `internal/*` with `amnezigo`. Remove any unused imports. The file should only import `cobra` and `amnezigo`.

- [ ] **Step 2: Update internal/cli/add.go**

Transformations:
1. Replace imports: remove `"github.com/Arsolitt/amnezigo/internal/config"`, `"github.com/Arsolitt/amnezigo/internal/crypto"`, add `"github.com/Arsolitt/amnezigo"`
2. Remove `loadServerConfig`, `saveServerConfig`, `findNextAvailableIP` functions (now in root package)
3. In `runAdd`: create `mgr := amnezigo.NewManager(configPath)`, call `mgr.AddClient(args[0], addIPAddr)`
4. Remove `fmt.Errorf` wrappers for load/save (Manager handles these)
5. Keep flag definitions and output formatting

Result pattern:
```go
func runAdd(_ *cobra.Command, args []string) error {
    mgr := amnezigo.NewManager(cfgFile)
    peer, err := mgr.AddClient(args[0], addIPAddr)
    if err != nil {
        return err
    }
    fmt.Printf("✓ Client '%s' added successfully\n", peer.Name)
    fmt.Printf("  IP Address: %s\n", peer.AllowedIPs)
    fmt.Printf("  Public Key: %s\n", peer.PublicKey)
    return nil
}
```

- [ ] **Step 3: Update internal/cli/remove.go**

Transformations:
1. Replace imports: remove `internal/config`, add `amnezigo`
2. Remove `loadServerConfig`, `saveServerConfig` (if still present)
3. In `runRemove`: use `mgr := amnezigo.NewManager(cfgFile)`, `mgr.RemoveClient(args[0])`

Result pattern:
```go
func runRemove(_ *cobra.Command, args []string) error {
    mgr := amnezigo.NewManager(cfgFile)
    if err := mgr.RemoveClient(args[0]); err != nil {
        return err
    }
    fmt.Printf("✓ Client '%s' removed successfully\n", args[0])
    return nil
}
```

- [ ] **Step 4: Update internal/cli/edit.go**

Transformations:
1. Replace imports: remove `internal/config`, `internal/network`, add `amnezigo`
2. Remove blank `var _ config.ServerConfig` line
3. In `runEdit`: use `mgr := amnezigo.NewManager(editConfigPath)`, `mgr.Load()`, `mgr.Save()`
4. Use `amnezigo.ExtractSubnet()` instead of local `extractSubnet()`
5. Use `amnezigo.GeneratePostUp()` and `amnezigo.GeneratePostDown()` instead of `network.GeneratePostUp/Down()`

- [ ] **Step 5: Update internal/cli/list.go**

Transformations:
1. No internal package imports to change (list.go doesn't import config/crypto/etc. directly — it uses `loadServerConfig` from add.go in same package)
2. In `runList`: use `mgr := amnezigo.NewManager(cfgFile)`, `mgr.Load()`, `mgr.ListClients()`

Result pattern:
```go
func runList(_ *cobra.Command, _ []string) error {
    mgr := amnezigo.NewManager(cfgFile)
    clients := mgr.ListClients()
    if len(clients) == 0 {
        fmt.Println("No clients configured")
        return nil
    }
    // ... tabwriter formatting unchanged ...
}
```

- [ ] **Step 6: Update internal/cli/export.go**

Transformations:
1. Replace imports: remove `internal/config`, `internal/crypto`, `internal/obfuscation`, add `amnezigo`
2. Remove `loadServerConfig`, `exportClient` functions
3. Keep `getExternalIP` (CLI-specific HTTP call)
4. In `runExport`: use `mgr := amnezigo.NewManager(cfgFile)`, `mgr.ExportClient()` or `mgr.BuildClientConfig()`
5. Write client config file using `amnezigo.WriteClientConfig()`

- [ ] **Step 7: Update internal/cli/init.go**

Transformations:
1. Replace imports: remove `internal/config`, `internal/crypto`, `internal/network`, `internal/obfuscation`, add `amnezigo`
2. Remove: `isValidIPAddr`, `extractSubnet`, `generateRandomPort`, `detectMainInterface`, `writeConfigFile` (now in root)
3. Use `amnezigo.IsValidIPAddr()`, `amnezigo.ExtractSubnet()`, `amnezigo.GenerateRandomPort()`, `amnezigo.DetectMainInterface()`, `amnezigo.GenerateKeyPair()`, `amnezigo.GenerateServerConfig()`, `amnezigo.GeneratePostUp()`, `amnezigo.GeneratePostDown()`, `amnezigo.SaveServerConfig()`
4. Keep: `getEndpointV4`, `getEndpointV6`, `saveMainConfigPath` (CLI-specific)

- [ ] **Step 8: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 9: Run CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: all CLI tests pass

- [ ] **Step 10: Commit**

```bash
git add internal/cli/ && git commit -m "refactor: switch CLI to use root package API"
```

---

### Task 13: Move tests from internal/ to root

Move test files from `internal/config/`, `internal/crypto/`, `internal/obfuscation/`, `internal/network/` to root, updating package names and type references.

**Files:**
- Move: `internal/config/parser_test.go` → `parser_test.go`
- Move: `internal/config/writer_test.go` → `writer_test.go`
- Move: `internal/config/types_test.go` → `types_test.go`
- Move: `internal/crypto/keys_test.go` → `keys_test.go`
- Move: `internal/obfuscation/generator_test.go` → `generator_test.go`
- Move: `internal/obfuscation/cps_test.go` → `cps_test.go`
- Move: `internal/obfuscation/cps_mtu_test.go` → `cps_mtu_test.go`
- Move: `internal/network/iptables_test.go` → `iptables_test.go`

- [ ] **Step 1: Move and update parser_test.go**

Copy `internal/config/parser_test.go` with:
1. `package config` → `package amnezigo`
2. Remove `config.` prefixes from all type references
3. Remove `"github.com/Arsolitt/amnezigo/internal/config"` import

- [ ] **Step 2: Move and update writer_test.go**

Same transformations as parser_test.go.

- [ ] **Step 3: Move and update types_test.go**

Same transformations.

- [ ] **Step 4: Move and update keys_test.go**

Copy `internal/crypto/keys_test.go` with:
1. `package crypto` → `package amnezigo`
2. Remove `"github.com/Arsolitt/amnezigo/internal/crypto"` import (if any)

- [ ] **Step 5: Move and update generator_test.go**

Copy `internal/obfuscation/generator_test.go` with:
1. `package obfuscation` → `package amnezigo`
2. Remove `"github.com/Arsolitt/amnezigo/internal/obfuscation"` import
3. Remove `config.` prefixes, replace with direct type names
4. Remove `"github.com/Arsolitt/amnezigo/internal/config"` import

- [ ] **Step 6: Move and update cps_test.go**

Same transformations as generator_test.go.

- [ ] **Step 7: Move and update cps_mtu_test.go**

Same transformations.

- [ ] **Step 8: Move and update iptables_test.go**

Copy `internal/network/iptables_test.go` with:
1. `package network` → `package amnezigo`
2. Remove `"github.com/Arsolitt/amnezigo/internal/network"` import (if any)

- [ ] **Step 9: Run all root package tests**

Run: `go test -v .`
Expected: all tests pass (including new manager_test.go and moved tests)

- [ ] **Step 10: Commit**

```bash
git add *_test.go && git commit -m "refactor: move tests to root package"
```

---

### Task 14: Delete old internal/ packages

Remove the old internal packages that have been replaced by root package code. Keep `internal/cli/`.

**Files:**
- Delete: `internal/config/` (entire directory)
- Delete: `internal/crypto/` (entire directory)
- Delete: `internal/obfuscation/` (entire directory)
- Delete: `internal/network/` (entire directory)

- [ ] **Step 1: Delete old internal packages**

```bash
rm -rf internal/config internal/crypto internal/obfuscation internal/network
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "refactor: remove old internal packages (replaced by root package)"
```

---

### Task 15: Update build configuration and documentation

Update references to the new CLI path in Dockerfile, AGENTS.md, and README.

**Files:**
- Modify: `Dockerfile`
- Modify: `AGENTS.md`
- Modify: `README.md` (if it references `go install`)
- Modify: `README.ru.md` (if it references `go install`)

- [ ] **Step 1: Update Dockerfile**

Find the `go install` or `go build` command and update the binary path:
- Change `go install github.com/Arsolitt/amnezigo@latest` → `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest`
- Or if using `go build`, change to `go build -o /app/amnezigo ./cmd/amnezigo/`

- [ ] **Step 2: Update AGENTS.md**

- Change `go install github.com/Arsolitt/amnezigo@latest` → `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest`
- Change `go build -o build/amnezigo .` → `go build -o build/amnezigo ./cmd/amnezigo/`
- Update package organization section to reflect new structure

- [ ] **Step 3: Update README files if needed**

Check both README.md and README.ru.md for any `go install` references and update them.

- [ ] **Step 4: Commit**

```bash
git add Dockerfile AGENTS.md README.md README.ru.md && git commit -m "docs: update build instructions for new cmd/amnezigo/ path"
```

---

### Task 16: Full verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 2: Run build**

Run: `go build -o build/amnezigo ./cmd/amnezigo/`
Expected: binary builds successfully

- [ ] **Step 3: Run linter**

Run: `golangci-lint run --fix`
Expected: no errors (or auto-fixed)

- [ ] **Step 4: Run linter again to confirm clean**

Run: `golangci-lint run`
Expected: no errors

- [ ] **Step 5: Commit any lint fixes**

```bash
git add -A && git commit -m "chore: fix lint issues after library refactoring"
```
