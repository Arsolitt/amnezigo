# Edge AWG Client Config Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add edge server support (peers that connect to the hub) and restructure CLI into `client` and `edge` command groups.

**Architecture:** `PeerConfig` gains a `Role` field. `ServerConfig.Peers` is renamed to `ServerConfig.Clients` and a new `ServerConfig.Edges` slice is added. Parser enforces `#_Role` on every `[Peer]`. Edge export reuses `ClientConfig`. CLI commands are restructured into `client` and `edge` groups.

**Tech Stack:** Go, cobra CLI, standard library crypto/rand

---

## Task 1: Add Role field to PeerConfig and split ServerConfig

**Files:**
- Modify: `types.go:10-15` (ServerConfig struct)
- Modify: `types.go:34-42` (PeerConfig struct)
- Modify: `types_test.go`
- Test: `manager_test.go` (will be updated in later tasks — only structural changes here)

- [ ] **Step 1: Add Role field to PeerConfig**

In `types.go`, add `Role string` to `PeerConfig` between `Name` and `PrivateKey`:

```go
type PeerConfig struct {
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
	Name              string
	Role              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
}
```

- [ ] **Step 2: Rename Peers to Clients and add Edges in ServerConfig**

In `types.go`, change `ServerConfig`:

```go
type ServerConfig struct {
	Clients     []PeerConfig
	Edges       []PeerConfig
	Interface   InterfaceConfig
	Obfuscation ServerObfuscationConfig
}
```

- [ ] **Step 3: Add role constants**

Add at the top of `types.go` after imports:

```go
const (
	RoleClient = "client"
	RoleEdge   = "edge"
)
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: Build failure — many files reference `cfg.Peers`. That's expected; we fix them in subsequent tasks.

---

## Task 2: Update parser to enforce #_Role and split Clients/Edges

**Files:**
- Modify: `parser.go:19-176` (ParseServerConfig)
- Modify: `parser_test.go` (all tests — add #_Role to test inputs, update assertions)

- [ ] **Step 1: Write failing test — parser rejects missing #_Role**

In `parser_test.go`, add at the end:

```go
func TestParsePeerRejectsMissingRole(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`

	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for peer without #_Role, got nil")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("expected error about missing role, got: %v", err)
	}
}
```

Run: `go test -run TestParsePeerRejectsMissingRole ./...`
Expected: FAIL (parser doesn't check for role yet)

- [ ] **Step 2: Write failing test — parser rejects invalid #_Role**

```go
func TestParsePeerRejectsInvalidRole(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = server
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`

	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}
}
```

Run: `go test -run TestParsePeerRejectsInvalidRole ./...`
Expected: FAIL

- [ ] **Step 3: Write failing test — parser splits clients and edges**

```go
func TestParseSplitsClientsAndEdges(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Role = client
#_Name = user1
PublicKey = pub1
AllowedIPs = 10.8.0.2/32

[Peer]
#_Role = edge
#_Name = edge1
PublicKey = pub2
AllowedIPs = 10.8.0.3/32

[Peer]
#_Role = client
#_Name = user2
PublicKey = pub3
AllowedIPs = 10.8.0.4/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	if len(cfg.Clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(cfg.Clients))
	}
	if len(cfg.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(cfg.Edges))
	}
	if cfg.Clients[0].Name != "user1" {
		t.Errorf("expected client name 'user1', got '%s'", cfg.Clients[0].Name)
	}
	if cfg.Clients[0].Role != RoleClient {
		t.Errorf("expected role '%s', got '%s'", RoleClient, cfg.Clients[0].Role)
	}
	if cfg.Edges[0].Name != "edge1" {
		t.Errorf("expected edge name 'edge1', got '%s'", cfg.Edges[0].Name)
	}
	if cfg.Edges[0].Role != RoleEdge {
		t.Errorf("expected role '%s', got '%s'", RoleEdge, cfg.Edges[0].Role)
	}
}
```

Run: `go test -run TestParseSplitsClientsAndEdges ./...`
Expected: FAIL

- [ ] **Step 4: Update parser implementation**

In `parser.go`, update `ParseServerConfig`:

1. Change `cfg.Peers` to `cfg.Clients` and `cfg.Edges` in the append logic.
2. Add `Role` case to the `#_` switch for `sectionPeer`.
3. After finalizing a peer section, validate role is set and route to correct slice.

Replace the peer section handling. The key changes:

In the `#_` field parsing for peers, add:
```go
case "Role":
    currentPeer.Role = value
```

Change the section transition (when a new section starts and we need to flush the current peer):
```go
if currentSection == sectionPeer && currentPeer.PublicKey != "" {
    switch currentPeer.Role {
    case RoleClient:
        cfg.Clients = append(cfg.Clients, currentPeer)
    case RoleEdge:
        cfg.Edges = append(cfg.Edges, currentPeer)
    default:
        return ServerConfig{}, fmt.Errorf("peer '%s': missing or invalid #_Role (must be 'client' or 'edge')", currentPeer.Name)
    }
    currentPeer = PeerConfig{}
}
```

