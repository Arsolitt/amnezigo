# Manifest Reference

> Complete field reference for `amnezigo.json` / `.amnezigo.jsonnet` — the single declarative file that drives `amnezigo generate`.

## Table of Contents

- [Top-level `Manifest`](#top-level-manifest)
- [Network configuration (`NetworkConfig`)](#network-configuration-networkconfig)
- [Obfuscation profile (`ObfuscationManifest`)](#obfuscation-profile-obfuscationmanifest)
- [Peer declaration (`PeerManifest`)](#peer-declaration-peermanifest)
- [Header range (`HeaderRange`)](#header-range-headerrange)
- [Pointer-nil semantics](#pointer-nil-semantics)
- [Manifest discovery](#manifest-discovery)
- [Related](#related)

---

## Top-level `Manifest`

The root object parsed from the manifest file. Declares one schema version, shared network settings, a shared obfuscation profile, and a flat map of peers.

| Field | JSON key | Go type | Required | Default | Description |
|---|---|---|---|---|---|
| `Version` | `version` | `int` | **yes** | — | Schema version; **must be `1`**. No `omitempty` — `{}` decodes to `0` and is rejected. |
| `Network` | `network` | `NetworkConfig` | no | zero-value | Shared interface settings (MTU, DNS). See [Network configuration](#network-configuration-networkconfig). |
| `Obfuscation` | `obfuscation` | `ObfuscationManifest` | no | zero-value | Shared obfuscation profile; all S/H/J fields are nullable pointers. See [Obfuscation profile](#obfuscation-profile-obfuscationmanifest). |
| `Peers` | `peers` | `map[string]PeerManifest` | **yes** | — | Flat peer map. The **map key is the peer name** (used as the output directory name); there is no `name` field on `PeerManifest`. See [Peer declaration](#peer-declaration-peermanifest). |

**Version validation** (`loader.go`):

| `version` value | Result |
|---|---|
| `0` (or absent) | error: `missing or zero version field` |
| `1` | OK (only supported schema) |
| any other | error: `unsupported schema version N (expected 1)` |

**Helper methods on `*Manifest`:**

| Method | Returns | Notes |
|---|---|---|
| `ServerPeer()` | `(name string, count int)` | A valid manifest has `count == 1`. When `count > 1`, `name` is one of the servers (arbitrary — map iteration order); check `count` first. |
| `ServerPeerName()` | `string` | **Panics** if `count != 1`. Call only after validation (the generate pipeline does). |
| `PeerNames()` | `[]string` | All peer names sorted alphabetically (deterministic iteration). |

Minimal shape:

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": { "protocol": "quic" },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "phone":  { "address": "10.0.0.2/32" }
  }
}
```

## Network configuration (`NetworkConfig`)

Global because AWG/WG MTU is per-interface (not per-peer) and DNS applies uniformly to client configs.

| Field | JSON key | Go type | Required | Default | Description |
|---|---|---|---|---|---|
| `MTU` | `mtu` | `int` | no | `1280` (when `0`/absent) | Per-interface MTU; shared by the server and every client. |
| `DNS` | `dns` | `[]string` | no | `nil` | Upstream resolvers; comma-joined into the `DNS =` line of **client** configs only. Not written to the server config. |

> **Note:** The `1280` default is applied by the generate pipeline, not by the loader — `NetworkConfig.MTU` is a plain `int`, so `0`/absent propagates and is substituted at generation time (`pipeline.go`, server and client paths).

## Obfuscation profile (`ObfuscationManifest`)

Shared AmneziaWG obfuscation profile. All numeric fields use **pointer types** so "set to 0" is distinct from "unset" — see [Pointer-nil semantics](#pointer-nil-semantics).

| Field | JSON key | Go type | Required | Default (`nil` →) | Description |
|---|---|---|---|---|---|
| `Protocol` | `protocol` | `string` | no | — | **Decorative at the manifest level — NOT consumed by `generate`.** Only the per-peer `peers[].protocol` field drives I-packet shape (defaults to `quic`). Kept in the schema for documentation/Jsonnet use. |
| `S1` | `s1` | `*int` | no | random | S-prefix 1. Generator legal range `[0,64]`. |
| `S2` | `s2` | `*int` | no | random | S-prefix 2. Range `[0,64]`. |
| `S3` | `s3` | `*int` | no | random | S-prefix 3. Range `[0,64]`. |
| `S4` | `s4` | `*int` | no | random | S-prefix 4. Range `[0,32]`. |
| `H1` | `h1` | `*HeaderRange` | no | random | Header range 1; must be non-overlapping and avoid WG message type-ids `[1..4]`. |
| `H2` | `h2` | `*HeaderRange` | no | random | Header range 2. |
| `H3` | `h3` | `*HeaderRange` | no | random | Header range 3. |
| `H4` | `h4` | `*HeaderRange` | no | random | Header range 4. |
| `Jc` | `jc` | `*int` | no | random | Junk count. Generator range `[0,10]`. |
| `Jmin` | `jmin` | `*int` | no | random | Junk minimum. Range `[64,1024]`; must avoid padded+raw WG sizes. |
| `Jmax` | `jmax` | `*int` | no | random | Junk maximum. Must be `> jmin`. |

> **Warning:** `obfuscation.protocol` is informational only. To select a protocol that actually affects generation, set `peers.<name>.protocol` (per peer). See [Obfuscation](./obfuscation.md) for the param semantics and constraints, and [Presets](./presets.md) for ready-made S/H/J value sets.

For range/format details on the S-prefix constraint (`S1+56 ≠ S2`), H-range validity (uint32 `5–2147483647`, non-overlapping, span `≥ 10⁷`), and junk-range ordering, see [Obfuscation](./obfuscation.md).

**Helper methods on `*ObfuscationManifest`:**

| Method | Returns | Notes |
|---|---|---|
| `HasAnyValue()` | `bool` | `true` if any S/H/J pointer is non-nil. `false` ⇒ fully-random generation (fast-path in `resolveObfuscation`). |
| `ToSharedObfuscation()` | `ServerObfuscationConfig` | Dereferences non-nil pointer fields; nil fields become zero-values, signalling random generation downstream. |

## Peer declaration (`PeerManifest`)

Declares a single peer. The peer **name** is the `peers` map key, not a struct field.

| Field | JSON key | Go type | Required | Default | Description |
|---|---|---|---|---|---|
| `Address` | `address` | `string` | **yes** | — | Interface address as CIDR (e.g. `10.0.0.1/24` for the server, `10.0.0.2/32` for a client). |
| `Endpoint` | `endpoint` | `string` | no | `""` | `host:port`. Server marker (together with non-zero `listen_port`). |
| `ListenPort` | `listen_port` | `int` | no | `0` | UDP listen port. Server marker (together with non-empty `endpoint`). |
| `Protocol` | `protocol` | `string` | no | `quic` | **Per-peer obfuscation protocol** — this is the field `generate` actually reads. Drives I1–I5 generation for this peer. |
| `Keepalive` | `keepalive` | `*int` | no | `nil` | `PersistentKeepalive` seconds for the client's `[Peer]`. `nil`/`0` ⇒ off. |
| `TunName` | `tun_name` | `string` | no | `awg0` | Server interface name; written as `#_TunName` metadata. |
| `MainIface` | `main_iface` | `string` | no | `""` | Server egress interface (e.g. `eth0`). When set, `PostUp`/`PostDown` iptables rules are generated. |

> **Note:** There is no `PresharedKey`, `PublicKey`, or `PrivateKey` field on `PeerManifest` — crypto material is generated/persisted by the credentials layer, not declared in the manifest.

**Server detection rule** (`PeerManifest.IsServer()`, exact):

```go
return p.Endpoint != "" && p.ListenPort != 0
```

A peer with only one of `endpoint` / `listen_port` set is a **client**. A valid manifest contains **at most one** server peer (enforced by validation; `ServerPeerName()` panics otherwise).

## Header range (`HeaderRange`)

Inclusive `min`–`max` pair used by the H1–H4 obfuscation headers.

| Field | JSON key | Go type | Description |
|---|---|---|---|
| `Min` | `min` | `uint32` | Range lower bound (inclusive). |
| `Max` | `max` | `uint32` | Range upper bound (inclusive). |

```json
"h1": { "min": 100, "max": 5000000 }
```

Serialized as `min-max` in INI metadata. The **server** config stores H1–H4 as ranges; each **client** config stores them as resolved point values. See [Output Format](./output-format.md).

## Pointer-nil semantics

The `*int` and `*HeaderRange` pointer fields exist to distinguish "user set this to `0`" from "user did not set this field" — JSON `"s1": 0` is different from omitting `s1`.

| Pointer type | `nil` (omitted) | explicit value (incl. `0`) |
|---|---|---|
| `*int` — `S1`–`S4`, `Jc`, `Jmin`, `Jmax` | random fallback in `resolveObfuscation` | pinned to that integer |
| `*HeaderRange` — `H1`–`H4` | random non-overlapping range | pinned to `{min, max}` |
| `*int` — `Keepalive` | `PersistentKeepalive = 0` (off) | `PersistentKeepalive = <value>` |

`ObfuscationManifest.HasAnyValue()` returns `false` only when **all** S/H/J pointer fields are `nil` — the fast-path that selects fully-random generation. See [Obfuscation](./obfuscation.md) for how randomness is drawn and validated.

## Manifest discovery

The loader discovers the manifest from a project directory. Jsonnet takes precedence over plain JSON.

| File | Extension | Precedence | Evaluation |
|---|---|---|---|
| `.amnezigo.jsonnet` | `.jsonnet` | 1 (highest) | Jsonnet VM → JSON string → parsed as `Manifest` |
| `amnezigo.json` | `.json` | 2 | Parsed directly as JSON → `Manifest` |

If neither file exists, `LoadManifest` returns an error: `no manifest file found in <dir> (expected .amnezigo.jsonnet or amnezigo.json)`.

**Loader API (`loader.go`):**

| Function | Signature | Notes |
|---|---|---|
| `LoadManifest` | `(dir string, jpathDirs []string) (Manifest, error)` | Discovery by directory; Jsonnet precedence. `jpathDirs` defaults to `[dir/lib]` when nil/empty. |
| `LoadManifestFromFile` | `(path string, jpathDirs []string) (Manifest, error)` | Explicit path; `.jsonnet` extension ⇒ Jsonnet VM, otherwise plain JSON. `jpathDirs` defaults to `[parentDir/lib]`. |

> **Tip:** `--jpath` on `amnezigo generate` adds Jsonnet library search paths (for `import "presets.libsonnet"`). See [Jsonnet](./jsonnet.md).

## Related

- [Manifest Examples](./manifest-examples.md) — copy-paste JSON manifests (minimal, multi-peer, preset-based, IPv6).
- [Obfuscation](./obfuscation.md) — full S/H/J param semantics, ranges, and constraints.
- [Presets](./presets.md) — 7 ready-made S/H/J/Jc/Jmin/Jmax value sets to copy into `obfuscation`.
- [Jsonnet](./jsonnet.md) — `.amnezigo.jsonnet` evaluation, `--jpath`, and `presets.libsonnet` usage.
- [Output Format](./output-format.md) — INI structure, `#_`-prefixed metadata lines, and how `HeaderRange` serializes.
- [CLI Reference](./cli-reference.md) — `generate` flags (`--project`, `--jpath`, `--peer`).
