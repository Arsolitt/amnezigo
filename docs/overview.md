# Overview

> What amnezigo is, the declarative model it is built on, and the three commands it exposes.

## Table of Contents

- [What Amnezigo Is](#what-amnezigo-is)
- [The Declarative Manifest Concept](#the-declarative-manifest-concept)
- [What `amnezigo generate` Produces](#what-amnezigo-generate-produces)
- [Key Concepts](#key-concepts)
- [The Three Commands](#the-three-commands)
- [Where To Go Next](#where-to-go-next)

---

## What Amnezigo Is

amnezigo is a **configuration generator** for [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0 — it is **not a daemon** and never runs a tunnel. It reads one declarative manifest, resolves AmneziaWG obfuscation parameters and X25519 cryptography, and emits ready-to-deploy `awg0.conf` files (one server config plus one per client peer). The same logic is also exposed as an importable Go library.

| Property | Value |
|---|---|
| Language | Go |
| Module | `github.com/Arsolitt/amnezigo` |
| Go version | 1.26+ |
| License | GPL-3.0 |
| Commands | `generate`, `validate`, `analyze` |
| Output artifact | INI `awg0.conf` per peer; optional `amnezigo.vpn` import link per client (`--vpn-links`) |
| Library entry point | `amnezigo.Generate` |

## The Declarative Manifest Concept

The entire network topology — global settings, obfuscation profile, and all peers — is declared in a **single file**: `amnezigo.json` or `.amnezigo.jsonnet`. The manifest drives `amnezigo generate`; there is no imperative add/edit flow. The `version` field MUST be `1` (the only schema version the loader supports). If both files exist, `.amnezigo.jsonnet` takes precedence and is evaluated to JSON before parsing.

| Model | Status | Commands |
|---|---|---|
| Imperative (`init`/`add`/`edit`/`remove`/`export`/`list`) | **Removed** (commit `226e4b8`) | none |
| Declarative manifest | **Current** | `generate`, `validate`, `analyze` |

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": { "protocol": "quic" },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": { "address": "10.0.0.2/32" }
  }
}
```

See [Manifest Reference](./manifest-reference.md) for every field and [Jsonnet](./jsonnet.md) for the `.amnezigo.jsonnet` evaluation rules.

## What `amnezigo generate` Produces

`generate` writes one `awg0.conf` per peer under the output directory. The server peer's config holds shared obfuscation parameters (S/H/J) plus a `[Peer]` block for every client; each client config points back at the server. INI files carry `#_`-prefixed metadata lines (e.g. `#_Name`, `#_TunName`) alongside the WireGuard keys that let later runs reuse peer credentials (see [Credentials](./credentials.md) and [Output Format](./output-format.md)).

```text
output/
├── server/            # the server peer (Endpoint + ListenPort set)
│   └── awg0.conf      # [Interface] + one [Peer] per client
└── phone/             # a client peer
    ├── awg0.conf      # [Interface] + single [Peer] → server
    └── amnezigo.vpn   # optional: AmneziaVPN import link (--vpn-links)
```

With `--vpn-links`, each client peer also gets an `amnezigo.vpn` file containing a `vpn://` import link for the AmneziaVPN app (see [VPN Import Links](./vpn-links.md)).

Generation is **two-pass atomic**: every config is built in memory first; if any build fails, no files are written to disk.

## Key Concepts

### Server peer vs client peer

A peer is the **server** iff it has both a non-empty `endpoint` AND a non-zero `listen_port`. Every other peer is a client. A valid manifest contains **exactly one** server peer; validation rejects zero or more than one.

| Peer kind | Detection rule | Role | Count |
|---|---|---|---|
| Server | `endpoint != ""` AND `listen_port != 0` | Listens, holds shared obfuscation + all client `[Peer]` blocks | exactly 1 |
| Client | otherwise | Single `[Peer]` block pointing at the server | 0 or more |

### Shared obfuscation vs per-client custom packets

| Scope | What is shared | Where it lives |
|---|---|---|
| Network-wide (shared) | S1–S4 size prefixes, H1–H4 header ranges, junk packet params (`Jc`/`Jmin`/`Jmax`) | Server config; copied into every client config |
| Per-client (unique) | I1–I5 custom packet strings (CPS) | Differ per client; `I5` is always empty |

See [Obfuscation](./obfuscation.md) for parameter ranges and the CPS tag grammar.

### Generation flow

```text
amnezigo.json | .amnezigo.jsonnet
        │
        ▼   LoadManifest (jsonnet precedence; version must be 1)
   Manifest
        │
        ▼   Generate  (two-pass atomic: build all in memory, then write)
   output/<server>/awg0.conf
   output/<peer>/awg0.conf
        │
        ▼   validate / analyze
   findings / heuristics
```

## The Three Commands

| Command | Purpose | Reference |
|---|---|---|
| `amnezigo generate` | Read the manifest and write per-peer `awg0.conf` files (reusing persisted keys unless `--full-reset`). | [CLI Reference](./cli-reference.md) |
| `amnezigo validate` | Check one or more generated configs against AmneziaWG v2.0 invariants. | [CLI Reference](./cli-reference.md) |
| `amnezigo analyze` | Inspect obfuscation strength with heuristic risk findings. | [CLI Reference](./cli-reference.md) |

The CLI is a thin Cobra wrapper over the root `amnezigo` package (`cmd/amnezigo/main.go` → `cli.Execute()`).

## Where To Go Next

- [Installation](./installation.md) — build or install the `amnezigo` binary.
- [Quick Start](./quick-start.md) — minimal manifest to generated configs in three commands.
- [Manifest Reference](./manifest-reference.md) — every manifest field, type, and default.
- [Obfuscation](./obfuscation.md) — obfuscation parameters, CPS tags, and protocol templates.
