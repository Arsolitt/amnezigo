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
			Jc: 5, Jmin: 100, Jmax: 200, S1: 10, S2: 20, S3: 30, S4: 5,
			H1: HeaderRange{Min: 100, Max: 200},
			H2: HeaderRange{Min: 300, Max: 400},
			H3: HeaderRange{Min: 500, Max: 600},
			H4: HeaderRange{Min: 700, Max: 800},
		},
		Clients: []PeerConfig{},
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
		Clients: []PeerConfig{},
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

	if len(loaded.Clients) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(loaded.Clients))
	}
	if loaded.Clients[0].Name != "testclient" {
		t.Errorf("expected peer name 'testclient', got '%s'", loaded.Clients[0].Name)
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
		Clients: []PeerConfig{},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)
	_, _ = mgr.AddClient("dup", "")

	_, err := mgr.AddClient("dup", "")
	if err == nil {
		t.Fatal("expected error for duplicate client name")
	}
}

func TestManagerAddClientDuplicateEdgeName(t *testing.T) {
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
		Edges: []PeerConfig{
			{Name: "shared", Role: RoleEdge, PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	_, err := mgr.AddClient("shared", "")
	if err == nil {
		t.Fatal("expected error when client name conflicts with existing edge name")
	}
}

func TestManagerAddEdgeDuplicateClientName(t *testing.T) {
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
		Clients: []PeerConfig{
			{Name: "shared", Role: RoleClient, PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	_, err := mgr.AddEdge("shared", "")
	if err == nil {
		t.Fatal("expected error when edge name conflicts with existing client name")
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
		Clients: []PeerConfig{},
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
		Clients: []PeerConfig{
			{Name: "keep", Role: RoleClient, PublicKey: "pub1", AllowedIPs: "10.8.0.2/32"},
			{Name: "remove", Role: RoleClient, PublicKey: "pub2", AllowedIPs: "10.8.0.3/32"},
		},
	}

	mgr := NewManager(path)
	_ = mgr.Save(cfg)

	err := mgr.RemoveClient("remove")
	if err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	loaded, _ := mgr.Load()
	if len(loaded.Clients) != 1 {
		t.Fatalf("expected 1 peer after removal, got %d", len(loaded.Clients))
	}
	if loaded.Clients[0].Name != "keep" {
		t.Errorf("expected remaining peer 'keep', got '%s'", loaded.Clients[0].Name)
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
		Clients: []PeerConfig{},
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
		Clients: []PeerConfig{
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
		Clients: []PeerConfig{
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
		Clients: []PeerConfig{
			{
				Name:         "exportme",
				PrivateKey:   "clientpriv=",
				PublicKey:    "clientpub=",
				PresharedKey: "psk=",
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
		Clients: []PeerConfig{},
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
		Clients: []PeerConfig{},
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
	if len(loaded.Clients) != 0 {
		t.Error("edge should not be in Clients")
	}
}

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
