# Unify Clients and Edges into Single Peer List

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the client/edge distinction from the entire codebase, replacing it with a single unified peer list for hub-and-spoke topology.

**Architecture:** All peers stored in `ServerConfig.Peers []PeerConfig`. No `#_Role` in config files. Flat CLI commands (`add`, `list`, `export`, `remove`). Export always uses full tunnel (`0.0.0.0/0, ::/0`) with DNS `1.1.1.1, 8.8.8.8`.

**Tech Stack:** Go, Cobra CLI, INI config format

---

### Task 1: Update types.go

**Files:**
- Modify: `types.go`

- [ ] **Step 1: Remove role constants and Role field, merge Clients/Edges into Peers**

Replace lines 5-8 (role constants) and lines 16-21 (ServerConfig) and line 44 (Role field in PeerConfig):

```go
// ServerConfig represents the full WireGuard server configuration.
type ServerConfig struct {
	Peers       []PeerConfig
	Interface   InterfaceConfig
	Obfuscation ServerObfuscationConfig
}
```

In `PeerConfig`, remove the `Role string` field (line 44). The struct becomes:

```go
type PeerConfig struct {
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
	Name              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
}
```

- [ ] **Step 2: Verify build fails** (expected — all downstream code references removed symbols)

Run: `go build ./...`
Expected: compilation errors referencing `RoleClient`, `RoleEdge`, `Role`, `Clients`, `Edges`

- [ ] **Step 3: Commit**

```bash
git add types.go
git commit -m "refactor: remove Role field and merge Clients/Edges into Peers"
```

---

### Task 2: Update parser.go

**Files:**
- Modify: `parser.go`

- [ ] **Step 1: Remove role-based routing in ParseServerConfig**

Replace the mid-stream peer finalization block (lines 35-48). Currently:

```go
if currentSection == sectionPeer && currentPeer.PublicKey != "" {
    switch currentPeer.Role {
    case RoleClient:
        cfg.Clients = append(cfg.Clients, currentPeer)
    case RoleEdge:
        cfg.Edges = append(cfg.Edges, currentPeer)
    default:
        return ServerConfig{}, fmt.Errorf(...)
    }
    currentPeer = PeerConfig{}
}
```

Replace with:

```go
if currentSection == sectionPeer && currentPeer.PublicKey != "" {
    cfg.Peers = append(cfg.Peers, currentPeer)
    currentPeer = PeerConfig{}
}
```

- [ ] **Step 2: Remove `case "Role":` from peer metadata switch (lines 71-72)**

Remove these two lines from the `switch fieldName` block inside `case sectionPeer`:

```go
case "Role":
    currentPeer.Role = value
```

- [ ] **Step 3: Replace end-of-file peer finalization (lines 180-192)**

Currently:

```go
if currentSection == sectionPeer && currentPeer.PublicKey != "" {
    switch currentPeer.Role {
    case RoleClient:
        cfg.Clients = append(cfg.Clients, currentPeer)
    case RoleEdge:
        cfg.Edges = append(cfg.Edges, currentPeer)
    default:
        return ServerConfig{}, fmt.Errorf(...)
    }
}
```

Replace with:

```go
if currentSection == sectionPeer && currentPeer.PublicKey != "" {
    cfg.Peers = append(cfg.Peers, currentPeer)
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: compilation errors in writer.go, manager.go, tests (not in parser.go)

- [ ] **Step 5: Commit**

```bash
git add parser.go
git commit -m "refactor: remove role-based routing from parser"
```

---

### Task 3: Update writer.go

**Files:**
- Modify: `writer.go`

- [ ] **Step 1: Replace two peer loops with single loop (lines 55-61)**

Currently:

```go
for _, peer := range cfg.Clients {
    writePeerSection(w, peer, RoleClient)
}

for _, peer := range cfg.Edges {
    writePeerSection(w, peer, RoleEdge)
}
```

Replace with:

```go
for _, peer := range cfg.Peers {
    writePeerSection(w, peer)
}
```

- [ ] **Step 2: Update writePeerSection signature and remove role emission (lines 66-85)**

Replace the entire function with:

```go
func writePeerSection(w io.Writer, peer PeerConfig) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "[Peer]")
	if peer.Name != "" {
		fmt.Fprintf(w, "#_Name = %s\n", peer.Name)
	}
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

