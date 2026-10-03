# Overview

> What amnezigo is, the declarative model it is built on, and how one manifest becomes deployable AmneziaWG configs.

## Table of Contents

- [What Amnezigo Is](#what-amnezigo-is)
- [The Declarative Manifest Concept](#the-declarative-manifest-concept)
- [What `amnezigo generate` Produces](#what-amnezigo-generate-produces)
- [The AWG Version Model](#the-awg-version-model)
- [Transport Protection at a Glance](#transport-protection-at-a-glance)
- [Key Concepts](#key-concepts)
- [The Commands](#the-commands)
- [Project Layout](#project-layout)
- [Related](#related)

---

## What Amnezigo Is

amnezigo is a **configuration generator** for [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) 2.0 / 3.0 / 3.1 (default 3.1). It is **not a daemon** and never runs a tunnel: it reads one declarative manifest, resolves the AmneziaWG obfuscation parameters and X25519 cryptography, and emits ready-to-deploy `awg0.conf` files — one server config plus one per client peer. The same logic is exposed as an importable Go library, so the CLI and your own programs share one implementation.

| Property | Value |
| --- | --- |
| Language | Go |
| Module | `github.com/Arsolitt/amnezigo` |
| Go version | 1.26.1 |
| License | GPL-3.0 |
| AmneziaWG versions | 2.0, 3.0, 3.1 (default 3.1) |
| Commands | `generate`, `validate`, `analyze`, `version` |
| Output artifact | INI `awg0.conf` per peer; optional `amnezigo.vpn` import link per client (`--vpn-links`) |
| Library entry point | `amnezigo.Generate` |

## The Declarative Manifest Concept

The entire network topology — global settings, obfuscation profile, and all peers — is declared in a **single file**: `amnezigo.json` or `.amnezigo.jsonnet`. That file drives `amnezigo generate`; there is no imperative add/edit flow. The `version` field of the manifest MUST be `1` (the only schema version the loader supports) — `0` fails with `missing or zero version field`, any other value with `unsupported schema version %d (expected 1)`.

If both files exist, `.amnezigo.jsonnet` wins and is evaluated to JSON before parsing. If neither exists, loading fails with `no manifest file found in <dir> (expected .amnezigo.jsonnet or amnezigo.json)`. See [Manifest Reference](./manifest-reference.md) for every field and [Jsonnet](./jsonnet.md) for the `.amnezigo.jsonnet` evaluation rules.

| Model | Status | Commands |
| --- | --- | --- |
| Imperative (`init`/`add`/`edit`/`remove`/`export`/`list`) | **Removed** | none |
| Declarative manifest | **Current** | `generate`, `validate`, `analyze` |

A minimal 3.1 manifest looks like this:

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": { "awg_version": "3.1" },
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

`obfuscation.awg_version` is optional and defaults to `3.1`; leave it out and you get the same output. Set `"awg_version": "2.0"` to target a legacy runtime, or `"3.0"` for 3.0 — the version decides which generation of INI keys the run is allowed to emit (see [The AWG Version Model](#the-awg-version-model)).

## What `amnezigo generate` Produces

`generate` writes one `awg0.conf` per peer under the output directory (`<project>/output` by default, `--output` to change it). The server peer's config holds the shared obfuscation parameters (S/H/J plus the 3.x transport-protection block) and one `[Peer]` section per client; each client config points back at the server. INI files carry `#_`-prefixed metadata lines (such as `#_Name` and `#_TunName`) alongside the WireGuard keys, and later runs reuse peer credentials recovered from those same files (see [Credentials](./credentials.md) and [Output Format](./output-format.md)).

```text
output/
├── server/            # the server peer (endpoint + listen_port set)
│   └── awg0.conf      # [Interface] + one [Peer] per client
└── phone/             # a client peer
    ├── awg0.conf      # [Interface] + single [Peer] → server
    └── amnezigo.vpn   # optional: AmneziaVPN import link (--vpn-links)
```

With `--vpn-links`, each client peer also gets an `amnezigo.vpn` file containing a `vpn://` import link for the AmneziaVPN app (see [VPN Import Links](./vpn-links.md)).

Generate computes every config **in memory** before writing anything, so a build failure aborts the run with the filesystem untouched. The write phase itself is not transactional: directories are created with mode `0750` and files written with `0600` one by one, with no temp-file-and-rename step and no rollback, so an I/O error partway through can leave earlier files updated and later ones stale.

> **Danger:** `--full-reset` discards every persisted credential — the server and client key pairs, every PresharedKey, and the 3.x `HeaderProtectionKey`. All previously distributed configs stop working with the newly generated server config. Treat it as fleet-wide re-provisioning, not a local refresh.

## The AWG Version Model

amnezigo models three AmneziaWG protocol generations, each a superset of the previous one (`AWG20`, `AWG30`, `AWG31` in the Go API, where `String()` returns the dotted label). `obfuscation.awg_version` selects the target; it accepts exactly `""` (unset), `"2.0"`, `"3.0"`, and `"3.1"`, and anything else fails with `unsupported awg_version %q (expected "2.0", "3.0", or "3.1")`. The default is `3.1` (`DefaultAWGVersion`), so a manifest that never mentions a version generates 3.1 output.

| `awg_version` | Enum | What it adds on top of the previous generation |
| --- | --- | --- |
| `"2.0"` | `AWG20` | Junk packets (`Jc`/`Jmin`/`Jmax`), S-prefixes (`S1`–`S4`), header ranges (`H1`–`H4`), CPS I-packets (`I1`–`I5`) |
| `"3.0"` | `AWG30` | `HeaderProtectionKey`, `ContentPaddingAddition`, and five uint16 ranges: `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts` |
| `"3.1"` | `AWG31` | `RandomTrailers` and `DisableCookies` |

The generation is also a **gate**, not just a label. Manifest fields belonging to a newer generation are hard errors rather than silently ignored, because engine builds reject INI keys they do not know (`Line unrecognized`), so silently dropping a field would hide a configuration mistake:

- `header_protection`, `content_padding`, `rekey_after_time`, `rekey_timeout`, `reject_after_time`, `keepalive_timeout`, and `max_handshake_attempts` require version 3.0 or later.
- `random_trailers` and `disable_cookies` require version 3.1.

For example, a manifest that pins `"awg_version": "2.0"` but sets `random_trailers` fails with:

```text
obfuscation.random_trailers requires awg_version 3.1 or later (got "2.0")
```

The version is enforced a second time in the writer: the seven 3.0 keys (`HeaderProtectionKey` plus the six ranges) are emitted only at version 3.0 or later, and `RandomTrailers`/`DisableCookies` additionally require 3.1 — so a hand-built `ServerObfuscationConfig` with a zero version produces exactly the 2.0 key set. For the full design of the 3.x layer — every knob, default, range rule, and INI key — see [Transport Protection](./transport-protection.md).

## Transport Protection at a Glance

Starting with 3.0 (and on by default in 3.0 and 3.1), the generator emits a device-level transport-protection block into both the server and every client config. Under 3.1 defaults it looks like this:

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

The mental model:

- **Header protection** is a shared ChaCha20 key that encrypts packet headers. The key is 32 random bytes, emitted base64-encoded (44 characters) as `HeaderProtectionKey`, and must match on both ends. Set `header_protection: false` to turn it off; the other 3.x keys are still emitted.
- **S-prefix floor.** Header protection encrypts the 4-byte message header at offset `S<n>`, and the ChaCha20 nonce is the first 12 bytes of the packet — so an S-prefix shorter than 12 bytes is refused. A smaller pinned value aborts generation: `header protection requires S1-S4 >= 12 (got S<n>=<value>)`.
- **Header ranges.** With header protection on and `H1`–`H4` left unset, the generator uses the WireGuard message type-ids as ranges: `H1 = 1-1`, `H2 = 2-2`, `H3 = 3-3`, `H4 = 4-4`. The type field is encrypted now, so the old type-id avoidance no longer applies.
- **Random trailers.** With trailers on and all four S values unset, one random draw is used for `S1 = S2 = S3 = S4` (the 3.1 uniform-S recommendation). Pinning even one S value switches the other three to independent draws, which makes `RandomTrailers` and differing S values coexist and triggers the `TRL001` warning on validation.
- **Key persistence.** The header-protection key is generated once, stored inside `output/<server>/awg0.conf`, and recovered from there on every later run; only `--full-reset` rotates it.

> **Danger:** `HeaderProtectionKey` is a normal (non-`#_`) INI key duplicated in the server and every client config. Anyone who can read any config can decrypt header-protected packet headers. Treat client configs as secrets — they already contain private keys.

## Key Concepts

### Server peer vs client peer

A peer is the **server** if and only if it has both a non-empty `endpoint` and a non-zero `listen_port`. Every other peer is a client. A valid manifest contains **exactly one** server peer; generation fails with `exactly one server peer required, found <n>` for zero or more than one.

| Peer kind | Detection rule | Role | Count |
| --- | --- | --- | --- |
| Server | `endpoint != ""` AND `listen_port != 0` | Listens, holds shared obfuscation + all client `[Peer]` blocks | exactly 1 |
| Client | otherwise | Single `[Peer]` block pointing at the server | 0 or more |

### Shared obfuscation vs per-client custom packets

| Scope | What is shared | Where it lives |
| --- | --- | --- |
| Network-wide (shared) | S1–S4 size prefixes, H1–H4 header ranges, junk packet params (`Jc`/`Jmin`/`Jmax`), the 3.x transport-protection block | Server config; copied into every client config |
| Per client | I1–I5 custom packet strings (CPS) | Computed per client from its protocol template (default `quic`) and the packet-size budget; a named template is deterministic, so peers with the same protocol and MTU get identical strings, while `protocol: random` draws fresh strings on every run. An interval the template leaves empty is omitted from the config (the default 3.1 output emits `I1`–`I4`) |

See [Obfuscation](./obfuscation.md) for parameter ranges and the CPS tag grammar.

### Credentials persist, unpinned parameters do not

Credential reuse is asymmetric, and the distinction matters:

- **Keys persist.** The server key pair, each client's private key and PresharedKey, and the header-protection key are recovered from the existing output tree and kept byte-identical across runs.
- **Unpinned obfuscation parameters are re-drawn every run.** Any of S1–S4, `Jc`/`Jmin`/`Jmax`, and the H ranges that you did not pin in the manifest are resolved fresh on each `generate`, usually from `crypto/rand`. Two consecutive runs on the same unpinned manifest therefore produce different S values and junk parameters while all keys stay stable — template-derived I-packets are unchanged, and only peers on the `random` protocol get fresh I-packets. The one exception is header protection: with it on and every H field unset, the H ranges are the constant type-ids rather than random.

> **Warning:** Because unpinned parameters change on every run, every client must re-import its config after any regeneration that was not pinned. If you need a stable deployment, pin the obfuscation values in the manifest — and see [Gotchas](./gotchas.md) for the full list.

### Generation flow

```text
amnezigo.json | .amnezigo.jsonnet
        │
        ▼  LoadManifest (Jsonnet wins; version must be 1)
   Manifest
        │
        ▼  Generate
   1. identify the single server peer
   2. load persisted credentials from the output tree
   3. resolve obfuscation (version gates, 3.x defaults, S floor, header-protection key)
   4. resolve peer credentials (reuse persisted keys or generate X25519 pairs + PSKs)
   5. build the server config (in memory)
   6. re-parse the server config and validate it → findings
   7. build client configs (sorted by name, filtered by --peer)
   8. collect all FileOutput values
        │
        ▼  9. write output/<server>/awg0.conf, output/<peer>/awg0.conf [+ amnezigo.vpn]
   `Warnings: N` block when the server validation produced findings
```

Two details are worth calling out:

- Step 6 re-parses the server config the generator just built and runs the same validation rules `amnezigo validate` uses, so findings such as `TRL001` surface right after the file list as a `Warnings: N` block. Generate reports them but still exits 0.
- Step 9 is skipped by `--dry-run`, which prints `Dry run — no files written` and still lists the byte counts of everything it would have written. `--peer` filters **client** peers only: the server config is always regenerated, and an unmatched peer name silently produces no client config rather than an error.

> **Warning:** Always set `network.mtu` explicitly. With the field omitted the client config still shows `MTU = 1280`, but client CPS generation reads the raw (zero) manifest value before that default is applied, so all five client I-packets collapse to the bare `<t>` and lose their protocol mimicry. (The server path is unaffected.)

## The Commands

| Command | Purpose | Reference |
| --- | --- | --- |
| `amnezigo generate` | Read the manifest and write per-peer `awg0.conf` files, reusing persisted keys unless `--full-reset`. Prints the relative paths and byte counts (not the output directory itself), plus a `Warnings: N` block from the server-config validation. | [CLI Reference](./cli-reference.md) |
| `amnezigo validate` | Check exactly one generated server config against the AWG size invariants the generator enforces. Exits `1` on any error, or on any warning with `--strict`. | [CLI Reference](./cli-reference.md) |
| `amnezigo analyze` | Inspect obfuscation strength with heuristic RISK001–RISK009 findings. Informational only: it always exits 0 on findings. | [CLI Reference](./cli-reference.md) |
| `amnezigo version` | Print the build stamp `amnezigo <version> (<commit>)`. A plain `go build` reports `amnezigo dev (none)`. | [CLI Reference](./cli-reference.md) |

> **Note:** `analyze --seed` does not currently make the report reproducible — the seed is accepted by the CLI but never read by the analyzer, so I-packet sizes are re-drawn from `crypto/rand` on each run. The flag's help text is aspirational.

The CLI is a thin Cobra wrapper over the root `amnezigo` package: `cmd/amnezigo/main.go` only calls `cli.Execute()`, and each subcommand translates flags into the library's option structs.

## Project Layout

```text
amnezigo/
├── *.go                  # root package: pipeline, generators, parsers, validation
├── internal/cli/         # cobra commands: generate, validate, analyze, version
├── internal/buildinfo/   # version/commit stamps reported by `amnezigo version`
├── cmd/amnezigo/         # main package: calls cli.Execute()
├── e2e/                  # container end-to-end suite (opt-in, `e2e` build tag)
├── testdata/loader/      # manifest fixtures used by the loader tests
├── hack/noticegen/       # license-bundle generator for NOTICE and licenses/
├── docs/                 # human documentation
└── assets/               # project logo
```

The root package never imports Cobra; the CLI depends on the library, not the other way around. The e2e suite is opt-in behind the `e2e` build tag and drives the built CLI as a black box against real `amneziawg-go` 3.1 containers.

## Related

- [Installation](./installation.md) — build or install the `amnezigo` binary, including the runtime version requirement.
- [Quick Start](./quick-start.md) — minimal manifest to generated configs in three commands.
- [Manifest Reference](./manifest-reference.md) — every manifest field, type, and default.
- [Transport Protection](./transport-protection.md) — the AWG 3.x layer end-to-end: knobs, defaults, gates, and INI keys.
- [CLI Reference](./cli-reference.md) — all four commands, flags, exit codes, and findings.
- [Gotchas](./gotchas.md) — project-wide pitfalls, from version gating to runtime alignment.
