# Refactoring Obfuscation Parameters and CLI

## Overview

Refactor Amnezigo to support per-client I1-I5 generation, H1-H4 ranges, configurable interface names, and dynamic client-to-client switching.

## Goals

1. Store IPv4/IPv6 endpoints at init time, not at export time
2. Allow switching client-to-client at any time via edit command
3. Allow custom interface name at init
4. Allow custom config path at init
5. Move I1-I5 generation from init to export (per-client, per-protocol)
6. Change H1-H4 from single values to non-overlapping ranges

## Architecture

### Data Structures

#### HeaderRange

```go
type HeaderRange struct {
    Min, Max uint32
}
```

#### ServerObfuscationConfig

```go
type ServerObfuscationConfig struct {
    Jc, Jmin, Jmax int
    S1, S2, S3, S4 int
    H1, H2, H3, H4 HeaderRange
}
```

#### ClientObfuscationConfig

```go
type ClientObfuscationConfig struct {
    ServerObfuscationConfig
    I1, I2, I3, I4, I5 string
}
```

#### InterfaceConfig (updated)

```go
type InterfaceConfig struct {
    PrivateKey     string
    PublicKey      string
    Address        string
    ListenPort     int
    MTU            int
    PostUp         string
    PostDown       string
    MainIface      string
    TunName        string
    EndpointV4     string  // stored as metadata comment
    EndpointV6     string  // stored as metadata comment
    ClientToClient bool    // stored as metadata comment
}
```

#### PeerConfig (updated)

```go
type PeerConfig struct {
    Name              string
    PrivateKey        string
    PublicKey         string
    PresharedKey      string
    AllowedIPs        string
    CreatedAt         time.Time
    ClientObfuscation *ClientObfuscationConfig  // nil until first export
}
```

#### ServerConfig (updated)

```go
type ServerConfig struct {
    Interface   InterfaceConfig
    Peers       []PeerConfig
    Obfuscation ServerObfuscationConfig
}
```

### H1-H4 Range Generation

#### Constraints

- Range: `5 - 2,147,483,647`
- 4 non-overlapping ranges
- Minimum range size: `10,000,000`
- Order doesn't matter (H1 can be > H2)

#### Algorithm

```
1. Generate 8 random numbers (2 per H1-H4)
2. For each pair: ensure min < max
3. Validate: (max - min) >= 10,000,000
4. Sort ranges by min
5. Validate: end of previous < start of next
6. If validation fails, regenerate
7. Max attempts: 1000
```

#### Example Output

```ini
H1 = 1455256372-2135530789
H2 = 2137190950-2145904314
H3 = 2146855739-2147473085
H4 = 2147475247-2147476419
```

### Config File Format

```ini
[Interface]
PrivateKey = ...
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
PostUp = iptables ...
PostDown = iptables ...

#_EndpointV4 = 1.2.3.4:55424
#_EndpointV6 = [2001:db8::1]:55424
#_ClientToClient = true
#_TunName = awg0

H1 = 1455256372-2135530789
H2 = 2137190950-2145904314
H3 = 2146855739-2147473085
H4 = 2147475247-2147476419
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10

[Peer]
#_Name = laptop
PublicKey = ...
AllowedIPs = 10.8.0.2/32
```

**Note:** Metadata fields (EndpointV4, EndpointV6, ClientToClient, TunName) stored as `#_` comments to avoid AmneziaWG validation errors.

## CLI Changes

### init command

#### New flags

```
--iface-name       Interface name (default: awg0)
--config           Server config path (default: awg0.conf)
--endpoint-v4      IPv4:port (auto-detect if not specified)
--endpoint-v6      IPv6:port (optional)
--client-to-client Allow client-to-client traffic
```

#### Removed flags

```
--protocol         Moved to export command
```

#### Examples

```bash
# Minimum
amnezigo init --ipaddr 10.8.0.1/24

# Full
amnezigo init --ipaddr 10.8.0.1/24 \
    --config /etc/amnezia/server.conf \
    --iface-name wg0 \
    --endpoint-v4 1.2.3.4:51820 \
    --endpoint-v6 [2001:db8::1]:51820 \
    --client-to-client
```

#### Endpoint auto-detection

- IPv4: `https://icanhazip.com` (existing behavior)
- IPv6: `https://ipv6.icanhazip.com` (if available, non-blocking)

### export command

#### New flag

```
--protocol         Obfuscation protocol: random, quic, dns, dtls, stun (default: random)
```

#### Behavior changes

- Endpoint taken from server config (EndpointV4 preferred over EndpointV6)
- I1-I5 generated at export time based on protocol
- Fallback to auto-detection if no endpoint in config

#### Examples

```bash
amnezigo export laptop --protocol quic
amnezigo export --protocol dns
```

#### Endpoint selection logic

```go
if serverCfg.Interface.EndpointV4 != "" {
    endpoint = serverCfg.Interface.EndpointV4
} else if serverCfg.Interface.EndpointV6 != "" {
    endpoint = serverCfg.Interface.EndpointV6
} else {
    endpoint = autoDetect()
}
```

### edit command (new)

#### Usage

```bash
amnezigo edit --client-to-client true|false
```

#### Behavior

1. Load server config
2. If disabling client-to-client (was enabled):
   - Print iptables command to remove forwarding rule immediately
   - Example: `iptables -D FORWARD -i awg0 -o awg0 -j ACCEPT`
3. Update `InterfaceConfig.ClientToClient`
4. Regenerate PostUp/PostDown
5. Rewrite config
6. Print restart reminder

#### Example output

```
✓ Client-to-client disabled
  Run this command to apply immediately:
    iptables -D FORWARD -i awg0 -o awg0 -j ACCEPT
  
  Restart AmneziaWG service to apply all changes
```

## Error Handling

### Export without endpoint

```go
if endpoint == "" {
    endpoint, err = autoDetectEndpoint()
    if err != nil {
        return fmt.Errorf("no endpoint available. Use --endpoint or re-run init with --endpoint-v4")
    }
}
```

### Old config format (missing TunName)

```go
if cfg.Interface.TunName == "" {
    cfg.Interface.TunName = "awg0"
}
```

### H1-H4 generation failure

```go
const maxAttempts = 1000
if attempts >= maxAttempts {
    return fmt.Errorf("failed to generate non-overlapping header ranges")
}
```

### IPv6 unavailable

```go
if err != nil {
    cfg.Interface.EndpointV6 = ""  // optional, not an error
}
```

## Files to Modify

1. `internal/config/types.go` - New data structures
2. `internal/config/writer.go` - Write new config format
3. `internal/config/parser.go` - Parse new config format
4. `internal/obfuscation/generator.go` - GenerateHeaderRanges()
5. `internal/cli/init.go` - New flags, endpoint detection
6. `internal/cli/export.go` - Protocol flag, I1-I5 generation
7. `internal/cli/edit.go` - New command (create)

## Future Considerations

This architecture supports future mesh topologies:

- **Star topology:** Server has ServerObfuscation, clients get ClientObfuscation at export
- **Mesh topology:** Each node can have both ServerObfuscation (for receiving) and ClientObfuscation (for initiating)

The separation of ServerObfuscationConfig and ClientObfuscationConfig allows clean extension without architectural changes.
