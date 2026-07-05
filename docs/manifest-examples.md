# Manifest Examples

> Complete, valid `amnezigo.json` / `.amnezigo.jsonnet` manifests with field-by-field explainers. Every value below is lifted from source fixtures or the preset registry — no invented fields or numbers.

## Table of Contents

- [Conventions](#conventions)
- [Minimal Two-Peer Manifest](#minimal-two-peer-manifest)
- [Three-Peer Manifest (Per-Peer Protocol)](#three-peer-manifest-per-peer-protocol)
- [Fully-Random Obfuscation](#fully-random-obfuscation)
- [Explicit Obfuscation (Preset Values)](#explicit-obfuscation-preset-values)
- [Jsonnet Manifest Skeleton](#jsonnet-manifest-skeleton)

---

## Conventions

Two rules hold for every example on this page:

1. **`version` MUST be `1`.** It is the only schema version the loader accepts (`currentManifestVersion` in `loader.go`). Any other value — including `0` and omitted — fails to load.
2. **`.amnezigo.jsonnet` takes precedence over `amnezigo.json`.** When both files exist in the project directory, Jsonnet wins and the JSON file is ignored entirely.

Run any example against `amnezigo generate` after saving it; check it without writing files with `--dry-run`:

```shell
$ amnezigo generate --dry-run
$ amnezigo generate
```

For the full field reference, see [./manifest-reference.md](./manifest-reference.md).

## Minimal Two-Peer Manifest

One server peer (`endpoint` + `listen_port` set) and one client peer. Byte-accurate copy of `testdata/loader/valid/amnezigo.json`; the obfuscation values are identical to the `home-balanced` preset.

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "protocol": "quic",
    "s1": 30, "s2": 35, "s3": 20, "s4": 12,
    "h1": {"min": 100, "max": 5000000},
    "h2": {"min": 10000000, "max": 200000000},
    "h3": {"min": 400000000, "max": 800000000},
    "h4": {"min": 1000000000, "max": 2100000000},
    "jc": 5, "jmin": 250, "jmax": 750
  },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": {
      "address": "10.0.0.2/32"
    }
  }
}
```

| Field | Value | Why |
|---|---|---|
| `version` | `1` | Required; only accepted schema version. |
| `network.mtu` | `1280` | Interface MTU shared by all peers; AWG MTU is per-interface, not per-peer. |
| `obfuscation.protocol` | `"quic"` | Informational label only — `generate` does not consume the top-level protocol. Per-peer `protocol` (below) drives I-packet templates. |
| `obfuscation.s1`–`s4` | `30, 35, 20, 12` | AWG handshake size prefixes (bytes). `home-balanced` values; produce pairwise-distinct padded sizes (S1+148=178, etc.). |
| `obfuscation.h1`–`h4` | `{min,max}` ranges | Header-range windows (uint32). Four ranges are non-overlapping and sit above the WG type-id window `[1..4]`. |
| `obfuscation.jc`/`jmin`/`jmax` | `5 / 250 / 750` | Junk-packet count and size range. `[jmin..jmax]` excludes all padded and raw WG sizes. |
| `peers.server.address` | `10.0.0.1/24` | Server tunnel address with subnet mask — the /24 covers the client pool. |
| `peers.server.endpoint` | `vpn.example.com:51820` | Non-empty endpoint is one of the two server-peer markers (`IsServer`). |
| `peers.server.listen_port` | `51820` | Non-zero listen port is the other server-peer marker. Exactly one server peer is valid. |
| `peers.phone.address` | `10.0.0.2/32` | Client peer; host route (/32). No `endpoint`/`listen_port` → treated as a client. |

## Three-Peer Manifest (Per-Peer Protocol)

One server plus two clients, each pinning a distinct `protocol`. This illustrates **per-peer protocol selection**: the `peers.<name>.protocol` field chooses the I-packet template for that peer, independently.

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "protocol": "quic",
    "s1": 30, "s2": 35, "s3": 20, "s4": 12,
    "h1": {"min": 100, "max": 5000000},
    "h2": {"min": 10000000, "max": 200000000},
    "h3": {"min": 400000000, "max": 800000000},
    "h4": {"min": 1000000000, "max": 2100000000},
    "jc": 5, "jmin": 250, "jmax": 750
  },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820,
      "protocol": "quic"
    },
    "phone": {
      "address": "10.0.0.2/32",
      "protocol": "quic"
    },
    "laptop": {
      "address": "10.0.0.3/32",
      "protocol": "dns"
    }
  }
}
```

| Field | Value | Why |
|---|---|---|
| `peers.server.protocol` | `"quic"` | I-packet template for the server peer. |
| `peers.phone.protocol` | `"quic"` | Phone client mimics QUIC; independent of the server's choice. |
| `peers.laptop.protocol` | `"dns"` | Laptop client mimics DNS — a different template on the same shared S/H/J profile. |
| (all other fields) | as above | The obfuscation S/H/J block is server-wide and shared; only the per-peer template differs. |

> **Note:** valid per-peer protocols are `quic`, `dns`, `dtls`, `stun`, `sip`, `rtp`, plus `random`. See [./obfuscation.md](./obfuscation.md). An empty `protocol` defaults to `quic`.

## Fully-Random Obfuscation

Omit every S/H/J field. Each pointer (`*int`, `*HeaderRange`) is `nil`, so `resolveObfuscation` fills all eleven values with random numbers that satisfy the AWG 2.0 invariants.

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {},
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": {
      "address": "10.0.0.2/32"
    }
  }
}
```

| Field | Value | Why |
|---|---|---|
| `obfuscation` | `{}` | Empty object → all S1–S4, H1–H4, Jc/Jmin/Jmax pointers are `nil` → random fallback (`HasAnyValue()` returns false). |
| `obfuscation.protocol` | (omitted) | Non-pointer string; empty by omission. The per-peer template (also empty here) defaults to `quic`. |
| (partial variants) | e.g. set only `s1`–`s4` | Per-field fallback: any non-nil field is used as-is; only nil fields are randomized. |

> **Tip:** partial specification is legal — pin the fields you care about and let the rest randomize. Full rules in [./manifest-reference.md](./manifest-reference.md); parameter ranges in [./obfuscation.md](./obfuscation.md).

## Explicit Obfuscation (Preset Values)

There is **no `preset` field** in the manifest schema. To use a preset, copy its numeric values into `obfuscation.*`. This block reproduces the **`mobile-aggressive`** preset verbatim from `presets.go` (heavy-DPI carrier networks).

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "protocol": "dns",
    "s1": 60, "s2": 60, "s3": 50, "s4": 24,
    "h1": {"min": 50, "max": 10000000},
    "h2": {"min": 50000000, "max": 500000000},
    "h3": {"min": 700000000, "max": 1200000000},
    "h4": {"min": 1500000000, "max": 2147000000},
    "jc": 8, "jmin": 500, "jmax": 1000
  },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": {
      "address": "10.0.0.2/32"
    }
  }
}
```

