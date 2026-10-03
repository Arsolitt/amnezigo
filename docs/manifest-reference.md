# Manifest Reference

> Complete field reference for `amnezigo.json` / `.amnezigo.jsonnet` — the single declarative file that drives `amnezigo generate`.

## Table of Contents

- [Top-level `Manifest`](#top-level-manifest)
- [Network configuration (`NetworkConfig`)](#network-configuration-networkconfig)
- [Obfuscation profile (`ObfuscationManifest`)](#obfuscation-profile-obfuscationmanifest)
- [Peer declaration (`PeerManifest`)](#peer-declaration-peermanifest)
- [Header range (`HeaderRange`)](#header-range-headerrange)
- [U16Range](#u16range)
- [Pointer-nil semantics](#pointer-nil-semantics)
- [Manifest discovery](#manifest-discovery)
- [Related](#related)

---

## Top-level `Manifest`

The root object parsed from the manifest file. It declares one schema version, shared network settings, a shared obfuscation profile, and a flat map of peers.

| Field | JSON key | Go type | Required | Default | Description |
| --- | --- | --- | --- | --- | --- |
| `Version` | `version` | `int` | **yes** | — | JSON schema version; **must be `1`**. No `omitempty` — `{}` decodes to `0` and is rejected. |
| `Network` | `network` | `NetworkConfig` | no | zero-value | Shared interface settings (MTU, DNS). See [Network configuration](#network-configuration-networkconfig). |
| `Obfuscation` | `obfuscation` | `ObfuscationManifest` | no | zero-value | Shared obfuscation profile, including the AWG 3.x transport-protection knobs. |
| `Peers` | `peers` | `map[string]PeerManifest` | **yes** | — | Flat peer map. The **map key is the peer name** (used as the output directory name, `output/<peer>/awg0.conf`); there is no `name` field on `PeerManifest`. |

**Version validation** (`loader.go`; `<path>` is the manifest file path):

| `version` value | Result |
| --- | --- |
| `0` (or absent) | error: `<path>: missing or zero version field` |
| `1` | OK (only supported schema) |
| any other | error: `<path>: unsupported schema version <N> (expected 1)` |

> **Note:** The manifest **schema** version (`version: 1`) is independent of the AWG **protocol** generation (`obfuscation.awg_version`). See [Transport Protection (AWG 3.x)](./transport-protection.md) for the generation model.

> **Note:** Unknown top-level and nested JSON keys are silently ignored during parsing, so a typo in a field name does not fail loading and future schema fields do not break older binaries. Verify key spellings against this reference.

**Helper methods on `*Manifest`:**

| Method | Returns | Notes |
| --- | --- | --- |
| `ServerPeer()` | `(name string, count int)` | `count` is the number of peers satisfying `IsServer()`. A valid manifest has `count == 1`; when `count > 1`, `name` is one of the servers (arbitrary — map iteration order), so check `count` first. `generate` calls this method, not `ServerPeerName()`. |
| `ServerPeerName()` | `string` | **Panics** with `ServerPeerName: expected exactly 1 server peer, found <N>` when `count != 1`. Use it only after you have verified there is exactly one server peer; library validation code should use `ServerPeer()`. |
| `PeerNames()` | `[]string` | All peer names sorted alphabetically. `generate` does not call this method; it builds and sorts its own peer-name slices, with the same resulting order. |

Minimal shape (the AWG 3.1 defaults):

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": { "awg_version": "3.1" },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "phone":  { "address": "10.0.0.2/32" }
  }
}
```

More worked manifests live on the [Manifest Examples](./manifest-examples.md) page.

## Network configuration (`NetworkConfig`)

Global because AWG/WG MTU is per-interface (not per-peer) and DNS applies uniformly to client configs.

| Field | JSON key | Go type | Required | Default | Description |
| --- | --- | --- | --- | --- | --- |
| `MTU` | `mtu` | `int` | no | `0` in the manifest; resolves to `1280` in the emitted configs | Per-interface MTU; shared by the server and every client. |
| `DNS` | `dns` | `[]string` | no | `nil` | Upstream resolvers, joined with `", "` into the `DNS =` line of **client** configs only. The server config never contains a `DNS` line. |

> **Note:** The `1280` default is applied by the generate pipeline, not by the loader — `NetworkConfig.MTU` is a plain `int`, so unset stays `0` until generation time.

> **Warning:** Leaving `network.mtu` unset does not fully default. The emitted configs get `MTU = 1280`, but client-side CPS generation still reads the raw manifest value (`0`), so `I1`–`I5` collapse to the bare `<t>` fallback (4 bytes) and lose their protocol mimicry. Set `mtu` explicitly until this is fixed.

## Obfuscation profile (`ObfuscationManifest`)

Shared obfuscation profile. S/H/J parameters use pointer types so an absent key means "generate a value"; the AWG 3.x fields use the same convention on top of a version model, where a nil value selects the version default. Ready-made value sets are on the [Presets](./presets.md) page; the CPS grammar and constraint details of S/H/J are on the [Obfuscation](./obfuscation.md) page.

### S/H/J parameters

| Field | JSON key | Go type | Default (`nil`) | Description |
| --- | --- | --- | --- | --- |
| `S1` | `s1` | `*int` | generated | S-prefix 1. Generated draw: `[12, 65)` under header protection, `[0, 65)` otherwise. An explicit `0` is **regenerated** (zeros are treated as unset for the value), though the key still counts as present for the uniform-S shortcut. |
| `S2` | `s2` | `*int` | generated | S-prefix 2. Same rules as `S1`. |
| `S3` | `s3` | `*int` | generated | S-prefix 3. Same rules as `S1`. |
| `S4` | `s4` | `*int` | generated | S-prefix 4. Draw is capped lower: `[12, 33)` under header protection, `[0, 33)` otherwise. |
| `H1`–`H4` | `h1`–`h4` | `*HeaderRange` | generated | Header ranges. Generated ranges are non-overlapping, with `Min >= 5`, `Max <= 2147483647`, span `>= 10000000`, and never include the WG message type-ids 1–4. With header protection on and **all four** H unset, the fixed `1-1`/`2-2`/`3-3`/`4-4` set is used instead. |
| `Jc` | `jc` | `*int` | generated | Junk packet count. Generator range `[0, 10]`; generation retries until non-zero unless all three junk fields are explicit. |
| `Jmin` | `jmin` | `*int` | generated | Junk minimum. Range `[64, 1024]`; must not collide with the padded/raw WG sizes. |
| `Jmax` | `jmax` | `*int` | generated | Junk maximum. Range `[64, 1024]`, with `Jmin < Jmax`. |
| `Protocol` | `protocol` | `string` | — | **Decorative at the manifest level — NOT consumed by `generate`.** Only the per-peer `peers[].protocol` field drives I-packet shape (defaults to `quic`). No code path reads this field. |

The four padded handshake sizes — `S1+148`, `S2+92`, `S3+64`, `S4+32` — must be pairwise distinct (six comparisons; `S1+56 != S2` is only one of them). Generated draws are retried until the invariant holds; a fully pinned set is not checked against it during generation (only the S `>= 12` floor is still enforced, and `validate` reports violations later).

**Generated-value rules and pinned values:**

- **Header-protection floor.** With header protection active, all four S values must be `>= 12`, because the header cipher encrypts the 4-byte message header at offset `S<n>` using a ChaCha20 nonce made from the first 12 bytes of the packet. An explicit value below 12 aborts generation with `header protection requires S1-S4 >= 12 (got S<N>=<value>)`; generated values start at 12.
- **Uniform S.** When `random_trailers` is on and **all four `s1`–`s4` keys are absent**, one value is drawn and applied to `S1 = S2 = S3 = S4` (from `[12, 33)` under header protection; the 3.1 receiver classifies packet types by size). Setting any one of the four keys — including an explicit `0`, which is otherwise regenerated — disables the uniform shortcut, and the other fields fall back to independent draws.
- **H shortcut.** The fixed `1-1`/`2-2`/`3-3`/`4-4` default applies only when header protection is on **and all four** H fields are nil. If even one is explicit, the other three are drawn as large generated ranges (which never intersect the type-ids 1–4), so a lone `h1: {min: 1, max: 1}` does not imply `2-2`/`3-3`/`4-4`.
- **Pinned values are verbatim.** Explicit non-zero S/H/J values are used as-is; the pipeline enforces only the S floor. A hand-picked set that violates the size invariants is reported later by `validate`/`analyze` rather than enforced during generation — see [Validation & Analysis](./validation.md).

### AWG 3.x transport protection

Ten fields control the AWG 3.0/3.1 transport-protection layer. The generation gates are hard errors, not silent ignores. Defaults and gates are resolved by the pipeline, not by the loader.

| Field | JSON key | Go type | Default | Minimum `awg_version` | Description |
| --- | --- | --- | --- | --- | --- |
| `AWGVersion` | `awg_version` | `string` | `""` selects `"3.1"` | — | Target protocol generation. Accepts exactly `""`, `"2.0"`, `"3.0"`, `"3.1"`. |
| `HeaderProtection` | `header_protection` | `*bool` | `true` (3.0/3.1) | `"3.0"` | `false` removes `HeaderProtectionKey` from the output and lifts the S `>= 12` floor. |
| `ContentPadding` | `content_padding` | `*U16Range` | `{2, 10}` | `"3.0"` | Content-padding addition range. |
| `RekeyAfterTime` | `rekey_after_time` | `*U16Range` | `{120, 180}` | `"3.0"` | Rekey-after-time seconds. |
| `RekeyTimeout` | `rekey_timeout` | `*U16Range` | `{5, 8}` | `"3.0"` | Rekey timeout seconds. |
| `RejectAfterTime` | `reject_after_time` | `*U16Range` | `{180, 240}` | `"3.0"` | Reject-after-time seconds. |
| `KeepaliveTimeout` | `keepalive_timeout` | `*U16Range` | `{8, 12}` | `"3.0"` | Keepalive timeout seconds. |
| `MaxHandshakeAttempts` | `max_handshake_attempts` | `*U16Range` | `{16, 20}` | `"3.0"` | Maximum handshake attempts. |
| `RandomTrailers` | `random_trailers` | `*bool` | `true` (3.1 only) | `"3.1"` | Append random trailers. Always `false` under 2.0/3.0. |
| `DisableCookies` | `disable_cookies` | `*bool` | `true` (3.1 only) | `"3.1"` | Disable cookie replies. Always `false` under 2.0/3.0. |

**Version gates.** A 3.x field present in the manifest under a lower target version fails generation with:

```text
obfuscation.<field> requires awg_version <version> or later (got "<got>")
```

The gate is triggered by the field's **presence**, not its value — for example, `"header_protection": false` with `awg_version: "2.0"` is still an error.

**`awg_version` errors.** Any value outside the accepted set fails with `unsupported awg_version "<value>" (expected "2.0", "3.0", or "3.1")`. The field is a JSON string; a non-string value fails during JSON parsing. The CLI wraps resolution errors as `generating configs: resolve obfuscation: ...`.

**Range semantics.** A nil range field selects the 3.0+ default; an explicit `{"min": 0, "max": 0}` disables the corresponding engine key (it is omitted from the emitted INI and the engine keeps its own default). Invalid ranges fail with:

| Condition | Error |
| --- | --- |
| Exactly one zero bound | `obfuscation.<field>: bounds must both be zero or both non-zero (got <min>-<max>)` |
| `max < min` | `obfuscation.<field>: max (<max>) is below min (<min>)` |

> **Note:** Setting `header_protection: false` does not disable the other 3.x keys. The six ranges and the 3.1 booleans are still emitted, and the H ranges switch from the fixed `1-1`…`4-4` set to large generated ranges.

> **Warning:** Under the default 3.1 target, the header-protection key is generated once and persisted inside the emitted server config; header-protection and 3.1 behavior therefore depend on the previous `output/<server>/awg0.conf`. See [Transport Protection (AWG 3.x)](./transport-protection.md) for persistence and reset semantics.

### Helper methods on `*ObfuscationManifest`

| Method | Returns | Notes |
| --- | --- | --- |
| `HasAnyValue()` | `bool` | `true` when any of the eleven S/H/J pointer fields is non-nil; the 3.x fields are deliberately ignored. `false` does **not** mean "fully random generation" under 3.x — S may be drawn uniformly, H takes the fixed default, and the 3.x keys are still emitted. The pipeline never calls this helper; it reads each field directly. |
| `ToSharedObfuscation()` | `ServerObfuscationConfig` | Dereferences non-nil S/H/J pointers only. It leaves `Version` at its zero value and drops every 3.x field, so the result is not a resolvable 3.x config. |

## Peer declaration (`PeerManifest`)

Declares a single peer. The peer **name** is the `peers` map key, not a struct field.

| Field | JSON key | Go type | Required | Default | Description |
| --- | --- | --- | --- | --- | --- |
| `Address` | `address` | `string` | **yes** | — | Interface address as CIDR (e.g. `10.0.0.1/24` for the server, `10.0.0.2/32` for a client). The only field without `omitempty`. |
| `Endpoint` | `endpoint` | `string` | no | `""` | `host:port`. Server marker (together with non-zero `listen_port`). Client configs take the endpoint from the server peer. |
| `ListenPort` | `listen_port` | `int` | no | `0` | UDP listen port. Server marker (together with non-empty `endpoint`). |
| `Protocol` | `protocol` | `string` | no | `quic` | **Per-peer obfuscation protocol** — this is the field `generate` actually reads. One of `quic`, `dns`, `dtls`, `stun`, `sip`, `rtp`, `random`. |
| `Keepalive` | `keepalive` | `*int` | no | `nil` | `PersistentKeepalive` seconds for the client's `[Peer]`. Both `nil` and explicit `0` emit `PersistentKeepalive = 0`. Ignored on the server peer. |
| `TunName` | `tun_name` | `string` | no | `awg0` (server peer) | Server interface name; written as `#_TunName` metadata. Ignored on client peers. |
| `MainIface` | `main_iface` | `string` | no | `""` | Server egress interface (e.g. `eth0`). When set, `PostUp`/`PostDown` iptables and ip6tables rules are generated. Ignored on client peers. |
| `DisplayName` | `display_name` | `string` | no | `""` | Displayed server name in the AmneziaVPN app when the client config is imported via a `vpn://` link; empty omits the link's `description` field, letting the app fall back to `hostName`. Consumed by the link encoder only — it never appears in `awg0.conf`. See [VPN Import Links](./vpn-links.md). |

> **Note:** There is no `PresharedKey`, `PublicKey`, or `PrivateKey` field on `PeerManifest` — crypto material is generated and persisted by the credentials layer, not declared in the manifest.

**Server detection rule** (`PeerManifest.IsServer()`, exact):

```go
return p.Endpoint != "" && p.ListenPort != 0
```

A peer with only one of `endpoint` / `listen_port` set is a **client**. A manifest must contain **exactly one** server peer: zero or two-plus servers make `generate` fail with `exactly one server peer required, found <N>`. In particular, a client peer that also sets both `endpoint` and `listen_port` becomes a second server and aborts generation; adding `listen_port` alone does not change its role.

## Header range (`HeaderRange`)

Inclusive `min`–`max` pair used by the H1–H4 obfuscation headers.

| Field | JSON key | Go type | Description |
| --- | --- | --- | --- |
| `Min` | `min` | `uint32` | Range lower bound (inclusive). |
| `Max` | `max` | `uint32` | Range upper bound (inclusive). |

```json
"h1": { "min": 100, "max": 5000000 }
```

H ranges are always serialized in INI as the two-part `Min-Max` form, even when `Min == Max` — the header-protected defaults appear as `H1 = 1-1` … `H4 = 4-4`. Both the server and every client config store the ranges, not resolved point values. See [Output Format](./output-format.md).

## U16Range

Inclusive `uint16` range used by the six AWG 3.x range parameters (`content_padding` and the five timers). It mirrors the engine's own notation.

| Field | JSON key | Go type | Description |
| --- | --- | --- | --- |
| `Min` | `min` | `uint16` | Range lower bound (inclusive); values above 65535 do not fit. |
| `Max` | `max` | `uint16` | Range upper bound (inclusive). |

```json
"content_padding": { "min": 2, "max": 10 }
```

- The **zero value `{0, 0}` is the disabled state**: `IsZero()` is true and the key is omitted from the emitted INI, so the engine keeps its default. Because `{0,0}` is reserved, a range cannot start at 0 (`{0, 5}` is an error).
- A **missing bound decodes to `0`**, so an object whose present bound is also zero yields the disabled `{0, 0}` state: `{}` and `{"min": 0}` both disable the key rather than selecting the default. A single non-zero bound is not the zero value — `{"min": 5}` fails with `obfuscation.<field>: bounds must both be zero or both non-zero (got 5-0)`. To keep the default, omit the field entirely.
- `String()` renders `N` when `Min == Max` and `Min-Max` otherwise; the writer uses this rendering for the INI value.

## Pointer-nil semantics

The pointer fields exist to distinguish "user set this to `0`" from "user did not set this field" — JSON `"s1": 0` is nominally different from omitting `s1`. For S values the distinction is lost at the value level (a zero is regenerated), but pointer presence still matters: the uniform-S shortcut checks whether the keys are absent, not whether they are zero.

| Pointer type | `nil` (omitted) | explicit value |
| --- | --- | --- |
| `*int` — `S1`–`S4` | generated | pinned if non-zero; an explicit `0` is regenerated like a missing value, but still disables the uniform-S shortcut |
| `*int` — `Jc`, `Jmin`, `Jmax` | generated for the missing fields | when **all three** are explicit they are copied verbatim (including `jc: 0`); if any one is nil, the missing fields are generated and a zero `jc` is replaced |
| `*HeaderRange` — `H1`–`H4` | generated; fixed `1-1`…`4-4` under header protection when all four are nil | pinned range (used verbatim) |
| `*U16Range` — the six 3.x ranges | version default | pinned range; `{0, 0}` disables the key; mixed zero bounds and `max < min` are errors |
| `*bool` — `header_protection`, `random_trailers`, `disable_cookies` | version default | overrides the default |
| `*int` — `Keepalive` | `PersistentKeepalive = 0` (off) | `PersistentKeepalive = <value>` |

## Manifest discovery

The loader discovers the manifest from a project directory. Jsonnet takes precedence over plain JSON.

| File | Extension | Precedence | Evaluation |
| --- | --- | --- | --- |
| `.amnezigo.jsonnet` | `.jsonnet` | 1 (highest) | Jsonnet VM → JSON string → parsed as `Manifest` |
| `amnezigo.json` | `.json` | 2 | Parsed directly as JSON → `Manifest` |

If neither file exists, `LoadManifest` returns an error: `no manifest file found in <dir> (expected .amnezigo.jsonnet or amnezigo.json)`.

**Loader API (`loader.go`):**

| Function | Signature | Notes |
| --- | --- | --- |
| `LoadManifest` | `(dir string, jpathDirs []string) (Manifest, error)` | Discovery by directory; Jsonnet wins outright when both files exist. `jpathDirs` defaults to `[dir/lib]` when nil/empty. |
| `LoadManifestFromFile` | `(path string, jpathDirs []string) (Manifest, error)` | Explicit path; a `.jsonnet` suffix routes through the Jsonnet VM, anything else through plain JSON. `jpathDirs` defaults to `[parentDir/lib]`. |

Loader behavior worth knowing:

- An explicit `jpathDirs` (or `--jpath` on `amnezigo generate`) **replaces** the implicit `<dir>/lib` default rather than appending to it; relative entries resolve against the process working directory. These are user-created library directories — amnezigo bundles no Jsonnet libraries.
- `.jsonnet` detection is a case-sensitive suffix check, and `LoadManifest` only ever looks for the exact filenames `.amnezigo.jsonnet` and `amnezigo.json`.
- File I/O and parsing failures are wrapped with the path: `read <path>: <err>` and `parse <path>: <err>` for JSON, `evaluate jsonnet <path>: <err>` and `parse jsonnet output from <path>: <err>` for Jsonnet.
- `Manifest.Version` is validated only by these loaders. A `Manifest` built in Go with `Version: 0` generates fine.

See [Jsonnet](./jsonnet.md) for `.amnezigo.jsonnet` evaluation and library imports.

## Related

- [Transport Protection (AWG 3.x)](./transport-protection.md) — the 3.x layer end-to-end: version model, INI key emission, header protection, persistence.
- [Manifest Examples](./manifest-examples.md) — worked manifests: minimal, 3.1 defaults, pinned 2.0, disabled ranges, Jsonnet.
- [Obfuscation](./obfuscation.md) — S/H/J semantics, CPS grammar, and protocol templates.
- [Output Format](./output-format.md) — `awg0.conf` key order, `#_` metadata, and server/client differences.
- [Jsonnet](./jsonnet.md) — `.amnezigo.jsonnet` evaluation, `--jpath`, and user-supplied `lib/` library imports.
- [Validation & Analysis](./validation.md) — finding codes (including HPK001 and TRL001) and exit codes.
