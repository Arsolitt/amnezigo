# Unify Clients and Edges into Single Peer List

## Problem

The codebase splits peers into `Clients` and `Edges` via `#_Role` metadata, adding complexity (parallel method sets, separate CLI command groups, parser routing) without providing meaningful value in a hub-and-spoke topology. All peers share the same key generation, IP pool, and obfuscation logic.

## Decision

Remove the client/edge distinction entirely. All peers are stored in a single `Peers []PeerConfig` slice. CLI commands become flat (`add`, `list`, `export`, `remove`) without subcommand groups.

Breaking change: existing configs with `#_Role` must be manually edited to remove it.

## Changes

### 1. Types (`types.go`)

- Remove `RoleClient` and `RoleEdge` constants
- Remove `Role string` field from `PeerConfig`
- Replace `Clients []PeerConfig` + `Edges []PeerConfig` with `Peers []PeerConfig` in `ServerConfig`

### 2. Parser (`parser.go`)

- Remove `#_Role` parsing from the `#_` metadata switch
- Remove role-based switch that routes peers to `Clients` vs `Edges`
- All `[Peer]` sections append to `cfg.Peers`
- If `#_Role` is encountered in a config, it is silently ignored (treated as an unknown metadata field)
- The `PublicKey != ""` guard before appending remains

### 3. Writer (`writer.go`)

- Replace two loops (clients, then edges) with single `for _, peer := range cfg.Peers`
- `writePeerSection` signature changes to `(w io.Writer, peer PeerConfig)` — no role parameter
- `writePeerSection` no longer writes `#_Role = ...` line
- `WriteClientConfig`: remove the conditional `DNS != ""` check — DNS is always written

### 4. Manager (`manager.go`)

Delete all edge methods: `AddEdge`, `RemoveEdge`, `FindEdge`, `ListEdges`, `BuildEdgeConfig`, `ExportEdge`.

Rename client methods:

| Old Name | New Name |
|---|---|
| `AddClient` | `AddPeer` |
| `RemoveClient` | `RemovePeer` |
| `FindClient` | `FindPeer` |
| `ListClients` | `ListPeers` |
| `BuildClientConfig` | `BuildPeerConfig` |
| `ExportClient` | `ExportPeer` |

`isNameTaken()` checks uniqueness across `cfg.Peers` only.

`resolveClientIP()` considers only `cfg.Peers` when finding the next available IP.

`BuildPeerConfig` export defaults:
- AllowedIPs: `0.0.0.0/0, ::/0`
- DNS: `1.1.1.1, 8.8.8.8`
- PersistentKeepalive: 25

`ExportPeer` returns `(ClientConfig, error)` — same as current `ExportClient`.

### 5. CLI (`internal/cli/`)

Delete `client.go` and `edge.go`.

Create flat command files: `add.go`, `list.go`, `export.go`, `remove.go`.

Register directly in `cli.go`:
```go
rootCmd.AddCommand(NewAddCommand())
rootCmd.AddCommand(NewListCommand())
rootCmd.AddCommand(NewExportCommand())
rootCmd.AddCommand(NewRemoveCommand())
```

Command behavior:
- `amnezigo add <name> [--ip IP]` — add a peer, auto-assign IP if not specified
- `amnezigo list` — list all peers
- `amnezigo export [name] [--protocol PROTO] [--endpoint ADDR]` — export one or all peers
- `amnezigo remove <name>` — remove a peer

`export` accepts optional `[name]` — exports all peers if no name given. Output file: `<name>.conf`.

Flags remain the same: `--protocol` and `--endpoint` for export, `--ip` for add.

### 6. Tests

Delete:
- `internal/cli/edge_test.go`
- `TestParseSplitsClientsAndEdges`
- `TestParsePeerRejectsMissingRole`
- `TestParsePeerRejectsInvalidRole`
- `TestWriteServerConfigEmitsRole`
- `TestManagerAddClientDuplicateEdgeName`
- `TestManagerAddEdgeDuplicateClientName`
- `TestManagerBuildEdgeConfig`

Update remaining tests:
- `cfg.Clients` → `cfg.Peers`
- `AddClient` → `AddPeer`, `RemoveClient` → `RemovePeer`, etc.
- `BuildClientConfig` → `BuildPeerConfig`

Add new tests for renamed methods: `TestManagerAddPeer`, `TestManagerRemovePeer`, `TestManagerFindPeer`, `TestManagerListPeers`, `TestManagerBuildPeerConfig`, `TestManagerExportPeer`.

CLI tests: `TestAddCommand`, `TestListCommand`, `TestExportCommand`, `TestRemoveCommand`.

### 7. INI Config Format

Before:
```ini
[Peer]
#_Name = laptop
#_Role = client
#_PrivateKey = ...
PublicKey = ...
```

After:
```ini
[Peer]
#_Name = laptop
#_PrivateKey = ...
PublicKey = ...
```

No `#_Role` line. All peers are peers.

## Files Affected

- `types.go` — remove role constants, `Role` field, merge `Clients`/`Edges` into `Peers`
- `parser.go` — remove role parsing and routing
- `writer.go` — single peer loop, no role emission
- `manager.go` — rename methods, delete edge methods
- `manager_test.go` — update and delete tests
- `parser_test.go` — update and delete tests
- `writer_test.go` — update and delete tests
- `internal/cli/cli.go` — register flat commands
- `internal/cli/client.go` — delete
- `internal/cli/edge.go` — delete
- `internal/cli/edge_test.go` — delete
- `internal/cli/add.go` — new (flat add command)
- `internal/cli/list.go` — new (flat list command)
- `internal/cli/export.go` — new (flat export command)
- `internal/cli/remove.go` — new (flat remove command)
- `AGENTS.md` — update API reference and project structure
- `README.md` — update CLI reference and usage examples
