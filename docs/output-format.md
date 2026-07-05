# Output Format

> WireGuard-compatible INI layout that `amnezigo generate` writes — one `awg0.conf` per peer, extended with AmneziaWG obfuscation keys and `#_`-prefixed metadata comments.

## Table of Contents

- [Output Layout](#output-layout)
- [Server Config Anatomy](#server-config-anatomy)
- [Client Config Anatomy](#client-config-anatomy)
- [Metadata Comments (`#_`)](#metadata-comments-_)
- [Atomicity and I/O](#atomicity-and-io)
- [Related](#related)

---

## Output Layout

`generate` writes **one directory per peer**, each named after the peer's map key
in the manifest, each holding a single file named `awg0.conf`
(`outputConfigName = "awg0.conf"`, `credentials.go:11`).

```text
output/
├── <server>/awg0.conf      # [Interface] + one [Peer] block per client
├── <client-1>/awg0.conf    # client [Interface] (with I1–I5) + server [Peer]
└── <client-2>/awg0.conf    # client [Interface] (with I1–I5) + server [Peer]
```

| Property | Value | Source |
|---|---|---|
| File name | `awg0.conf` (constant `outputConfigName`) | `credentials.go:11` |
| Server path | `<OutputDir>/<serverName>/awg0.conf` | `pipeline.go:489-492` |
| Client path | `<OutputDir>/<peerName>/awg0.conf` | `pipeline.go:500-503` |
| Directory mode | `0750` (created with `os.MkdirAll`) | `pipeline.go:516` |
| File mode | `0600` (written with `os.WriteFile`) | `pipeline.go:521` |
| Server config | Always written, even under `--peer` filter | `pipeline.go:489-492` |
| `--peer <name>` | Writes only the listed clients + the server | `pipeline.go:494-503` |
| `--dry-run` | Configs computed in memory; nothing written | `pipeline.go:510` |

The server peer's directory name is its manifest key (commonly `server`), not a
hardcoded literal — any valid manifest key is allowed.

---

## Server Config Anatomy

A server config has one `[Interface]` section followed by one `[Peer]` section
per client peer (clients sorted alphabetically for deterministic output,
`pipeline.go:282`). Field emission order follows `WriteServerConfig`
(`writer.go:11-68`) and `writePeerSection` (`writer.go:70-87`) exactly.

```ini
[Interface]
PrivateKey = ePDIloPaQ6+VVKneZLLSi/FAJKyr4LEha76Mx9pn8l0=
PublicKey = ovcr9onMRe15/uY/bP8Pp8QpLZ9i/Wf4KAK1mSwdvkM=
Address = 10.0.0.1/24
ListenPort = 51820
MTU = 1280
PostUp = iptables -A FORWARD -i awg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i awg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
Jc = 5
Jmin = 250
Jmax = 750
S1 = 30
S2 = 35
S3 = 20
S4 = 12
H1 = 100-5000000
H2 = 10000000-200000000
H3 = 400000000-800000000
H4 = 1000000000-2100000000
#_ClientToClient = false
#_TunName = awg0
#_MainIface = eth0

[Peer]
#_Name = phone
PublicKey = dzDIyAqIptJvxjCXYAS4zz5kt1CRLuDabJDgUqzHCFo=
PresharedKey = Pk7j36a8FwPNbgc0hmFq/3dU1WDT3bcoRsJVK+9UAcE=
AllowedIPs = 10.0.0.2/32
```

> **Note:** `generate` does **not** populate `EndpointV4`/`EndpointV6`
> (`buildServerConfig` in `pipeline.go:218-267` never sets them), so generated
> server configs carry no `#_EndpointV4` / `#_EndpointV6` line. The writer
> supports them (`writer.go:49-54`) for hand-authored or library-built configs.

> **Note:** `generate` also sets no peer `PrivateKey` or `CreatedAt`, so generated
> server configs carry **no `#_PrivateKey` or `#_GenKeyTime`** in `[Peer]` (the
> writer supports both for hand-authored or library-built `PeerConfig`). See
> [Credentials](./credentials.md).

### `[Interface]` keys

| Key | Emitted when | Format | Meaning |
|---|---|---|---|
| `PrivateKey` | always | base64 | Server X25519 private key |
| `PublicKey` | `PublicKey != ""` | base64 | Server public key (derived from `PrivateKey`) |
| `Address` | always | CIDR | Server tunnel address (e.g. `10.0.0.1/24`) |
| `ListenPort` | always | decimal | UDP listen port |
| `MTU` | always | decimal | Tunnel MTU (default `1280` from manifest) |
| `DNS` | `DNS != ""` | CSV | Resolver list (e.g. `1.1.1.1, 8.8.8.8`) |
| `PersistentKeepalive` | `!= 0` | decimal | Seconds between keepalives (rarely set on server) |
| `PostUp` | `!= ""` | shell | iptables `-A` rules; set when `main_iface` declared (`pipeline.go:258-266`) |
| `PostDown` | `!= ""` | shell | iptables `-D` rules; mirrors `PostUp` |
| `Jc` | always | decimal | Junk count |
| `Jmin` | always | decimal | Junk-range minimum |
| `Jmax` | always | decimal | Junk-range maximum |
| `S1`–`S4` | always | decimal | Prefix sizes |
| `H1`–`H4` | always | `min-max` | Header ranges (inclusive both ends, `uint32`) |

### `[Interface]` metadata (`#_`-prefixed)

| Line | Emitted when | Format | Meaning |
|---|---|---|---|
| `#_EndpointV4` | `EndpointV4 != ""` | `host:port` | IPv4 endpoint (writer supports; `generate` does not set) |
| `#_EndpointV6` | `EndpointV6 != ""` | `[host]:port` | IPv6 endpoint (writer supports; `generate` does not set) |
| `#_ClientToClient` | always | `true`/`false` | Inter-client routing flag (currently always `false`) |
| `#_TunName` | `TunName != ""` | string | Tunnel interface name (defaults to `awg0`) |
| `#_MainIface` | `MainIface != ""` | string | Egress interface used for NAT/forwarding rules |

### `[Peer]` keys (one section per client, sorted by name)

| Key | Emitted when | Format | Meaning |
|---|---|---|---|
| `#_Name` | `Name != ""` | string | Client's manifest map key |
| `#_PrivateKey` | `PrivateKey != ""` | base64 | Writer-supported; `generate` **never sets it** (`buildServerConfig` omits peer `PrivateKey`), so it does not appear in generated configs. See [Credentials](./credentials.md). |
| `PublicKey` | always | base64 | Client's public key |
| `PresharedKey` | `!= ""` | base64 | Shared PSK for this server↔client pair |
| `AllowedIPs` | always | CIDR | Client's own `/32` address |
| `#_GenKeyTime` | `!CreatedAt.IsZero()` | RFC3339 | Writer-supported; `generate` never sets `CreatedAt`, so it does not appear in generated configs. |

> **Note:** The server config writes **no `I1`–`I5`**. CPS strings are
> client-only — each client's `awg0.conf` carries its own I-values; the server
> has no I-packets to emit (`writer.go:46`).

---

## Client Config Anatomy

A client config has one `[Interface]` (the client's own identity + obfuscation
including its per-client CPS) and one `[Peer]` (the server). Emission order
follows `WriteClientConfig` (`writer.go:89-135`) exactly.

```ini
[Interface]
PrivateKey = cF3x...base64...=
Address = 10.0.0.2/24
DNS = 1.1.1.1, 8.8.8.8
MTU = 1280
Jc = 5
Jmin = 250
Jmax = 750
S1 = 30
S2 = 35
S3 = 20
S4 = 12
H1 = 100-5000000
H2 = 10000000-200000000
H3 = 400000000-800000000
H4 = 1000000000-2100000000
I1 = <b 0x17><r 64><rc 4><rd 4>
I2 = <b 0x18><r 64><rc 4><rd 4>
I3 = <b 0x19><r 64><rc 4><rd 4>
I4 = <b 0x1a><r 64><rc 4><rd 4>

[Peer]
PublicKey = ovcr9onMRe15/uY/bP8Pp8QpLZ9i/Wf4KAK1mSwdvkM=
PresharedKey = Pk7j36a8FwPNbgc0hmFq/3dU1WDT3bcoRsJVK+9UAcE=
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```

### `[Interface]` keys

| Key | Emitted when | Format | Meaning |
|---|---|---|---|
| `PrivateKey` | always | base64 | Client X25519 private key |
| `Address` | always | CIDR | Client tunnel address |
| `DNS` | always (even empty) | CSV | Resolver list — written unconditionally (`writer.go:94`) |
| `MTU` | always | decimal | Tunnel MTU |
| `Jc`, `Jmin`, `Jmax` | always | decimal | Shared junk params (echo the server) |
| `S1`–`S4` | always | decimal | Shared prefix sizes (echo the server) |
| `H1`–`H4` | always | `min-max` | Shared header ranges (echo the server) |
| `I1`–`I5` | only when non-empty | CPS string | Per-client custom packet strings (client-only; regenerated every run) |

> **Note:** `I5` is almost always empty (`maxISize = MTU − 49 − 149 − S1` leaves no
> room at default MTU 1280 with typical S1), so it is omitted. `I1`–`I4` carry
> the CPS payload. See [Obfuscation](./obfuscation.md).

### `[Peer]` keys (the server)

| Key | Emitted when | Format | Meaning |
|---|---|---|---|
| `PublicKey` | always | base64 | Server public key |
| `PresharedKey` | always | base64 | Shared PSK |
| `Endpoint` | always | `host:port` | Server endpoint |
| `AllowedIPs` | always | CIDR list | **Hardcoded `0.0.0.0/0, ::/0`** full tunnel — not configurable via manifest |
| `PersistentKeepalive` | always | decimal | Seconds; **always written, even when `0`** (`writer.go:132`) |

> **Note:** The client config is the **only** place a peer's `I1`–`I5` and
> private `PrivateKey` live. The server config deliberately carries neither the
> CPS strings nor a trusted private key — see [Credentials](./credentials.md).

---

## Metadata Comments (`#_`)

Amnezigo stashes bookkeeping inside INI comments. A bare `#` is an ordinary
comment that WireGuard and Amnezigo both ignore. A `#_`-prefixed line is
**structured metadata** that `ParseServerConfigWithOptions` reads back
(`parser.go:117-148`).

| Prefix | Treatment | Reader |
|---|---|---|
| `#` (bare) | Ignored comment (skipped before key/value split) | `parser.go:77` |
| `#_` | Parsed metadata: stripped of `#_`, value unquoted, routed by section | `parser.go:117-148` |

### Recognised `#_` lines

| Line | Section | Parsed into | Read by |
|---|---|---|---|
| `#_Name` | `[Peer]` | `PeerConfig.Name` | `parser.go:124-125` — matches peer to manifest key |
| `#_PrivateKey` | `[Peer]` | `PeerConfig.PrivateKey` | `parser.go:126-127` — parsed, but `generate` never writes it and the loader ignores it; the client's own `awg0.conf` is the trusted source |
| `#_GenKeyTime` | `[Peer]` | `PeerConfig.CreatedAt` | `parser.go:128-131` — key-generation timestamp (RFC3339); `generate` never sets it, so it does not appear in generated configs |
| `#_EndpointV4` | `[Interface]` | `InterfaceConfig.EndpointV4` | `parser.go:135-136` |
| `#_EndpointV6` | `[Interface]` | `InterfaceConfig.EndpointV6` | `parser.go:137-138` |
| `#_ClientToClient` | `[Interface]` | `InterfaceConfig.ClientToClient` | `parser.go:139-140` — parsed as Go bool (`true`/`false`) |
| `#_TunName` | `[Interface]` | `InterfaceConfig.TunName` | `parser.go:141-142` |
| `#_MainIface` | `[Interface]` | `InterfaceConfig.MainIface` | `parser.go:143-144` |

The `#_` metadata is what lets later `generate` runs **reuse peer credentials**
across regenerations. The recovery flow is documented in
[Credentials & Key Reuse](./credentials.md).

> **Warning:** `ParseServerConfig` only recognises the keys above. Anything else
> under `#_` is silently dropped (non-strict) or surfaced as a `KEY001` warning
> (strict mode). There is **no `ParseClientConfig`** — client configs are read
> only by the lightweight `extractClientCredentials` scanner
> (`credentials.go:122`) for `PrivateKey`/`PresharedKey` recovery.

---

## Atomicity and I/O

Two write paths exist in the codebase. **`generate` uses the non-atomic one.**

| Operation | Function | Mechanism | Atomic? |
|---|---|---|---|
| `generate` file writes | inline in `Generate` | `os.WriteFile(fullPath, content, 0600)` | **No** — partial output possible on crash (`pipeline.go:521`) |
| Server config save (library) | `SaveServerConfig` | write `path.tmp` → `os.Rename` | **Yes** (`writer.go:139-154`) |
| Directory creation | inline in `Generate` | `os.MkdirAll(dir, 0750)` | n/a |
| Client config save (library) | none — no `SaveClientConfig` | — | — |

> **Danger:** `generate` writes each file directly with `os.WriteFile`. A
> mid-run crash (signal, panic, power loss) can leave a **partially-written
> `output/` tree**. `SaveServerConfig` exists and is atomic, but `Generate`
> does **not** call it. Treat the `output/` directory as regenerated-from-state,
> not transactional. See [Gotchas](./gotchas.md).

---

## Related

- [Credentials & Key Reuse](./credentials.md) — how `#_` metadata drives key
  reuse across runs (client private keys are recovered from each peer's own config).
- [Validation & Analysis](./validation.md) — `validate` parses this format with
  `ParseServerConfigWithOptions` and reports findings.
- [Obfuscation](./obfuscation.md) — meaning and ranges of `Jc`/`Jmin`/`Jmax`,
  `S1`–`S4`, `H1`–`H4`, and the `I1`–`I5` CPS grammar.
- [Gotchas](./gotchas.md) — non-atomic `generate` writes, hardcoded client
  `AllowedIPs`, unconditional `PersistentKeepalive`, and more.
