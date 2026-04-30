# Obfuscation Parameters

> How AmneziaWG traffic obfuscation works and what each parameter controls.

## Table of Contents

- [Overview](#overview)
- [Junk Packet Parameters](#junk-packet-parameters)
- [Size Prefixes](#size-prefixes)
- [Header Ranges](#header-ranges)
- [Custom Packet Strings](#custom-packet-strings)
- [CPS Tag Syntax](#cps-tag-syntax)
- [Protocol Templates](#protocol-templates)
- [Server vs Client Obfuscation](#server-vs-client-obfuscation)
- [MTU Considerations](#mtu-considerations)

---

## Overview

AmneziaWG wraps WireGuard traffic in an obfuscation layer to make it resemble legitimate protocol traffic. This defeats deep packet inspection (DPI) by disguising the VPN tunnel as QUIC, DNS, DTLS, STUN, SIP, or other supported traffic shapes.

Obfuscation has two layers:

1. **Server-side parameters** — generated once at `init` and stored in the server config (`Jc`, `Jmin`, `Jmax`, `S1`–`S4`, `H1`–`H4`)
2. **Client-side CPS strings** — generated per-peer at export time (`I1`–`I5`)

See the [configuration](configuration.md) page for where these parameters appear in config files, or the [library-usage](library-usage.md) page for the programmatic API.

---

## Junk Packet Parameters

Junk packets are injected between real WireGuard packets to add noise and disrupt traffic analysis.

| Parameter | Range | Description |
|-----------|-------|-------------|
| `Jc` | 0–10 | Number of junk packets to send after each real packet |
| `Jmin` | 64–1024 | Minimum junk packet size in bytes |
| `Jmax` | 64–1024 | Maximum junk packet size in bytes |

`Jmin` must be less than `Jmax`. The generator ensures this by swapping the values if needed, or incrementing `Jmax` by 1 if they end up equal.

Setting `Jc` to `0` disables junk packet injection entirely.

---

## Size Prefixes

Size prefixes (`S1`–`S4`) modify packet sizes to prevent size-based fingerprinting.

| Parameter | Range | Description |
|-----------|-------|-------------|
| `S1` | 0–64 | Size prefix 1 |
| `S2` | 0–64 | Size prefix 2 |
| `S3` | 0–64 | Size prefix 3 |
| `S4` | 0–32 | Size prefix 4 |

> **Note:** There is a hidden constraint: `S1 + 56` must **not** equal `S2`. This avoids an Init/Response size collision in the AmneziaWG protocol. The generator retries until this constraint is satisfied — you don't need to worry about it when using the built-in generation functions.

---

## Header Ranges

Header ranges (`H1`–`H4`) define the range of valid header values that obfuscated packets can use.

### Structure

Each header range is a `HeaderRange` with two fields:

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `Min` | `uint32` | 5–2147483647 | Lower bound of the range |
| `Max` | `uint32` | 5–2147483647 | Upper bound of the range |

### Constraints

- `Min` must be less than `Max`
- Each range must span at least 10,000,000 values (`Max - Min >= 10,000,000`)
- All four ranges must be non-overlapping
- Ranges are sorted by `Min` value after generation

> **Warning:** `GenerateHeaderRanges` panics if it cannot generate 4 non-overlapping ranges after 1000 attempts. In practice this is extremely unlikely given the valid range (5 to 2,147,483,647).

---

## Custom Packet Strings

Custom Packet Strings (CPS) are the core of protocol mimicry. They define what the obfuscated packet payloads look like.

There are five CPS intervals: `I1` through `I5`. Each interval is a string of CPS tags that construct a fake protocol payload:

| Interval | Description |
|----------|-------------|
| `I1` | Interval 1 — largest payload, used for initial handshake |
| `I2` | Interval 2 — medium-large payload |
| `I3` | Interval 3 — medium payload |
| `I4` | Interval 4 — smallest payload |
| `I5` | Interval 5 — always empty for named protocol templates |

> **Note:** I1–I5 are **client-only** fields. They are not stored in the server config — they are generated fresh for each peer at export time. See [configuration](configuration.md) for details on what appears in each config type.

---

## CPS Tag Syntax

CPS strings are composed of tags enclosed in angle brackets. Each tag produces a specific type of payload data.

### Tag Types

| Tag | CPS Type | Description | Size |
|-----|----------|-------------|------|
| `<b 0xNN>` | `bytes` | Fixed byte sequence (hex) | `len(NN)/2` bytes |
| `<r N>` | `random` | N random bytes | N bytes |
| `<rc N>` | `random_chars` | N random ASCII characters | N bytes |
| `<rd N>` | `random_digits` | N random digit characters | N bytes |
| `<t>` | `timestamp` | Timestamp value | 8 bytes |
| `<d>` | `data` | Reuse a value from an earlier I-packet at runtime | 0 bytes (passthrough) |

### Examples

```text
<b 0xc0ff>          — Two fixed bytes: 0xC0, 0xFF
<r 8>               — Eight random bytes
<rc 7>              — Seven random ASCII characters (e.g., "aB3xK9p")
<rd 2>              — Two random digits (e.g., "47")
<t>                 — 8-byte timestamp
<d>                 — Runtime passthrough: reuse a value from an earlier I-packet
                       (only meaningful inside templates with multi-interval chaining)
```

A complete chained template might look like:

```text
I1: <b 0xc0ff><b 0x08><r 8><t>          — fresh DCID + timestamp
I2: <b 0xc0ff><b 0x08><d><b 0x01>       — same DCID via <d>, then packet number
```

A complete CPS interval might look like:

```text
<b 0xc0ff><b 0x00000001><b 0x08><r 8><t><r 40>
```

This produces: 2 fixed bytes + 4 fixed bytes + 1 fixed byte + 8 random bytes + 8 timestamp bytes + 40 random bytes = 63 bytes total.

> **Warning:** `BuildCPSTag` returns an empty string for unknown tag types without raising an error. If a CPS string ends up shorter than expected, check for typos in tag types.

> **Note:** When generating random CPS tags, value ranges are: `<b 0xNN>` produces 4–16 bytes of hex, `<r N>`, `<rc N>`, and `<rd N>` use 5–40 as the range for `N`.

> **Note:** `<d>` requires AmneziaWG 2.0 userspace (Go reference, AmneziaVPN mobile and desktop clients). The legacy `amneziawg-linux-kernel-module` does not recognise it and will reject configs containing `<d>` with "unknown tag". Kernel-module-only deployments must use `--protocol random` (which never emits `<d>`) or omit `<d>` from custom templates.

---

## Protocol Templates

Protocol templates define the tag structure for each `I1`–`I4` interval. When you export a peer with `--protocol quic`, the corresponding template is used to generate the CPS strings.

### Available Protocols

| Protocol | What It Mimics | Key Characteristics |
|----------|---------------|---------------------|
| `quic` | QUIC Version 1 | Long Header form `0xC0FF`, version `0x00000001` |
| `dns` | DNS A record query | Query type `0x0001`, class IN `0x0001` |
| `dtls` | DTLS 1.2 ClientHello | Version `0xFEFD`, specific cipher suites |
| `stun` | STUN Binding Request | Magic cookie `0x2112A442`, message type `0x0001` |
| `sip` | SIP OPTIONS request | Method literal `OPTIONS `, ASCII line-delimited headers, no timestamp |
| `random` | None (random tags) | 3–6 random tags per interval, at most one `<t>` per interval |

### Template Structure

All named protocol templates share these properties:

- `I1` is the largest interval (most tags)
- `I4` is the smallest interval (fewest tags)
- `I5` is always empty

> **Note:** While `I1` is typically the largest interval and `I4` the smallest, the exact ordering of `I2` and `I3` relative to `I1` varies by protocol. For example, the STUN template has `I2` larger than `I1`.

### The "random" Protocol

There are two distinct code paths for random behavior:

1. **`--protocol random`** in `GenerateCPS` — generates simple random CPS with 3–6 random tags per interval. This is a distinct "random" mode.
2. **Unknown protocol string** in `getTemplate()` — silently falls back to a randomly selected named protocol template (QUIC, DNS, DTLS, STUN, or SIP). This is **not** the same as simple random CPS.

Use the `export --protocol` option to select a protocol. See [cli-reference](cli-reference.md) for details.

---

## Server vs Client Obfuscation

The obfuscation system splits parameters between server and client configs:

### Server Config

The server config contains the shared obfuscation parameters, generated once at `init`:

- `Jc`, `Jmin`, `Jmax` — junk packet settings
- `S1`–`S4` — size prefix values
- `H1`–`H4` — header ranges (stored as `"min-max"` string format, e.g., `"10000000-20000000"`)

These are written to the `[Interface]` section of the server config.

### Client Config

The client config embeds all server parameters **plus** the CPS strings:

- `I1`–`I5` — Custom Packet Strings (generated per-peer at export)
- All server-side parameters are copied to the client

> **Tip:** Since `I1`–`I5` are generated at export time, each peer can have different CPS strings even if they use the same protocol. This means every client's traffic looks unique to observers.

### Key Difference: Header Ranges

On the **server**, `H1`–`H4` are stored as **ranges** (min–max pairs). The server accepts any header value within those ranges. On the **client**, `H1`–`H4` are stored as **point values** (min equals max) — a single value picked within the server's range. This ensures the client sends headers the server will accept.

> **Note:** The conversion from server ranges to client point values happens in `GenerateConfig()`, which is called during export. The server config itself only stores ranges.

---

## MTU Considerations

CPS strings have a maximum length determined by the MTU. If the generated CPS is too long, tags are progressively removed until it fits.

### The Formula

```
maxISize = MTU - 49 - 149 - S1
```

Where:
- **49** — protocol overhead reserve
- **149** — handshake size
- **S1** — size prefix 1 (larger S1 means less room for CPS)

### Examples

| MTU | S1 | maxISize | Formula |
|-----|----|----|---------|
| 1280 | 32 | 1050 | 1280 − 49 − 149 − 32 = 1050 |
| 1420 | 64 | 1158 | 1420 − 49 − 149 − 64 = 1158 |

### Fallback Behavior

When a protocol template's CPS exceeds `maxISize`:

1. Tags are removed one at a time from the **end** of the interval
2. If still too large, more tags are removed
3. The ultimate fallback is a single `<t>` tag (8 bytes), which always fits

This means even with a very low MTU or large S1 value, obfuscation will still work — just with less realistic protocol mimicry.

> **Tip:** If you need full protocol mimicry, ensure your MTU is large enough. The default WireGuard MTU of 1420 works well for all protocol templates.