Do the same for the end-of-file flush:
```go
if currentSection == sectionPeer && currentPeer.PublicKey != "" {
    switch currentPeer.Role {
    case RoleClient:
        cfg.Clients = append(cfg.Clients, currentPeer)
    case RoleEdge:
        cfg.Edges = append(cfg.Edges, currentPeer)
    default:
        return ServerConfig{}, fmt.Errorf("peer '%s': missing or invalid #_Role (must be 'client' or 'edge')", currentPeer.Name)
    }
}
```

Add `"fmt"` to imports if not already present.

- [ ] **Step 5: Update existing parser tests to include #_Role**

All existing test inputs in `parser_test.go` have `[Peer]` sections without `#_Role`. Add `#_Role = client` to each `[Peer]` section in test inputs, and change all `cfg.Peers` references to `cfg.Clients`.

Tests to update:
- `TestParseServerConfig` — add `#_Role = client`, change `cfg.Peers` → `cfg.Clients`
- `TestParseMultiplePeers` — add `#_Role = client`, change `cfg.Peers` → `cfg.Clients`
- `TestParsePostScripts` — add `#_Role = client`
- `TestParsePeerPresharedKey` — add `#_Role = client`, change `cfg.Peers` → `cfg.Clients`

- [ ] **Step 6: Run tests to verify**

Run: `go test -run TestParse ./...`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add types.go parser.go parser_test.go
git commit -m "refactor: add Role field to PeerConfig, split ServerConfig into Clients/Edges, enforce #_Role in parser"
```

---

## Task 3: Update writer to emit #_Role

**Files:**
- Modify: `writer.go:10-75` (WriteServerConfig)
- Modify: `writer_test.go` (all tests referencing Peers)

- [ ] **Step 1: Write failing test — writer emits #_Role for clients**

Add to `writer_test.go`:

```go
func TestWriteServerConfigEmitsRole(t *testing.T) {
	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv", Address: "10.0.0.1/24",
			ListenPort: 51820, MTU: 1420,
		},
		Clients: []PeerConfig{
			{Name: "c1", Role: RoleClient, PublicKey: "cpub", AllowedIPs: "10.0.0.2/32"},
		},
		Edges: []PeerConfig{
			{Name: "e1", Role: RoleEdge, PublicKey: "epub", AllowedIPs: "10.0.0.3/32"},
		},
	}

	var buf strings.Builder
	if err := WriteServerConfig(&buf, cfg); err != nil {
		t.Fatalf("WriteServerConfig failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "#_Role = client") {
		t.Error("expected #_Role = client for client peer")
	}
	if !strings.Contains(output, "#_Role = edge") {
		t.Error("expected #_Role = edge for edge peer")
	}

	// Verify #_Role appears before PublicKey
	roleIdx := strings.Index(output, "#_Role = client")
	pubIdx := strings.Index(output, "PublicKey = cpub")
	if roleIdx == -1 || pubIdx == -1 || roleIdx > pubIdx {
		t.Error("#_Role should appear before PublicKey")
	}
}
```

Run: `go test -run TestWriteServerConfigEmitsRole ./...`
Expected: FAIL

- [ ] **Step 2: Update WriteServerConfig**

In `writer.go`, update `WriteServerConfig`:

1. Replace the `cfg.Peers` loop with two loops: one for `cfg.Clients` and one for `cfg.Edges`.
2. In each `[Peer]` section, add `#_Role` line after `#_Name`.

```go
for _, peer := range cfg.Clients {
    fmt.Fprintln(w, "")
    fmt.Fprintln(w, "[Peer]")
    if peer.Name != "" {
        fmt.Fprintf(w, "#_Name = %s\n", peer.Name)
    }
    fmt.Fprintf(w, "#_Role = client\n")
    if peer.PrivateKey != "" {
        fmt.Fprintf(w, "#_PrivateKey = %s\n", peer.PrivateKey)
    }
    fmt.Fprintf(w, "PublicKey = %s\n", peer.PublicKey)
    if peer.PresharedKey != "" {
        fmt.Fprintf(w, "PresharedKey = %s\n", peer.PresharedKey)
    }
    fmt.Fprintf(w, "AllowedIPs = %s\n", peer.AllowedIPs)
    if !peer.CreatedAt.IsZero() {
        fmt.Fprintf(w, "#_GenKeyTime = %s\n", peer.CreatedAt.Format(time.RFC3339))
    }
}

for _, peer := range cfg.Edges {
    fmt.Fprintln(w, "")
    fmt.Fprintln(w, "[Peer]")
    if peer.Name != "" {
        fmt.Fprintf(w, "#_Name = %s\n", peer.Name)
    }
    fmt.Fprintf(w, "#_Role = edge\n")
    if peer.PrivateKey != "" {
        fmt.Fprintf(w, "#_PrivateKey = %s\n", peer.PrivateKey)
    }
    fmt.Fprintf(w, "PublicKey = %s\n", peer.PublicKey)
    if peer.PresharedKey != "" {
        fmt.Fprintf(w, "PresharedKey = %s\n", peer.PresharedKey)
    }
    fmt.Fprintf(w, "AllowedIPs = %s\n", peer.AllowedIPs)
    if !peer.CreatedAt.IsZero() {
        fmt.Fprintf(w, "#_GenKeyTime = %s\n", peer.CreatedAt.Format(time.RFC3339))
    }
}
```

