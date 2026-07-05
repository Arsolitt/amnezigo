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

amnezigo ships **7 named presets** in the `presetRegistry` (`presets.go`). Each is a fixed bundle of obfuscation parameters (`S1`–`S4`, `H1`–`H4`, `Jc`/`Jmin`/`Jmax`, `MTU`) tuned for a network environment. The S-prefixes are chosen so the four AWG-padded handshake sizes (`S1+148`, `S2+92`, `S3+64`, `S4+32`) are pairwise distinct, the junk range `[Jmin..Jmax]` excludes all padded and raw WireGuard sizes, and the `H1`–`H4` ranges are non-overlapping and above the WireGuard type-id window `[1..4]`. Presets are data — they are resolved at author time and the manifest carries only the resolved parameters.

| Preset | Intent |
|---|---|
| `lan-conservative` | Corporate LANs with minimal DPI; low overhead over deep obfuscation. |
| `home-balanced` | General-purpose default for home internet; tolerates light DPI. |
| `mobile-aggressive` | Carrier networks with heavy DPI (e.g. MTS, Beeline); maximum entropy. |
| `stealth-paranoid` | Hostile DPI — national firewalls, deep statistical inspection; highest throughput cost (~3 %/packet). |
| `standard-1420` | Same masking as `home-balanced` at WireGuard MTU 1420, with more I-packet headroom. |
| `low-overhead` | Bandwidth-constrained links (satellite, metered, slow cellular); minimum overhead, still valid. |
| `test-minimal` | Smallest valid parameter set; **integration testing and CI only — not for production.** |

## Per-preset parameters

All values are pulled verbatim from `presets.go`. Header ranges are shown as `min–max`.

### lan-conservative

| Param | Value |
|---|---|
| `S1` | 10 |
| `S2` | 10 |
| `S3` | 15 |
| `S4` | 5 |
| `Jc` | 3 |
| `Jmin` | 160 |
| `Jmax` | 240 |
| `H1` | 10–1000000 |
| `H2` | 2000000–100000000 |
| `H3` | 200000000–500000000 |
| `H4` | 700000000–2000000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `random` |

**When to use:** corporate LANs with minimal DPI where low overhead is preferred over deep obfuscation.

### home-balanced

| Param | Value |
|---|---|
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

**When to use:** good default for home internet connections where some DPI may exist but is not aggressive.

### mobile-aggressive

| Param | Value |
|---|---|
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

**When to use:** carrier networks with heavy DPI inspection (MTS, Beeline, etc.).

### stealth-paranoid

| Param | Value |
|---|---|
| `S1` | 30 |
| `S2` | 24 |
| `S3` | 20 |
| `S4` | 40 |
| `Jc` | 10 |
| `Jmin` | 300 |
| `Jmax` | 1100 |
| `H1` | 50–50000000 |
| `H2` | 100000000–600000000 |
| `H3` | 700000000–1300000000 |
| `H4` | 1500000000–2147000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `quic` |

**When to use:** maximum steady-state masking for hostile DPI (national firewalls, deep statistical inspection). Highest throughput cost (~3 % per packet).

### standard-1420

| Param | Value |
|---|---|
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

**When to use:** balanced profile at the classic WireGuard MTU 1420 — same masking strength as `home-balanced` but with more I-packet headroom (`maxISize` 1190). Use when the link MTU allows 1420.

> **Warning:** `S4 + MTU` exceeds the IPv6 Ethernet budget by 16 B. Use IPv4 outer transport or a jumbo-frame link.

### low-overhead

| Param | Value |
|---|---|
| `S1` | 12 |
| `S2` | 10 |
| `S3` | 8 |
| `S4` | 8 |
| `Jc` | 2 |
| `Jmin` | 180 |
| `Jmax` | 320 |
| `H1` | 50–50000000 |
| `H2` | 100000000–400000000 |
| `H3` | 500000000–900000000 |
| `H4` | 1100000000–2100000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `dns` |

**When to use:** bandwidth-constrained links (satellite, metered, slow cellular). `S4` sits at the RISK003 floor (8 B); trades masking strength for throughput while remaining fully valid and obfuscated.

### test-minimal

