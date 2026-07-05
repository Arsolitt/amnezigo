# Obfuscation

> AmneziaWG 2.0 obfuscation parameters, size-classification invariants, the CPS
> tag grammar, and the six protocol templates (`quic`, `dns`, `dtls`, `stun`,
> `sip`, `rtp`) plus `random`.

## Table of Contents

- [Obfuscation parameters overview](#obfuscation-parameters-overview)
- [Server-shared vs per-client](#server-shared-vs-per-client)
- [Random vs explicit values](#random-vs-explicit-values)
- [AWG 2.0 size invariants](#awg-20-size-invariants)
- [CPS tag grammar](#cps-tag-grammar)
- [Protocol templates](#protocol-templates)
- [Choosing values](#choosing-values)
- [Related](#related)

---

## Obfuscation parameters overview

All values live under the manifest's top-level `obfuscation` object (see
[./manifest-reference.md](./manifest-reference.md)). Pointer types (`*int`,
`*HeaderRange`) distinguish "user set to 0" from "user did not set this field";
`nil` triggers random generation (see [Random vs explicit values](#random-vs-explicit-values)).

| Param | Type | Range / units | Role |
|---|---|---|---|
| `s1`–`s3` | `*int` | `0`–`64` (random: `rand.Int(.., 65)`) | Handshake size markers added to raw WG message sizes (`148`, `92`, `64`, `32`). Drive the four AWG-padded handshake sizes `S1+148`, `S2+92`, `S3+64`. |
| `s4` | `*int` | `0`–`32` (random: `rand.Int(.., 33)`) | Marker for the transport message size `S4+32`. Smaller range because the transport payload is the smallest WG message. |
| `h1`–`h4` | `*HeaderRange` | `uint32`, `Min` ∈ `[5, 2147483647]`, span `Max−Min ≥ 10000000` (`headerMinRange`) | Init-packet header byte ranges. Serialized `min-max` (inclusive). Must exclude WG type-ids `[1..4]` and be pairwise non-overlapping. |
| `jc` | `*int` | `0`–`10` (random: `rand.Int(.., 11)`) | Junk-packet count emitted per handshake window. |
| `jmin` | `*int` | `64`–`1024` | Inclusive lower bound of the junk-packet size range. |
| `jmax` | `*int` | `64`–`1024`, `jmin < jmax` enforced | Inclusive upper bound of the junk-packet size range. `jmin > jmax` is a structural error (`ErrEmptyJunkRange`). |
| `protocol` | `string` | `quic` \| `dns` \| `dtls` \| `stun` \| `sip` \| `rtp` | **Top-level field is NOT consumed by `generate`** (see [Protocol templates](#protocol-templates)). Kept for documentation only. |

> **Source:** `generator.go` (`s4RangeMax=33`, `sPrefixRangeMax=65`, `jcRangeMax=11`,
> `junkMinValue=64`, `headerMinValue=5`, `headerMaxValue=2147483647`,
> `headerMinRange=10000000`), `manifest.go` (`ObfuscationManifest`).

---

## Server-shared vs per-client

The obfuscation profile splits into a **network-wide shared layer** (resolved
once from `obfuscation.*`, identical for every peer) and a **per-client CPS
layer** (the I1–I5 strings, generated independently for each client peer every
run).

| Layer | Fields | Scope | Lifecycle |
|---|---|---|---|
| Server-shared | `s1`–`s4`, `h1`–`h4`, `jc`, `jmin`, `jmax` | All peers in the manifest | Resolved once per `generate`; written into the server config and mirrored into every client config. |
| Per-client | `i1`–`i5` (CPS strings) | One set per client peer | Regenerated **every run** for each client; never stored in the server config. |

Consequences:

- The server `[Interface]` block carries `S_*`, `H_*`, `Jc/Jmin/Jmax` but **no
  I-strings**. Each client's `awg0.conf` carries its own I1–I5 plus the shared
  S/H/J values.
- Re-running `amnezigo generate` produces fresh I1–I5 for every client while
  reusing the persisted shared values (unless `--full-reset`). See
  [./credentials.md](./credentials.md) and [./output-format.md](./output-format.md).
- The per-peer `peers[].protocol` field selects which protocol template shapes
  that client's I1–I5 (see [Protocol templates](#protocol-templates)).

> **Source:** `pipeline.go` (`GenerateCPS` called once per client peer),
> `types.go` (`ServerObfuscationConfig` has no I-fields; `ClientObfuscationConfig`
> embeds it and adds `I1`–`I5`).

---

## Random vs explicit values

Every S/H/J field uses a pointer. `nil` means "generate randomly"; a non-nil
value is used as-is. `ObfuscationManifest.HasAnyValue()` is the fast-path check
before per-field fallback.

| Field state | Resolution | Where |
|---|---|---|
| All S/H/J pointers `nil` | Fully random profile: `GenerateSPrefixes`, `GenerateHeaderRanges`, `GenerateJunkParamsWithForbidden` | `HasAnyValue() == false` → random path |
| Some pointers set, others `nil` | Set fields used as-is; `nil` fields fall back to random generation for that field only | per-field fallback in `resolveObfuscation` |
| `s1` set, `s2`–`s4` `nil` | `GenerateSPrefixesWithS1(fixedS1)` derives `s2`–`s4` so the six padded-size pairs stay distinct | `generator.go` |
| `h1`–`h4` set | Used verbatim; must already be valid (`ValidateHeaderRange`) and non-overlapping | `validation.go` |
| `peers[].protocol` empty | Defaults to `quic` | `pipeline.go` (`if protocol == "" { protocol = protocolQUIC }`) |

For the full field-by-field pointer/nil semantics, see
[./manifest-reference.md](./manifest-reference.md).

> **Source:** `manifest.go` (`ObfuscationManifest` doc, `HasAnyValue`),
> `generator.go` (`GenerateSPrefixesWithS1`), `pipeline.go`.

---

## AWG 2.0 size invariants

`ValidatePacketSizes` enforces the AWG 2.0 size-classification invariant. The
generator (`GenerateConfig`, `GenerateJunkParamsWithForbidden`) produces values
that already satisfy it; `amnezigo validate` re-checks parsed configs. Check
order: S-pairs → I-packets → junk range.

| # | Invariant | Collision kind | Error |
|---|---|---|---|
| 1 | The four AWG-padded sizes `S1+148`, `S2+92`, `S3+64`, `S4+32` are pairwise distinct (6 pairs). | `s-pair` | `*PacketSizeCollisionError` |
| 2 | No I-packet length equals any of the four padded sizes. | `i-packet` | `*PacketSizeCollisionError` |
| 3 | The junk range `[jmin..jmax]` contains none of the four padded sizes. | `junk-range` | `*PacketSizeCollisionError` |
| 4 | The junk range `[jmin..jmax]` contains none of the four raw WG sizes (`148`, `92`, `64`, `32`). | `junk-range` | `*PacketSizeCollisionError` |
| 5 | `jmin <= jmax` (else the range is empty). | structural | `ErrEmptyJunkRange` |
| 6 | Each `H1`–`H4` range excludes WG message type-ids `[1..4]` and has `Max >= Min`. | header | `ValidateHeaderRange` error |
| 7 | `H1`–`H4` are pairwise non-overlapping (generator sorts by `Min` and requires `Max_i < Min_{i+1}`). | header | generator retry / `validateHeaderRanges` |

> **Note:** invariant #1 is stated on **padded** sizes, not raw `S` values —
> e.g. `S1+148 == S2+92` is a collision even though `S1 != S2`.

> **Source:** `validation.go` (`ValidatePacketSizes`, `ValidateHeaderRange`,
> `paddedSizes`), `generator.go` (`junkRangeOK`, `headerRangesValid`). See
> [./validation.md](./validation.md) for the `validate` command and finding
> codes.

---

## CPS tag grammar

Custom Packet Strings (I1–I5) are built from tags defined in `cps.go`. The
generator emits the tag **literals**; the AmneziaWG receiver expands them at
packet-emit time.

| Tag | Meaning | Length (bytes) | Notes |
|---|---|---|---|
| `<b 0xNN>` | Literal hex bytes | `len(NN) / 2` | Constant protocol headers (e.g. QUIC `c0ff`, STUN magic cookie `2112a442`). |
| `<r N>` | `N` cryptographically random bytes | `N` | Binary fields (DCIDs, transaction IDs, payloads). |
| `<rc N>` | `N` chars from `[a-zA-Z]` (52 letters, lowercase first) | `N` | Text-looking fields (DNS labels, SIP tokens). Alphabet source: `amneziawg-go device/obf_randchars.go`. |
| `<rd N>` | `N` random ASCII digits | `N` | Numeric-looking fields; also the collision-perturbation suffix. |
| `<t>` | Unix timestamp, `uint32` big-endian | **4** | Source: `amneziawg-go device/obf_timestamp.go` (`binary.BigEndian.PutUint32`). |
| `<d>` | Data passthrough — reuses a value from an earlier I-packet position | 0 (at generation) | **Requires AWG 2.0 userspace.** The legacy Linux kernel module rejects `<d>` with "unknown tag". |

Rules and corrections:

- **`<t>` is 4 bytes**, not 8. (The old `docs/obfuscation.md` claimed 8 bytes —
  that was wrong.)
- **`<c>` (counter) does NOT exist.** It was kernel-module-only and is rejected
  by `amneziawg-go` and every AmneziaVPN client; `BuildCPSTag("c", …)` returns
  the empty-string sentinel.
- **At most one `<t>` per interval.** Random mode enforces this via the
  `usedUniqueTag` flag in `generateRandomTags`; named templates that use
  timestamps carry exactly one `<t>` per interval.
- The smallest legal CPS is **4 bytes** (a bare `<t>`); the MTU bound is strict
  (`calculateCPSLength(cps) < maxISize`).
- `<d>` is excluded from random mode — it only makes sense in templated
  multi-interval flows where an earlier interval produces the reused value
  (e.g. QUIC I2 reuses I1's DCID).

> **Source:** `cps.go` (`BuildCPSTag`, `mapTagType`, `calculateCPSLength`,
> `generateRandomTags`, constants `cpsTimestampSize=4`, `cpsRcAlphabet`).

---

## Protocol templates

Each client peer's I1–I5 are shaped by a protocol template chosen via the
**per-peer** `peers[].protocol` field. The top-level `obfuscation.protocol` is
**not consumed by `generate`**.

| Protocol | Leading bytes / mimicry | Source file | Notes |
|---|---|---|---|
| `quic` *(default)* | QUIC Long Header: `c0ff` + version `00000001`, 8-byte DCID | `quic.go` | I2 reuses I1's DCID via `<d>` (session continuity). |
| `dns` | DNS query: 2-byte txn ID + flags `0100`, `<rc>` labels | `dns.go` | Standard query, recursion desired. |
| `dtls` | DTLS 1.2 ClientHello: `16` (Handshake) + `fefd` | `dtls.go` | Timestamp seeded into the ClientHello random field. |
| `stun` | STUN Binding Request: `0001` + magic cookie `2112a442` | `stun.go` | 12-byte transaction ID via `<r 12>`. |
| `sip` | ASCII SIP OPTIONS request (`OPTIONS sip:` …) | `sip.go` | Text protocol; heavy use of `<rc>`. |
| `rtp` | RTP fixed header: `8000` (V=2, P=0, X=0, CC=0, M=0, PT=0 PCMU) | `rtp.go` | RFC 3550; per-packet timestamp via `<t>`, payload sizes mimic G.711 frames. |
| `random` | No fixed header; mixed `<b>`/`<r>`/`<rc>`/`<rd>`/`<t>` tags | `cps.go` (`generateRandomTags`) | Default only for `analyze`; `generate` defaults empty `protocol` to `quic`. |

Selection rules:

- `peers[].protocol` empty → `quic` (`pipeline.go`).
- `peers[].protocol` = one of the six names → that template via `getTemplate`.
- `analyze --protocol` accepts exactly `random,quic,dns,dtls,stun,sip,rtp`; the
  `random` selector picks one of the six named templates uniformly at random.
  See [./cli-reference.md](./cli-reference.md).
- All six named templates leave **I5 empty** by convention; `random` mode
  generates all five intervals.
- Named templates that use timestamps carry **exactly one `<t>` per interval**.

> **Source:** `protocols.go` (`getTemplate`, protocol constants), `quic.go`,
> `dns.go`, `dtls.go`, `stun.go`, `sip.go`, `rtp.go`, `pipeline.go`,
> `internal/cli/analyze.go`.

---

## Choosing values

There is no `preset` manifest field. To use a curated bundle, copy its values
into `obfuscation.*` (see [./presets.md](./presets.md) for all seven bundles).

| Concern | Rule | Source |
|---|---|---|
| I-packet upper bound | `maxISize = MTU − 49 − 149 − S1` (`reserve=49`, `handshakeSize=149`); strict `<`. Default MTU `1280`. | `cps.go` (`calculateMaxISize`) |
| I-packet lower bound | Smallest legal CPS is 4 bytes (bare `<t>`). | `cps.go` (`cpsAcceptable`) |
| Collision avoidance | The generator's `buildAndValidateCPS` shrinks then perturbs (`<t><rd N>`, `N ∈ [1..8]`) to avoid the four padded sizes. | `cps.go` |
| Named-template byte budgets | Descending `I1 ≥ I2 ≥ I3 ≥ I4`, `I5` empty — mimics a real protocol's packet-size distribution (e.g. RTP 92/52/36/20 B). Convention, not a validated invariant. | `rtp.go`, `quic.go` |
| S/H/J defaults | Prefer a preset over hand-picking; random generation is collision-safe via `GenerateConfig`. | `presets.go`, `generator.go` |

For ready-made S/H/J combinations, see [./presets.md](./presets.md).

---

## Related

- [./presets.md](./presets.md) — the seven named value bundles (`lan-conservative`,
  `home-balanced`, `mobile-aggressive`, `stealth-paranoid`, `standard-1420`,
  `low-overhead`, `test-minimal`).
- [./manifest-reference.md](./manifest-reference.md) — `obfuscation.*` field
  reference and pointer/nil semantics.
- [./validation.md](./validation.md) — `validate` rules, finding codes, and the
  `analyze` heuristics.
- [./output-format.md](./output-format.md) — how S/H/J and I1–I5 serialize into
  `awg0.conf` (`#_`-prefixed metadata, `HeaderRange` as `min-max`).
- [./cli-reference.md](./cli-reference.md) — `analyze --protocol` flag values.