- [ ] **Step 3: Update existing writer tests**

In `writer_test.go`:
- `TestWriteServerConfig` — change `Peers:` to `Clients:` and add `Role: RoleClient` to peer entries.
- `TestWriteServerConfigWithPresharedKey` — same change.

- [ ] **Step 4: Run tests**

Run: `go test -run TestWrite ./...`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add writer.go writer_test.go
git commit -m "refactor: update WriteServerConfig to emit #_Role and write Clients/Edges"
```

---

## Task 4: Update Manager — migrate Peers → Clients, add Edge CRUD

**Files:**
- Modify: `manager.go:37-73` (AddClient — set Role)
- Modify: `manager.go:106-131` (RemoveClient — operate on Clients)
- Modify: `manager.go:133-147` (FindClient — operate on Clients)
- Modify: `manager.go:149-156` (ListClients — operate on Clients)
- Modify: `manager.go:158-231` (ExportClient/BuildClientConfig — operate on Clients)
- Add: `manager.go` (AddEdge, RemoveEdge, FindEdge, ListEdges, BuildEdgeConfig, ExportEdge)
- Modify: `manager_test.go` (all tests — Peers → Clients, add edge tests)

- [ ] **Step 1: Write failing tests for edge Manager methods**

Add to `manager_test.go`:

```go
func TestManagerAddEdge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	initialCfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "serverpriv=", PublicKey: "serverpub=",
			Address: "10.8.0.1/24", ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200}, H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600}, H4: HeaderRange{Min: 700, Max: 800},
		},
		Clients: []PeerConfig{},
	}

	mgr := NewManager(path)
	if err := mgr.Save(initialCfg); err != nil {
		t.Fatalf("initial Save failed: %v", err)
	}

	edge, err := mgr.AddEdge("moscow", "")
	if err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}

	if edge.Name != "moscow" {
		t.Errorf("expected name 'moscow', got '%s'", edge.Name)
	}
	if edge.Role != RoleEdge {
		t.Errorf("expected role '%s', got '%s'", RoleEdge, edge.Role)
	}
	if edge.PublicKey == "" {
		t.Error("expected non-empty PublicKey")
	}

	loaded, _ := mgr.Load()
	if len(loaded.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(loaded.Edges))
	}
	if loaded.Clients[0:] != nil && len(loaded.Clients) != 0 {
		t.Error("edge should not be in Clients")
	}
}
```

Run: `go test -run TestManagerAddEdge ./...`
Expected: FAIL (AddEdge not defined)

- [ ] **Step 2: Write failing tests for RemoveEdge, FindEdge, ListEdges**

```go
func TestManagerRemoveEdge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200}, H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600}, H4: HeaderRange{Min: 700, Max: 800},
		},
		Edges: []PeerConfig{
			{Name: "keep", Role: RoleEdge, PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "remove", Role: RoleEdge, PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	if err := mgr.RemoveEdge("remove"); err != nil {
		t.Fatalf("RemoveEdge failed: %v", err)
	}

	loaded, _ := mgr.Load()
	if len(loaded.Edges) != 1 || loaded.Edges[0].Name != "keep" {
		t.Error("expected only 'keep' edge remaining")
	}
}

func TestManagerFindEdge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200}, H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600}, H4: HeaderRange{Min: 700, Max: 800},
		},
		Edges: []PeerConfig{
			{Name: "target", Role: RoleEdge, PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	edge, err := mgr.FindEdge("target")
	if err != nil {
		t.Fatalf("FindEdge failed: %v", err)
	}
	if edge.Name != "target" {
		t.Errorf("expected name 'target', got '%s'", edge.Name)
	}
}

func TestManagerListEdges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "priv=", Address: "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200}, H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600}, H4: HeaderRange{Min: 700, Max: 800},
		},
		Edges: []PeerConfig{
			{Name: "a", Role: RoleEdge, PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "b", Role: RoleEdge, PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	edges := mgr.ListEdges()
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
}
```

- [ ] **Step 3: Update AddClient to set Role**

In `manager.go`, in `AddClient`, change the `newPeer` construction:

```go
newPeer := PeerConfig{
    Name:         name,
    Role:         RoleClient,
    PrivateKey:   privateKey,
    PublicKey:    publicKey,
    PresharedKey: psk,
    AllowedIPs:   clientIP + "/32",
    CreatedAt:    time.Now(),
}
```

Change `serverCfg.Peers` to `serverCfg.Clients` in AddClient:
```go
serverCfg.Clients = append(serverCfg.Clients, newPeer)
```

- [ ] **Step 4: Update RemoveClient, FindClient, ListClients to operate on Clients**

In `manager.go`:
- `RemoveClient`: change `serverCfg.Peers` → `serverCfg.Clients`
- `FindClient`: change `serverCfg.Peers` → `serverCfg.Clients`
- `ListClients`: change `serverCfg.Peers` → `serverCfg.Clients`

- [ ] **Step 5: Update ExportClient/BuildClientConfig to operate on Clients**

In `manager.go`:
- `ExportClient`: change `serverCfg.Peers` → `serverCfg.Clients`
- `BuildClientConfig`: change `serverCfg.Peers` → `serverCfg.Clients`

- [ ] **Step 6: Add Edge CRUD methods**

Add to `manager.go`:

```go
func (m *Manager) AddEdge(name, ip string) (PeerConfig, error) {
    serverCfg, err := m.Load()
    if err != nil {
        return PeerConfig{}, fmt.Errorf("failed to load server config: %w", err)
    }

    for _, edge := range serverCfg.Edges {
        if edge.Name == name {
            return PeerConfig{}, fmt.Errorf("edge with name '%s' already exists", name)
        }
    }

    edgeIP, err := m.resolveClientIP(ip, serverCfg)
    if err != nil {
        return PeerConfig{}, err
    }

    privateKey, publicKey := GenerateKeyPair()
    psk := GeneratePSK()

    newEdge := PeerConfig{
        Name:         name,
        Role:         RoleEdge,
        PrivateKey:   privateKey,
        PublicKey:    publicKey,
        PresharedKey: psk,
        AllowedIPs:   edgeIP + "/32",
        CreatedAt:    time.Now(),
    }

    serverCfg.Edges = append(serverCfg.Edges, newEdge)

    if err := m.Save(serverCfg); err != nil {
        return PeerConfig{}, fmt.Errorf("failed to save server config: %w", err)
    }

    return newEdge, nil
}

func (m *Manager) RemoveEdge(name string) error {
    serverCfg, err := m.Load()
    if err != nil {
        return fmt.Errorf("failed to load server config: %w", err)
    }

    edgeIndex := -1
    for i, edge := range serverCfg.Edges {
        if edge.Name == name {
            edgeIndex = i
            break
        }
    }

    if edgeIndex == -1 {
        return fmt.Errorf("edge '%s' not found", name)
    }

    serverCfg.Edges = append(serverCfg.Edges[:edgeIndex], serverCfg.Edges[edgeIndex+1:]...)

    if err := m.Save(serverCfg); err != nil {
        return fmt.Errorf("failed to save server config: %w", err)
    }

    return nil
}

func (m *Manager) FindEdge(name string) (*PeerConfig, error) {
    serverCfg, err := m.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load server config: %w", err)
    }

    for i := range serverCfg.Edges {
        if serverCfg.Edges[i].Name == name {
            return &serverCfg.Edges[i], nil
        }
    }

    return nil, fmt.Errorf("edge '%s' not found", name)
}

