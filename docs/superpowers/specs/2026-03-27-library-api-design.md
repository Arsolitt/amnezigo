# Library API Design

## Goal

Make amnezigo usable as a Go library, not just a CLI tool. External projects should be able to parse configs, create new ones, add/remove clients, generate keys, and build client configs.

## Current State

All business logic lives under `internal/`, which is not importable by external Go projects. Significant logic (file I/O, IP allocation, client config building, endpoint detection) is embedded in CLI command handlers.

## Approach

Hybrid API: low-level package functions for fine-grained control, plus a high-level `Manager` type for common workflows. All code in a single root `package amnezigo` for flat, ergonomic API (`amnezigo.GenerateKeyPair()`).

## Package Structure

```
amnezigo/
├── cmd/amnezigo/main.go        # CLI entry point (package main)
├── internal/cli/                # CLI commands (package cli, thin wrappers)
├── types.go                     # ServerConfig, PeerConfig, ClientConfig, etc.
├── parser.go                    # ParseServerConfig(io.Reader)
├── writer.go                    # WriteServerConfig, WriteClientConfig
├── manager.go                   # Manager type (Load/Save/AddClient/RemoveClient/Export)
├── keys.go                      # GenerateKeyPair, DerivePublicKey, GeneratePSK
├── generator.go                 # GenerateHeaders, GenerateSPrefixes, etc.
├── cps.go                       # GenerateCPS, generateCPSConfig
├── protocols.go                 # Protocol registry
├── dns.go / quic.go / dtls.go / stun.go  # Protocol templates
├── iptables.go                  # GeneratePostUp, GeneratePostDown
├── helpers.go                   # FindNextAvailableIP, ExtractSubnet
└── *_test.go                    # All tests
```

All files in root `package amnezigo`. External import:

```go
import "github.com/Arsolitt/amnezigo"
// amnezigo.GenerateKeyPair()
// amnezigo.ParseServerConfig(r)
// amnezigo.Manager{ConfigPath: "awg0.conf"}
```

## Key Changes from Current Code

1. Move `config/`, `crypto/`, `obfuscation/`, `network/` from `internal/` to root package
2. Move `main.go` to `cmd/amnezigo/main.go` (Go binary convention)
3. `internal/cli/` stays internal — CLI implementation, not library API
4. Remove duplicate `obfuscation.HeaderRange` — use single `HeaderRange` type
5. Extract business logic from CLI into public functions (see below)

## High-level API: Manager

```go
type Manager struct {
    ConfigPath string
}

func NewManager(configPath string) *Manager
```

Methods:

- `Load() (ServerConfig, error)` — read and parse config from disk
- `Save(cfg ServerConfig) error)` — atomic write via `.tmp` + rename
- `AddClient(name, ip string) (PeerConfig, error)` — generate keys, PSK, assign IP (auto if empty), add to config, save. Returns created peer.
- `RemoveClient(name string) error` — remove peer by name, save
- `FindClient(name string) (*PeerConfig, error)` — find peer by name (read-only)
- `ListClients() []PeerConfig` — return all clients
- `ExportClient(name, protocol, endpoint string) (ClientConfig, error)` — build client config with obfuscation. Returns `ClientConfig` (does not write to disk).
- `BuildClientConfig(peer PeerConfig, protocol string) (ClientConfig, error)` — lower-level version without endpoint detection

`Manager` is not thread-safe. Callers handle concurrency if needed.

## Low-level API

### Parsing and Writing

- `ParseServerConfig(r io.Reader) (ServerConfig, error)`
- `WriteServerConfig(w io.Writer, cfg ServerConfig) error`
- `WriteClientConfig(w io.Writer, cfg ClientConfig) error`
- `LoadServerConfig(path string) (ServerConfig, error)` — convenience file wrapper
- `SaveServerConfig(path string, cfg ServerConfig) error` — atomic file write

### Cryptography

- `GenerateKeyPair() (privateKey, publicKey string)`
- `DerivePublicKey(privateKey string) string`
- `GeneratePSK() string`

### Obfuscation

- `GenerateHeaders() Headers`
- `GenerateSPrefixes() SPrefixes`
- `GenerateJunkParams() JunkParams`
- `GenerateHeaderRanges() [4]HeaderRange`
- `GenerateServerConfig(mtu, s1, jc int) ServerObfuscationConfig`
- `GenerateClientConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig`
- `GenerateCPS(protocol string, mtu, s1, jc int) (i1, i2, i3, i4, i5 string)`

### Network

- `GeneratePostUp(tunName, mainIface, subnet string, clientToClient bool) string`
- `GeneratePostDown(tunName, mainIface, subnet string, clientToClient bool) string`

### Utilities

- `FindNextAvailableIP(serverAddress string, existingIPs []string) (string, error)`
- `ExtractSubnet(ipaddr string) string`
- `GenerateRandomPort() (int, error)`
- `DetectMainInterface() string`
- `IsValidIPAddr(ipaddr string) bool`

## CLI Migration

CLI commands in `internal/cli/` become thin wrappers:

1. Parse flags
2. Create `amnezigo.NewManager(configPath)`
3. Call Manager method
4. Format output to stdout

Functions extracted from CLI to library: `loadServerConfig` → `Manager.Load`, `saveServerConfig` → `Manager.Save`, `findNextAvailableIP` → `FindNextAvailableIP`, `extractSubnet` → `ExtractSubnet`, `exportClient` → `Manager.ExportClient`, `detectMainInterface` → `DetectMainInterface`, `generateRandomPort` → `GenerateRandomPort`, `isValidIPAddr` → `IsValidIPAddr`.

Functions that stay in CLI only: `getExternalIP`, `getEndpointV4`, `getEndpointV6`, `saveMainConfigPath` — these are CLI-specific (HTTP calls to external services, writing `.main.config` file).

### init command

Uses library functions (`GenerateKeyPair`, `ExtractSubnet`, `GeneratePostUp`, `GenerateServerConfig`, `DetectMainInterface`, `GenerateRandomPort`, `SaveServerConfig`) but HTTP endpoint detection remains in CLI.

### edit command

Calls `mgr.Load()`, modifies fields, calls `mgr.Save()`. Uses `GeneratePostUp/Down` for iptables regeneration.

## Types

All types move to root package. Single `HeaderRange` (duplicate from obfuscation package removed):

- `HeaderRange{Min, Max uint32}`
- `ServerConfig{Peers, Interface, Obfuscation}`
- `InterfaceConfig{PrivateKey, PublicKey, Address, PostUp, PostDown, MainIface, TunName, EndpointV4, EndpointV6, ListenPort, MTU, ClientToClient}`
- `PeerConfig{CreatedAt, ClientObfuscation, Name, PrivateKey, PublicKey, PresharedKey, AllowedIPs}`
- `ServerObfuscationConfig{Jc, Jmin, Jmax, S1, S2, S3, S4, H1, H2, H3, H4}`
- `ClientObfuscationConfig{I1, I2, I3, I4, I5, ServerObfuscationConfig}`
- `ClientConfig{Peer, Interface}`
- `ClientInterfaceConfig{PrivateKey, Address, DNS, Obfuscation, MTU}`
- `ClientPeerConfig{PublicKey, PresharedKey, Endpoint, AllowedIPs, PersistentKeepalive}`
- `Headers{H1, H2, H3, H4 uint32}`
- `SPrefixes{S1, S2, S3, S4 int}`
- `JunkParams{Jc, Jmin, Jmax int}`

## Tests

Existing tests for parser, writer, keys, obfuscation, network, iptables move with the code to root package. CLI tests remain in `internal/cli/` and continue working through the public API.
