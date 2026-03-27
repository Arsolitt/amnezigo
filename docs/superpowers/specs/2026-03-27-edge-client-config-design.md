# Edge AWG Client Config Generation

Date: 2026-03-27

## Goal

Enable amnezigo to generate client-style AWG server configs for edge servers that connect to a hub's AWG tunnel. Add CLI command groups for managing clients and edges separately.

## Background

Currently `ServerConfig` and `BuildClientConfig` represent two distinct concepts:
- `ServerConfig`: AWG server with `[Interface]` + `[Peer]` sections. Used for hubs with multiple clients.
- `ClientConfig` (from `BuildClientConfig`): Client-side config for end-user devices (laptop, phone). Used for export.

The gap: an edge server needs an AWG config file that connects to the hub as a peer (initiates the connection), not serves local clients. Additionally, the CLI needs separate command groups for client and edge management.

## Architecture

- `PeerConfig` gains a `Role` field to distinguish clients from edges.
- `ServerConfig` splits `Peers` into `Clients` and `Edges`.
- Edge export reuses `ClientConfig` type — edge is an AWG client from the tunnel perspective.
- CLI restructured into `client` and `edge` command groups.
- Parser enforces `#_Role` on every `[Peer]` section.

## Types

### PeerConfig

Add `Role` field:

```go
type PeerConfig struct {
    CreatedAt         time.Time
    ClientObfuscation *ClientObfuscationConfig
    Name              string
    Role              string  // "client" or "edge"
    PrivateKey        string
    PublicKey         string
    PresharedKey      string
    AllowedIPs        string
}
```

### ServerConfig

Rename `Peers` to `Clients`, add `Edges`:

```go
type ServerConfig struct {
    Clients     []PeerConfig
    Edges       []PeerConfig
    Interface   InterfaceConfig
    Obfuscation ServerObfuscationConfig
}
```

No new types are introduced. `ClientConfig` and `ClientInterfaceConfig` remain unchanged.

## INI Format

All peers (clients and edges) use `[Peer]` sections with a mandatory `#_Role` field:

```ini
[Peer]
#_Name = user1
#_Role = client
#_PrivateKey = <priv>
PublicKey = <pub>
PresharedKey = <psk>
AllowedIPs = 10.110.2.2/32
#_GenKeyTime = 2026-03-27T...

[Peer]
#_Name = moscow-edge
#_Role = edge
#_PrivateKey = <priv>
PublicKey = <pub>
PresharedKey = <psk>
AllowedIPs = 10.110.2.3/32
#_GenKeyTime = 2026-03-27T...
```

## Parser (`parser.go`)

- Every `[Peer]` section must have `#_Role`.
- If `#_Role` is missing or not `"client"`/`"edge"` — return parse error.
- `"client"` → append to `ServerConfig.Clients`.
- `"edge"` → append to `ServerConfig.Edges`.

## Writer (`writer.go`)

- `WriteServerConfig` writes all `Clients` as `[Peer]` with `#_Role = client`.
- `WriteServerConfig` writes all `Edges` as `[Peer]` with `#_Role = edge`.
- `WriteClientConfig` — no changes. Used for both client and edge exports.

## Library API (manager.go)

### New methods

- `AddEdge(name, ip string) (PeerConfig, error)` — generates keys, PSK, assigns IP, appends to `Edges` with `Role: "edge"`. Saves config.
- `BuildEdgeConfig(name, protocol, endpoint string) (ClientConfig, error)` — builds `ClientConfig` for edge deployment:
  - Interface: edge private key, edge IP/32, empty DNS, MTU + obfuscation (server params + generated I1-I5).
  - Peer: hub public key, PSK, endpoint, `AllowedIPs = hub_ip/32`, PersistentKeepalive = 25.
- `ExportEdge(name, protocol, endpoint string) ([]byte, error)` — serializes `BuildEdgeConfig` via `WriteClientConfig`.
- `RemoveEdge(name string) error`
- `FindEdge(name string) (*PeerConfig, error)`
- `ListEdges() []PeerConfig`

### Updated methods

- `AddClient` — sets `Role: "client"` explicitly on the new peer.
- All existing `Client*` methods (`AddClient`, `RemoveClient`, `FindClient`, `ListClients`, `ExportClient`, `BuildClientConfig`) — operate on `Clients` instead of `Peers`.

## CLI

Two command groups replace current flat commands:

### `amnezigo client`

- `client add <name> [--ip <ip>]` — add client to hub config.
- `client list` — list all clients.
- `client export <name> --protocol <proto> [--endpoint <endpoint>]` — generate client AWG config.
- `client remove <name>` — remove client from hub config.

### `amnezigo edge`

- `edge add <name> [--ip <ip>]` — add edge to hub config.
- `edge list` — list all edges.
- `edge export <name> --protocol <proto> [--endpoint <endpoint>]` — generate edge AWG config.
- `edge remove <name>` — remove edge from hub config.

### File structure

- `internal/cli/client.go` — all client subcommands (`NewClientCommand()`, `NewClientAddCommand()`, etc.)
- `internal/cli/edge.go` — all edge subcommands (`NewEdgeCommand()`, `NewEdgeAddCommand()`, etc.)
- `internal/cli/cli.go` — root command, registers `client` and `edge` groups.

Current files `add.go`, `list.go`, `export.go`, `remove.go` are replaced by `client.go`. `init.go` remains as-is. `edit.go` is out of scope for this spec.

## Edge Config Details

### AllowedIPs

Edge peer's `AllowedIPs` on the edge-side config = hub's AWG IP only (e.g., `10.110.2.1/32`). The edge is a traffic endpoint, not a router. User traffic arrives via VLESS+Reality on the public interface or via SOCKS5 on the private AWG interface — both originate from the hub side.

### Obfuscation

Edge uses the same server-side obfuscation parameters (Jc, Jmin, Jmax, S1-S4, H1-H4) as the hub. I1-I5 CPS strings are generated per-edge at export time, same as for clients.

### No iptables

Edge does not need PostUp/PostDown iptables rules. It does not do NAT or forwarding. The edge is a traffic endpoint.

## Testing

- Unit tests for all new Manager methods (`AddEdge`, `RemoveEdge`, `FindEdge`, `ListEdges`, `BuildEdgeConfig`, `ExportEdge`).
- Parser tests: valid client/edge roles, missing role (error), invalid role (error).
- Writer tests: verify `#_Role` is written for each peer.
- CLI tests: smoke tests for `edge add`, `edge list`, `edge export`, `edge remove`.
- Update existing tests that reference `Peers` → `Clients`.