| Param | Value |
|---|---|
| `S1` | 5 |
| `S2` | 7 |
| `S3` | 7 |
| `S4` | 7 |
| `Jc` | 1 |
| `Jmin` | 200 |
| `Jmax` | 250 |
| `H1` | 5–10000 |
| `H2` | 20000–50000 |
| `H3` | 100000–500000 |
| `H4` | 1000000–5000000 |
| `MTU` | 1280 |
| `DefaultProtocol` | `random` |

> **Danger:** smallest valid parameter set — **integration testing and CI only. Not intended for production use.**

## How to use a preset

Presets are **not** a manifest field and **not** a CLI flag. `generate` never resolves a preset by name — the manifest carries only resolved obfuscation parameters. To use a preset, **copy its values into the `obfuscation.*` block** of your manifest (or into a Jsonnet library that emits them).

| Step | Action |
|---|---|
| 1 | Pick a preset from the tables above. |
| 2 | Copy every value (`s1`–`s4`, `h1`–`h4` as `{min,max}`, `jc`, `jmin`, `jmax`, `network.mtu`) into the manifest. |
| 3 | Run `amnezigo generate`. The resolved parameters are what reach the output configs. |

There is no `preset` key in the manifest schema and no `--preset` flag on any command.

`home-balanced` copied into a manifest:

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "protocol": "quic",
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
    "h4": { "min": 1000000000, "max": 2100000000 }
  },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "laptop": { "address": "10.0.0.2/24" }
  }
}
```

For field semantics and the full manifest schema see [Manifest Reference](./manifest-reference.md). For applying a preset programmatically in Jsonnet, see [Jsonnet](./jsonnet.md).

## API

The preset API lives in package `amnezigo` (`presets.go`). See [Library Usage](./library-usage.md) for the full Go API.

| Symbol | Signature | Description |
|---|---|---|
| `GetPreset` | `func GetPreset(name string) (Preset, error)` | Returns the preset with the given name. On an unknown name, returns an error listing all available preset names. |
| `ListPresets` | `func ListPresets() []Preset` | Returns a copy of all 7 built-in presets. |
| `Preset.ToServerObfuscation` | `func (p Preset) ToServerObfuscation() ServerObfuscationConfig` | Converts a preset's `Jc`/`Jmin`/`Jmax`, `S1`–`S4`, and `H1`–`H4` into a `ServerObfuscationConfig` (no `MTU`, no `DefaultProtocol`). |

```go
import "github.com/Arsolitt/amnezigo"

p, err := amnezigo.GetPreset("home-balanced")
if err != nil {
    log.Fatal(err)
}
cfg := p.ToServerObfuscation() // ServerObfuscationConfig ready for a server peer
```

## Gotchas

| Gotcha | Detail |
|---|---|
| No `preset` manifest field | The schema has no `preset` key, and no CLI command accepts `--preset`. Copy values into `obfuscation.*` manually or via Jsonnet. Any reference to a preset-selection flow from the removed imperative CLI, or to a `Manager` API, is **stale** — that CLI was deleted. |
| `DefaultProtocol` is metadata only | It does **not** drive `generate`. The per-peer `peers[].protocol` is what generate uses (defaults to `quic` when empty). |
| Presets resolve at author time | The manifest carries only resolved parameters; `generate` never re-resolves a preset name. |
| `test-minimal` is not for production | It is the smallest valid set, intended for integration tests and CI only. |
| Every preset is pre-validated | Each passes `ValidatePacketSizes` and `ValidateHeaderRange` with non-overlapping `H1`–`H4` ranges — you do not need to re-validate copied values. |

For project-wide gotchas, see [Gotchas](./gotchas.md).

## Related

- [Obfuscation](./obfuscation.md) — what each parameter means, ranges, and the CPS tag grammar.
- [Manifest Reference](./manifest-reference.md) — full manifest schema including `obfuscation.*` and `peers[].protocol`.
- [Library Usage](./library-usage.md) — Go API reference for `GetPreset`, `ListPresets`, and `Preset.ToServerObfuscation`.
- [Jsonnet](./jsonnet.md) — applying preset values programmatically via `lib/presets.libsonnet`.