func (m *Manager) ListEdges() []PeerConfig {
    serverCfg, err := m.Load()
    if err != nil {
        return nil
    }
    return serverCfg.Edges
}
```

- [ ] **Step 7: Add BuildEdgeConfig and ExportEdge**

```go
func (m *Manager) BuildEdgeConfig(name, protocol, endpoint string) (ClientConfig, error) {
    serverCfg, err := m.Load()
    if err != nil {
        return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
    }

    var edge PeerConfig
    found := false
    for _, e := range serverCfg.Edges {
        if e.Name == name {
            edge = e
            found = true
            break
        }
    }
    if !found {
        return ClientConfig{}, fmt.Errorf("edge '%s' not found", name)
    }

    edgeIP := strings.TrimSuffix(edge.AllowedIPs, "/32")

    serverPublicKey := serverCfg.Interface.PublicKey
    if serverPublicKey == "" {
        serverPublicKey = DerivePublicKey(serverCfg.Interface.PrivateKey)
    }

    hubIP := extractHubIP(serverCfg.Interface.Address)

    i1, i2, i3, i4, i5 := GenerateCPS(
        protocol,
        serverCfg.Interface.MTU,
        serverCfg.Obfuscation.S1,
        0,
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
            PrivateKey:  edge.PrivateKey,
            Address:     edgeIP + "/32",
            DNS:         "",
            MTU:         serverCfg.Interface.MTU,
            Obfuscation: obfConfig,
        },
        Peer: ClientPeerConfig{
            PublicKey:           serverPublicKey,
            PresharedKey:        edge.PresharedKey,
            Endpoint:            endpoint,
            AllowedIPs:          hubIP + "/32",
            PersistentKeepalive: defaultPersistentKeepalive,
        },
    }

    return clientConfig, nil
}

func extractHubIP(address string) string {
    ip, _, err := net.ParseCIDR(address)
    if err != nil {
        return address
    }
    return ip.String()
}

