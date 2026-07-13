# CLI Reference

> Every command and flag exposed by the `amnezigo` binary, verified against `internal/cli/*.go`.

## Table of Contents

- [Command overview](#command-overview)
- [amnezigo generate](#amnezigo-generate)
- [amnezigo validate](#amnezigo-validate)
- [amnezigo analyze](#amnezigo-analyze)
- [Exit codes](#exit-codes)
- [Global flags](#global-flags)

---

## Command overview

amnezigo exposes exactly **three** subcommands. The old imperative CLI
(`init`, `add`, `edit`, `remove`, `export`, `list`) was removed in commit
`226e4b8` and no longer exists; do not look for it.

| Command | Synopsis | Args | Exits non-zero? |
|---|---|---|---|
| [`amnezigo generate`](#amnezigo-generate) | Read a manifest, emit per-peer `awg0.conf` configs. | `NoArgs` | Yes — on manifest-load or generation error (exit `1`). |
| [`amnezigo validate <config>`](#amnezigo-validate) | Lint an existing AmneziaWG server config against AWG 2.0 invariants. | `ExactArgs(1)` — path to a server `awg0.conf` | Yes — exit `1` on any error, or on any warning when `--strict` is set. |
| [`amnezigo analyze`](#amnezigo-analyze) | Heuristic risk report (`RISK001`–`RISK009`) for a server config. | `NoArgs` | No — always exits `0` on success. Findings are informational. Exits `1` only on config-load or `--output`-format error. |

The root command itself is:

```text
amnezigo — AmneziaWG v2.0 Configuration Generator
        Declarative AmneziaWG v2.0 configuration generator.
```

---

## amnezigo generate

Reads `amnezigo.json` or `.amnezigo.jsonnet` from the project directory and
writes one `awg0.conf` per peer (server + clients) to the output directory. See
[Manifest Reference](./manifest-reference.md) for the input format and
[Output Format](./output-format.md) for the generated file layout.

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--project` | string | current dir (`os.Getwd`) | Project directory containing the manifest. |
| `--output` | string | `<project>/output` | Output directory for generated configs. |
| `--full-reset` | bool | `false` | Regenerate **all** credentials (X25519 keys + PSK) instead of reusing persisted ones. See [Credentials](./credentials.md). |
| `--dry-run` | bool | `false` | Compute configs without writing files. Prints `Dry run — no files written` first. |
| `--peer` | stringSlice (repeatable) | none | Generate only the named client peer(s). The server config is **always** generated regardless of this filter; unknown peer names are silently ignored. |
| `--jpath` | stringSlice | none | Jsonnet library search paths (passed to the manifest loader). See [Jsonnet](./jsonnet.md). |
| `--vpn-links` | bool | `false` | Generate AmneziaVPN `vpn://` import links for each client peer. Emits an additional `amnezigo.vpn` file per client. See [VPN Import Links](./vpn-links.md). |

> **Note:** `--peer` filters only *client* peers. The server config is always
> emitted. A name not present in the manifest is silently skipped.

### Output

```text
Generated <N> config(s):
  <peer>/awg0.conf (<bytes>)
  ...
Warnings: <N>          # only if Generate returned findings
[<SEV> <CODE>] ...
```

### Examples

```shell
# Generate from ./amnezigo.json into ./output/
$ amnezigo generate

# Generate from a specific project, write configs elsewhere
$ amnezigo generate --project ~/sites/vpn --output /etc/amnezia

# Only regenerate the "laptop" client; reuse all other credentials
$ amnezigo generate --peer laptop

# Regenerate every key (do not reuse persisted material)
$ amnezigo generate --full-reset

# Preview without touching the filesystem
$ amnezigo generate --dry-run

# Use a Jsonnet manifest with a custom library path
$ amnezigo generate --jpath ./lib

# Generate configs + AmneziaVPN import links
$ amnezigo generate --vpn-links
```

---

## amnezigo validate

Lints a **server** config (`awg0.conf`) against the same invariants the
generator enforces — packet-size classification, header-range validity,
junk-range ordering, S-prefix distinctness, and deprecated/unknown keys. See
[Validation & Analysis](./validation.md) for the rule catalogue.

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--output` | string | `text` | Output format: `text` or `json`. Any other value is a hard error (`unknown --output format`). |
| `--strict` | bool | `false` | Treat warnings as errors **for the exit code only**. Does not change which findings are emitted. |
| `--quiet` | bool | `false` | Suppress the summary line in `text` mode. Ignored in `json` mode (the JSON `summary` object is always present). |

> **Critical behavior — `validate` always parses with `Strict: true`.** The
> parser is invoked as `ParseServerConfigWithOptions(f, ParseOptions{Strict: true})`,
> so unknown keys and raw `<c>` tags are collected as `CPS001`/`KEY001` warnings
> *regardless* of `--strict`. The `--strict` flag changes **only** the exit
> code: with `--strict`, any warning flips the exit code to `1`.

### Text output

One line per finding via `Finding.OneLine()`, then a summary line (suppressed
by `--quiet`):

```text
[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>
  <optional multi-line detail>
✓ <path>: <E> errors, <W> warnings, <I> info
```

`<SEVERITY>` is uppercased (`ERROR`, `WARNING`, `INFO`); the `:line` and
`(key=…)` segments are omitted when empty. The leading marker is `✓` when
there are no errors, `✗` otherwise.

### JSON output

```json
{
  "file": "awg0.conf",
  "findings": [
    {
      "message": "header range overlaps H1",
      "code": "HDR002",
      "severity": "warning",
      "location": { "file": "awg0.conf", "line": 12, "key": "#_H2" }
    }
  ],
  "summary": { "errors": 0, "warnings": 1, "info": 0 }
}
```

`findings` is never serialized as `null` — an empty list renders as `[]`.

### Examples

```shell
# Plain text lint of a generated server config
$ amnezigo validate output/server/awg0.conf
[WARNING CPS001] output/server/awg0.conf:42 (key=I1): raw <c> tag rejected by amneziawg-go and AmneziaVPN clients
✓ output/server/awg0.conf: 0 errors, 1 warnings, 0 info

# Machine-readable JSON
$ amnezigo validate output/server/awg0.conf --output json

# Fail the CI gate on warnings too
$ amnezigo validate output/server/awg0.conf --strict --quiet
```

---

## amnezigo analyze

Runs heuristic analysis on a **server** config and reports potential weaknesses
(`RISK001`–`RISK009`), plus profiles of handshake sizes, junk parameters, header
ranges, and I-packet distributions. See [Validation & Analysis](./validation.md)
and [Obfuscation](./obfuscation.md).

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--protocol` | string | `random` | Obfuscation protocol template. **Accepted values (exact):** `random`, `quic`, `dns`, `dtls`, `stun`, `sip`, `rtp`. |
| `--peer` | string | `""` (all) | Analyze only this peer (empty = all peers). |
| `--output` | string | `text` | Output format: `text` or `json`. Any other value returns an error. |
| `--samples` | int | `0` | Number of samples for distribution analysis. `0` = snapshot only (no distribution). |
| `--seed` | uint64 | `0` | PRNG seed for reproducible I-packet generation. `0` = `crypto/rand` (non-deterministic). When non-zero, seeds `math/rand/v2` PCG with `(seed, seed)`. |
| `--config` | string | `awg0.conf` | Server config file path. **Relative to CWD** — to analyze a generated config you must point this at `output/<server>/awg0.conf` or `cd` there first. |

> **Critical:** `analyze` **always exits `0` on success.** Every finding is
> informational. The only non-zero exit paths are a config-load failure or an
> invalid `--output` value.

### Examples

```shell
# Default: random-protocol risk report on ./awg0.conf
$ amnezigo analyze --config output/server/awg0.conf

# Force the QUIC template and reproducible samples
$ amnezigo analyze --config output/server/awg0.conf \
    --protocol quic --samples 100 --seed 42

# JSON report for a single peer
$ amnezigo analyze --config output/server/awg0.conf \
    --peer laptop --output json
```

---

## Exit codes

| Command | Condition | Exit |
|---|---|---|
| `generate` | Success. | `0` |
| `generate` | Manifest load or generation error. | `1` |
| `validate` | No errors, and no warnings (or `--strict` unset). | `0` |
| `validate` | ≥ 1 error. | `1` |
| `validate` | `--strict` set AND ≥ 1 warning. | `1` |
| `validate` | Unknown `--output` value, or input file open/parse failure. | `1` |
| `analyze` | Success (findings are informational and never affect the exit code). | `0` |
| `analyze` | Config-load failure or invalid `--output` value. | `1` |

`generate` exits via `rootCmd.Execute()` → `os.Exit(1)` on any returned error.
`validate` exits via the package-level `exitFn` (`os.Exit`; overridable in
tests). `analyze` simply returns its error to cobra.

---

## Global flags

The root command declares **no project-specific global flags**. (A `cfgFile`
variable exists in `cli.go` but is unused.) The only flags available on every
command are the ones cobra registers automatically:

| Flag | Scope | Description |
|---|---|---|
| `-h`, `--help` | root + every subcommand | Show help for the command and exit. |

`--version` is **not** registered (the root command sets no `Version` field).
