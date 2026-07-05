# Quick Start

> The three-command round-trip: write a manifest, `generate` configs, `validate` the server, `analyze` for risk.

## Table of Contents

- [The Round-Trip](#the-round-trip)
- [Minimal Manifest](#minimal-manifest)
- [Generate Output Layout](#generate-output-layout)
- [Validate & Analyze](#validate--analyze)
- [Where Next](#where-next)

---

## The Round-Trip

amnezigo is a config generator with exactly three commands. The end-to-end flow is manifest → `generate` → `validate` → `analyze`:

| Step | Command | Result |
|---|---|---|
| 1. Write manifest | _(create `amnezigo.json`)_ | One declarative manifest: a shared obfuscation profile + exactly one server peer + N client peers. See [Minimal Manifest](#minimal-manifest). |
| 2. Generate | `$ amnezigo generate` | Reads `amnezigo.json` (or `.amnezigo.jsonnet`), resolves X25519 keys + per-peer CPS, writes one `awg0.conf` per peer under `output/`. |
| 3. Validate | `$ amnezigo validate output/server/awg0.conf` | Parses the server config with `Strict:true` and runs every AWG 2.0 invariant; exits non-zero on any `error` finding. |
| 4. Analyze | `$ amnezigo analyze --config output/server/awg0.conf` | Heuristic risk report (RISK001–009) + size/range/distribution profile. **Exits 0 on successful load; findings never affect the exit code.** |

Run from a project directory that contains the manifest:

```shell
$ amnezigo generate
$ amnezigo validate output/server/awg0.conf
$ amnezigo analyze --config output/server/awg0.conf
```

For the full flag set of each command, see [./cli-reference.md](./cli-reference.md).

---

## Minimal Manifest

A minimal valid manifest — one server peer (`endpoint` + `listen_port` set), one client peer, and a fully-specified `quic` obfuscation block. Verbatim from `testdata/loader/valid/amnezigo.json`:

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

| Field | Purpose |
|---|---|
| `version` | Manifest schema version; **must be `1`**. |
| `network.mtu` | Per-interface MTU; defaults to `1280` when `0`/unset. |
| `obfuscation.protocol` | Top-level protocol label — informational during `generate`. The per-peer `protocol` (default `quic`) is what drives I-packet generation. |
| `obfuscation.s1`–`s4` | Magic-header prefix sizes (bytes). |
| `obfuscation.h1`–`h4` | Header magic ranges, `{min,max}` (uint32). |
| `obfuscation.jc`/`jmin`/`jmax` | Junk-packet count and size range. |
| `peers.<name>.address` | Interface address (CIDR); required on every peer. |
| `peers.<name>.endpoint` + `listen_port` | Both set → this peer is the **server**. Exactly one server peer is required. |

For every field, type, and default, see [./manifest-reference.md](./manifest-reference.md).

---

## Generate Output Layout

`generate` writes one config per peer under the output directory (default `./output`). The subdirectory name is the peer key from the manifest:

```text
output/
├── server/        # the server peer (endpoint + listen_port set)
│   └── awg0.conf
└── phone/         # one directory per client peer, named after the peer key
    └── awg0.conf
```

Each `awg0.conf` is an INI file (`[Interface]` / `[Peer]`) carrying `#_`-prefixed metadata lines. See [./output-format.md](./output-format.md).

---

## Validate & Analyze

### `validate` — AWG 2.0 invariants

`validate` runs the same checks the generator enforces, against an existing server config:

| Rule family | On failure |
|---|---|
| Required fields (`PrivateKey`, `Address`, `ListenPort`) | `error` · `FLD001` |
| S-prefix constraints (`S1+56 ≠ S2`, bounds) | `error` · `PSC00x` |
| Junk-range ordering (`jmin < jmax`) | `error` · `JNK001` |
| Header-range validity, non-overlap, span ≥ 10⁷ | `error` · `PSC00x` |
| Pre-parse warnings (unknown keys, raw `<c>` tag) | `warning` |

The config is always parsed with `Strict:true`. Exit `0` when there are no errors; exit `1` on any `error` (or any `warning` when `--strict` is set). Flag details: [./cli-reference.md](./cli-reference.md). Finding codes: [./validation.md](./validation.md).

### `analyze` — informational only

| Property | Value |
|---|---|
| Input | server `awg0.conf` (`--config`, default `awg0.conf`) |
| Output | RISK001–009 heuristics + handshake-size / junk / header-range / I-packet profile |
| Exit code | **always `0`** — findings are informational, never errors |

Risk-code meanings: [./validation.md](./validation.md).

---

## Where Next

- [./manifest-reference.md](./manifest-reference.md) — every manifest field, type, and default.
- [./obfuscation.md](./obfuscation.md) — obfuscation parameter ranges and the CPS tag grammar.
- [./output-format.md](./output-format.md) — INI structure and `#_` metadata lines.
- [./credentials.md](./credentials.md) — how keys are reused across runs unless `--full-reset`.
- [./cli-reference.md](./cli-reference.md) — full flag tables for `generate`, `validate`, `analyze`.