func (m *Manager) ExportEdge(name, protocol, endpoint string) ([]byte, error) {
    clientCfg, err := m.BuildEdgeConfig(name, protocol, endpoint)
    if err != nil {
        return nil, err
    }

    var buf bytes.Buffer
    if err := WriteClientConfig(&buf, clientCfg); err != nil {
        return nil, fmt.Errorf("failed to write edge config: %w", err)
    }

    return buf.Bytes(), nil
}
```

Add `"bytes"` to imports.

- [ ] **Step 8: Write failing test for BuildEdgeConfig**

```go
func TestManagerBuildEdgeConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	cfg := ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "serverpriv=", PublicKey: "serverpub=",
			Address:    "10.8.0.1/24",
			ListenPort: 12345, MTU: 1280,
		},
		Obfuscation: ServerObfuscationConfig{
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200}, H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600}, H4: HeaderRange{Min: 700, Max: 800},
		},
		Edges: []PeerConfig{
			{
				Name: "moscow", Role: RoleEdge,
				PrivateKey: "edgepriv=", PublicKey: "edgepub=",
				PresharedKey: "psk=", AllowedIPs: "10.8.0.3/32",
			},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	edgeCfg, err := mgr.BuildEdgeConfig("moscow", "random", "1.2.3.4:12345")
	if err != nil {
		t.Fatalf("BuildEdgeConfig failed: %v", err)
	}

	if edgeCfg.Peer.Endpoint != "1.2.3.4:12345" {
		t.Errorf("expected endpoint '1.2.3.4:12345', got '%s'", edgeCfg.Peer.Endpoint)
	}
	if edgeCfg.Peer.PublicKey != "serverpub=" {
		t.Errorf("expected server public key, got '%s'", edgeCfg.Peer.PublicKey)
	}
	if edgeCfg.Interface.PrivateKey != "edgepriv=" {
		t.Errorf("expected edge private key, got '%s'", edgeCfg.Interface.PrivateKey)
	}
	if edgeCfg.Peer.AllowedIPs != "10.8.0.1/32" {
		t.Errorf("expected AllowedIPs '10.8.0.1/32' (hub IP), got '%s'", edgeCfg.Peer.AllowedIPs)
	}
	if edgeCfg.Interface.DNS != "" {
		t.Errorf("expected empty DNS for edge, got '%s'", edgeCfg.Interface.DNS)
	}
}
```

- [ ] **Step 9: Update existing manager tests — Peers → Clients**

In `manager_test.go`, all test functions that create `ServerConfig` with `Peers:` must change to `Clients:`. Add `Role: RoleClient` to each peer entry. Change all `loaded.Peers` references to `loaded.Clients`.

Affected tests:
- `TestManagerLoadSave` — `Peers: []PeerConfig{}` → `Clients: []PeerConfig{}`
- `TestManagerAddClient` — same, plus verify `peer.Role == RoleClient`
- `TestManagerAddClientDuplicate` — same
- `TestManagerAddClientWithIP` — same
- `TestManagerRemoveClient` — `Peers:` → `Clients:`, add `Role: RoleClient`, `loaded.Peers` → `loaded.Clients`
- `TestManagerRemoveClientNotFound` — `Peers:` → `Clients:`
- `TestManagerFindClient` — `Peers:` → `Clients:`, add `Role: RoleClient`
- `TestManagerListClients` — `Peers:` → `Clients:`, add `Role: RoleClient`, `loaded.Peers` → `loaded.Clients`
- `TestManagerExportClient` — `Peers:` → `Clients:`, add `Role: RoleClient`
- `TestLoadServerConfig` — `Peers:` → `Clients:`
- `TestSaveServerConfig` — `Peers:` → `Clients:`

- [ ] **Step 10: Run all tests**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 11: Commit**

```bash
git add manager.go manager_test.go
git commit -m "feat: add edge CRUD methods to Manager, migrate Peers to Clients"
```

---

## Task 5: Update init command

**Files:**
- Modify: `internal/cli/init.go:128-145` (runInit — Peers → Clients)

- [ ] **Step 1: Update init command**

In `internal/cli/init.go`, change:
```go
Peers: []amnezigo.PeerConfig{},
```
to:
```go
Clients: []amnezigo.PeerConfig{},
```

- [ ] **Step 2: Run init tests**

Run: `go test -run TestInit ./internal/cli/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cli/init.go
git commit -m "refactor: update init command to use Clients instead of Peers"
```

---

## Task 6: Restructure CLI — client command group

**Files:**
- Create: `internal/cli/client.go`
- Modify: `internal/cli/cli.go` (register client group)
- Delete: `internal/cli/add.go`, `internal/cli/list.go`, `internal/cli/export.go`, `internal/cli/remove.go`
- Modify: `internal/cli/add_test.go`, `internal/cli/list_test.go`, `internal/cli/export_test.go`, `internal/cli/remove_test.go` (update for #_Role in test configs)

- [ ] **Step 1: Create client.go with all client subcommands**

Create `internal/cli/client.go`:

```go
package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	clientIPAddr    string
	clientProtocol  string
)

