# Output Format

> WireGuard-compatible INI layout that `amnezigo generate` writes — one `awg0.conf` per peer, extended with AmneziaWG obfuscation keys, the AWG 3.x transport-protection block, and `#_`-prefixed metadata comments.

## Table of Contents

- [Output Layout](#output-layout)
- [Server Config Anatomy](#server-config-anatomy)
- [Client Config Anatomy](#client-config-anatomy)
- [Transport Protection Keys (AWG 3.x)](#transport-protection-keys-awg-3x)
- [Metadata Comments (`#_`)](#metadata-comments-_)
- [Atomicity and I/O](#atomicity-and-io)
- [Related](#related)

---

## Output Layout

`generate` writes **one directory per peer**, each named after the peer's map key in the manifest and holding that peer's `awg0.conf` (`outputConfigName = "awg0.conf"`, `credentials.go:11`); client directories can additionally hold an `amnezigo.vpn` import link.

```text
output/
├── <server>/awg0.conf        # server config: [Interface] + one [Peer] block per client
├── <client-1>/awg0.conf      # client config: [Interface] (with I-packets) + server [Peer]
├── <client-1>/amnezigo.vpn   # only with --vpn-links: AmneziaVPN import link
├── <client-2>/awg0.conf
└── <client-2>/amnezigo.vpn   # only with --vpn-links
```

| Property | Value | Source |
| --- | --- | --- |
| File name | `awg0.conf` (constant `outputConfigName`) | `credentials.go:11` |
| Server path | `<OutputDir>/<serverName>/awg0.conf` | `pipeline.go:722-723` |
| Client path | `<OutputDir>/<peerName>/awg0.conf` | `pipeline.go:733-734` |
| VPN link path | `<OutputDir>/<peerName>/amnezigo.vpn` (client peers only) | `pipeline.go:738`; `vpnlink.go:16,259` |
| Directory mode | `0750` (created with `os.MkdirAll`) | `pipeline.go:752` |
| File mode | `0600` (written with `os.WriteFile`) | `pipeline.go:757` |
| Server config | Always written, even under `--peer` filter | `pipeline.go:722-723` |
| `--peer <name>` | Writes only the listed clients + the server | `pipeline.go:706-727` |
| `--dry-run` | Configs computed in memory; nothing written | `pipeline.go:746` |
| `--vpn-links` | Emits `amnezigo.vpn` per client peer alongside `awg0.conf` | `pipeline.go:738` |

The server peer's directory name is its manifest key (commonly `server`), not a hardcoded literal — any valid manifest key is allowed.

The `amnezigo.vpn` file is emitted **only** when `--vpn-links` (or `GenerateOptions.VPNLinks: true`) is set, and **only for client peers** — the server directory never contains one. The file holds a single `vpn://` URL string that the AmneziaVPN app can import directly. See [VPN Import Links](./vpn-links.md).

---

## Server Config Anatomy

A server config has one `[Interface]` section followed by one `[Peer]` section per client peer (clients sorted alphabetically for deterministic output, `pipeline.go:497`). Field emission order follows `WriteServerConfig` (`writer.go:11-69`) and `writePeerSection` (`writer.go:71-89`) exactly.

The excerpt below is one complete generated server config, produced by `amnezigo generate` on a two-peer manifest with `network.mtu: 1280` and the default `obfuscation.awg_version: "3.1"` (the project's `testdata/loader/valid/amnezigo-31.json` fixture). Keys, junk parameters, and S prefixes are freshly generated on every run, so your values will differ; the key order, the AWG 3.x block, and the defaults applied to unset manifest fields will not.

```ini
[Interface]
PrivateKey = wGzFK2aTCilO9PUXSpc/l+hnwPtaJqZiXIoLdJUwLmA=
PublicKey = V54qnnPPxnNmNgNJi/uRtkGag71gwEl3LNNYIrgeVgM=
Address = 10.0.0.1/24
ListenPort = 51820
MTU = 1280
Jc = 9
Jmin = 368
Jmax = 903
S1 = 19
S2 = 19
S3 = 19
S4 = 19
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = IyEy+jPeys7vndSabynFVgIJnjJOXDG6RphQCg0fGhY=
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
#_ClientToClient = false
#_TunName = awg0

[Peer]
#_Name = phone
PublicKey = CcObX3gO5jaeJZXOQWiARixYHRX5l2vKx5zRI67CwFM=
PresharedKey = somob5Yoy5p3z3VP9Kkv1fkEZ8g5LimRdgCecqFDdIs=
AllowedIPs = 10.0.0.2/32
```

> **Note:** `network.dns` is unset in this manifest, and the server writer omits `DNS` for the server entirely — `buildServerConfig` never populates `InterfaceConfig.DNS`, so a generated server config never contains a DNS line (`pipeline.go:453-463`).

> **Note:** `generate` does **not** populate `EndpointV4`/`EndpointV6` (`buildServerConfig` never sets them), so generated server configs carry no `#_EndpointV4` / `#_EndpointV6` line. The writer supports them (`writer.go:50-55`) for hand-authored or library-built configs.

> **Note:** `generate` also sets no peer `PrivateKey` or `CreatedAt`, so generated server configs carry **no `#_PrivateKey` or `#_GenKeyTime`** in `[Peer]` (the writer supports both for hand-authored or library-built `PeerConfig`, `writer.go:77-87`). See [Credentials & Key Reuse](./credentials.md).

### `[Interface]` keys

| Key | Emitted when | Format | Meaning |
| --- | --- | --- | --- |
| `PrivateKey` | always | base64 | Server X25519 private key |
| `PublicKey` | `PublicKey != ""` | base64 | Server public key (derived from `PrivateKey`) |
| `Address` | always | CIDR | Server tunnel address (e.g. `10.0.0.1/24`) |
| `ListenPort` | always | decimal | UDP listen port |
| `MTU` | always | decimal | Tunnel MTU (defaults to `1280` when `network.mtu` is unset, `pipeline.go:465-467`) |
| `DNS` | `DNS != ""` | CSV | Resolver list — the writer supports it, but `generate` never sets it, so generated server configs never contain this line (`pipeline.go:453-463`) |
| `PersistentKeepalive` | `!= 0` | decimal | Seconds between keepalives — the pipeline never sets it, so generated server configs omit it (`writer.go:24-26`) |
| `PostUp` | `!= ""` | shell | iptables `-A` rules; set when `main_iface` is declared (`pipeline.go:473-482`) |
| `PostDown` | `!= ""` | shell | iptables `-D` rules; mirrors `PostUp` (`pipeline.go:473-482`) |
| `Jc` | always | decimal | Junk count |
| `Jmin` | always | decimal | Junk-range minimum |
| `Jmax` | always | decimal | Junk-range maximum |
| `S1`–`S4` | always | decimal | Prefix sizes |
| `H1`–`H4` | always | `min-max` | Header ranges (inclusive both ends, `uint32`) |
| `HeaderProtectionKey` | AWG 3.0+ and non-empty | base64 | 32-byte ChaCha20 header-protection key; the identical value appears in every client config |
| `ContentPaddingAddition` | AWG 3.0+ and non-zero | `N` or `N-M` | Content-padding byte range |
| `RekeyAfterTime` | AWG 3.0+ and non-zero | `N` or `N-M` | Rekey-after-time range |
| `RekeyTimeout` | AWG 3.0+ and non-zero | `N` or `N-M` | Rekey-timeout range |
| `RejectAfterTime` | AWG 3.0+ and non-zero | `N` or `N-M` | Packet-rejection age range |
| `KeepaliveTimeout` | AWG 3.0+ and non-zero | `N` or `N-M` | Keepalive-timeout range |
| `MaxHandshakeAttempts` | AWG 3.0+ and non-zero | `N` or `N-M` | Handshake-attempt range |
| `RandomTrailers` | AWG 3.1+ (always emitted) | `on`/`off` | Random trailers toggle |
| `DisableCookies` | AWG 3.1+ (always emitted) | `on`/`off` | Cookie-reply toggle |

When `main_iface` is set, the `PostUp`/`PostDown` pair is emitted after `MTU`. Each value spans two physical lines — all IPv4 rules joined with `; ` on the first line, the matching IPv6 rules on the second (`pipeline.go:473-482`). Abridged with `...`:

```ini
PostUp = iptables -A INPUT -i awg0 -j ACCEPT; ...; iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
ip6tables -A INPUT -i awg0 -j ACCEPT; ...; ip6tables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
PostDown = iptables -D INPUT -i awg0 -j ACCEPT; ...; iptables -t nat -D POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
ip6tables -D INPUT -i awg0 -j ACCEPT; ...; ip6tables -t nat -D POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
```

### `[Interface]` metadata (`#_`-prefixed)

| Line | Emitted when | Format | Meaning |
| --- | --- | --- | --- |
| `#_EndpointV4` | `EndpointV4 != ""` | `host:port` | IPv4 endpoint (writer supports; `generate` does not set) |
| `#_EndpointV6` | `EndpointV6 != ""` | `[host]:port` | IPv6 endpoint (writer supports; `generate` does not set) |
| `#_ClientToClient` | always | `true`/`false` | Inter-client routing flag (currently always `false`; `pipeline.go:461`) |
| `#_TunName` | `TunName != ""` | string | Tunnel interface name (defaults to `awg0` when `tun_name` is unset, `pipeline.go:468-470`) |
| `#_MainIface` | `MainIface != ""` | string | Egress interface used for NAT/forwarding rules; set together with `PostUp`/`PostDown` |

### `[Peer]` keys (one section per client, sorted by name)

| Key | Emitted when | Format | Meaning |
| --- | --- | --- | --- |
| `#_Name` | `Name != ""` | string | Client's manifest map key; required for credential reuse across runs (`writer.go:72-76`) |
| `#_PrivateKey` | `PrivateKey != ""` | base64 | Writer-supported; `generate` **never sets it** (`buildServerConfig` omits peer `PrivateKey`), so it does not appear in generated configs. See [Credentials & Key Reuse](./credentials.md). |
| `PublicKey` | always | base64 | Client's public key |
| `PresharedKey` | `!= ""` | base64 | Shared PSK for this server↔client pair |
| `AllowedIPs` | always | CIDR | Client's own `/32` address |
| `#_GenKeyTime` | `!CreatedAt.IsZero()` | RFC3339 | Writer-supported; `generate` never sets `CreatedAt`, so it does not appear in generated configs (`writer.go:85-87`). |

> **Note:** The server config writes **no `I1`–`I5`**. `ServerObfuscationConfig` has no I fields at all — the server writer has no I-packet support — so CPS strings are client-only: each client's `awg0.conf` carries its own I-values (`writer.go:46-47`, `types.go:77-108`).

---

## Client Config Anatomy

A client config has one `[Interface]` (the client's own identity + obfuscation including its per-client CPS) and one `[Peer]` (the server). Emission order follows `WriteClientConfig` (`writer.go:131-179`) exactly.

This is the same run's `phone/awg0.conf`:

```ini
[Interface]
PrivateKey = gO4kQcjfUHcFo9mTYg5OncmHItZ/0w5MT7mQ+XvEL1k=
Address = 10.0.0.2/32
DNS = 
MTU = 1280
Jc = 9
Jmin = 368
Jmax = 903
S1 = 19
S2 = 19
S3 = 19
S4 = 19
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = IyEy+jPeys7vndSabynFVgIJnjJOXDG6RphQCg0fGhY=
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
I1 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0040><b 0x00><b 0x01><t><r 40>
I2 = <b 0xc0ff><b 0x00000001><b 0x08><d><b 0x00><b 0x00><b 0x0020><b 0x01><t><r 20>
I3 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0010><b 0x01><t><r 10>
I4 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0005><b 0x01><t><r 5>

[Peer]
PublicKey = V54qnnPPxnNmNgNJi/uRtkGag71gwEl3LNNYIrgeVgM=
PresharedKey = somob5Yoy5p3z3VP9Kkv1fkEZ8g5LimRdgCecqFDdIs=
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 0
```

> **Note:** the `DNS = ` line above is real output for a manifest without `network.dns` — the client writer always emits the key, so the line is present with an empty value (and a trailing space). When DNS resolvers are configured, the client line is a `", "`-joined list, for example `DNS = 1.1.1.1, 8.8.8.8` (`writer.go:135`, `pipeline.go:597`).

### `[Interface]` keys

| Key | Emitted when | Format | Meaning |
| --- | --- | --- | --- |
| `PrivateKey` | always | base64 | Client X25519 private key |
| `Address` | always | CIDR | Client tunnel address |
| `DNS` | always (even empty) | CSV | Resolver list — written unconditionally (`writer.go:135`) |
| `MTU` | always | decimal | Tunnel MTU (defaults to `1280`, `pipeline.go:580-582`) |
| `Jc`, `Jmin`, `Jmax` | always | decimal | Shared junk params (echo the server) |
| `S1`–`S4` | always | decimal | Shared prefix sizes (echo the server) |
| `H1`–`H4` | always | `min-max` | Shared header ranges (echo the server) |
| `HeaderProtectionKey` | AWG 3.0+ and non-empty | base64 | The server's header-protection key (identical value) |
| `ContentPaddingAddition` | AWG 3.0+ and non-zero | `N` or `N-M` | Content-padding byte range |
| `RekeyAfterTime` | AWG 3.0+ and non-zero | `N` or `N-M` | Rekey-after-time range |
| `RekeyTimeout` | AWG 3.0+ and non-zero | `N` or `N-M` | Rekey-timeout range |
| `RejectAfterTime` | AWG 3.0+ and non-zero | `N` or `N-M` | Packet-rejection age range |
| `KeepaliveTimeout` | AWG 3.0+ and non-zero | `N` or `N-M` | Keepalive-timeout range |
| `MaxHandshakeAttempts` | AWG 3.0+ and non-zero | `N` or `N-M` | Handshake-attempt range |
| `RandomTrailers` | AWG 3.1+ (always emitted) | `on`/`off` | Random trailers toggle |
| `DisableCookies` | AWG 3.1+ (always emitted) | `on`/`off` | Cookie-reply toggle |
| `I1`–`I5` | only when non-empty | CPS string | Per-client custom packet strings (client-only; emitted in fixed `I1`→`I5` order) |

> **Note:** `I5` is intentionally empty in every named protocol template (a named-template convention — see `quic.go:79-80`), and the writer skips empty I-values, so the four I-lines above are the complete set for the default `quic` protocol (`writer.go:152-169`). The `maxISize = MTU − 49 − 149 − S1` bound applies to the intervals that are defined: an interval that would not fit is shrunk by dropping trailing tags (`cps.go:14-15, 67-70, 158-172`). See [Obfuscation](./obfuscation.md) for the CPS tag grammar.

> **Warning:** always set `network.mtu` explicitly. If the manifest omits it, the emitted config still says `MTU = 1280`, but CPS generation runs before that default is applied and reads MTU 0, so **every I-packet degenerates to the bare `<t>` interval** — protocol mimicry is lost. A live run without a `network` section produced `I1 = <t>` through `I5 = <t>` (`pipeline.go:577-584`). See [Gotchas](./gotchas.md).

### `[Peer]` keys (the server)

| Key | Emitted when | Format | Meaning |
| --- | --- | --- | --- |
| `PublicKey` | always | base64 | Server public key |
| `PresharedKey` | always | base64 | Shared PSK |
| `Endpoint` | always | `host:port` | Server endpoint |
| `AllowedIPs` | always | CIDR list | **Hardcoded `0.0.0.0/0, ::/0`** full tunnel — not configurable via manifest |
| `PersistentKeepalive` | always | decimal | Seconds; **always written, even when `0`** (`writer.go:174`) |

> **Note:** The client config is the **only** place a peer's `I1`–`I5` and private `PrivateKey` live. The server config deliberately carries neither the CPS strings nor a trusted peer private key — see [Credentials & Key Reuse](./credentials.md).

---

## Transport Protection Keys (AWG 3.x)

The nine AWG 3.x keys are device-level `[Interface]` keys. `generate` resolves them once and writes the identical block into the server config and every client config (client configs carry their own `I1`–`I5` lines after it). They are real INI keys, not `#_` metadata, and they are emitted in a fixed position: immediately after `H4`.

- On the server: after `H4`, before the `#_` metadata block (`writer.go:45-47`).
- On clients: after `H4`, before `I1` (`writer.go:149-151`).

Emission is version-gated because pre-3.x engine builds reject unknown INI keys (`Line unrecognized`), so a key is never written for a lower target version (`writer.go:107-114`).

### Version gates

| Target | Emitted transport keys |
| --- | --- |
| AWG 2.0 (`awg_version: "2.0"`) or a zero-value `Version` | none — the whole block is suppressed |
| AWG 3.0 (`awg_version: "3.0"`) | `HeaderProtectionKey` (only when non-empty) + the six ranges (each only when non-zero) |
| AWG 3.1 (`awg_version: "3.1"`, the default) | the 3.0 set, plus `RandomTrailers` and `DisableCookies` — always emitted, even when `off` |

> **Warning:** the gate is driven by `ServerObfuscationConfig.Version`, not by the manifest. Library code that builds this struct by hand and leaves `Version` at its zero value emits **no transport keys at all**, even when the 3.x fields are populated — that default keeps legacy configs emitting exactly the 2.0 key set (`writer.go:112`, `version.go:12-21`).

### Value formats and omission rules

- Range values render as `N` when `Min == Max`, otherwise `N-M` (`U16Range.String()`, `types.go:33-40`). A 5–5 timeout range is written `RekeyTimeout = 5`, not `5-5`; the default is `RekeyTimeout = 5-8`.
- A range whose bounds are both `0` (`IsZero()`) is **skipped entirely** — the key does not appear, so the engine keeps its own default (`writer.go:100-105`).
- The two booleans are spelled `on`/`off`, never `true`/`false`, and are never omitted under AWG 3.1 — `RandomTrailers = off` and `DisableCookies = off` are normal output (`writer.go:91-96`, `writer.go:124-127`).
- `HeaderProtectionKey` is skipped when empty. With header protection turned off under 3.1, that line disappears while the six ranges and both booleans are still emitted.
- `H1`–`H4` do **not** follow these folding rules: they are always written as `min-max`, even when the bounds are equal (`H1 = 1-1` under header protection), and are never omitted (`writer.go:42-45`).

### Defaults for unset manifest fields

Under a 3.0+ target, a manifest that omits these fields gets the values below; the six default ranges jitter around the WireGuard protocol constants 120 s rekey, 5 s rekey timeout, 180 s reject, 10 s keepalive, and 18 handshake attempts (`pipeline.go:250-257`).

| Key | Emitted value when the manifest field is unset |
| --- | --- |
| `ContentPaddingAddition` | `2-10` |
| `RekeyAfterTime` | `120-180` |
| `RekeyTimeout` | `5-8` |
| `RejectAfterTime` | `180-240` |
| `KeepaliveTimeout` | `8-12` |
| `MaxHandshakeAttempts` | `16-20` |
| `RandomTrailers` | `on` (3.1) |
| `DisableCookies` | `on` (3.1) |

The manifest fields that drive these keys are documented in [Manifest Reference](./manifest-reference.md); the whole layer is covered end-to-end in [Transport Protection (AWG 3.x)](./transport-protection.md).

### Reading the keys back

`ParseServerConfig` recognises all nine keys (`parser.go:224-262`). The parser has no notion of the target engine version — there is no version key in the INI format — so a 3.1 key set is accepted even in a config that will be used with a 2.0 engine (`parser.go:50-62`).

- `RandomTrailers` / `DisableCookies` parse case-insensitively: `on`/`1` are true, `off`/`0` are false; `true`/`false` are rejected with `expected on/off/0/1, got "<value>"` (`parser.go:354-365`).
- Range parse failures are wrapped with the key name: for example `invalid ContentPaddingAddition "abc": expected "N" or "N-M"` and `invalid RekeyTimeout "10-5": max (5) is below min (10)` (`parser.go:336-375`).
- `HeaderProtectionKey` must be 44-character base64 decoding to exactly 32 bytes, otherwise parsing fails with `invalid HeaderProtectionKey "<value>": must be 44-char base64 of 32 bytes` (`parser.go:224-230`).
- Round-tripping is exact: a written 3.1 block re-parses to the same config, and the parser accepts both the `N` and the `N-M` form (`parser_test.go:405-490`).

> **Note:** `generate` does not validate its in-memory struct — it re-parses the server bytes it just emitted and attaches `ValidateServerConfig` findings (for example `TRL001`) to the result, while a parse failure is silently ignored (`pipeline.go:688-694`). See [Validation & Analysis](./validation.md).

> **Danger:** `HeaderProtectionKey` is a plain INI key duplicated in the server and every client config — anyone who can read a client config can decrypt header-protected packet headers. Treat every `awg0.conf` as a secret (client configs already contain private keys), and see [Credentials & Key Reuse](./credentials.md) for how the key is persisted and rotated.

---

## Metadata Comments (`#_`)

Amnezigo stashes bookkeeping inside INI comments. A bare `#` is an ordinary comment that WireGuard and Amnezigo both ignore. A `#_`-prefixed line is **structured metadata** that `ParseServerConfigWithOptions` reads back (`parser.go:125-159`).

| Prefix | Treatment | Reader |
| --- | --- | --- |
| `#` (bare) | Ignored comment (skipped before the key/value split) | `parser.go:85` |
| `#_` | Parsed metadata: stripped of `#_`, value unquoted, routed by section | `parser.go:125-159` |

### Recognised `#_` lines

| Line | Section | Parsed into | Read by |
| --- | --- | --- | --- |
| `#_Name` | `[Peer]` | `PeerConfig.Name` | `parser.go:132-133` — matches peer to manifest key |
| `#_PrivateKey` | `[Peer]` | `PeerConfig.PrivateKey` | `parser.go:134-135` — parsed, but `generate` never writes it and the loader ignores it; the client's own `awg0.conf` is the trusted source |
| `#_GenKeyTime` | `[Peer]` | `PeerConfig.CreatedAt` | `parser.go:136-139` — key-generation timestamp (RFC3339); unparseable values are ignored; `generate` never sets it |
| `#_EndpointV4` | `[Interface]` | `InterfaceConfig.EndpointV4` | `parser.go:143-144` |
| `#_EndpointV6` | `[Interface]` | `InterfaceConfig.EndpointV6` | `parser.go:145-146` |
| `#_ClientToClient` | `[Interface]` | `InterfaceConfig.ClientToClient` | `parser.go:147-148` — parsed as `value == "true"`, so only lowercase `true` is true |
| `#_TunName` | `[Interface]` | `InterfaceConfig.TunName` | `parser.go:149-150` |
| `#_MainIface` | `[Interface]` | `InterfaceConfig.MainIface` | `parser.go:151-152` |

The `#_` metadata is what lets later `generate` runs **reuse peer credentials** across regenerations. The recovery flow is documented in [Credentials & Key Reuse](./credentials.md).

> **Warning:** `ParseServerConfig` only recognises the keys above. Anything else under `#_` is silently dropped in both modes — the parser continues past the `#_` branch before its strict key check, so `KEY001` (`parser.go:276-280`) fires only for unrecognized regular `[Interface]`/`[Peer]` keys. There is **no `ParseClientConfig`** — client configs are read only by the lightweight `extractClientCredentials` scanner (`credentials.go:131`) for `PrivateKey`/`PresharedKey` recovery.

---

## Atomicity and I/O

Two write paths exist in the codebase. **`generate` uses the non-atomic one.**

| Operation | Function | Mechanism | Atomic? |
| --- | --- | --- | --- |
| `generate` file writes | write loop in `Generate` | `os.WriteFile(fullPath, content, 0600)` | **No** — no temp file, no rollback (`pipeline.go:746-760`) |
| Server config save (library) | `SaveServerConfig` | write `path.tmp` → `os.Rename` | **Yes** (`writer.go:181-196`) |
| Directory creation | write loop in `Generate` | `os.MkdirAll(dir, 0750)` | n/a |
| Client config save (library) | none — there is no `SaveClientConfig` | — | — |

The pipeline is **two-pass**: every config is built in memory before the first file is written, so a build failure (for example, an invalid obfuscation value) leaves the output tree untouched. The write loop itself, however, has no transactional guarantee.

> **Danger:** `generate` writes each file directly with `os.WriteFile`. A crash mid-loop (signal, panic, power loss, disk full) can leave a **partially updated `output/` tree** with earlier files new and later files stale, and there is no rollback. `SaveServerConfig` exists and is atomic, but `Generate` does **not** call it. Treat the `output/` directory as regenerated-from-state, not transactional. See [Gotchas](./gotchas.md).

---

## Related

- [Transport Protection (AWG 3.x)](./transport-protection.md) — the 3.x layer end-to-end: manifest knobs, version gates, header-protection key persistence, validation codes.
- [Manifest Reference](./manifest-reference.md) — the manifest fields that resolve to every key shown here, including the ten AWG 3.x fields.
- [Obfuscation](./obfuscation.md) — meaning and ranges of `Jc`/`Jmin`/`Jmax`, `S1`–`S4`, `H1`–`H4`, and the `I1`–`I5` CPS grammar.
- [Credentials & Key Reuse](./credentials.md) — how `#_` metadata drives key reuse across runs, and how the header-protection key is persisted.
- [VPN Import Links](./vpn-links.md) — the `vpn://` import link format and the `--vpn-links` flag that emits `amnezigo.vpn` per client peer.
- [Gotchas](./gotchas.md) — non-atomic `generate` writes, the omitted-`network.mtu` I-packet collapse, hardcoded client `AllowedIPs`, and more.
