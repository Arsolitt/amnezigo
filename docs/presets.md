# Presets

> Seven curated bundles of AmneziaWG obfuscation parameters. Copy a preset's values into the manifest — there is no `preset` manifest field and no `--preset` flag.

## Table of Contents

- [Preset overview](#preset-overview)
- [Per-preset parameters](#per-preset-parameters)
- [How to use a preset](#how-to-use-a-preset)
- [API](#api)
- [Gotchas](#gotchas)
- [Related](#related)

---

## Preset overview

amnezigo ships **seven named presets** in the unexported `presetRegistry` (`presets.go`). Each preset is a fixed bundle of obfuscation parameters: `S1`–`S4`, `H1`–`H4`, `Jc`/`Jmin`/`Jmax`, the 3.1 knobs `ContentPadding`, `RandomTrailers` and `DisableCookies`, plus an `MTU` and a default cover protocol. Presets are data — `generate` never resolves a preset by name, and the manifest carries only resolved parameters. See [Obfuscation](./obfuscation.md) for the S/H/J semantics and [Transport Protection (AWG 3.x)](./transport-protection.md) for the 3.1 knob layer.

Every preset is constructed so that:

- The four AWG-padded handshake sizes (`S1+148`, `S2+92`, `S3+64`, `S4+32`) are pairwise distinct.
- The junk range `[Jmin..Jmax]` excludes all padded sizes and the raw WireGuard sizes 148/92/64/32.
- `H1`–`H4` are non-overlapping, ascending, and avoid the WireGuard type-id window `[1..4]`.
- `S1`–`S4` stay at or above the 12 B header-protection floor.
- Every preset passes `ValidatePacketSizes` and `ValidateHeaderRange` — but "pre-validated" does not mean "warning-free" (see the findings notes under [Per-preset parameters](#per-preset-parameters)).

| Preset | Intent | `DefaultProtocol` | `MTU` | `ContentPadding` | `RandomTrailers` | `DisableCookies` |
| --- | --- | --- | --- | --- | --- | --- |
| `lan-conservative` | Corporate LANs with minimal DPI; low overhead over deep obfuscation. | `random` | 1280 | `{0, 0}` (disabled) | `false` | `true` |
| `home-balanced` | General-purpose default for home internet; tolerates light DPI. | `quic` | 1280 | `{2, 10}` | `true` | `true` |
| `mobile-aggressive` | Carrier networks with heavy DPI (MTS, Beeline, etc.); maximum entropy. | `dns` | 1280 | `{2, 10}` | `true` | `true` |
| `stealth-paranoid` | Hostile DPI (national firewalls, deep statistical inspection); highest throughput cost (~2.5% per packet). | `quic` | 1280 | `{4, 12}` | `true` | `true` |
| `standard-1420` | Balanced profile at classic WireGuard MTU 1420; more I-packet headroom than `home-balanced`. | `quic` | 1420 | `{2, 8}` | `true` | `true` |
| `low-overhead` | Bandwidth-constrained links (satellite, metered, slow cellular); minimum overhead. | `dns` | 1280 | `{2, 4}` | `false` | `true` |
| `test-minimal` | Smallest valid parameter set — integration testing and CI only, not for production. | `random` | 1280 | `{0, 0}` (disabled) | `false` | `false` |

`DefaultProtocol` is metadata for the preset; `generate` uses the per-peer `peers.<name>.protocol` field (defaulting to `quic` when empty). Copy the preset's protocol there if you want its cover traffic.

## Per-preset parameters

All values below are pulled verbatim from `presets.go`. Header ranges are shown as `min–max`; `ContentPadding` is a `U16Range` (`{min, max}` in Go, `"content_padding": {"min": …, "max": …}` in the manifest). A `{0, 0}` range disables the corresponding engine key, so `ContentPaddingAddition` is omitted from the config.

`generate`/`validate` and `analyze` judge presets at different layers: `generate` and `validate` run the structural checks (finding codes such as `TRL001`), while `analyze` runs heuristics (`RISK001`–`RISK009`). A preset can pass every structural check and still trip a heuristic. `analyze` regenerates I-packet sizes on every run, so `RISK002` (narrow I-packet cluster) can appear intermittently; the per-preset findings below report the checks derived from the preset's own parameters. See [Validation & Analysis](./validation.md) for both code families.

> **Note:** The four presets with `RandomTrailers: true` have deliberately unequal `S` values, so every `generate` prints `TRL001` (`RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification`). This is expected, not a preset defect — configs are still written and the command reports `Warnings: 1`.

### lan-conservative

| Param | Value |
| --- | --- |
| `S1` | 12 |
| `S2` | 12 |
| `S3` | 15 |
| `S4` | 12 |
| `Jc` | 3 |
| `Jmin` | 161 |
| `Jmax` | 240 |
| `H1` | 10–1000000 |
| `H2` | 2000000–100000000 |
| `H3` | 200000000–500000000 |
| `H4` | 700000000–2000000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `random` |
| `ContentPadding` | `{0, 0}` (disabled) |
| `RandomTrailers` | `false` |
| `DisableCookies` | `true` |
| Padded sizes | `S1+148`=160, `S2+92`=104, `S3+64`=79, `S4+32`=44 |

**When to use:** corporate LANs with minimal DPI where low overhead is preferred over deep obfuscation.

**Findings:** no `TRL001` (trailers off). The preset's parameters trip one `analyze` heuristic (`RISK007`): `H1` width 999,991 is below 1,000,000.

### home-balanced

| Param | Value |
| --- | --- |
| `S1` | 30 |
| `S2` | 35 |
| `S3` | 20 |
| `S4` | 12 |
| `Jc` | 5 |
| `Jmin` | 250 |
| `Jmax` | 750 |
| `H1` | 100–5000000 |
| `H2` | 10000000–200000000 |
| `H3` | 400000000–800000000 |
| `H4` | 1000000000–2100000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `quic` |
| `ContentPadding` | `{2, 10}` |
| `RandomTrailers` | `true` |
| `DisableCookies` | `true` |
| Padded sizes | `S1+148`=178, `S2+92`=127, `S3+64`=84, `S4+32`=44 |

**When to use:** a good general-purpose default for home internet connections where some DPI may exist but is not aggressive.

**Findings:** expected `TRL001` warning. The preset's parameters trip no `analyze` heuristics.

### mobile-aggressive

| Param | Value |
| --- | --- |
| `S1` | 60 |
| `S2` | 60 |
| `S3` | 50 |
| `S4` | 24 |
| `Jc` | 8 |
| `Jmin` | 500 |
| `Jmax` | 1000 |
| `H1` | 50–10000000 |
| `H2` | 50000000–500000000 |
| `H3` | 700000000–1200000000 |
| `H4` | 1500000000–2147000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `dns` |
| `ContentPadding` | `{2, 10}` |
| `RandomTrailers` | `true` |
| `DisableCookies` | `true` |
| Padded sizes | `S1+148`=208, `S2+92`=152, `S3+64`=114, `S4+32`=56 |

**When to use:** carrier networks with heavy DPI inspection (MTS, Beeline, etc.).

**Findings:** expected `TRL001` warning. The preset's parameters trip one `analyze` heuristic (`RISK005`, info): padded Response 152 is within 4 B of the raw 148 B WireGuard handshake initiation.

### stealth-paranoid

| Param | Value |
| --- | --- |
| `S1` | 30 |
| `S2` | 24 |
| `S3` | 20 |
| `S4` | 32 |
| `Jc` | 10 |
| `Jmin` | 300 |
| `Jmax` | 1100 |
| `H1` | 50–50000000 |
| `H2` | 100000000–600000000 |
| `H3` | 700000000–1300000000 |
| `H4` | 1500000000–2147000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `quic` |
| `ContentPadding` | `{4, 12}` |
| `RandomTrailers` | `true` |
| `DisableCookies` | `true` |
| Padded sizes | `S1+148`=178, `S2+92`=116, `S3+64`=84, `S4+32`=64 |

**When to use:** maximum steady-state masking for hostile DPI (national firewalls, deep statistical inspection). Large `S4` pads every transport packet; highest throughput cost (~2.5% per packet).

**Findings:** expected `TRL001` warning. The preset's parameters trip no `analyze` heuristics, but note that the padded Transport size 64 equals the raw cookie-reply size 64 — `RISK005` requires a strictly non-zero difference, so this exact collision is silently not flagged.

### standard-1420

| Param | Value |
| --- | --- |
| `S1` | 32 |
| `S2` | 28 |
| `S3` | 20 |
| `S4` | 16 |
| `Jc` | 5 |
| `Jmin` | 250 |
| `Jmax` | 800 |
| `H1` | 100–5000000 |
| `H2` | 10000000–200000000 |
| `H3` | 400000000–800000000 |
| `H4` | 1000000000–2100000000 |
| `MTU` | 1420 |
| `DefaultProtocol` | `quic` |
| `ContentPadding` | `{2, 8}` |
| `RandomTrailers` | `true` |
| `DisableCookies` | `true` |
| Padded sizes | `S1+148`=180, `S2+92`=120, `S3+64`=84, `S4+32`=48 |

**When to use:** balanced profile at the classic WireGuard MTU 1420 — same masking strength as `home-balanced` but with more I-packet headroom (`maxISize` 1190) for richer protocol mimicry. Use when the link MTU allows 1420.

> **Warning:** `S4` + `MTU` exceeds the IPv6 Ethernet budget by 16 B. Use an IPv4 outer transport or a jumbo-frame link.

**Findings:** expected `TRL001` warning. The preset's parameters trip no `analyze` heuristics.

### low-overhead

| Param | Value |
| --- | --- |
| `S1` | 12 |
| `S2` | 12 |
| `S3` | 12 |
| `S4` | 12 |
| `Jc` | 2 |
| `Jmin` | 180 |
| `Jmax` | 320 |
| `H1` | 50–50000000 |
| `H2` | 100000000–400000000 |
| `H3` | 500000000–900000000 |
| `H4` | 1100000000–2100000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `dns` |
| `ContentPadding` | `{2, 4}` |
| `RandomTrailers` | `false` |
| `DisableCookies` | `true` |
| Padded sizes | `S1+148`=160, `S2+92`=104, `S3+64`=76, `S4+32`=44 |

**When to use:** bandwidth-constrained links (satellite, metered, slow cellular). `S` values sit at the header-protection floor (12 B), junk count is low, and the cover protocol is DNS; trades masking strength for throughput while remaining fully valid and obfuscated.

**Findings:** no structural warnings — uniform `S` with trailers off means no `TRL001` — and the preset's parameters trip no `analyze` heuristics.

### test-minimal

| Param | Value |
| --- | --- |
| `S1` | 12 |
| `S2` | 13 |
| `S3` | 12 |
| `S4` | 12 |
| `Jc` | 1 |
| `Jmin` | 200 |
| `Jmax` | 250 |
| `H1` | 5–10000 |
| `H2` | 20000–50000 |
| `H3` | 100000–500000 |
| `H4` | 1000000–5000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `random` |
| `ContentPadding` | `{0, 0}` (disabled) |
| `RandomTrailers` | `false` |
| `DisableCookies` | `false` |
| Padded sizes | `S1+148`=160, `S2+92`=105, `S3+64`=76, `S4+32`=44 |

> **Danger:** smallest valid parameter set — **integration testing and CI only. Not intended for production use.** It is also the only preset with `DisableCookies: false`, so copying it into a production manifest silently leaves cookie replies enabled.

**Findings:** no `TRL001` (trailers off). The preset's parameters trip three `analyze` heuristics (`RISK007`): `H1` width 9,996, `H2` width 30,001, `H3` width 400,001 — all below 1,000,000.

> **Note:** `analyze`'s `RISK007` is skipped entirely when the analyzed config carries a `HeaderProtectionKey`, so the H-width verdicts above describe the preset parameters on their own. A 3.1 config generated with header protection (the default) will not report `RISK007`.

## How to use a preset

Presets are **not** a manifest field and **not** a CLI flag. The manifest schema has no `preset` key, no command accepts `--preset`, and `GetPreset`/`ListPresets` have no non-test callers in the repository — a preset reaches the output only through the numbers you copy. Presets resolve at author time, so copy values into `obfuscation.*` by hand or emit the resolved JSON from a Jsonnet library you write (amnezigo ships none).

| Step | Action |
| --- | --- |
| 1 | Pick a preset from the tables above. |
| 2 | Copy `s1`–`s4`, `h1`–`h4` (as `{min, max}`), `jc`, `jmin`, `jmax`, `content_padding` (as `{min, max}` — write `{"min": 0, "max": 0}` explicitly to disable it), `random_trailers` and `disable_cookies` into `obfuscation.*`. |
| 3 | Copy `MTU` into `network.mtu` and the preset's `DefaultProtocol` into the peer's `protocol` field. |
| 4 | Run `amnezigo generate`. The resolved parameters are what reach the output configs. |

`home-balanced` copied into a manifest:

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "awg_version": "3.1",
    "s1": 30,
    "s2": 35,
    "s3": 20,
    "s4": 12,
    "jc": 5,
    "jmin": 250,
    "jmax": 750,
    "h1": { "min": 100, "max": 5000000 },
    "h2": { "min": 10000000, "max": 200000000 },
    "h3": { "min": 400000000, "max": 800000000 },
    "h4": { "min": 1000000000, "max": 2100000000 },
    "content_padding": { "min": 2, "max": 10 },
    "random_trailers": true,
    "disable_cookies": true
  },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "laptop": { "address": "10.0.0.2/24", "protocol": "quic" }
  }
}
```

`awg_version` is optional; unset defaults to `3.1`. All presets are 3.1 data by construction.

> **Warning:** Copy the three 3.1 knobs verbatim — none of them is disabled by omission. An omitted `content_padding` resolves to the 3.1 default `{2, 10}`, and omitted `random_trailers`/`disable_cookies` resolve to `true`. To reproduce `lan-conservative`, `low-overhead` or `test-minimal` you must write `"random_trailers": false` (and `"disable_cookies": false` for `test-minimal`), and to reproduce the disable-padding presets you must write `"content_padding": { "min": 0, "max": 0 }`.

> **Warning:** Set `network.mtu` explicitly. When `network.mtu` is omitted, the client config gets `MTU = 1280` but its `I1`–`I5` currently collapse to the `<t>` placeholder — the client path passes the raw (0) MTU to the I-packet generator before the 1280 default is applied (the server path is correct). Until that defect is fixed, copy the preset's MTU instead of relying on the default.

For the full field semantics see [Manifest Reference](./manifest-reference.md); for more worked inputs see [Manifest Examples](./manifest-examples.md); for emitting preset values programmatically see [Jsonnet](./jsonnet.md).

## API

The preset API lives in package `amnezigo` (`presets.go`); see [Library Usage](./library-usage.md) for the full Go API.

| Symbol | Signature | Description |
| --- | --- | --- |
| `GetPreset` | `func GetPreset(name string) (Preset, error)` | Returns the named preset. On an unknown name, returns `unknown preset %q; available presets: %v` listing all seven names in registry order. |
| `ListPresets` | `func ListPresets() []Preset` | Returns a copy of the seven-entry registry, so callers cannot mutate the built-in presets. |
| `Preset.ToServerObfuscation` | `func (p Preset) ToServerObfuscation() ServerObfuscationConfig` | Copies `Jc`/`Jmin`/`Jmax`, `S1`–`S4`, `H1`–`H4`, `ContentPadding`, `RandomTrailers` and `DisableCookies`, and hardcodes `Version: AWG31`. Omits `MTU`, `Name`, `Description` and `DefaultProtocol`; leaves `HeaderProtectionKey` empty because the generate pipeline supplies it separately. |

`Preset` field reference:

| Field | Type | Notes |
| --- | --- | --- |
| `Name` | `string` | Registry key used by `GetPreset`. |
| `Description` | `string` | Human-readable intent; non-empty for every built-in preset. |
| `DefaultProtocol` | `string` | Cover protocol for the preset; copy into `peers[].protocol`. |
| `H1`–`H4` | `HeaderRange` | Non-overlapping, ascending, outside the WireGuard type-id window `[1..4]`. |
| `MTU` | `int` | Positive and ≤ 1500; copy into `network.mtu`. Not copied by `ToServerObfuscation`. |
| `S1`–`S4` | `int` | All ≥ 12 (header-protection floor); padded sizes pairwise distinct. |
| `Jc`, `Jmin`, `Jmax` | `int` | Junk packet count and range; `Jmin < Jmax`, range excludes padded and raw WireGuard sizes. |
| `ContentPadding` | `U16Range` | AWG 3.1 content-padding addition; `{0, 0}` disables the key. |
| `RandomTrailers` | `bool` | AWG 3.1 random-trailer padding. |
| `DisableCookies` | `bool` | AWG 3.1 cookie replies. |

> **Note:** There is no `Version` field on `Preset`. Presets are 3.1 data by construction — the version appears only as the `Version: AWG31` value that `ToServerObfuscation` sets.

```go
import "github.com/Arsolitt/amnezigo"

p, err := amnezigo.GetPreset("home-balanced")
if err != nil {
    log.Fatal(err)
}
cfg := p.ToServerObfuscation() // ServerObfuscationConfig, Version AWG31
// cfg.HeaderProtectionKey is empty; the pipeline generates it.
// For network.mtu / peers[].protocol, read p.MTU and p.DefaultProtocol.
```

## Gotchas

| Gotcha | Detail |
| --- | --- |
| No `preset` manifest field | The schema has no `preset` key, no command accepts `--preset`, and the manifest comment is explicit that presets are resolved at the Jsonnet level; the manifest only carries resolved parameters. Copy values manually or via your own Jsonnet library. |
| No `Preset.Version` field | The struct has no version field; `ToServerObfuscation` hardcodes `Version: AWG31`, so all presets are 3.1 data by construction. |
| `ToServerObfuscation` drops `MTU` and `DefaultProtocol` | It also drops `Name` and `Description`, and leaves `HeaderProtectionKey` empty. Copy all four fields yourself if you build a manifest or config from the struct. |
| Four presets always emit `TRL001` | `home-balanced`, `mobile-aggressive`, `stealth-paranoid` and `standard-1420` enable random trailers with unequal `S` values; `generate` still succeeds and prints `Warnings: 1`. |
| Pre-validated is not warning-free | Every preset passes the structural checks, but three trip heuristics: `lan-conservative` `RISK007` (H1), `mobile-aggressive` `RISK005` (padded Response 152 vs raw 148), `test-minimal` `RISK007` ×3 (H1/H2/H3). |
| `stealth-paranoid`'s 64 B transport padding is silently unflagged | The padded Transport size 64 equals the raw cookie-reply size 64; `RISK005` requires a strictly non-zero difference, so the collision produces no finding. |
| `DefaultProtocol` is metadata only | It does not drive `generate`. The per-peer `peers[].protocol` is what generate uses (defaults to `quic` when empty). |
| `ContentPadding: {0, 0}` omits the INI key | Zero ranges are skipped during emission, so `ContentPaddingAddition` is absent from the config rather than written as `0-0`. Under AWG 3.1, `RandomTrailers` and `DisableCookies` are always emitted, spelled `on`/`off`. |
| Omitting a 3.1 knob does not reproduce a preset | An omitted `content_padding` resolves to the 3.1 default `{2, 10}`, and omitted `random_trailers`/`disable_cookies` resolve to `true`. `lan-conservative`, `low-overhead` and `test-minimal` need `random_trailers: false` written out; `test-minimal` also needs `disable_cookies: false`; and the disable-padding presets need `content_padding: {min: 0, max: 0}`. |
| `test-minimal` is not for production | It is the smallest valid set, intended for integration tests and CI, and the only preset with `DisableCookies: false`. |
| Copy `network.mtu` explicitly | `ToServerObfuscation` does not carry MTU, and omitting `network.mtu` currently breaks client I-packet sizes (they collapse to `<t>` while `MTU` defaults to 1280). |

For project-wide gotchas, see [Gotchas](./gotchas.md).

## Related

- [Obfuscation](./obfuscation.md) — what each S/H/J parameter means, its constraints, and the CPS tag grammar.
- [Transport Protection (AWG 3.x)](./transport-protection.md) — the `ContentPadding`, `RandomTrailers` and `DisableCookies` layer end to end.
- [Manifest Reference](./manifest-reference.md) — full manifest schema, including `obfuscation.*`, `network.mtu` and `peers[].protocol`.
- [Library Usage](./library-usage.md) — Go API reference for `GetPreset`, `ListPresets` and `Preset.ToServerObfuscation`.
- [Validation & Analysis](./validation.md) — validator finding codes such as `TRL001` and heuristic `RISK*` codes.
- [Jsonnet](./jsonnet.md) — emitting resolved preset values from a Jsonnet manifest generator.
