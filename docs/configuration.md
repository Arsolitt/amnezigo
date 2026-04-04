# Configuration Files

> How Amnezigo server and client configuration files are structured, parsed, and written.

## Table of Contents

- [Overview](#overview)
- [Server Configuration (awg0.conf)](#server-configuration-awg0conf)
- [Client/Peer Configuration](#clientpeer-configuration)
- [Metadata Fields](#metadata-fields)
- [Atomic Writes](#atomic-writes)
- [iptables Rules](#iptables-rules)

---

## Overview

Amnezigo uses INI-format configuration files with `[Interface]` and `[Peer]` sections. There are two types:

- **Server config** — a single file (typically `awg0.conf`) containing the server's `[Interface]` and all peer `[Peer]` sections. This is the source of truth.
- **Client config** — generated at export time, one per peer. Contains I1-I5 Custom Packet Strings unique to each peer.

Both formats are WireGuard-compatible INI, extended with AmneziaWG obfuscation fields (`Jc`, `Jmin`, `Jmax`, `S1`-`S4`, `H1`-`H4`) and commented metadata fields (`#_` prefix).

> **Note:** The parser silently ignores unrecognized keys, invalid integer values, and lines without `=`. This means a round-trip (write -> parse -> write) may lose data from unknown fields. See [cli-reference.md](cli-reference.md) for commands that create and modify configs, and [library-usage.md](library-usage.md) for programmatic config I/O.

---

## Server Configuration (awg0.conf)

A server config has one `[Interface]` section followed by one or more `[Peer]` sections:

```ini
[Interface]
PrivateKey = <server-private-key>
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
PostUp = iptables -t nat ...
PostDown = iptables -t nat ...
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
#_EndpointV4 = 1.2.3.4:51820
#_EndpointV6 = [2001:db8::1]:51820
#_ClientToClient = false
#_TunName = awg0
#_MainIface = eth0

[Peer]
#_Name = laptop
#_PrivateKey = <peer-private-key>
PublicKey = <peer-public-key>
AllowedIPs = 10.8.0.2/32
```

### [Interface] Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `PrivateKey` | string | Yes | Server private key (base64, 44 chars) |
| `PublicKey` | string | No | Server public key (base64, 44 chars). Written only if non-empty. |
| `Address` | string | Yes | Server IP with CIDR (e.g., `10.8.0.1/24`) |
| `ListenPort` | int | Yes | WireGuard listen port (10000-65535) |
| `MTU` | int | Yes | Interface MTU (default 1280) |
| `DNS` | string | No | DNS servers. Written only if non-empty. |
| `PersistentKeepalive` | int | No | Keepalive interval in seconds. Written only if non-zero. |
| `PostUp` | string | No | iptables rules joined with `; `. Written only if non-empty. |
| `PostDown` | string | No | iptables rules joined with `; `. Written only if non-empty. |

### Obfuscation Parameters (server [Interface])

These fields configure traffic obfuscation. See [obfuscation.md](obfuscation.md) for detailed explanations of each parameter.

| Field | Type | Description |
|---|---|---|
| `Jc` | int | Junk count (0-10) |
| `Jmin` | int | Junk minimum packet size (64-1024) |
| `Jmax` | int | Junk maximum packet size (64-1024, must be > Jmin) |
| `S1` | int | Size prefix 1 (0-64) |
| `S2` | int | Size prefix 2 (0-64) |
| `S3` | int | Size prefix 3 (0-64) |
| `S4` | int | Size prefix 4 (0-32) |
| `H1` | range | Header range 1 (`min-max`, uint32 values) |
| `H2` | range | Header range 2 (`min-max`, uint32 values) |
| `H3` | range | Header range 3 (`min-max`, uint32 values) |
| `H4` | range | Header range 4 (`min-max`, uint32 values) |

> **Warning:** `H1`-`H4` are stored as `min-max` string format (e.g., `191091632-238083235`). If the parser fails to parse a header range, it silently returns an empty value — no error is propagated.

### [Peer] Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `PublicKey` | string | Yes | Peer public key (base64, 44 chars) |
| `PresharedKey` | string | No | Preshared key. Written only if non-empty. |
| `AllowedIPs` | string | Yes | Peer allowed IPs (always `/32` for star topology) |

> **Tip:** A `[Peer]` section is only added to the config when it has a non-empty `PublicKey`. A `[Peer]` section with no `PublicKey` is silently dropped by the parser.

---

## Client/Peer Configuration

Client configs are generated at export time — they are never stored or parsed from the server config. Each peer gets its own I1-I5 CPS strings that make its traffic look like a specific protocol (QUIC, DNS, DTLS, STUN).

```ini
[Interface]
PrivateKey = <peer-private-key>
Address = 10.8.0.2/32
DNS = 1.1.1.1, 8.8.8.8
MTU = 1280
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
I1 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0040><b 0x00><b 0x01><t><r 40>
I2 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0020><b 0x01><t><r 20>
I3 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0010><b 0x01><t><r 16>
I4 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0005><b 0x01><t><r 5>
I5 =

[Peer]
PublicKey = <server-public-key>
PresharedKey = <psk>
Endpoint = 1.2.3.4:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```

### Client [Interface] Fields

The client inherits obfuscation parameters (`Jc`, `Jmin`, `Jmax`, `S1`-`S4`, `H1`-`H4`) from the server config. It additionally includes:

| Field | Type | Description |
|---|---|---|
| `PrivateKey` | string | Client private key |
| `Address` | string | Client IP with `/32` |
| `DNS` | string | DNS servers (defaults to `1.1.1.1, 8.8.8.8` if server has none) |
| `MTU` | int | Interface MTU (from server config) |
| `I1` | string | CPS interval 1. Written only if non-empty. |
| `I2` | string | CPS interval 2. Written only if non-empty. |
| `I3` | string | CPS interval 3. Written only if non-empty. |
| `I4` | string | CPS interval 4. Written only if non-empty. |
| `I5` | string | CPS interval 5. Written only if non-empty. |

### Client [Peer] Fields

| Field | Type | Description |
|---|---|---|
| `PublicKey` | string | Server public key |
| `PresharedKey` | string | Preshared key |
| `Endpoint` | string | Server endpoint (`host:port` or `[ipv6]:port`) |
| `AllowedIPs` | string | Always `0.0.0.0/0, ::/0` (routes all traffic through VPN) |
| `PersistentKeepalive` | int | Keepalive interval (defaults to 25 if server config has 0) |

> **Note:** I1-I5 are **client-only** fields. They are not parsed from or written to the server config. They are generated per-peer at export time based on the selected protocol. See [obfuscation.md](obfuscation.md) for protocol-specific CPS patterns.

---

## Metadata Fields

Metadata fields use a `#_` prefix and appear as comments in the config file. This prefix prevents WireGuard from interpreting them as config directives, while allowing amnezigo to parse them.

### Interface Metadata

| Field | Description |
|---|---|
| `#_EndpointV4` | IPv4 endpoint (e.g., `1.2.3.4:51820`). Written only if non-empty. |
| `#_EndpointV6` | IPv6 endpoint (e.g., `[::1]:51820`). Written only if non-empty. |
| `#_ClientToClient` | Client-to-client traffic enabled (`true`/`false`). Always written. |
| `#_TunName` | Tunnel interface name (e.g., `awg0`). Written only if non-empty. |
| `#_MainIface` | Main network interface (e.g., `eth0`). Written only if non-empty. |

### Peer Metadata

| Field | Description |
|---|---|
| `#_Name` | Peer display name. Written only if non-empty. |
| `#_PrivateKey` | Peer private key (base64, 44 chars). Written only if non-empty. Stored here so the server can export client configs without requiring the peer's private key separately. |
| `#_GenKeyTime` | Key generation time (RFC3339 format). Written only if non-zero. |

> **Warning:** Regular comments use `#` without an underscore — only `#_` prefixed comments are parsed as metadata. Metadata values have surrounding quotes trimmed during parsing (`strings.Trim(value, "\"'")`).

---

## Atomic Writes

When saving a server config, amnezigo uses atomic writes to prevent corruption:

1. Write the full config to `<path>.tmp`
2. Rename `<path>.tmp` to `<path>` (atomic on most filesystems)
3. On error, clean up the `.tmp` file

This is handled by `SaveServerConfig()` in the library API. See [library-usage.md](library-usage.md) for the programmatic interface.

---

## iptables Rules

Amnezigo auto-generates iptables rules stored in the `PostUp` and `PostDown` fields. `PostUp` rules are applied when the interface comes up; `PostDown` uses the same rules with `-D` (delete) instead of `-A` (append) to tear them down.

Rules are joined with `; ` for WireGuard's `PostUp`/`PostDown` hooks.

### Standard Rules (without client-to-client)

```shell
iptables -A INPUT -i awg0 -j ACCEPT
iptables -A OUTPUT -o awg0 -j ACCEPT
iptables -A FORWARD -i awg0 -o eth0 -s 10.8.0.0/24 -j ACCEPT
iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A FORWARD -i eth0 -o awg0 -d 10.8.0.0/24 -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
```

### Additional Rule with Client-to-Client

When client-to-client traffic is enabled, one extra rule is added:

```shell
iptables -A FORWARD -i awg0 -o awg0 -j ACCEPT
```

This allows peers to communicate directly with each other through the tunnel without going through the main interface.

> **Tip:** Use `amnezigo edit --client-to-client true` to enable client-to-client traffic. This regenerates the `PostUp`/`PostDown` rules in the server config. See [cli-reference.md](cli-reference.md) for details.

### Parameters

| Parameter | Description |
|---|---|
| `tunName` | Tunnel interface name (e.g., `awg0`) — from `#_TunName` metadata |
| `mainIface` | Main network interface (e.g., `eth0`) — from `#_MainIface` metadata |
| `subnet` | VPN subnet (e.g., `10.8.0.0/24`) — extracted from server `Address` |
| `clientToClient` | Whether to add the tunnel-to-tunnel forwarding rule |