// NewClientCommand creates the client command group.
func NewClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Manage client peers",
	}

	cmd.AddCommand(NewClientAddCommand())
	cmd.AddCommand(NewClientListCommand())
	cmd.AddCommand(NewClientExportCommand())
	cmd.AddCommand(NewClientRemoveCommand())

	return cmd
}

// NewClientAddCommand creates the client add subcommand.
func NewClientAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new client to the server configuration",
		Long: `Add a new WireGuard client to the AmneziaWG server configuration.

Generates a keypair for the client and adds it to the server's peer list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo client add laptop
  amnezigo client add phone --ipaddr 10.8.0.50
`,
		Args: cobra.ExactArgs(1),
		RunE: runClientAdd,
	}
	cmd.Flags().StringVar(&clientIPAddr, "ipaddr", "", "Client IP address (e.g., 10.8.0.5)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peer, err := mgr.AddClient(args[0], clientIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Client '%s' added successfully\n", peer.Name)
	fmt.Printf("  IP Address: %s\n", peer.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", peer.PublicKey)

	return nil
}

// NewClientListCommand creates the client list subcommand.
func NewClientListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured clients",
		Long: `List all WireGuard clients configured in the AmneziaWG server configuration.

Displays a table with client name, IP address, and creation time.

Example:
  amnezigo client list
`,
		RunE: runClientList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	clients := mgr.ListClients()
	if len(clients) == 0 {
		fmt.Println("No clients configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, peer := range clients {
		timestamp := ""
		if !peer.CreatedAt.IsZero() {
			timestamp = peer.CreatedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", peer.Name, peer.AllowedIPs, timestamp)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}

// NewClientExportCommand creates the client export subcommand.
func NewClientExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export client configuration(s)",
		Long: `Export WireGuard client configuration(s) for the specified client(s).

If a name is specified, exports only that client's configuration.
If no name is specified, exports all clients' configurations.

Example:
  amnezigo client export laptop
  amnezigo client export --protocol quic laptop
  amnezigo client export
`,
		Args: cobra.MaximumNArgs(1),
		RunE: runClientExport,
	}
	cmd.Flags().StringVar(&clientProtocol, "protocol", "random", "Obfuscation protocol")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := resolveExportEndpoint(serverCfg)

	clientsToExport, err := selectClientsToExport(serverCfg.Clients, args)
	if err != nil {
		return err
	}

	return writeClientConfigs(mgr, clientsToExport, endpoint)
}

// NewClientRemoveCommand creates the client remove subcommand.
func NewClientRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a client from the server configuration",
		Long: `Remove a WireGuard client from the AmneziaWG server configuration.

Removes the peer with the specified name from the server's peer list.

Example:
  amnezigo client remove laptop
`,
		Args: cobra.ExactArgs(1),
		RunE: runClientRemove,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runClientRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemoveClient(args[0]); err != nil {
		return err
	}
	fmt.Printf("Client '%s' removed successfully\n", args[0])
	return nil
}

func resolveExportEndpoint(serverCfg amnezigo.ServerConfig) string {
	if serverCfg.Interface.EndpointV4 != "" {
		return serverCfg.Interface.EndpointV4
	}
	if serverCfg.Interface.EndpointV6 != "" {
		return serverCfg.Interface.EndpointV6
	}
	externalIP, err := getExternalIP()
	if err != nil {
		externalIP = "YOUR_SERVER_IP"
	}
	return fmt.Sprintf("%s:%d", externalIP, serverCfg.Interface.ListenPort)
}

func selectClientsToExport(peers []amnezigo.PeerConfig, args []string) ([]amnezigo.PeerConfig, error) {
	if len(args) == 0 {
		return peers, nil
	}
	clientName := args[0]
	for _, peer := range peers {
		if peer.Name == clientName {
			return []amnezigo.PeerConfig{peer}, nil
		}
	}
	return nil, fmt.Errorf("client '%s' not found", clientName)
}

func writeClientConfigs(mgr *amnezigo.Manager, clients []amnezigo.PeerConfig, endpoint string) error {
	for _, client := range clients {
		clientCfg, err := mgr.BuildClientConfig(client, clientProtocol, endpoint)
		if err != nil {
			return fmt.Errorf("failed to export client '%s': %w", client.Name, err)
		}

		configPath := client.Name + ".conf"
		file, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		defer file.Close()

		if err := amnezigo.WriteClientConfig(file, clientCfg); err != nil {
			return fmt.Errorf("failed to write client config: %w", err)
		}

		fmt.Printf("Exported client '%s' to %s.conf\n", client.Name, client.Name)
	}
	return nil
}

