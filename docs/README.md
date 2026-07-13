# Documentation

> Reference documentation for **Amnezigo** — the AmneziaWG v2.0 configuration generator.

Amnezigo reads a single declarative manifest and emits ready-to-deploy `awg0.conf`
files. These pages are the human-readable reference. The AI-friendly single-file
documentation remains at [llms-full.txt](./llms-full.txt).

## Getting started

| Page | Description |
|------|-------------|
| [Overview](./overview.md) | What amnezigo is, the declarative manifest model, and what `generate` produces |
| [Installation](./installation.md) | Install via `go install`, build from source, or Docker |
| [Quick Start](./quick-start.md) | The generate → validate → analyze round-trip on one page |

## Reference

| Page | Description |
|------|-------------|
| [Manifest Reference](./manifest-reference.md) | Every manifest field, type, default, and rule |
| [Manifest Examples](./manifest-examples.md) | Worked manifests: minimal, multi-peer, random, explicit, Jsonnet |
| [CLI Reference](./cli-reference.md) | All commands and flags with exact behavior and exit codes |
| [Library Usage](./library-usage.md) | The importable Go API: `Generate`, `LoadManifest`, keys, validation, analysis |
| [Output Format](./output-format.md) | The `awg0.conf` INI layout and `#_` metadata |
| [VPN Import Links](./vpn-links.md) | The `vpn://` import link format and the `--vpn-links` flag |
| [Obfuscation](./obfuscation.md) | S/H/J parameters, size invariants, CPS grammar, protocol templates |
| [Presets](./presets.md) | The 7 named obfuscation bundles and their exact values |

## Guides

| Page | Description |
|------|-------------|
| [Jsonnet](./jsonnet.md) | Authoring manifests with `.amnezigo.jsonnet` |
| [Credentials & Key Reuse](./credentials.md) | How keys are generated, stored, and reused across runs |
| [Validation & Analysis](./validation.md) | `validate` invariants and `analyze` heuristic risk codes |
| [Gotchas](./gotchas.md) | Project-wide pitfalls and how to avoid them |
