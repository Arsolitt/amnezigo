# Obfuscation

> The obfuscation parameter layer: `S1`-`S4` size prefixes, `H1`-`H4` header ranges, `Jc`/`Jmin`/`Jmax` junk parameters, the AWG 3.x transport-protection knobs and their version gates, the size-classification invariants, the CPS tag grammar, and the six named protocol templates plus `random`.

## Table of Contents

- [Obfuscation parameters overview](#obfuscation-parameters-overview)
- [AWG version and version gates](#awg-version-and-version-gates)
- [Server-shared vs per-client](#server-shared-vs-per-client)
- [Random vs explicit values](#random-vs-explicit-values)
- [Size invariants](#size-invariants)
- [CPS tag grammar](#cps-tag-grammar)
- [Protocol templates](#protocol-templates)
- [Choosing values](#choosing-values)
- [Related](#related)

---

## Obfuscation parameters overview

All parameters live under the manifest's top-level `obfuscation` object (see [./manifest-reference.md](./manifest-reference.md)). Two layers exist: the S/H/J layer inherited from AmneziaWG 2.0, and the transport-protection layer added by AWG 3.0/3.1. This page covers both at the parameter level; the engine semantics behind the 3.x layer — the header cipher, its nonce, trailer classification, key persistence — are documented in [./transport-protection.md](./transport-protection.md).

Manifest values use pointer types so that `nil` ("not set") is distinguishable from an explicit `0`. The general rule: `nil` means "draw a value" (or "select the version default"), while an explicit non-nil value is used as-is.

### S/H/J parameters

| Param | Type | Default when unset | Role |
| --- | --- | --- | --- |
| `s1`-`s3` | `*int` | drawn in `[minS, 65)` — `0`-`64`, or `12`-`64` under header protection | Size prefixes for the handshake classes: padded Initiation `S1+148`, Response `S2+92`, Cookie reply `S3+64`. |
| `s4` | `*int` | drawn in `[minS, 33)` — `0`-`32`, or `12`-`32` under header protection | Prefix for transport messages: `S4+32`. |
| `h1`-`h4` | `*HeaderRange` (`min`/`max` are `uint32`) | `{1,1}`, `{2,2}`, `{3,3}`, `{4,4}` when header protection is on and all four are unset; otherwise generated | Header byte-range substitution. Generated ranges have `Min >= 5`, `Max <= 2147483647`, width `>= 10000000`, are sorted, non-overlapping, and never intersect `[1..4]`. |
| `jc` | `*int` | drawn in `0`-`10`, retried until non-zero | Junk packets emitted per handshake window. |
| `jmin`/`jmax` | `*int` | drawn in `64`-`1024`, with `Jmin < Jmax` | Inclusive junk-packet size range. Generated ranges exclude every padded and raw WG size. |
| `protocol` | `string` | informational — never read by `generate` | Per-peer `peers[].protocol` is what selects the CPS template; the top-level field has no effect. |

> **Note:** the generator's S ranges do not bound explicit values. A hand-picked `s4` of `116` is written to the config as-is; explicit S values are only checked against the header-protection floor and the padded-size invariant.

### AWG 3.x parameters

| Param | Type | Resolved default | Gate | INI key |
| --- | --- | --- | --- | --- |
| `awg_version` | `string` | `"3.1"` (`DefaultAWGVersion`) | — | selects the emitted key set |
| `header_protection` | `*bool` | `true` for `>= 3.0` | `3.0` | `HeaderProtectionKey` |
| `content_padding` | `*U16Range` | `2-10` | `3.0` | `ContentPaddingAddition` |
| `rekey_after_time` | `*U16Range` | `120-180` | `3.0` | `RekeyAfterTime` |
| `rekey_timeout` | `*U16Range` | `5-8` | `3.0` | `RekeyTimeout` |
| `reject_after_time` | `*U16Range` | `180-240` | `3.0` | `RejectAfterTime` |
| `keepalive_timeout` | `*U16Range` | `8-12` | `3.0` | `KeepaliveTimeout` |
| `max_handshake_attempts` | `*U16Range` | `16-20` | `3.0` | `MaxHandshakeAttempts` |
| `random_trailers` | `*bool` | `true` for `3.1`, `false` for `3.0` | `3.1` | `RandomTrailers` (`on`/`off`) |
| `disable_cookies` | `*bool` | `true` for `>= 3.1` | `3.1` | `DisableCookies` (`on`/`off`) |

`U16Range` is `{"min": N, "max": M}` with `uint16` bounds. A `nil` field takes the resolved default; `{"min": 0, "max": 0}` disables the key (it is omitted from the INI); a mixed range such as `{"min": 0, "max": 5}` is an error, as is `max` below `min`:

```text
obfuscation.content_padding: bounds must both be zero or both non-zero (got 0-5)
obfuscation.content_padding: max (2) is below min (5)
```

A resolved range renders as `N` when both bounds are equal and `Min-Max` otherwise (`{"min": 5, "max": 5}` becomes `5`). Header ranges are always two-part: `{"min": 1, "max": 1}` becomes `H1 = 1-1`.

The defaults above produce this block in a 3.1 config (emission order and gating rules are covered in [./output-format.md](./output-format.md)):

```ini
HeaderProtectionKey = <44-char base64>
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
```

`HeaderProtectionKey` is a generated 32-byte ChaCha20 key, base64 StdEncoding (44 characters), persisted across runs and reused unless `--full-reset` (see [./credentials.md](./credentials.md)). Header protection is switched by the key being present, not by a separate flag — `header_protection: false` simply leaves the key empty.

---

## AWG version and version gates

`obfuscation.awg_version` pins the target engine generation. An unset field selects `"3.1"`; any other value fails with `unsupported awg_version "3" (expected "2.0", "3.0", or "3.1")`.

| `awg_version` | Meaning | Flags default | Emitted 3.x keys |
| --- | --- | --- | --- |
| `"2.0"` | legacy S/H/J + I1-I5 | header protection off | none |
| `"3.0"` | adds the header-protection key and the six uint16 ranges | `header_protection: true`; `random_trailers` and `disable_cookies` false | `HeaderProtectionKey` plus the six ranges |
| `"3.1"` | adds trailers and cookie suppression | all three `true` | the 3.0 set plus `RandomTrailers`/`DisableCookies` |

Gates are presence-based, not value-based: a field the target version cannot express is rejected even when its value would be a no-op.

| Fields | Minimum `awg_version` |
| --- | --- |
| `header_protection`, `content_padding`, `rekey_after_time`, `rekey_timeout`, `reject_after_time`, `keepalive_timeout`, `max_handshake_attempts` | `"3.0"` |
| `random_trailers`, `disable_cookies` | `"3.1"` |

Every gate failure has the form:

```text
obfuscation.<field> requires awg_version <min> or later (got "<have>")
```

Because `header_protection: false` still counts as "field present", a 2.0 manifest that tries to disable header protection fails like this (the CLI adds its own context prefix):

```text
generating configs: resolve obfuscation: obfuscation.header_protection requires awg_version 3.0 or later (got "2.0")
```

Keys are never emitted for a lower target version: pre-3.x engine builds reject unknown INI keys with `Line unrecognized`. See [./transport-protection.md](./transport-protection.md) for the full version model.

---

## Server-shared vs per-client

The obfuscation profile splits into a network-wide shared layer and a per-client CPS layer.

| Layer | Fields | Scope | Lifecycle |
| --- | --- | --- | --- |
| Server-shared | `S1`-`S4`, `H1`-`H4`, `Jc`/`Jmin`/`Jmax`, the nine 3.x keys (including `HeaderProtectionKey`) | every peer in the manifest | resolved once per `generate`; written into the server config and mirrored into every client config |
| Per-client | `I1`-`I5` | one set per client peer | computed per client from its protocol template; not persisted — deterministic for named templates, `random` redraws each run |

- The server `[Interface]` block carries S/H/J and the 3.x block but no I-strings. Each client's `awg0.conf` carries its own I1-I5 plus the shared values.
- Only keys are persisted and reused (peer keypairs, PSKs, the header-protection key). Unpinned S/H/J values are re-drawn on every run, and a client with `protocol: random` gets a fresh I1-I5 tag sequence each run (named templates recompute deterministically from the same inputs). Re-import every client after an unpinned regeneration.
- The per-peer `peers[].protocol` field selects which template shapes that client's I1-I5; empty means `quic`.

---

## Random vs explicit values

Resolution treats each field independently. `ObfuscationManifest.HasAnyValue()` and `ToSharedObfuscation()` cover only S/H/J — the 3.x fields are read directly by the resolver.

| Field state | Resolution |
| --- | --- |
| All S/H/J pointers nil | independent draws: `GenerateSPrefixes`, `GenerateHeaderRanges` (or the reference `1..4` set), `GenerateJunkParamsWithForbidden` |
| `s1` explicit, `s2`-`s4` nil | `S1` is kept, `S2`-`S4` are drawn; the merged set is not re-checked during resolution, so a collision between the pinned value and a generated one is reported by validation (PSC001) rather than repaired |
| Any explicit S | used verbatim, never re-drawn; under header protection `S < 12` aborts generation with `header protection requires S1-S4 >= 12 (got S3=8)` |
| `"s1": 0` | treated as unset — zero S values are replaced by generated ones until all four are non-zero |
| `random_trailers` on and all four S nil | uniform S: one draw in `[minS, 33)` copied into `S1=S2=S3=S4`. Under header protection that is `12`-`32`; with header protection off the draw may be `0`, which is retried, so the effective range is `1`-`32`. |
| Header protection on, all four H nil | the reference set `{1,1}`, `{2,2}`, `{3,3}`, `{4,4}` |
| Header protection on, some H explicit | explicit values verbatim, nil H filled from `GenerateHeaderRanges()`; the mixture is not overlap-checked |
| Header protection off with H nil | `GenerateHeaderRanges()` — wide ranges that never intersect `[1..4]` |
| All three J explicit | copied verbatim, collision checks skipped |
| `"jc": 0` | replaced by a non-zero draw unless `jmin` and `jmax` are also explicit |
| 3.x range or bool nil | resolved version default |
| 3.x range `{0, 0}` | key disabled and omitted from the INI |
| `peers[].protocol` empty | `quic` |

Notes:

- Uniform S exists for AWG 3.1 trailers: with `random_trailers` enabled the receiver classifies packets by size, so equal S values are recommended. Pinning any S disables the shortcut, and unequal pinned S with trailers enabled produces the `TRL001` warning — never an error.
- The library helper `GenerateSPrefixesWithS1(minS, fixedS1)` is a different path: it keeps a caller-supplied `S1` and draws `S2`-`S4` so all six padded-size pairs are checked against that fixed value. `GenerateConfig`/`GenerateServerConfig` use it; the manifest pipeline uses the merge described above.
- Explicit values are never repaired. A pinned set that produces colliding padded sizes fails `validate` (PSC001); a pinned overlapping H range is only caught by the engine when the config is applied.

---

## Size invariants

The four AWG-padded handshake sizes are `S1+148` (Initiation), `S2+92` (Response), `S3+64` (Cookie reply) and `S4+32` (Transport). `ValidatePacketSizes` enforces the size-classification invariant. S, junk and H values are generated collision-free; the I-packet-vs-padded-size guarantee is specific to the library `GenerateConfig` path (see row 2).

| # | Invariant | Where enforced |
| --- | --- | --- |
| 1 | The four padded sizes are pairwise distinct (six pairs; `S1+56 != S2` is only one of them). | generator retry (`pairsDistinct`); `ValidatePacketSizes` |
| 2 | No I-packet length equals any padded size. | `buildAndValidateCPS` plus the `GenerateConfig` post-condition (`ValidatePacketSizes` called with I-packet sizes). The `validate` CLI cannot check this one — it passes `nil` I-packet sizes, so `PSC003` is library-only. |
| 3 | `[Jmin..Jmax]` contains none of the four padded sizes. | `GenerateJunkParamsWithForbidden`; `validate` |
| 4 | `[Jmin..Jmax]` contains none of the raw WG sizes (`148`, `92`, `64`, `32`). | same as #3 |
| 5 | `Jmin <= Jmax`. | generator swaps/bumps; `validate` (JNK001) |
| 6 | Every H range has `Max >= Min`, and must not intersect `[1..4]` unless a header-protection key is present. | `ValidateHeaderRange`; HDR001 is suppressed under header protection |
| 7 | H ranges are pairwise non-overlapping: sorted by `Min`, with `Max_i < Min_{i+1}` (touching ranges are rejected). | generation only (`GenerateHeaderRanges`); `validate` does not check overlap |

Notes:

- Invariant #1 is stated on **padded** sizes, not raw S values: `S1+148 == S2+92` is a collision even though `S1 != S2`.
- The `[1..4]` exclusion in #6 is a vanilla-WireGuard defence. Without header protection, a range containing the WG message type-ids would let genuine WireGuard traffic reach the peer. With header protection the 4-byte type is encrypted, so `H1..H4 = 1-1..4-4` is the legitimate reference configuration.
- Overlapping explicit H ranges pass `validate` but are rejected by the engine when the config is applied (`headers must not overlap`). This rule comes from the engine, not from amnezigo.
- `GenerateHeaderRanges` retries by drawing an entirely new set (up to 1000 attempts, then a panic); it never nudges individual bounds.
- A pinned value can be outside the generator's ranges. The generator caps `S4` at `32` while `S1`-`S3` cap at `64`, but `ValidatePacketSizes` is a generic primitive: it accepts a hand-picked `S4` of `116` as long as the padded sizes stay pairwise distinct.

For the finding codes (`PSC001`, `HDR001`, `HPK001`, `TRL001`, ...) see [./validation.md](./validation.md).

---

## CPS tag grammar

Custom Packet Strings (I1-I5) are sequences of tags. The generator emits the tag literals; the AmneziaWG receiver expands them when packets are emitted.

| Tag | Meaning | Length (bytes) |
| --- | --- | --- |
| `<b 0xNN>` | literal hex bytes (a missing `0x` prefix is added automatically) | `len(NN) / 2` |
| `<r N>` | `N` cryptographically random bytes | `N` |
| `<rc N>` | `N` random ASCII letters from `[a-zA-Z]` (52 letters, lowercase first) | `N` |
| `<rd N>` | `N` random decimal digits | `N` |
| `<t>` | big-endian `uint32` Unix timestamp | `4` |
| `<d>` | runtime passthrough — reuses a value from an earlier interval | `0` at generation |
| anything else | unknown tag, including `<c>` and the engine-only `ds`/`dz` | `0` |

Rules and common mistakes:

- amnezigo implements exactly `b`, `t`, `r`, `rc`, `rd` and `d`. `BuildCPSTag` returns the empty-string sentinel for `c` and every other unknown type.
- `<c>` (counter) is deliberately unsupported: it is recognised only by the legacy Linux kernel module, while `amneziawg-go` and every AmneziaVPN client reject it with `unknown tag`. A raw `<c>` literal in a config produces the strict-parse warning CPS001.
- `ds` and `dz` exist in `amneziawg-go` 3.1's CPS parser but are not implemented in amnezigo. A hand-written `<dz 32>` counts as 0 bytes, silently shrinking the interval in every size calculation.
- Unknown tags and unparsable counts are lenient: `<r abc>` counts as 0 bytes. A malformed tag does not fail validation — it makes the interval look shorter.
- `<rc>` is letters only. For a mixed letter/digit field, concatenate tags: `<rc 4><rd 2>`.
- `<t>` is 4 bytes, not 8. At most one `<t>` may appear per interval; random mode enforces this and the named templates use at most one.
- `<d>` is never emitted by random mode — a standalone random interval has no earlier interval to source the passthrough value from. It appears in hand-written or template intervals such as QUIC I2.
- The MTU bound is strict: an interval whose computed length equals the budget is rejected, matching historical behaviour.

Byte-accounting examples, using the same parser as generation-time validation:

| CPS | `CPSLength` |
| --- | --- |
| `<b 0xdeadbeef><t>` | 8 |
| `<r 10><rc 5><rd 3>` | 18 |
| `<t>` | 4 |
| `<b 0xff><d>` | 1 |
| `<d>` | 0 |
| `<c>` | 0 |

---

## Protocol templates

Each client peer's I1-I5 are shaped by a protocol template selected through the per-peer `peers[].protocol` field. The budgets below are measured with `CPSLength` on the rendered templates; every named template fills I1-I4 and leaves I5 empty.

| Protocol | Fixed shape | I1 / I2 / I3 / I4 (bytes) | Notes |
| --- | --- | --- | --- |
| `quic` (default) | QUIC long header `c0ff` + version `00000001` + 8-byte DCID | 65 / 36 / 34 / 29 | I2 reuses I1's DCID through `<d>` (session continuity). |
| `dns` | random 2-byte transaction ID, flags `0100`, label bytes (`<rc>`; I3 uses `<rd>`) | 33 / 22 / 20 / 19 | Standard query, recursion desired. |
| `dtls` | `16` (handshake) + `fefd` (DTLS 1.2 ClientHello) | 73 / 67 / 67 / 67 | |
| `stun` | `0001` Binding Request + magic cookie `2112a442` | 20 / 24 / 20 / 20 | I2 is larger than I1 — the budgets are per-template, not monotonic. |
| `sip` | ASCII `OPTIONS sip:` request | 312 / 203 / 140 / 87 | Source comments quote ~360/~240/~170/~120; those are approximate, the measured values are exact. |
| `rtp` | `0x80` fixed header (V=2, P=0, X=0, CC=0) | 92 / 52 / 36 / 20 | Payload sizes mimic G.711 frames. |
| `random` | no fixed shape — mixed tags | per run | 3-6 tags, at most one `<t>`; `<b>` values are 4-16 bytes, `<r>`/`<rc>`/`<rd>` values 5-40; all five intervals are generated. |

Rendered examples (MTU 1280, S1 32):

```text
quic I1 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0040><b 0x00><b 0x01><t><r 40>  (65 B)
stun I1 = <b 0x0001><b 0x0000><b 0x2112a442><r 12>  (20 B)
rtp  I1 = <b 0x8000><r 2><t><r 4><r 80>  (92 B)
sip  I1 = <b 0x4f5054494f4e53><b 0x20><b 0x7369703a><rc 8>...  (312 B)
```

Selection rules:

- `peers[].protocol` empty → `quic`. One of the six names → that template.
- `"random"` runs the random tag-sequence mode — it is not a named template.
- Any other non-empty string is not rejected: the template lookup's default branch picks one of the six named templates uniformly at random. A typo therefore yields a plausible but unexpected protocol shape.
- `amnezigo analyze` defaults `--protocol` to `random` and also does not reject unknown names; `generate` defaults an empty per-peer protocol to `quic`.
- The top-level `obfuscation.protocol` field is never read by `generate`.
- `ListProtocols()` returns the seven names sorted alphabetically: `dns`, `dtls`, `quic`, `random`, `rtp`, `sip`, `stun`.

---

## Choosing values

| Concern | Rule |
| --- | --- |
| I-packet upper bound | `maxISize = MTU - 49 - 149 - S1` (49 bytes of IP/UDP header plus a safety byte; 149 is the fixed handshake size plus a safety byte). The bound is strict. With the default MTU `1280` and `S1 = 32` the budget is 1050 bytes. |
| I-packet lower bound | 4 bytes (a bare `<t>`). |
| Collision avoidance | `buildAndValidateCPS` drops tags from the end until the interval fits, then tries `<t><rd N>` for `N = 1..8`, then falls back to `<t>`; padded-size avoidance applies only when a non-empty forbidden set is threaded through. The library `GenerateConfig` passes `PaddedSizes(...)` and re-checks with `ValidatePacketSizes`; the `generate` pipeline calls `GenerateCPS`, which hardcodes an empty forbidden set (and ignores its 4th argument), so CLI-generated I1-I5 are not sized against the padded handshake sizes. Random mode retries up to 1000 draws with the same perturbation ladder. |
| Template budgets | The measured sizes above are consequences of the template definitions, not a validated invariant — they are not required to descend (STUN I2 > I1). |
| Stable configs | Unpinned S/H/J values change on every `generate` run, and `protocol: random` produces fresh I1-I5 tags each run (named templates are deterministic). Pin S/H/J in the manifest, or copy a curated bundle from [./presets.md](./presets.md). |

> **Warning:** CLI-generated I-packets are not padded-size-avoided. A manifest can therefore silently emit an I-packet whose length equals one of the four padded handshake sizes — for example `S3 = 1` makes the padded cookie reply `65` bytes, the same as the default QUIC I1 — and neither `generate` nor `validate` reports it. Use the library `GenerateConfig` path, or re-check emitted intervals with `ValidatePacketSizes`, if you need the stronger guarantee.

> **Warning:** An omitted `network.mtu` collapses client I1-I5 to `<t>`. CPS generation reads the raw `network.mtu` value before it is defaulted, so the MTU budget is negative and every interval falls back to the 4-byte bare timestamp — while the emitted client config still says `MTU = 1280`. Always set `network.mtu` explicitly.

> **Tip:** `CPSLength` and `GenerateCPS(protocol, mtu, s1, _)` are exported, so you can reproduce these budgets and intervals in your own tooling (note that `GenerateCPS` uses an empty forbidden set). See [./library-usage.md](./library-usage.md).

---

## Related

- [./transport-protection.md](./transport-protection.md) — the AWG 3.x layer end-to-end: header-protection key lifecycle, engine semantics, INI emission, and validation codes.
- [./manifest-reference.md](./manifest-reference.md) — field-by-field `obfuscation.*` reference and the pointer/nil semantics.
- [./presets.md](./presets.md) — the seven curated S/H/J bundles, including their 3.x `ContentPadding`/`RandomTrailers`/`DisableCookies` values.
- [./validation.md](./validation.md) — `validate`/`analyze` finding codes that consume these parameters (HPK001, TRL001, HDR001, PSC001, RISK heuristics).
- [./output-format.md](./output-format.md) — how S/H/J, the nine 3.x keys, and I1-I5 serialize into `awg0.conf`.
- [./library-usage.md](./library-usage.md) — the exported generators and helpers (`GenerateSPrefixes`, `GenerateConfig`, `GenerateCPS`, `CPSLength`, `ListProtocols`).