func getExternalIP() (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://icanhazip.com", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get external IP: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	return ip, nil
}
```

Note: `resolveExportEndpoint`, `selectClientsToExport`, `writeClientConfigs`, `getExternalIP` are moved from `export.go` to `client.go` since they're now only used by client export.

- [ ] **Step 2: Delete old command files**

Delete: `internal/cli/add.go`, `internal/cli/list.go`, `internal/cli/export.go`, `internal/cli/remove.go`

- [ ] **Step 3: Update cli.go root command**

Replace `cli.go`:

```go
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

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(NewClientCommand())
	rootCmd.AddCommand(NewEdgeCommand())

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
```

Note: `NewEdgeCommand()` will be defined in Task 7. For now, comment it out or create a stub to allow compilation.

- [ ] **Step 4: Create edge.go stub**

Create `internal/cli/edge.go` with a minimal stub:

```go
package cli

import "github.com/spf13/cobra"

// NewEdgeCommand creates the edge command group.
func NewEdgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Manage edge server peers",
	}
	return cmd
}
```

- [ ] **Step 5: Update CLI tests for #_Role**

All test configs in `add_test.go`, `list_test.go`, `export_test.go`, `remove_test.go` have `[Peer]` sections without `#_Role`. Add `#_Role = client` to each.

Also update function calls:
- `NewAddCommand()` → `NewClientAddCommand()`
- `NewListCommand()` → `NewClientListCommand()`
- `NewExportCommand()` → `NewClientExportCommand()`
- `NewRemoveCommand()` → `NewClientRemoveCommand()`
- `runRemove(...)` → `runClientRemove(...)`
- `addIPAddr` → `clientIPAddr`
- `exportProtocol` → `clientProtocol`

- [ ] **Step 6: Run tests**