- [ ] **Step 3: Remove DNS conditional in WriteClientConfig (lines 92-94)**

Replace:

```go
if cfg.Interface.DNS != "" {
    fmt.Fprintf(w, "DNS = %s\n", cfg.Interface.DNS)
}
```

With:

```go
fmt.Fprintf(w, "DNS = %s\n", cfg.Interface.DNS)
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: compilation errors in manager.go, CLI, tests (not in writer.go)

- [ ] **Step 5: Commit**

```bash
git add writer.go
git commit -m "refactor: unify peer writing and remove role emission"
```

---

### Task 4: Update manager.go

**Files:**
- Modify: `manager.go`

- [ ] **Step 1: Simplify isNameTaken (lines 36-48)**

Replace:

```go
func isNameTaken(name string, cfg ServerConfig) bool {
	for _, peer := range cfg.Clients {
		if peer.Name == name {
			return true
		}
	}
	for _, edge := range cfg.Edges {
		if edge.Name == name {
			return true
		}
	}
	return false
}
```

With:

```go
func isNameTaken(name string, cfg ServerConfig) bool {
	for _, peer := range cfg.Peers {
		if peer.Name == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Rename AddClient to AddPeer and update body (lines 50-88)**

Replace the entire `AddClient` method with:

```go
func (m *Manager) AddPeer(name, ip string) (PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return PeerConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	if isNameTaken(name, serverCfg) {
		return PeerConfig{}, fmt.Errorf("peer with name '%s' already exists", name)
	}

	peerIP, err := m.resolvePeerIP(ip, serverCfg)
	if err != nil {
		return PeerConfig{}, err
	}

	privateKey, publicKey := GenerateKeyPair()
	psk := GeneratePSK()

	newPeer := PeerConfig{
		Name:         name,
		PrivateKey:   privateKey,
		PublicKey:    publicKey,
		PresharedKey: psk,
		AllowedIPs:   peerIP + "/32",
		CreatedAt:    time.Now(),
	}

	serverCfg.Peers = append(serverCfg.Peers, newPeer)

	if err := m.Save(serverCfg); err != nil {
		return PeerConfig{}, fmt.Errorf("failed to save server config: %w", err)
	}

	return newPeer, nil
}
```

- [ ] **Step 3: Rename resolveClientIP to resolvePeerIP and simplify (lines 90-126)**

Replace the entire `resolveClientIP` method with:

```go
func (m *Manager) resolvePeerIP(ip string, serverCfg ServerConfig) (string, error) {
	if ip != "" {
		return ip, nil
	}

	_, ipnet, err := net.ParseCIDR(serverCfg.Interface.Address)
	if err != nil {
		return "", fmt.Errorf("invalid server address: %w", err)
	}

	existingIPs := make([]string, 0, len(serverCfg.Peers))
	for _, peer := range serverCfg.Peers {
		if before, ok := strings.CutSuffix(peer.AllowedIPs, "/32"); ok {
			peerIP := net.ParseIP(before)
			if peerIP != nil && ipnet.Contains(peerIP) {
				existingIPs = append(existingIPs, before)
			}
		}
	}

	peerIP, err := FindNextAvailableIP(serverCfg.Interface.Address, existingIPs)
	if err != nil {
		return "", fmt.Errorf("failed to assign IP address: %w", err)
	}

	return peerIP, nil
}
```

- [ ] **Step 4: Rename RemoveClient to RemovePeer (lines 128-154)**

Replace the entire method with:

```go
func (m *Manager) RemovePeer(name string) error {
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
		return fmt.Errorf("peer '%s' not found", name)
	}

	serverCfg.Peers = append(serverCfg.Peers[:peerIndex], serverCfg.Peers[peerIndex+1:]...)

	if err := m.Save(serverCfg); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	return nil
}
```

- [ ] **Step 5: Rename FindClient to FindPeer (lines 156-170)**

Replace the entire method with:

```go
func (m *Manager) FindPeer(name string) (*PeerConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load server config: %w", err)
	}

	for i := range serverCfg.Peers {
		if serverCfg.Peers[i].Name == name {
			return &serverCfg.Peers[i], nil
		}
	}

	return nil, fmt.Errorf("peer '%s' not found", name)
}
```

- [ ] **Step 6: Rename ListClients to ListPeers (lines 172-179)**

Replace the entire method with:

```go
func (m *Manager) ListPeers() []PeerConfig {
	serverCfg, err := m.Load()
	if err != nil {
		return nil
	}
	return serverCfg.Peers
}
```

- [ ] **Step 7: Rename ExportClient to ExportPeer (lines 181-203)**

Replace the entire method with:

```go
func (m *Manager) ExportPeer(name, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	var peer PeerConfig
	found := false
	for _, p := range serverCfg.Peers {
		if p.Name == name {
			peer = p
			found = true
			break
		}
	}
	if !found {
		return ClientConfig{}, fmt.Errorf("peer '%s' not found", name)
	}

	return m.BuildPeerConfig(peer, protocol, endpoint)
}
```

- [ ] **Step 8: Rename BuildClientConfig to BuildPeerConfig (lines 205-254)**

Replace the entire method with:

```go
func (m *Manager) BuildPeerConfig(peer PeerConfig, protocol, endpoint string) (ClientConfig, error) {
	serverCfg, err := m.Load()
	if err != nil {
		return ClientConfig{}, fmt.Errorf("failed to load server config: %w", err)
	}

	peerIP := strings.TrimSuffix(peer.AllowedIPs, "/32")
	allowedIPs := "0.0.0.0/0, ::/0"

	serverPublicKey := serverCfg.Interface.PublicKey
	if serverPublicKey == "" {
		serverPublicKey = DerivePublicKey(serverCfg.Interface.PrivateKey)
	}

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

	peerConfig := ClientConfig{
		Interface: ClientInterfaceConfig{
			PrivateKey:  peer.PrivateKey,
			Address:     peerIP + "/32",
			DNS:         "1.1.1.1, 8.8.8.8",
			MTU:         serverCfg.Interface.MTU,
			Obfuscation: obfConfig,
		},
		Peer: ClientPeerConfig{
			PublicKey:           serverPublicKey,
			PresharedKey:        peer.PresharedKey,
			Endpoint:            endpoint,
			AllowedIPs:          allowedIPs,
			PersistentKeepalive: defaultPersistentKeepalive,
		},
	}

	return peerConfig, nil
}
```

- [ ] **Step 9: Delete all edge methods and extractHubIP (lines 286-467)**

Delete everything from line 286 (`// AddEdge`) through line 467 (end of `ExportEdge`). This removes:
- `AddEdge`
- `RemoveEdge`
- `FindEdge`
- `ListEdges`
- `extractHubIP`
- `BuildEdgeConfig`
- `ExportEdge`

- [ ] **Step 10: Remove unused `bytes` import**

The `bytes` package was only used by `ExportEdge`. Remove `"bytes"` from the import block (line 3).

- [ ] **Step 11: Verify build**

Run: `go build ./...`
Expected: compilation errors only in CLI files and tests

- [ ] **Step 12: Commit**

```bash
git add manager.go
git commit -m "refactor: rename client methods to peer, delete all edge methods"
```

---

### Task 5: Update CLI — delete old files, create flat commands

**Files:**
- Delete: `internal/cli/client.go`
- Delete: `internal/cli/edge.go`
- Modify: `internal/cli/cli.go`
- Create: `internal/cli/add.go`
- Create: `internal/cli/list.go`
- Create: `internal/cli/export.go`
- Create: `internal/cli/remove.go`
- Modify: `internal/cli/init.go`

- [ ] **Step 1: Delete client.go and edge.go**

```bash
rm internal/cli/client.go internal/cli/edge.go
```

- [ ] **Step 2: Update cli.go — replace client/edge commands with flat commands**

Replace the full content of `internal/cli/cli.go` with:

```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	tabPadding     = 3
	separatorWidth = 76
)

var (
	cfgFile string
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "amnezigo",
		Short: "AmneziaWG v2.0 Configuration Generator for star topology",
		Long:  `Generate AmneziaWG v2.0 configurations for star topology networks.`,
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(NewAddCommand())
	rootCmd.AddCommand(NewListCommand())
	rootCmd.AddCommand(NewExportCommand())
	rootCmd.AddCommand(NewRemoveCommand())

	return rootCmd
}

func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Create internal/cli/add.go**

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var peerIPAddr string

func NewAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new peer to the server configuration",
		Long: `Add a new WireGuard peer to the AmneziaWG server configuration.

Generates a keypair for the peer and adds it to the server's peer list.
IP address can be auto-assigned or manually specified.

Example:
  amnezigo add laptop
  amnezigo add phone --ipaddr 10.8.0.50
`,
		Args: cobra.ExactArgs(1),
		RunE: runAdd,
	}
	cmd.Flags().StringVar(&peerIPAddr, "ipaddr", "", "Peer IP address (e.g., 10.8.0.5)")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runAdd(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peer, err := mgr.AddPeer(args[0], peerIPAddr)
	if err != nil {
		return err
	}

	fmt.Printf("Peer '%s' added successfully\n", peer.Name)
	fmt.Printf("  IP Address: %s\n", peer.AllowedIPs)
	fmt.Printf("  Public Key: %s\n", peer.PublicKey)

	return nil
}
```

- [ ] **Step 4: Create internal/cli/list.go**

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

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured peers",
		Long: `List all WireGuard peers configured in the AmneziaWG server configuration.

Displays a table with peer name, IP address, and creation time.

Example:
  amnezigo list
`,
		RunE: runList,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runList(_ *cobra.Command, _ []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	peers := mgr.ListPeers()
	if len(peers) == 0 {
		fmt.Println("No peers configured")
		return nil
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(writer, "NAME\tIP\tCREATED")
	fmt.Fprintln(writer, strings.Repeat("-", separatorWidth))

	for _, peer := range peers {
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
```

- [ ] **Step 5: Create internal/cli/export.go**

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

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

var peerProtocol string

func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export peer configuration(s)",
		Long: `Export WireGuard peer configuration(s).

If a name is specified, exports only that peer's configuration.
If no name is specified, exports all peers' configurations.

Example:
  amnezigo export laptop
  amnezigo export --protocol quic laptop
  amnezigo export
`,
		Args: cobra.MaximumNArgs(1),
		RunE: runExport,
	}
	cmd.Flags().StringVar(&peerProtocol, "protocol", "random", "Obfuscation protocol")
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runExport(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	serverCfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load server config: %w", err)
	}

	endpoint := resolveExportEndpoint(serverCfg)

	peersToExport, err := selectPeersToExport(serverCfg.Peers, args)
	if err != nil {
		return err
	}

	return writePeerConfigs(mgr, peersToExport, endpoint)
}

func selectPeersToExport(peers []amnezigo.PeerConfig, args []string) ([]amnezigo.PeerConfig, error) {
	if len(args) == 0 {
		return peers, nil
	}
	peerName := args[0]
	for _, peer := range peers {
		if peer.Name == peerName {
			return []amnezigo.PeerConfig{peer}, nil
		}
	}
	return nil, fmt.Errorf("peer '%s' not found", peerName)
}

func writePeerConfigs(mgr *amnezigo.Manager, peers []amnezigo.PeerConfig, endpoint string) error {
	for _, peer := range peers {
		peerCfg, err := mgr.BuildPeerConfig(peer, peerProtocol, endpoint)
		if err != nil {
			return fmt.Errorf("failed to export peer '%s': %w", peer.Name, err)
		}

		var buf bytes.Buffer
		if err := amnezigo.WriteClientConfig(&buf, peerCfg); err != nil {
			return fmt.Errorf("failed to write peer config: %w", err)
		}

		configPath := peer.Name + ".conf"
		if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Exported peer '%s' to %s\n", peer.Name, configPath)
	}
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

- [ ] **Step 6: Create internal/cli/remove.go**

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Arsolitt/amnezigo"
)

func NewRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a peer from the server configuration",
		Long: `Remove a WireGuard peer from the AmneziaWG server configuration.

Removes the peer with the specified name from the server's peer list.

Example:
  amnezigo remove laptop
`,
		Args: cobra.ExactArgs(1),
		RunE: runRemove,
	}
	cmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "config file path")
	return cmd
}

func runRemove(_ *cobra.Command, args []string) error {
	mgr := amnezigo.NewManager(cfgFile)
	if err := mgr.RemovePeer(args[0]); err != nil {
		return err
	}
	fmt.Printf("Peer '%s' removed successfully\n", args[0])
	return nil
}
```

- [ ] **Step 7: Update init.go — change Clients to Peers (line 143)**

Replace:

```go
Clients:     []amnezigo.PeerConfig{},
```

With:

```go
Peers:       []amnezigo.PeerConfig{},
```

- [ ] **Step 8: Verify build**

Run: `go build ./...`
Expected: compilation errors only in test files

- [ ] **Step 9: Commit**

```bash
git add internal/cli/
git commit -m "refactor: replace client/edge CLI commands with flat peer commands"
```

---

### Task 6: Update root package tests

**Files:**
- Modify: `parser_test.go`
- Modify: `writer_test.go`
- Modify: `manager_test.go`

- [ ] **Step 1: Update parser_test.go**

Changes needed:
1. All test inputs: remove `#_Role = client` lines from peer sections
2. All assertions: `cfg.Clients` → `cfg.Peers`

In `TestParseServerConfig` (line 18): remove `#_Role = client`. Lines 45-56: `cfg.Clients` → `cfg.Peers`.

In `TestParseMultiplePeers` (lines 109, 117): remove `#_Role = client`. Lines 130-150: `cfg.Clients` → `cfg.Peers`.

In `TestParsePostScripts` (line 165): remove `#_Role = client`.

In `TestParsePeerPresharedKey` (line 196): remove `#_Role = client`. Lines 208-219: `cfg.Clients` → `cfg.Peers`.

Delete entire test functions:
- `TestParsePeerRejectsMissingRole` (lines 222-242)
- `TestParsePeerRejectsInvalidRole` (lines 244-261)
- `TestParseSplitsClientsAndEdges` (lines 263-310)

- [ ] **Step 2: Update writer_test.go**

Changes needed:
1. All `Clients:` in struct literals → `Peers:`
2. Remove `Role: RoleClient` from all PeerConfig literals
3. Delete `TestWriteServerConfigEmitsRole` (lines 289-318)

In `TestWriteServerConfig` (line 23): `Clients:` → `Peers:`, remove `Role: RoleClient`.
In `TestWriteServerConfigWithPresharedKey` (line 139): `Clients:` → `Peers:`, remove `Role: RoleClient`.
`TestWriteClientConfig` — no changes needed (no role references).

- [ ] **Step 3: Update manager_test.go**

Rename test functions and update all references:

| Old | New |
|---|---|
| `TestManagerAddClient` | `TestManagerAddPeer` |
| `TestManagerAddClientDuplicate` | `TestManagerAddPeerDuplicate` |
| `TestManagerAddClientWithIP` | `TestManagerAddPeerWithIP` |
| `TestManagerRemoveClient` | `TestManagerRemovePeer` |
| `TestManagerRemoveClientNotFound` | `TestManagerRemovePeerNotFound` |
| `TestManagerFindClient` | `TestManagerFindPeer` |
| `TestManagerListClients` | `TestManagerListPeers` |
| `TestManagerExportClient` | `TestManagerExportPeer` |

In all test functions:
- `Clients:` → `Peers:` in ServerConfig literals
- Remove `Role: RoleClient` and `Role: RoleEdge` from PeerConfig literals
- `mgr.AddClient(` → `mgr.AddPeer(`
- `mgr.RemoveClient(` → `mgr.RemovePeer(`
- `mgr.FindClient(` → `mgr.FindPeer(`
- `mgr.ListClients()` → `mgr.ListPeers()`
- `mgr.ExportClient(` → `mgr.ExportPeer(`
- `loaded.Clients` → `loaded.Peers`
- Error messages: `"client"` → `"peer"` where they reference the renamed error strings

Delete entire test functions:
- `TestManagerAddClientDuplicateEdgeName` (lines 149-177)
- `TestManagerAddEdgeDuplicateClientName` (lines 179-207)
- `TestManagerAddEdge` (lines 492-536)
- `TestManagerRemoveEdge` (lines 538-569)
- `TestManagerFindEdge` (lines 571-600)
- `TestManagerListEdges` (lines 602-629)
- `TestManagerBuildEdgeConfig` (lines 631-678)

In `TestManagerLoadSave` (line 35): `Clients:` → `Peers:`.
In `TestLoadServerConfig` (line 437): `Clients:` → `Peers:`.
In `TestSaveServerConfig` (line 470): `Clients:` → `Peers:`.

- [ ] **Step 4: Run root package tests**

Run: `go test -v -count=1 .`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add parser_test.go writer_test.go manager_test.go
git commit -m "refactor: update root package tests for unified peer model"
```

---

### Task 7: Update CLI tests

**Files:**
- Delete: `internal/cli/edge_test.go`
- Modify: `internal/cli/add_test.go`
- Modify: `internal/cli/list_test.go`
- Modify: `internal/cli/export_test.go`
- Modify: `internal/cli/remove_test.go`

- [ ] **Step 1: Delete edge_test.go**

```bash
rm internal/cli/edge_test.go
```

- [ ] **Step 2: Update add_test.go**

Changes needed:
1. Remove `#_Role = client` from all test config strings (lines 174-175, etc.)
2. Replace `NewClientAddCommand()` → `NewAddCommand()` (lines 56, 125, 183, 229, 248, 300, 318, 427, 509)
3. Replace `clientIPAddr` → `peerIPAddr` (lines 22, 95, 146, 199, 269, 394, 479)
4. Remove unused `"time"` import (line 8) if not used elsewhere — it IS used in `TestCreatedAtTimestamp`, so keep it

- [ ] **Step 3: Update list_test.go**

Changes needed:
1. Remove `#_Role = client` from all test config strings (lines 50, 58, 218)
2. Replace `NewClientListCommand()` → `NewListCommand()` (lines 74, 163, 234, 314)
3. Replace `"No clients configured"` → `"No peers configured"` (line 178)

- [ ] **Step 4: Update export_test.go**

Changes needed:
1. Remove `#_Role = client` from all test config strings (lines 52, 177, 186, 257, 314, 421, 497, 564)
2. Replace `NewClientExportCommand()` → `NewExportCommand()` (lines 67, 201, 269, 329, 382, 437, 512, 579)
3. Replace `clientProtocol` → `peerProtocol` (lines are not reset directly but the variable name changed)

Note: the `"fmt"` import on line 4 and `"net/http"` / `"net/http/httptest"` on lines 5-6 are still needed.

- [ ] **Step 5: Update remove_test.go**

Changes needed:
1. Remove `#_Role = client` from all test config strings (lines 42, 49, 123)
2. Replace `runClientRemove` → `runRemove` (lines 64, 135)

- [ ] **Step 6: Run all CLI tests**

Run: `go test -v -count=1 ./internal/cli/`
Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git rm internal/cli/edge_test.go
git add internal/cli/add_test.go internal/cli/list_test.go internal/cli/export_test.go internal/cli/remove_test.go
git commit -m "refactor: update CLI tests for unified peer model"
```

---

### Task 8: Update documentation

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md`

- [ ] **Step 1: Update AGENTS.md**

Replace all references to the client/edge distinction:
- Project structure: remove `client.go` and `edge.go`, add `add.go`, `list.go`, `export.go`, `remove.go`
- Manager API: rename all client/edge methods to peer methods, delete edge methods
- Known Gotchas: remove `#_Role is mandatory` entry and `random protocol` edge reference if present
- Config I/O: no role-related mentions
- CLI: flat commands instead of groups

- [ ] **Step 2: Update README.md**

Replace all CLI examples from `amnezigo client add` / `amnezigo edge add` to `amnezigo add`. Remove edge-specific documentation. Update any references to `#_Role`.

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md README.md
git commit -m "docs: update documentation for unified peer model"
```

---

### Task 9: Final verification

- [ ] **Step 1: Run linter**

Run: `golangci-lint run --fix`
Expected: no errors

- [ ] **Step 2: Run all tests**

Run: `go test -v -count=1 ./...`
Expected: all tests pass

- [ ] **Step 3: Build binary**

Run: `go build -o build/amnezigo ./cmd/amnezigo/`
Expected: builds successfully

- [ ] **Step 4: Commit design doc**

```bash
git add docs/superpowers/specs/2026-03-29-unify-peers-design.md
git commit -m "docs: add unify-peers design spec"
```
