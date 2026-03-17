# Simplify AllowedIPs and Add PresharedKey to Server Peer Blocks

## Overview

Two changes to WireGuard configuration:
1. Replace complex AllowedIPs calculation with simple `0.0.0.0/0, ::/0`
2. Add PresharedKey to server-side peer blocks (not just client config)

## Changes

### 1. AllowedIPs Simplification

**Current:** `export.go` calls `network.CalculateAllowedIPs()` which computes complement of private IP ranges (~50 CIDR blocks).

**New:** Hardcode `0.0.0.0/0, ::/0` in client config export.

**Files:**
- `internal/cli/export.go:115` — replace `network.CalculateAllowedIPs(subnet)` with `"0.0.0.0/0, ::/0"`
- `internal/network/allowedips.go` — delete
- `internal/network/allowedips_test.go` — delete

### 2. PresharedKey in Server Peer Blocks

**Current:** PSK stored in `ServerConfig.PSK` (global), only written to client config.

**New:** Store PSK per-peer in server config peer blocks.

**Files:**
- `internal/config/types.go` — add `PresharedKey string` to `PeerConfig`
- `internal/config/writer.go:69` — write PresharedKey in server peer block
- `internal/config/parser.go` — parse PresharedKey from peer section
- `internal/cli/add.go` — generate and store PSK when adding peer
- Tests updated accordingly

## Server Config Example (After)

```ini
[Interface]
PrivateKey = ...
Address = 10.8.0.1/24
...

[Peer]
#_Name = laptop
PublicKey = ...
PresharedKey = <psk>
AllowedIPs = 10.8.0.2/32
```

## Client Config Example (After)

```ini
[Interface]
...

[Peer]
PublicKey = ...
PresharedKey = <psk>
Endpoint = 1.2.3.4:55424
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```
