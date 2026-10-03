# Transport Protection (AWG 3.x)

> The AWG 3.x device-level transport-protection layer: version gating, the ten manifest knobs, header-protection key lifecycle, the nine INI keys, engine semantics, and validation codes.

## Table of Contents

- [What transport protection is](#what-transport-protection-is)
- [Version model](#version-model)
- [Manifest knobs](#manifest-knobs)
- [Resolution semantics](#resolution-semantics)
- [INI keys and emission](#ini-keys-and-emission)
- [Engine and runtime semantics](#engine-and-runtime-semantics)
- [Validation](#validation)
- [Ecosystem integration](#ecosystem-integration)
- [Worked example](#worked-example)
- [Pitfalls](#pitfalls)
- [Related](#related)

---

## What transport protection is

AmneziaWG 2.0 [obfuscation](./obfuscation.md) shapes the outside of a tunnel: junk packets (`Jc`, `Jmin`, `Jmax`), size prefixes (`S1`–`S4`), header ranges (`H1`–`H4`), and custom packet strings (`I1`–`I5`). AWG 3.x adds a second, device-level layer on top:

- **Header protection** encrypts the 4-byte WireGuard message header with a symmetric ChaCha20 key, so the message type is no longer visible on the wire.
- **Content padding** and **random trailers** adjust packet sizes beyond the handshake.
- **Timer ranges** jitter the WireGuard rekey, rekey-timeout, reject, keepalive, and handshake-attempt constants.
- **Cookie replies** can be disabled entirely.

Unlike the 2.0 parameters, these are device-level keys: one `[Interface]` block, shared verbatim by the server and every client.

| Generation | `awg_version` | Adds |
| --- | --- | --- |
| AmneziaWG 2.0 | `"2.0"` | Jc/Jmin/Jmax, S1–S4, H1–H4 ranges, I1–I5 |
| AmneziaWG 3.0 | `"3.0"` | `HeaderProtectionKey`, `ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts` |
| AmneziaWG 3.1 | `"3.1"` (default) | `RandomTrailers`, `DisableCookies` |

The target generation is a hard output gate, not a hint. Older engine builds reject INI keys they do not know (`Line unrecognized` in amneziawg-tools `src/config.c`), so a 3.x key must never appear in a config aimed at a lower generation. Two mechanisms enforce that:

- `checkObfuscationVersionGates` in the resolution pipeline turns a 3.x manifest field under a lower `awg_version` into a hard error instead of silently dropping it.
- `writeTransportProtectionKeys` in the writer returns before emitting anything when `Version < AWG30`; the zero `AWGVersion` also keeps hand-built library configs on the exact 2.0 key set.

> **Warning:** The version gate is by field *presence*, not value: setting `header_protection: false` while `awg_version` is `"2.0"` still fails resolution.

## Version model

`version.go` defines the model as `type AWGVersion int`:

| Constant | Value | Label |
| --- | --- | --- |
| `AWG20` | 1 | `2.0` |
| `AWG30` | 2 | `3.0` |
| `AWG31` | 3 | `3.1` |

- `DefaultAWGVersion = AWG31`; an unset (or empty) `obfuscation.awg_version` resolves to 3.1.
- `ParseAWGVersion` accepts exactly `""`, `"2.0"`, `"3.0"`, `"3.1"`. Anything else fails with `unsupported awg_version %q (expected "2.0", "3.0", or "3.1")` — including values such as `"2"`, `"4.0"`, `"3.10"`, `"v3.1"`, and `"latest"`.
- `AWGVersion.String()` returns the dotted label, or `AWGVersion(%d)` for out-of-range values.
- The resolved version is stored in `ServerObfuscationConfig.Version`, so the writer, parser, and validator never need the manifest.

The zero value (`0`) sorts below `AWG20`, which is deliberate: a hand-built `ServerObfuscationConfig` that never sets `Version` emits exactly the legacy 2.0 key set. See [Library Usage](./library-usage.md) for the Go API.

## Manifest knobs

All ten knobs live in the `obfuscation` object and are optional; see [Manifest Reference](./manifest-reference.md) for the surrounding schema. `nil`/absent selects the version default, and the three booleans default to `true` for every version that understands them.

| JSON key | Go type | Absent resolves to | Minimum version |
| --- | --- | --- | --- |
| `awg_version` | `string` | `3.1` | — |
| `header_protection` | `*bool` | `true` | 3.0 |
| `content_padding` | `*U16Range` | `{2, 10}` | 3.0 |
| `rekey_after_time` | `*U16Range` | `{120, 180}` | 3.0 |
| `rekey_timeout` | `*U16Range` | `{5, 8}` | 3.0 |
| `reject_after_time` | `*U16Range` | `{180, 240}` | 3.0 |
| `keepalive_timeout` | `*U16Range` | `{8, 12}` | 3.0 |
| `max_handshake_attempts` | `*U16Range` | `{16, 20}` | 3.0 |
| `random_trailers` | `*bool` | `true` | 3.1 |
| `disable_cookies` | `*bool` | `true` | 3.1 |

`U16Range` is `{"min": uint16, "max": uint16}` (both bounds are `uint16`, so values above 65535 fail to unmarshal). Its semantics:

- An explicit `{"min": 0, "max": 0}` **disables** the key: the writer omits it and the engine keeps its default.
- A missing bound decodes as `0`, so `{"min": 0}` is `{0, 0}` — disabled, not "0 as the lower bound". `{"min": 0, "max": 5}` is a hard error.
- Mixed zero bounds fail with `obfuscation.<field>: bounds must both be zero or both non-zero (got %d-%d)`.
- Inverted bounds fail with `obfuscation.<field>: max (%d) is below min (%d)`.

What the same manifest resolves to per generation:

| Parameter | `"2.0"` | `"3.0"` | `"3.1"` |
| --- | --- | --- | --- |
| `header_protection` | off (field forbidden) | on (key generated) | on (key generated) |
| `content_padding` | off (no key) | `2-10` | `2-10` |
| five timer ranges | off (no keys) | defaults | defaults |
| `random_trailers` | off (field forbidden) | off (field forbidden) | on |
| `disable_cookies` | off (field forbidden) | off (field forbidden) | on |
| Emitted 3.x keys | none | key + six ranges | all nine |

Version gates are checked before anything else and report the first offending field, in declaration order:

| Field | Minimum | Error |
| --- | --- | --- |
| `header_protection` | 3.0 | `obfuscation.header_protection requires awg_version 3.0 or later (got "2.0")` |
| `content_padding` | 3.0 | `obfuscation.content_padding requires awg_version 3.0 or later (got "2.0")` |
| `rekey_after_time` | 3.0 | `obfuscation.rekey_after_time requires awg_version 3.0 or later (got "2.0")` |
| `rekey_timeout` | 3.0 | `obfuscation.rekey_timeout requires awg_version 3.0 or later (got "2.0")` |
| `reject_after_time` | 3.0 | `obfuscation.reject_after_time requires awg_version 3.0 or later (got "2.0")` |
| `keepalive_timeout` | 3.0 | `obfuscation.keepalive_timeout requires awg_version 3.0 or later (got "2.0")` |
| `max_handshake_attempts` | 3.0 | `obfuscation.max_handshake_attempts requires awg_version 3.0 or later (got "2.0")` |
| `random_trailers` | 3.1 | `obfuscation.random_trailers requires awg_version 3.1 or later (got "3.0")` |
| `disable_cookies` | 3.1 | `obfuscation.disable_cookies requires awg_version 3.1 or later (got "3.0")` |

The `(got "...")` suffix echoes the *resolved* version label, and `generate` wraps the error with `generating configs: resolve obfuscation: `:

```shell
$ amnezigo generate --project ./myproject
Error: generating configs: resolve obfuscation: obfuscation.content_padding requires awg_version 3.0 or later (got "2.0")
```

> **Note:** The 3.x knobs are not part of `HasAnyValue()` or `ToSharedObfuscation()`; those projection helpers still cover only S/H/J, so library code must read the 3.x fields directly.

## Resolution semantics

`resolveObfuscation` merges explicit manifest values with generated ones in a fixed order:

1. Parse `awg_version` (unset → 3.1).
2. Run the version gates.
3. Derive `headerProtection`, `randomTrailers`, `disableCookies` from the version, overridden by explicit pointers.
4. Seed explicit `S1`–`S4`.
5. Pick `minS`: `0` for 2.0, `headerProtectionNonceSize` (12) when header protection is on.
6. Decide whether S generation is uniform.
7. Fill missing S prefixes.
8. Check the S floor.
9. Fill missing header ranges.
10. Reuse or generate the header-protection key.
11. Apply the six 3.x ranges (version 3.0+ only).
12. Assign `RandomTrailers` and `DisableCookies`.
13. Fill missing junk parameters.

Because step 8 runs before steps 9–13, an S-floor violation is the error reported even when other fields are also wrong.

### Resolution flags

| Flag | 2.0 | 3.0 | 3.1 | Explicit override |
| --- | --- | --- | --- | --- |
| `headerProtection` | false | true | true | `header_protection` |
| `randomTrailers` | false | false | true | `random_trailers` |
| `disableCookies` | false | false | true | `disable_cookies` |

A flag can be turned off, but never forced on below its minimum version — the gates reject the field instead of ignoring it.

### S-prefix floor: 12

Header protection encrypts the 4-byte header at offset `S{n}` with a ChaCha20 nonce built from the first 12 bytes of the packet, so the engine refuses `S < 12` whenever a header-protection key is present. amnezigo encodes this as `headerProtectionNonceSize = 12` and enforces it during resolution:

```text
header protection requires S1-S4 >= 12 (got S3=8)
```

An explicit `s3: 8` under 3.1 makes `generate` exit 1 and write nothing:

```shell
$ amnezigo generate --project ./myproject
Error: generating configs: resolve obfuscation: header protection requires S1-S4 >= 12 (got S3=8)
```

Generated values always respect the floor: `S1`–`S3` are drawn from `[minS, 65)` and `S4` from `[minS, 33)`, and the four padded handshake sizes must be pairwise distinct. When header protection is off, `minS` is `0` and any `S` is legal — a 2.0 config can carry `S4 = 8`.

### Uniform S under random trailers

When `random_trailers` resolves to `true` **and all four `S` fields are absent**, amnezigo draws one value in `[minS, 33)` and uses it for `S1 = S2 = S3 = S4`. Equal size prefixes keep the four padded handshake sizes (`S+148`, `S+92`, `S+64`, `S+32`) unambiguous for the receiver's size-based packet classification.

The uniform path is skipped as soon as **any** `S` field is explicit — surviving values are then drawn independently (`S1`–`S3` from `[minS, 65)`, `S4` from `[minS, 33)`), which is exactly the case `TRL001` warns about. The rule also applies with header protection off, where `minS` is `0` and the drawn value can be as low as `1`.

### Header ranges under header protection

With header protection on and all four `H` fields absent, amnezigo emits the reference ranges:

```ini
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
```

The 4-byte message type is encrypted, so the WireGuard type-ids `1..4` are no longer observable and no longer have to be avoided. If even one `H` field is explicit, the remaining fields come from `GenerateHeaderRanges()` (wide random ranges that never intersect `1..4`) and the explicit values are kept verbatim. amnezigo does not check whether the resulting ranges overlap; the engine does, at `awg setconf` time, with `headers must not overlap`. With header protection off, generated wide ranges are always used.

### Header-protection key lifecycle

`GenerateHeaderProtectionKey` draws 32 bytes from `crypto/rand` and base64-encodes them with `StdEncoding`: a 44-character string with one `=` pad. The key is **not clamped** — it is a symmetric ChaCha20 key, not a Curve25519 scalar.

- The key is recovered from the previous run's server config (`PersistedCredentials.HeaderProtectionKey`) and reused whenever `--full-reset` is not set and the persisted value is non-empty.
- Otherwise a new key is generated. `--full-reset` rotates it together with every private key and PSK, so all previously distributed configs stop working.
- The same key is written to the server config and every client config.
- `--dry-run` still loads the persisted key (it does not rotate it); on a first run with no output tree it generates a fresh one that the next real run will not reuse.

> **Danger:** The emitted server config is the only place the key is persisted. A run that emits no key (target 2.0, or `header_protection: false`) drops it from the persisted state; switching back to 3.x generates a **new** key, and every client still holding the old one stops completing handshakes. See [Credentials & Key Reuse](./credentials.md).

## INI keys and emission

The nine keys are defined once in `transport_keys.go` — the exact spelling amneziawg-tools `src/config.c` parses — and shared by the writer, the parser, and the validators.

| INI key | Value form | Emitted when | Meaning |
| --- | --- | --- | --- |
| `HeaderProtectionKey` | 44-char base64 of 32 bytes | header protection active and key non-empty | ChaCha20 key for the 4-byte message header |
| `ContentPaddingAddition` | `N` or `N-M` | range non-zero, version ≥ 3.0 | content-padding amount (bytes) |
| `RekeyAfterTime` | `N` or `N-M` | range non-zero, version ≥ 3.0 | seconds; jitter around WireGuard's 120 s rekey |
| `RekeyTimeout` | `N` or `N-M` | range non-zero, version ≥ 3.0 | seconds; around the 5 s rekey timeout |
| `RejectAfterTime` | `N` or `N-M` | range non-zero, version ≥ 3.0 | seconds; around the 180 s reject |
| `KeepaliveTimeout` | `N` or `N-M` | range non-zero, version ≥ 3.0 | seconds; around the 10 s keepalive |
| `MaxHandshakeAttempts` | `N` or `N-M` | range non-zero, version ≥ 3.0 | handshake attempts; around the 18-attempt constant |
| `RandomTrailers` | `on` / `off` | always at version ≥ 3.1 | random trailer padding |
| `DisableCookies` | `on` / `off` | always at version ≥ 3.1 | disable cookie replies |

Formatting rules:

- A range is written as a single decimal number when `Min == Max` and as `Min-Max` otherwise; the parser accepts both and normalises a round trip the same way.
- Booleans are written `on`/`off`; the parser also accepts `1`/`0` case-insensitively.
- `HeaderProtectionKey` is the only 3.x value the parser rejects structurally at load: it must decode to exactly 32 bytes, otherwise `invalid HeaderProtectionKey "<value>": must be 44-char base64 of 32 bytes`.

Omission rules:

- `Version < AWG30` → nothing at all is emitted (2.0, and the zero-value `AWGVersion`).
- Empty `HeaderProtectionKey` → no key line (this is how `header_protection: false` appears in the output).
- A zero (`{0,0}`) range → that range key is omitted.
- `Version < AWG31` → neither `RandomTrailers` nor `DisableCookies`; at 3.1 both are always emitted, `on` or `off`.

Emission order is fixed, and the block is written immediately after `H4`:

```text
HeaderProtectionKey          (only when non-empty)
ContentPaddingAddition       (skipped when zero)
RekeyAfterTime               (skipped when zero)
RekeyTimeout                 (skipped when zero)
RejectAfterTime              (skipped when zero)
KeepaliveTimeout             (skipped when zero)
MaxHandshakeAttempts         (skipped when zero)
RandomTrailers               (3.1 only, on/off)
DisableCookies               (3.1 only, on/off)
```

In the **server** config the block is followed by the `#_` metadata comments (`#_ClientToClient`, `#_TunName`, ...). In a **client** config it is followed by the non-empty `I1`–`I5` lines, then a blank line and `[Peer]`. The full layout is documented in [Output Format](./output-format.md).

A real 3.1 server `[Interface]` tail from a live run:

```ini
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = eOZZWAHNFdNElFTe1Iz+KnCP7eHPUx1NmA7TQnolol4=
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
```

The client config repeats the same nine lines, with the identical key, between `H4` and `I1`.

## Engine and runtime semantics

**Header cipher.** Header protection encrypts the 4-byte WireGuard message header at offset `S{n}` with ChaCha20; the nonce is the first 12 bytes of the S-prefix, which is why `S >= 12` is a hard requirement. The key is 32 bytes shared by both ends.

**Key mismatch.** Two peers with different keys never complete a handshake. The container e2e test replaces one client's key with a different valid 44-char key and attempts pings for 10 seconds: both `latest-handshakes` timestamps stay `0`, while an untouched control pair handshakes and pings in both directions.

**The header really is ciphertext.** For a handshake initiation, the UDP payload is exactly `148 + S1` bytes; with header protection the little-endian `uint32` at offset `S1` is outside `1..4`, while with `header_protection: false` it lies inside the configured `H1` range.

**Header ranges.** The engine validates the four ranges itself:

- With a header-protection key, type-ids `1..4` are legal because they are encrypted — hence the `1-1`, `2-2`, `3-3`, `4-4` recommendation.
- Overlapping ranges are rejected by `awg setconf` with `headers must not overlap`; amnezigo's validator does not check overlap.

**Random trailers.** When a receiver has trailers enabled, it classifies packets by size and only accepts sizes above the expected base size. Equal `S` values keep the four padded handshake sizes unambiguous; unequal `S` values can misclassify padded handshake packets, which is what `TRL001` warns about.

**Timer ranges** are handed to the engine verbatim (the runtime dump uses the same `N`/`N-M` notation), and an explicit `rekey_after_time: {"min": 5, "max": 6}` really does force a rehandshake within seconds.

**Disable cookies** turns off the 3.1 cookie replies.

**Cross-version behaviour.** A 3.1 engine applies a 2.0 config unchanged: the runtime dump reports `header_protection_key` = `(none)` and both booleans `off`, and the tunnel comes up. The reverse does not hold: a pre-3.x engine (or a kernel implementation without the transport layer) refuses the unknown keys with `Line unrecognized`, so keep the runtime aligned — see [Installation](./installation.md).

## Validation

`validate` and `generate` share these codes. Conditions and exact messages:

| Code | Severity | Condition | Message |
| --- | --- | --- | --- |
| `HPK001` | error | `HeaderProtectionKey` present and any of `S1`–`S4` < 12 | `header protection requires S1-S4 >= 12 (got S1=%d, S2=%d, S3=%d, S4=%d)` |
| `HPK003` | error | `HeaderProtectionKey` present but not base64 of exactly 32 bytes (library-built configs; the parser rejects malformed values at load) | `HeaderProtectionKey is not 44-char base64 of 32 bytes` |
| `TRL001` | warning | `RandomTrailers` is on and `S1..S4` are not all equal | `RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification` |
| `TRM001` | error | one of the six ranges has `Max < Min`, or exactly one bound is zero | `invalid <Key> range: max (%d) is below min (%d)` / `invalid <Key> range: bounds must both be zero or both non-zero (got %d-%d)` |

`TRM001` iterates `ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts` and reports at most one structural problem per range; a disabled `{0,0}` range never produces a finding.

Two related codes change behaviour under header protection:

- `HDR001` (WG type-ids in an `H` range) is **suppressed** when `HeaderProtectionKey != ""` and `Max >= Min`, because `1..4` is the intended configuration; `HDR002` (`invalid header range: Max (%d) < Min (%d)`) still fires.
- `RISK007` (`<Hn> range width is <n> (< 1M) — narrow header range reduces entropy`) is skipped entirely when a header-protection key is present, so `H1..H4 = 1-1..4-4` no longer looks like a defect. The size-based heuristics (`RISK001`, `RISK003`–`RISK006`, `RISK008`, `RISK009`) keep applying.

`generate` re-parses the bytes it is about to write and appends `ValidateServerConfig` findings, so 3.x recommendations reach the CLI as `Warnings: <n>` plus one line per finding:

```shell
$ amnezigo generate --project ./myproject
Generated 2 config(s):
  server/awg0.conf (745 bytes)
  phone/awg0.conf (1028 bytes)

Warnings: 1
[WARNING TRL001]  (key=RandomTrailers): RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification
```

An existing config with several problems, checked with `validate`:

```shell
$ amnezigo validate server/awg0.conf
[ERROR HPK001] server/awg0.conf (key=HeaderProtectionKey): header protection requires S1-S4 >= 12 (got S1=4, S2=12, S3=12, S4=12)
  Header protection encrypts the 4-byte message header at offset S{n}; the ChaCha20 nonce is the first 12 bytes of the packet.
[WARNING TRL001] server/awg0.conf (key=RandomTrailers): RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification
  The receiver classifies packets by size and only accepts size > expected when trailers are enabled, so unequal S values can misclassify padded handshake packets.
[ERROR TRM001] server/awg0.conf (key=ContentPaddingAddition): invalid ContentPaddingAddition range: bounds must both be zero or both non-zero (got 0-5)
✗ server/awg0.conf: 2 errors, 1 warnings, 0 info
```

`validate` exits 1 when at least one error is present; warnings alone exit 0 unless `--strict` is set. Findings from `generate` describe the **server** config only — point `validate` at a client config to check it. All codes are listed in [Validation & Analysis](./validation.md).

## Ecosystem integration

### vpn:// import links

With `--vpn-links`, the `last_config` object of each `vpn://` link carries the nine 3.x values as separate string fields whose JSON names equal the INI key names: `HeaderProtectionKey`, `ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts`, `RandomTrailers`, `DisableCookies`. The values are copied verbatim from the client INI and omitted for pre-3.x configs; they live inside the obfuscation-enabled branch, which requires `Jc` to be present. See [VPN Import Links](./vpn-links.md).

### Presets

Preset data is 3.1 by construction: `Preset.ToServerObfuscation()` sets `Version: AWG31` and copies `ContentPadding`, `RandomTrailers`, and `DisableCookies`, leaving `HeaderProtectionKey` for the pipeline to fill. Because four presets pin unequal `S` values with trailers on (home-balanced `30/35/20/12`, mobile-aggressive `60/60/50/24`, stealth-paranoid `30/24/20/32`, standard-1420 `32/28/20/16`), `generate` prints a `TRL001` warning for them. See [Presets](./presets.md).

### Container e2e proof

The `e2e` suite drives the real engine through Docker — pinned to amneziawg-go `v3.1.20260828` and amneziawg-tools `v3.1.20260812`. Run it with:

```shell
$ go test -tags=e2e ./e2e/...
```

The suite needs a reachable Docker daemon and a working `/dev/net/tun` inside containers; it builds the CLI and a local `amnezigo-e2e-awg:31` image once, skips when Docker or TUN is unavailable, and fails when the CLI or image build breaks. Against the real engine it proves the 3.1 defaults end to end (with a byte-identical `HeaderProtectionKey` on both ends), that a mismatched key never handshakes, that the wire header is ciphertext, that a 2.0 config still tunnels on the 3.1 engine, that an explicit rekey range forces a rehandshake, and that overlapping `H` ranges are rejected.

### Runtime alignment

The generated 3.x keys are engine features, so the runtime must be 3.x. The shipped Docker image is pinned to `amneziavpn/amneziawg-go:3.1.20260828`; an older runtime (or a kernel implementation without the transport layer) cannot load the keys. See [Installation](./installation.md) for the image and release matrix.

## Worked example

A minimal manifest that accepts every 3.1 default:

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": { "awg_version": "3.1" },
  "peers": {
    "server": { "address": "10.77.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "phone": { "address": "10.77.0.2/32" }
  }
}
```

> **Warning:** Set `network.mtu` explicitly. If it is omitted, the client config still shows `MTU = 1280`, but its `I1`–`I5` collapse to `<t>`, because the pipeline resolves the I-packets before applying the MTU default. See [Gotchas](./gotchas.md).

Resolution result:

| Parameter | Resolved value |
| --- | --- |
| `awg_version` | `3.1` |
| Header protection | on; key generated (44-char base64) |
| `S1`–`S4` | one uniform draw in `[12, 33)` (this run: `26, 26, 26, 26`) |
| `H1`–`H4` | `1-1`, `2-2`, `3-3`, `4-4` |
| `ContentPaddingAddition` | `2-10` |
| `RekeyAfterTime` | `120-180` |
| `RekeyTimeout` | `5-8` |
| `RejectAfterTime` | `180-240` |
| `KeepaliveTimeout` | `8-12` |
| `MaxHandshakeAttempts` | `16-20` |
| `RandomTrailers` | `on` |
| `DisableCookies` | `on` |

The emitted server config (`output/server/awg0.conf`):

```ini
[Interface]
PrivateKey = 4Ajr9QS/l7r+K+S/hykK952Vy1tSuzTz8rTpn2Cg81o=
PublicKey = eeKBWGQfsUrb/fYOV022l/oDFybje9q6HXM5UbTJa00=
Address = 10.77.0.1/24
ListenPort = 51820
MTU = 1280
Jc = 9
Jmin = 310
Jmax = 637
S1 = 26
S2 = 26
S3 = 26
S4 = 26
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = eOZZWAHNFdNElFTe1Iz+KnCP7eHPUx1NmA7TQnolol4=
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
PublicKey = mJ3uYthO4VkyvdqrfwHmIkwrWDRDOSkPeItHQYmhR28=
PresharedKey = Iu9Hj3tKeXrjn5DMPnOAGGXddOIP/vsqXm38ON5d7fs=
AllowedIPs = 10.77.0.2/32
```

The client config carries the same nine lines (identical `HeaderProtectionKey`) between `H4` and `I1`.

### Verifying with `awg show <if> dump`

After applying the config to a node running a 3.x engine, `awg show awg0 dump` prints one tab-separated interface record. Its fields are positional; in order they are:

```text
private_key  public_key  listen_port  jc  jmin  jmax
s1  s2  s3  s4  h1  h2  h3  h4  i1  i2  i3  i4  i5
header_protection_key  content_padding_addition
rekey_after_time  rekey_timeout  reject_after_time  keepalive_timeout
max_handshake_attempts  random_trailers  disable_cookies  fwmark
```

For this example, the nine fields that follow `i1`–`i5` read back as:

| Dump field | Expected value |
| --- | --- |
| `header_protection_key` | the same 44-char base64 string as in the config |
| `content_padding_addition` | `2-10` |
| `rekey_after_time` | `120-180` |
| `rekey_timeout` | `5-8` |
| `reject_after_time` | `180-240` |
| `keepalive_timeout` | `8-12` |
| `max_handshake_attempts` | `16-20` |
| `random_trailers` | `on` |
| `disable_cookies` | `on` |

The positions are contract, not convenience: the e2e harness maps them by name (`dumpFieldNames` in `e2e/harness_test.go`) and its dump assertions fail if upstream ever reorders the record. To print just the nine 3.x values:

```shell
$ awg show awg0 dump | awk -F'\t' '{print $20, $21, $22, $23, $24, $25, $26, $27, $28}'
```

(`awg show all dump` prefixes the interface name, shifting every position by one.)

### Variant: `awg_version: "2.0"`

The `obfuscation` object becomes:

```json
{ "awg_version": "2.0" }
```

All nine keys disappear from both configs, and `S`/`H` return to free-form generation — `S < 12` is legal again. A live run produced `S4 = 8` and wide header ranges:

```ini
S1 = 44
S2 = 61
S3 = 52
S4 = 8
H1 = 704806984-1179235240
H2 = 1227731607-1640762541
H3 = 1826020069-1872072608
H4 = 1881254146-1937782749
#_ClientToClient = false
```

A 3.1 engine still accepts this config: the dump reports `header_protection_key` = `(none)` and both booleans `off`.

### Variant: `header_protection: false` under 3.1

The `obfuscation` object becomes:

```json
{ "awg_version": "3.1", "header_protection": false }
```

Only the key line is lost. The six ranges and both 3.1 booleans are still emitted, and the header ranges switch to generated wide ranges. Because header protection is off, `minS` is `0` and the uniform draw can go below 12 — a live run drew `S1..S4 = 1`:

```ini
S1 = 1
S2 = 1
S3 = 1
S4 = 1
H1 = 18680864-345187686
H2 = 958273519-1092850886
H3 = 2032947006-2050126642
H4 = 2098661837-2117151733
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
```

Under `awg_version: "3.0"` you get the key plus the six ranges, but neither `RandomTrailers` nor `DisableCookies`.

## Pitfalls

- **The key lives only in the emitted server config.** Switching to 2.0 or `header_protection: false` drops it; switching back rotates it, and every client still holding the old key stops handshaking until re-imported. `--full-reset` rotates it deliberately.
- **`header_protection: false` does not disable the rest of the layer.** The six ranges and both 3.1 booleans still ship.
- **Uniform S requires all four `S` fields to be absent.** Pinning one value makes the other three independent draws and triggers `TRL001`.
- **`{0,0}` means disabled, and a missing bound is a zero bound.** `{"min": 0}` is disabled; `{"min": 0, "max": 5}` is an error.
- **Overlapping `H` ranges are not a validator finding.** The engine rejects them only at `awg setconf` time with `headers must not overlap`.
- **An S-floor violation aborts generation before anything is written**, but the same condition is `HPK001` on a config that already exists.
- **Configs are secrets.** `HeaderProtectionKey` is written in plaintext next to the private keys, and anyone who can read any config can decrypt header-protected headers.
- **The 3.x fields are invisible to the S/H/J projection helpers** (`HasAnyValue`, `ToSharedObfuscation`); library code must read them directly.
- **Older engine builds cannot load the keys.** Keep the runtime and the Docker image 3.x-aligned.
- **`generate` findings cover the server config only.** Use `validate` on a client config to check it too.

## Related

- [Manifest Reference](./manifest-reference.md) — every manifest field, including the ten 3.x knobs.
- [Obfuscation](./obfuscation.md) — the 2.0 S/H/J/CPS layer and how it interacts with header protection.
- [Output Format](./output-format.md) — the full `awg0.conf` layout and key order.
- [Validation & Analysis](./validation.md) — all finding and RISK codes.
- [Credentials & Key Reuse](./credentials.md) — key persistence and `--full-reset`.
- [VPN Import Links](./vpn-links.md) — the `vpn://` envelope and its 3.x fields.
