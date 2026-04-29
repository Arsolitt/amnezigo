# Using as a Go Library

> Use Amnezigo as a Go library to generate and manage AmneziaWG configurations programmatically.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Manager API](#manager-api)
- [Config Parsing & Writing](#config-parsing--writing)
- [Key Generation](#key-generation)
- [Obfuscation](#obfuscation)
- [CPS Construction](#cps-construction)
- [Protocol Templates](#protocol-templates)
- [Network Helpers](#network-helpers)
- [iptables Rules](#iptables-rules)
- [Type Reference](#type-reference)
- [Gotchas & Important Notes](#gotchas--important-notes)

---

## Installation

```bash
go get github.com/Arsolitt/amnezigo
```

All business logic lives in the root package `amnezigo`. CLI commands are in `internal/cli` as thin wrappers over this package.

---

## Quick Start

This example demonstrates the full workflow: creating a manager, adding a peer, and exporting a client configuration.

```go
package main

import (
    "bytes"
    "fmt"
    "log"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    mgr := amnezigo.NewManager("awg0.conf")

    peer, err := mgr.AddPeer("laptop", "")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Added peer %s with IP %s\n", peer.Name, peer.AllowedIPs)

    clientCfg, err := mgr.ExportPeer("laptop", "quic", "1.2.3.4:51820")
    if err != nil {
        log.Fatal(err)
    }

    var buf bytes.Buffer
    if err := amnezigo.WriteClientConfig(&buf, clientCfg); err != nil {
        log.Fatal(err)
    }
    fmt.Println(buf.String())
}
```

> **Note:** Before adding peers, you need a server configuration file. Use the `amnezigo init` CLI command or build a `ServerConfig` struct manually and save it with `amnezigo.SaveServerConfig`.

---

## Manager API

The `Manager` struct provides high-level CRUD operations for server configurations and peers. It wraps config file I/O and peer management into a single interface.

```go
mgr := amnezigo.NewManager("awg0.conf")
```

### NewManager

```go
func NewManager(configPath string) *Manager
```

Creates a new Manager bound to a config file path. No file is read at this point.

### Load

```go
func (m *Manager) Load() (ServerConfig, error)
```

Reads and parses the server configuration from disk.

### Save

```go
func (m *Manager) Save(cfg ServerConfig) error
```

Writes the server configuration to disk using atomic writes (write to `.tmp`, then rename).

### AddPeer

```go
func (m *Manager) AddPeer(name, ip string) (PeerConfig, error)
```

Creates a new peer with generated keys. If `ip` is empty, the next available IP in the server's subnet is auto-assigned. Returns the created `PeerConfig`.

Errors: `"peer with name '<name>' already exists"`, `"failed to load server config: ..."`, `"failed to assign IP address: ..."`

```go
peer, err := mgr.AddPeer("phone", "")
peer, err := mgr.AddPeer("desktop", "10.8.0.50")
```

### RemovePeer

```go
func (m *Manager) RemovePeer(name string) error
```

Removes a peer by name.

Error: `"peer '<name>' not found"`

### FindPeer

```go
func (m *Manager) FindPeer(name string) (*PeerConfig, error)
```

Returns a pointer to the peer with the given name.

> **Warning:** The returned pointer points into the loaded config. Modifying it does **not** persist changes — you must call `Save` afterward.

Error: `"peer '<name>' not found"`

### ListPeers

```go
func (m *Manager) ListPeers() []PeerConfig
```

Returns all peers.

> **Warning:** Returns `nil` on load error instead of returning an error. Always check for `nil`.

```go
peers := mgr.ListPeers()
if peers == nil {
    // load error occurred
    return
}
for _, p := range peers {
    fmt.Println(p.Name, p.AllowedIPs)
}
```

### ExportPeer

```go
func (m *Manager) ExportPeer(name, protocol, endpoint string) (ClientConfig, error)
```

Generates a full client configuration for the named peer. The `protocol` parameter determines the I1-I5 CPS strings: `"quic"`, `"dns"`, `"dtls"`, `"stun"`, or `"random"`. The `endpoint` is the server address (e.g., `"1.2.3.4:51820"`).

The exported client config always uses `AllowedIPs = 0.0.0.0/0, ::/0` (routes all traffic). If the server's DNS is empty, it defaults to `"1.1.1.1, 8.8.8.8"`. If `PersistentKeepalive` is 0, it defaults to 25.

Error: `"peer '<name>' not found"`

### BuildPeerConfig

```go
func (m *Manager) BuildPeerConfig(peer PeerConfig, protocol, endpoint string) (ClientConfig, error)
```

Constructs a `ClientConfig` from an existing `PeerConfig`, protocol, and endpoint. Use this when you already have a `PeerConfig` and don't need to look it up by name.

---

## Config Parsing & Writing

Low-level I/O functions that work with any `io.Reader`/`io.Writer`.

### ParseServerConfig

```go
func ParseServerConfig(r io.Reader) (ServerConfig, error)
```

Parses an INI-format server config from any reader. See [configuration.md](configuration.md) for the full config format.

### WriteServerConfig

```go
func WriteServerConfig(w io.Writer, cfg ServerConfig) error
```

Writes a server config in INI format to any writer.

### WriteClientConfig

```go
func WriteClientConfig(w io.Writer, cfg ClientConfig) error
```

Writes a client config in INI format to any writer.

### LoadServerConfig / SaveServerConfig

```go
func LoadServerConfig(path string) (ServerConfig, error)
func SaveServerConfig(path string, cfg ServerConfig) error
```

Convenience wrappers that open files and delegate to `ParseServerConfig` / `WriteServerConfig`. `SaveServerConfig` uses atomic writes.

```go
cfg, err := amnezigo.LoadServerConfig("awg0.conf")
if err != nil {
    log.Fatal(err)
}
err = amnezigo.SaveServerConfig("awg0.conf", cfg)
```

> **Tip:** Use `strings.NewReader` and `bytes.Buffer` in tests instead of real files.

---

## Key Generation

Functions for generating WireGuard X25519 key pairs and preshared keys.

> **Danger:** `GenerateKeyPair`, `DerivePublicKey`, and `GeneratePSK` **panic** on error instead of returning errors. Only use these when failure is unrecoverable (e.g., in `main()` or during initialization).

### GenerateKeyPair

```go
func GenerateKeyPair() (string, string)
```

Returns `(privateKey, publicKey)` as base64 strings (44 chars each). Uses WireGuard key clamping: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`.

```go
privateKey, publicKey := amnezigo.GenerateKeyPair()
```

### DerivePublicKey

```go
func DerivePublicKey(privateKey string) string
```

Derives a public key from a base64-encoded private key. Panics on invalid base64 or wrong key length.

```go
pubKey := amnezigo.DerivePublicKey(privateKey)
```

### GeneratePSK

```go
func GeneratePSK() string
```

Generates a 256-bit preshared key as a 44-character base64 string.

```go
psk := amnezigo.GeneratePSK()
```

---

## Obfuscation

Functions for generating AmneziaWG obfuscation parameters. See [obfuscation.md](obfuscation.md) for detailed explanations of what these parameters mean.

### GenerateConfig

```go
func GenerateConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig
```

Generates all client obfuscation parameters: Jc/Jmin/Jmax, S1-S4, H1-H4, and I1-I5 CPS strings.

```go
obf := amnezigo.GenerateConfig("quic", 1280, 15, 3)
```

### GenerateServerConfig

```go
func GenerateServerConfig(_, s1, jc int) ServerObfuscationConfig
```

Generates server obfuscation parameters (Jc/Jmin/Jmax, S1-S4, H1-H4). Does not include I1-I5.

> **Note:** The first argument (`_`) is **ignored**. Only `s1` and `jc` are used.

```go
serverObf := amnezigo.GenerateServerConfig("quic", 15, 3)
```

### GenerateSPrefixes

```go
func GenerateSPrefixes() SPrefixes
```

Generates S1-S4 size prefixes. S1-S3 range from 0-64, S4 from 0-32. Enforces the constraint `S1+56 != S2` to avoid Init/Response size collisions.

### GenerateJunkParams

```go
func GenerateJunkParams() JunkParams
```

Generates junk packet parameters: Jc (0-10), Jmin and Jmax (64-1024, with Jmin < Jmax).

### GenerateCPS

```go
func GenerateCPS(protocol string, mtu, s1, _ int) (string, string, string, string, string)
```

Generates I1-I5 custom packet strings. The `protocol` parameter accepts `"random"`, `"quic"`, `"dns"`, `"dtls"`, or `"stun"`. The 4th argument is unused.

```go
i1, i2, i3, i4, i5 := amnezigo.GenerateCPS("quic", 1280, 15, 0)
```

### GenerateHeaderRanges

```go
func GenerateHeaderRanges() [4]HeaderRange
```

Generates 4 non-overlapping header ranges (H1-H4).

> **Danger:** Panics if non-overlapping ranges cannot be generated after 1000 attempts. Extremely unlikely in practice.

---

## CPS Construction

Low-level functions for building Custom Packet String tags and combining them.

### BuildCPSTag

```go
func BuildCPSTag(tagType, value string) string
```

Creates a single CPS tag. Supported types:

| Type | Description | Example | Output |
|------|-------------|---------|--------|
| `"b"` | Fixed bytes (hex) | `BuildCPSTag("b", "0xc0ff")` | `<b 0xc0ff>` |
| `"r"` | Random bytes | `BuildCPSTag("r", "8")` | `<r 8>` |
| `"rc"` | Random characters | `BuildCPSTag("rc", "7")` | `<rc 7>` |
| `"rd"` | Random digits | `BuildCPSTag("rd", "2")` | `<rd 2>` |
| `"t"` | Timestamp | `BuildCPSTag("t", "")` | `<t>` |

Returns an empty string for unknown tag types.

### BuildCPS

```go
func BuildCPS(tags []string) string
```

Concatenates CPS tags into a single CPS string.

```go
cps := amnezigo.BuildCPS([]string{
    amnezigo.BuildCPSTag("b", "0xc0ff"),
    amnezigo.BuildCPSTag("b", "0x01"),
    amnezigo.BuildCPSTag("r", "8"),
    amnezigo.BuildCPSTag("t"),
})
// cps: "<b 0xc0ff><b 0x01><r 8><t>"
```

---

## Protocol Templates

Protocol templates define the I1-I5 tag structure that mimics real protocol packets. Each returns an `I1I5Template` containing `[]TagSpec` slices for each interval.

### Available Templates

| Function | Protocol | Intervals |
|----------|----------|-----------|
| `QUICTemplate()` | QUIC Initial packets | I1-I4 (I5 empty) |
| `DNSTemplate()` | DNS Query packets | I1-I4 (I5 empty) |
| `DTLSTemplate()` | DTLS 1.2 ClientHello | I1-I4 (I5 empty) |
| `STUNTemplate()` | STUN Binding Request | I1-I4 (I5 empty) |

```go
tmpl := amnezigo.QUICTemplate()
for _, tag := range tmpl.I1 {
    fmt.Printf("Type: %s, Value: %s\n", tag.Type, tag.Value)
}
```

> **Note:** I5 is always empty for all named protocol templates. It is only populated in random/simple CPS mode.

---

## Network Helpers

Utility functions for IP address handling, port generation, and interface detection.

### IsValidIPAddr

```go
func IsValidIPAddr(ipaddr string) bool
```

Checks if a string is a valid IP address in CIDR notation.

```go
amnezigo.IsValidIPAddr("10.8.0.1/24")  // true
amnezigo.IsValidIPAddr("not-an-ip")     // false
```

### ExtractSubnet

```go
func ExtractSubnet(ipaddr string) string
```

Extracts the subnet from a CIDR address (e.g., `"10.8.0.1/24"` → `"10.8.0.0/24"`).

> **Note:** Returns the original string on parse error instead of returning an error.

### GenerateRandomPort

```go
func GenerateRandomPort() (int, error)
```

Generates a random port in the range [10000, 65535].

### DetectMainInterface

```go
func DetectMainInterface() string
```

Returns the first non-loopback, up interface that has addresses.

> **Note:** Returns an empty string on failure. No error is returned.

### FindNextAvailableIP

```go
func FindNextAvailableIP(serverAddress string, existingIPs []string) (string, error)
```

Finds the next available IP in the subnet, iterating from `.2` to `.254`. IPv4 only.

Errors: `"not an IPv4 address"`, `"invalid CIDR"`

> **Warning:** Returns an empty string and `nil` error if all IPs are exhausted. Check for empty string.

```go
ip, err := amnezigo.FindNextAvailableIP("10.8.0.1/24", []string{"10.8.0.2", "10.8.0.3"})
// ip: "10.8.0.4"
```

---

## iptables Rules

Functions for generating iptables NAT and forwarding rules for WireGuard interfaces.

### GeneratePostUp

```go
func GeneratePostUp(tunName, mainIface, subnet string, clientToClient bool) string
```

Generates iptables rules for the `PostUp` hook. Rules are joined with `; `. Returns 6 rules by default, or 7 when `clientToClient` is `true`.

### GeneratePostDown

```go
func GeneratePostDown(tunName, mainIface, subnet string, clientToClient bool) string
```

Same as `GeneratePostUp` but uses `-D` (delete) instead of `-A` (append), suitable for cleanup.

```go
postUp := amnezigo.GeneratePostUp("awg0", "eth0", "10.8.0.0/24", false)
postDown := amnezigo.GeneratePostDown("awg0", "eth0", "10.8.0.0/24", false)
```

---

## Type Reference

### Core Config Types

#### ServerConfig

Top-level server configuration containing the interface, obfuscation settings, and peers.

| Field | Type | Description |
|-------|------|-------------|
| `Interface` | `InterfaceConfig` | WireGuard interface settings |
| `Obfuscation` | `ServerObfuscationConfig` | Server obfuscation parameters |
| `Peers` | `[]PeerConfig` | Registered peers |

#### InterfaceConfig

WireGuard interface configuration.

| Field | Type | Description |
|-------|------|-------------|
| `TunName` | `string` | Tunnel interface name (e.g., `awg0`) |
| `PrivateKey` | `string` | Server private key (base64) |
| `PublicKey` | `string` | Server public key (base64) |
| `Address` | `string` | Server address in CIDR notation |
| `ListenPort` | `int` | Listening port |
| `MTU` | `int` | Maximum transmission unit |
| `DNS` | `string` | DNS servers (comma-separated) |
| `PostUp` | `string` | PostUp iptables commands |
| `PostDown` | `string` | PostDown iptables commands |
| `MainIface` | `string` | Main network interface for NAT |
| `EndpointV4` | `string` | IPv4 endpoint address |
| `EndpointV6` | `string` | IPv6 endpoint address |
| `PersistentKeepalive` | `int` | Keepalive interval in seconds |
| `ClientToClient` | `bool` | Allow peer-to-peer traffic |

#### PeerConfig

A single peer registered on the server.

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Peer display name |
| `PrivateKey` | `string` | Peer private key (base64) |
| `PublicKey` | `string` | Peer public key (base64) |
| `PresharedKey` | `string` | Preshared key (base64) |
| `AllowedIPs` | `string` | Peer allowed IPs (CIDR) |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `ClientObfuscation` | `*ClientObfuscationConfig` | Client obfuscation params (nil until export) |

### Client Config Types

#### ClientConfig

Top-level client configuration.

| Field | Type | Description |
|-------|------|-------------|
| `Interface` | `ClientInterfaceConfig` | Client interface settings |
| `Peer` | `ClientPeerConfig` | Server peer settings |

#### ClientInterfaceConfig

Client-side WireGuard interface configuration.

| Field | Type | Description |
|-------|------|-------------|
| `PrivateKey` | `string` | Client private key (base64) |
| `Address` | `string` | Client address (CIDR) |
| `DNS` | `string` | DNS servers |
| `MTU` | `int` | Maximum transmission unit |
| `Obfuscation` | `ClientObfuscationConfig` | Client obfuscation parameters |

#### ClientPeerConfig

Client-side server peer configuration.

| Field | Type | Description |
|-------|------|-------------|
| `PublicKey` | `string` | Server public key (base64) |
| `PresharedKey` | `string` | Preshared key (base64) |
| `Endpoint` | `string` | Server endpoint address |
| `AllowedIPs` | `string` | Routes through VPN (always `0.0.0.0/0, ::/0`) |
| `PersistentKeepalive` | `int` | Keepalive interval |

### Obfuscation Types

#### ServerObfuscationConfig

Server-side obfuscation parameters.

| Field | Type | Description |
|-------|------|-------------|
| `Jc` | `int` | Junk count |
| `Jmin` | `int` | Minimum junk size |
| `Jmax` | `int` | Maximum junk size |
| `S1` - `S4` | `int` | Size prefixes |
| `H1` - `H4` | `HeaderRange` | Header value ranges |

#### ClientObfuscationConfig

Client-side obfuscation parameters. Embeds all `ServerObfuscationConfig` fields plus I1-I5.

| Field | Type | Description |
|-------|------|-------------|
| *(embedded)* | `ServerObfuscationConfig` | Jc, Jmin, Jmax, S1-S4, H1-H4 |
| `I1` - `I5` | `string` | Custom Packet Strings |

#### HeaderRange

A range of header values stored as min-max.

| Field | Type | Description |
|-------|------|-------------|
| `Min` | `uint32` | Minimum header value |
| `Max` | `uint32` | Maximum header value |

#### SPrefixes

Size prefix parameters.

| Field | Type | Description |
|-------|------|-------------|
| `S1` - `S4` | `int` | Size prefix values |

#### JunkParams

Junk packet parameters.

| Field | Type | Description |
|-------|------|-------------|
| `Jc` | `int` | Junk count (0-10) |
| `Jmin` | `int` | Minimum junk size (64-1024) |
| `Jmax` | `int` | Maximum junk size (64-1024) |

#### TagSpec

A single CPS tag specification.

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Tag type (`b`, `r`, `rc`, `rd`, `c`, `t`) |
| `Value` | `string` | Tag value (hex for `b`, size for `r`/`rc`/`rd`) |

#### I1I5Template

A protocol template defining CPS tag structure for intervals I1-I5.

| Field | Type | Description |
|-------|------|-------------|
| `I1` - `I5` | `[]TagSpec` | Tag specifications for each interval |

#### Manager

High-level config manager.

| Field | Type | Description |
|-------|------|-------------|
| `ConfigPath` | `string` | Path to the server config file |

---

## Gotchas & Important Notes

### Panicking Functions

`GenerateKeyPair`, `DerivePublicKey`, `GeneratePSK`, and `GenerateHeaderRanges` all **panic** on error instead of returning errors. Only use these when failure is unrecoverable, or wrap them in `recover()`.

### ListPeers Returns Nil on Error

`Manager.ListPeers()` returns `nil` on load error — it does not return an error. Always check for `nil`:

```go
peers := mgr.ListPeers()
if peers == nil {
    // config load failed
}
```

### FindPeer Returns a Mutable Pointer

`Manager.FindPeer()` returns `*PeerConfig` pointing into the loaded config struct. Modifying the returned object does **not** persist changes. You must call `Save` to write changes.

### IPv4 Only for IP Allocation

`FindNextAvailableIP` only supports IPv4 addresses. Passing an IPv6 CIDR returns the error `"not an IPv4 address"`.

### Silent IP Exhaustion

`FindNextAvailableIP` returns an empty string and `nil` error when all IPs (.2 through .254) are exhausted. Check for an empty return value.

### GenerateServerConfig Ignores Protocol

The first argument to `GenerateServerConfig` (the protocol string) is **completely ignored**. Only `s1` and `jc` are used.

### Unused Argument in GenerateCPS

The 4th argument to `GenerateCPS` is unused (named `_` in the function signature).

### Silent Fallbacks

Several helper functions return fallback values instead of errors:
- `ExtractSubnet` — returns the original string on parse error
- `DetectMainInterface` — returns empty string on failure
- `BuildCPSTag` — returns empty string for unknown tag types

### I1-I5 Are Client-Only

I1-I5 fields are not present in server config files. They are generated at export time and only appear in client configurations.

### Peer Private Keys in Server Config

Peer private keys are stored in the server config as `#_PrivateKey` metadata comments. This is required for the export workflow.

### Atomic Writes

`SaveServerConfig` and `Manager.Save` use atomic writes: the config is first written to a `.tmp` file, then renamed to the target path. The `.tmp` file is cleaned up on error.