Run: `go test ./internal/cli/...`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/
git commit -m "refactor: restructure CLI into client command group"
```

---

## Task 7: Add edge CLI commands

**Files:**
- Modify: `internal/cli/edge.go` (full implementation)
- Create: `internal/cli/edge_test.go`

- [ ] **Step 1: Write failing test — edge add**

Create `internal/cli/edge_test.go`:

```go
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEdgeAddCommand(t *testing.T) {
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	t.Run("add edge with auto-assigned IP", func(t *testing.T) {
		cfgFile = ""
		clientIPAddr = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		initialConfig := `[Interface]
PrivateKey = abcdefghijklmnopqrstuvwxyz123456789012345=
Address = 10.8.0.1/24
ListenPort = 12345
MTU = 1280
Jc = 5
Jmin = 20
Jmax = 30
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400

[Peer]
#_Role = client
#_Name = existing
PublicKey = existingpub1234567890123456789012345678901234567890123456789012345678=
AllowedIPs = 10.8.0.2/32
`
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		cmd := NewEdgeAddCommand()
		cmd.SetArgs([]string{"--config", configPath, "moscow"})
		cfgFile = configPath
		if err := cmd.Execute(); err != nil {
			t.Fatalf("edge add failed: %v", err)
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		configStr := string(content)

		if !strings.Contains(configStr, `#_Name = moscow`) {
			t.Error("expected edge name in config")
		}
		if !strings.Contains(configStr, `#_Role = edge`) {
			t.Error("expected #_Role = edge")
		}
		if !strings.Contains(configStr, `AllowedIPs = 10.8.0.3/32`) {
			t.Error("expected auto-assigned IP 10.8.0.3/32 (skipping .2 used by client)")
		}
	})
}

func TestEdgeExportCommand(t *testing.T) {
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()

	t.Run("export edge config", func(t *testing.T) {
		cfgFile = ""
		clientProtocol = ""
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "awg0.conf")

		serverPriv, _ := amnezigo.GenerateKeyPair()
		edgePriv, edgePub := amnezigo.GenerateKeyPair()

		initialConfig := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
Jc = 3
Jmin = 64
Jmax = 64
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 100-100
H2 = 200-200
H3 = 300-300
H4 = 400-400
#_EndpointV4 = 1.2.3.4:55424

[Peer]
#_Role = edge
#_Name = moscow
#_PrivateKey = %s
PublicKey = %s
PresharedKey = edgepsk123
AllowedIPs = 10.8.0.3/32
#_GenKeyTime = 2024-03-17T12:00:00Z
`, serverPriv, edgePriv, edgePub)
		if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
			t.Fatal(err)
		}

		t.Chdir(tmpDir)

		cmd := NewEdgeExportCommand()
		cmd.SetArgs([]string{"--config", configPath, "moscow"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("edge export failed: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(tmpDir, "moscow.conf"))
		if err != nil {
			t.Fatalf("failed to read edge config: %v", err)
		}
		configStr := string(content)

		if !strings.Contains(configStr, "PrivateKey = "+edgePriv) {
			t.Error("expected edge private key in config")
		}
		if !strings.Contains(configStr, "AllowedIPs = 10.8.0.1/32") {
			t.Error("expected hub IP as AllowedIPs")
		}
		if strings.Contains(configStr, "DNS =") {
			t.Error("edge config should not have DNS")
		}
	})
}
```

Note: add required imports (`"fmt"` and `"github.com/Arsolitt/amnezigo"`) at the top.

- [ ] **Step 2: Implement full edge.go**

Replace `internal/cli/edge.go`:

```go
package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var (
	edgeIPAddr   string
	edgeProtocol string
)

// NewEdgeCommand creates the edge command group.
func NewEdgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Manage edge server peers",
	}

	cmd.AddCommand(NewEdgeAddCommand())
	cmd.AddCommand(NewEdgeListCommand())
	cmd.AddCommand(NewEdgeExportCommand())
	cmd.AddCommand(NewEdgeRemoveCommand())

	return cmd
}

// NewEdgeAddCommand creates the edge add subcommand.
func NewEdgeAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new edge server to the configuration",
		Long: `Add a new edge server to the AmneziaWG hub configuration.

Generates a keypair for the edge and adds it to the server's edge list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo edge add moscow
  amnezigo edge add berlin --ipaddr 10.8.0.50
`,
		Args: cobra.ExactArgs(1),
		RunE: runEdgeAdd,
	}
	cmd.Flags().StringVar(&edgeIPAddr, "ipaddr", "", "Edge IP address (e.g., 10.8.0.5)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	edge, err := mgr.AddEdge(args[0], edgeIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Edge '%s' added successfully\n", edge.Name)
	fmt.Printf("  IP Address: %s\n", edge.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", edge.PublicKey)

	return nil
}

// NewEdgeListCommand creates the edge list subcommand.
func NewEdgeListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured edge servers",
		Long: `List all edge servers configured in the AmneziaWG server configuration.

Displays a table with edge name, IP address, and creation time.

Example:
  amnezigo edge list
`,
		RunE: runEdgeList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	edges := mgr.ListEdges()
	if len(edges) == 0 {
		fmt.Println("No edges configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, edge := range edges {
		timestamp := ""
		if !edge.CreatedAt.IsZero() {
			timestamp = edge.CreatedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\n", edge.Name, edge.AllowedIPs, timestamp)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}

// NewEdgeExportCommand creates the edge export subcommand.
func NewEdgeExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <name>",
		Short: "Export edge server configuration",
		Long: `Export AWG configuration for the specified edge server.

The exported config is a client-style config where the edge connects to the hub.

Example:
  amnezigo edge export moscow
  amnezigo edge export --protocol quic moscow
`,
		Args: cobra.ExactArgs(1),
		RunE: runEdgeExport,
	}
	cmd.Flags().StringVar(&edgeProtocol, "protocol", "random", "Obfuscation protocol")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := resolveExportEndpoint(serverCfg)

	data, err := mgr.ExportEdge(args[0], edgeProtocol, endpoint)
	if err != nil {
		return err
	}

	configPath := args[0] + ".conf"
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write edge config: %w", err)
	}

	fmt.Printf("Exported edge '%s' to %s\n", args[0], configPath)
	return nil
}

// NewEdgeRemoveCommand creates the edge remove subcommand.
func NewEdgeRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an edge server from the configuration",
		Long: `Remove an edge server from the AmneziaWG hub configuration.

Example:
  amnezigo edge remove moscow
`,
		Args: cobra.ExactArgs(1),
		RunE: runEdgeRemove,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runEdgeRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemoveEdge(args[0]); err != nil {
		return err
	}
	fmt.Printf("Edge '%s' removed successfully\n", args[0])
	return nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/cli/...`
Expected: All PASS

- [ ] **Step 4: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 5: Run linter**

Run: `golangci-lint run --fix`
Expected: No errors (fix any auto-fixable issues)

- [ ] **Step 6: Commit**

```bash
git add internal/cli/
git commit -m "feat: add edge CLI command group with add, list, export, remove"
```

---

## Task 8: Update AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update AGENTS.md with new project structure and API**

Update the following sections in `AGENTS.md`:

1. **Project Structure**: Remove `add.go`, `list.go`, `export.go`, `remove.go`. Add `client.go`, `edge.go`.

2. **Library API - Manager**: Add edge methods:
   - `AddEdge(name, ip string) (PeerConfig, error)`
   - `RemoveEdge(name string) error`
   - `FindEdge(name string) (*PeerConfig, error)`
   - `ListEdges() []PeerConfig`
   - `BuildEdgeConfig(name, protocol, endpoint string) (ClientConfig, error)`
   - `ExportEdge(name, protocol, endpoint string) ([]byte, error)`

3. **Known Gotchas**: Add note about `#_Role` being mandatory on all `[Peer]` sections.

- [ ] **Step 2: Commit**

```bash
git add AGENTS.md
git commit -m "docs: update AGENTS.md with edge support and new CLI structure"
```

---

## Task 9: Final verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 2: Run linter**

Run: `golangci-lint run`
Expected: No errors

- [ ] **Step 3: Build binary**

Run: `go build -o build/amnezigo ./cmd/amnezigo/`
Expected: Build success

- [ ] **Step 4: Smoke test CLI help**

Run: `./build/amnezigo --help`
Expected: Shows `client` and `edge` command groups

Run: `./build/amnezigo client --help`
Expected: Shows add, list, export, remove

Run: `./build/amnezigo edge --help`
Expected: Shows add, list, export, remove