| Field | Value | Why |
|---|---|---|
| `s1`–`s4` | `60, 60, 50, 24` | `mobile-aggressive` prefixes — large, for maximum entropy under carrier DPI. |
| `h1`–`h4` | ranges above | `mobile-aggressive` header windows; wide, non-overlapping, above `[1..4]`. |
| `jc`/`jmin`/`jmax` | `8 / 500 / 1000` | High junk count, wide range — `mobile-aggressive` defaults. |
| `protocol` | `"dns"` | Matches the preset's `DefaultProtocol` (`protocolDNS`). |

> The full table of all seven presets and their exact values is in [./presets.md](./presets.md).

## Jsonnet Manifest Skeleton

Jsonnet is preferred when manifests grow: it adds comments, variables, imports, and functions on top of JSON. This skeleton imports a shared network fragment, defines a `local` client-builder function, and generates three clients with `std.range` + a comprehension. The object shape it evaluates to is identical to the JSON examples above.

```jsonnet
// .amnezigo.jsonnet
local net = import 'network.libsonnet';
local mkClient(addr) = { address: addr };
{
  version: 1,
  network: net,
  obfuscation: {
    protocol: 'quic',
    s1: 30, s2: 35, s3: 20, s4: 12,
    h1: { min: 100, max: 5000000 },
    h2: { min: 10000000, max: 200000000 },
    h3: { min: 400000000, max: 800000000 },
    h4: { min: 1000000000, max: 2100000000 },
    jc: 5, jmin: 250, jmax: 750,
  },
  peers: {
    server: {
      address: '10.0.0.1/24',
      endpoint: 'vpn.example.com:51820',
      listen_port: 51820,
    },
  } + {
    ['client' + n]: mkClient('10.0.0.' + n)
    for n in std.range(2, 4)
  },
}
```

| Element | Role | Why it matters |
|---|---|---|
| `local net = import 'network.libsonnet'` | Import binding | Reuses a shared `network` fragment across manifests; resolved via the default `lib/` jpath or `--jpath`. |
| `local mkClient(addr) = { address: addr }` | Local function | Defines a peer template once; called per client to avoid copy-paste. |
| `std.range(2, 4)` | std library function | Generates the host-address octets `2..4` programmatically. |
| `{ ... } + { [...] for n in std.range(2,4) }` | Object comprehension + merge | Builds `client2`, `client3`, `client4` peers and merges them with the server peer. |
| single quotes / trailing commas / unquoted keys | Jsonnet syntax | Legal in Jsonnet; the loader evaluates the file to JSON before parsing. |

> **Warning:** when `.amnezigo.jsonnet` exists, a sibling `amnezigo.json` is **ignored** — Jsonnet always wins. A missing import (e.g. `import 'nonexistent.libsonnet'`) is a hard load error with no silent fallback. Full rules: [./jsonnet.md](./jsonnet.md).
