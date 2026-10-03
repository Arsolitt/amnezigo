# CLI Reference

> Every command, flag, output shape, and exit code of the `amnezigo` binary, verified against `internal/cli/*.go`.

## Table of Contents

- [Command overview](#command-overview)
- [amnezigo generate](#amnezigo-generate)
- [amnezigo validate](#amnezigo-validate)
- [amnezigo analyze](#amnezigo-analyze)
- [amnezigo version](#amnezigo-version)
- [Exit codes](#exit-codes)
- [Finding codes](#finding-codes)
- [Global flags](#global-flags)

---

## Command overview

amnezigo exposes exactly **four** project subcommands — `generate`, `validate`, `analyze`, and `version`. Cobra additionally registers the built-in `help` command and the `completion` command, which prints an autocompletion script:

```shell
$ amnezigo completion bash   # also: zsh, fish, powershell
```

Running bare `amnezigo` prints the root help and exits `0`. An unknown subcommand prints `Error: unknown command "…" for "amnezigo"` plus cobra's `Run 'amnezigo --help' for usage.` line, then `Execute` prints the `Error:` line a second time and the process exits `1`.

| Command | Synopsis | Args | Exits non-zero? |
| --- | --- | --- | --- |
| [`amnezigo generate`](#amnezigo-generate) | Read a manifest and emit per-peer `awg0.conf` configs. | `NoArgs` | Yes — exit `1` on a manifest-load or generation error. |
| [`amnezigo validate <config>`](#amnezigo-validate) | Lint an existing server config against the generator's invariants. | `ExactArgs(1)` — path to a server `awg0.conf` | Yes — exit `1` on any error, or on any warning when `--strict` is set. |
| [`amnezigo analyze`](#amnezigo-analyze) | Heuristic risk report (`RISK001`–`RISK009`) for a server config. | `NoArgs` | No — always exits `0` on success; only config-load and `--output` errors exit `1`. |
| [`amnezigo version`](#amnezigo-version) | Print the build's version and commit stamp. | `NoArgs` | No — always exits `0`. |

The root help is:

```text
Declarative AmneziaWG v3.1 configuration generator.

Usage:
  amnezigo [command]

Available Commands:
  analyze     Analyze obfuscation config for potential weaknesses
  completion  Generate the autocompletion script for the specified shell
  generate    Generate AmneziaWG configs from manifest
  help        Help about any command
  validate    Validate an AmneziaWG server config against AWG size invariants
  version     Print the amnezigo version and commit

Flags:
  -h, --help   help for amnezigo

Use "amnezigo [command] --help" for more information about a command.
```

---

## amnezigo generate

Reads `amnezigo.json` or `.amnezigo.jsonnet` from the project directory and writes one `awg0.conf` per peer (server + clients) to the output directory. See [Manifest Reference](./manifest-reference.md) for the input format, [Output Format](./output-format.md) for the generated file layout, and [Credentials & Key Reuse](./credentials.md) for what persists between runs.

Usage: `amnezigo generate [flags]`.

### Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--project` | string | current working directory | Project directory containing the manifest. |
| `--output` | string | `<project>/output` | Output **directory** for generated configs. |
| `--full-reset` | bool | `false` | Regenerate all credentials and keys: discards persisted peer key pairs, PSKs, and the AWG 3.x header-protection key. |
| `--dry-run` | bool | `false` | Compute configs without writing files. Prints `Dry run — no files written` first, but still lists every file and its byte count. |
| `--peer` | stringSlice | none | Generate only the named client peer(s). The server config is always generated; unknown names are silently ignored. Repeat the flag or pass a comma-separated list (`--peer a,b`). |
| `--jpath` | stringSlice | none | Jsonnet library search paths; when set, they replace the default `<project>/lib` search path. Only affects `.amnezigo.jsonnet` manifests — passing it for an `amnezigo.json` project has no effect. See [Jsonnet](./jsonnet.md). |
| `--vpn-links` | bool | `false` | Generate an AmneziaVPN `vpn://` import link per client peer, written as `<peer>/amnezigo.vpn`. See [VPN Import Links](./vpn-links.md). |

> **Warning:** `--output` is not the same flag as `validate --output` / `analyze --output`: here it names a directory, there it selects the text/JSON format. Do not share one shell variable between commands.

> **Danger:** `--full-reset` rotates the header-protection key and every PSK, so all previously distributed configs stop working with the new server config. It is a fleet-wide re-provisioning action, not a local refresh.

### Output

```text
$ amnezigo generate --vpn-links
Generated 3 config(s):
  server/awg0.conf (743 bytes)
  phone/awg0.conf (1027 bytes)
  phone/amnezigo.vpn (1352 bytes)
```

The count covers **every** emitted file — including the extra `amnezigo.vpn` files added by `--vpn-links`. Paths are relative to the output directory, and the command never prints the output directory itself.

When the in-memory generated server config trips a validation rule, a warning block follows the file list:

```text

Warnings: 1
[WARNING TRL001]  (key=RandomTrailers): RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification
```

Generate findings carry no file or line. A finding that carries a `key` (e.g. `TRL001`) appends the key segment as ` (key=…)`, giving two spaces after `]`: `[WARNING TRL001]  (key=RandomTrailers): …`. Keyless findings render without the extra space: `[ERROR PSC001]: packet size collision …`. The block can list `ERROR`-severity findings even though the command still exits `0`.

> **Warning:** Re-running `generate` reuses persisted keys but re-draws every obfuscation parameter left unset in the manifest (`S1`–`S4`, `H1`–`H4`, `Jc`/`Jmin`/`Jmax`). A second run can therefore invalidate previously distributed client configs even though no key rotated — pin the parameters you have already handed out. See [Gotchas](./gotchas.md).

> **Note:** If the manifest omits `network.mtu`, the emitted client `MTU` defaults to 1280 but the client `I1`–`I5` values collapse to `<t>`, because the client pipeline generates the I-packets from the raw (zero) MTU before applying the default. Set `network.mtu` explicitly until this is fixed.

### Errors

```text
Error: loading manifest: no manifest file found in /path/to/project (expected .amnezigo.jsonnet or amnezigo.json)
Error: generating configs: resolve obfuscation: obfuscation.random_trailers requires awg_version 3.1 or later (got "2.0")
```

The second example shows a 3.x version gate failing: a manifest may only use a field its `awg_version` understands.

### Examples

```shell
# Generate from ./amnezigo.json into ./output/
$ amnezigo generate

# Generate from a specific project, write configs elsewhere
$ amnezigo generate --project ~/sites/vpn --output /etc/amnezia

# Only regenerate the "laptop" and "phone" clients; reuse all other credentials
$ amnezigo generate --peer laptop,phone

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

Lints a **server** config (`awg0.conf`) against the same invariants the generator enforces — packet-size classification, header-range validity, junk-range ordering, S-prefix distinctness, and unknown keys. See [Validation & Analysis](./validation.md) for the full rule catalogue.

The command's help text is:

```text
Validate runs every check the generator enforces (size collisions,
header ranges, required fields, deprecated tags) against an existing config.

Exit code:
  0 — no errors (warnings/info may still be printed)
  1 — at least one error (or any warning when --strict is set)

Examples:
  amnezigo validate /etc/amnezia/awg0.conf
  amnezigo validate awg0.conf --output json
  amnezigo validate awg0.conf --strict --quiet

Usage:
  amnezigo validate <config> [flags]

Flags:
  -h, --help            help for validate
      --output string   Output format: text|json (default "text")
      --quiet           Suppress summary line; print findings only
      --strict          Treat warnings as errors for exit code
```

Passing the wrong number of arguments prints cobra's error, e.g. `Error: accepts 1 arg(s), received 2`.

### Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--output` | string | `text` | Output format: `text` or `json`. Any other value is a hard error: `unknown --output format "xml" (want: text\|json)`. |
| `--strict` | bool | `false` | Treat warnings as errors **for the exit code only**. Does not change which findings are emitted. |
| `--quiet` | bool | `false` | Suppress the summary line in `text` mode. Ignored in `json` mode (the JSON `summary` object is always present). |

> **Critical behavior — `validate` always parses with `Strict: true`.** The parser is invoked as `ParseServerConfigWithOptions(f, ParseOptions{Strict: true})`, so unknown keys and raw `<c>` tags are collected as `CPS001`/`KEY001` warnings *regardless* of `--strict`. The `--strict` flag changes **only** the exit code: with `--strict`, any warning flips the exit code to `1`.

### Text output

One line per finding via `Finding.OneLine()`, with non-empty details indented by two spaces, then a summary line (suppressed by `--quiet`):

```text
[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>
  <optional multi-line detail>
✓ <path>: <E> errors, <W> warnings, <I> info
```

`<SEVERITY>` is uppercased (`ERROR`, `WARNING`, `INFO`); the `:line` and `(key=…)` segments are omitted when empty. The leading marker is `✓` when there are no errors, `✗` otherwise.

```shell
$ amnezigo validate awg0.conf
[WARNING CPS001] awg0.conf:42: raw <c> tag detected; rejected by amneziawg-go and AmneziaVPN clients
✓ awg0.conf: 0 errors, 1 warnings, 0 info
```

### JSON output

```json
{
  "file": "awg0.conf",
  "findings": [
    {
      "message": "unknown INI key \"I1\" in [Interface] section",
      "code": "KEY001",
      "severity": "warning",
      "location": {
        "file": "awg0.conf",
        "key": "I1",
        "line": 17
      }
    }
  ],
  "summary": {
    "errors": 0,
    "warnings": 1,
    "info": 0
  }
}
```

`findings` is never serialized as `null` — an empty list renders as `[]`. Within a finding, `detail` is omitted when empty, as are `location.line`/`location.key`; the whole `location` object is omitted only when every field is empty — e.g. `analyze`'s RISK findings. `validate` always stamps `location.file`, so its findings (PSE001 included) always carry a `location`.

### Examples

```shell
# Machine-readable JSON
$ amnezigo validate output/server/awg0.conf --output json

# Fail the CI gate on warnings too, printing findings only
$ amnezigo validate output/server/awg0.conf --strict --quiet
```

> **Note:** A config with no errors exits `0` and prints the `✓` summary; findings never go through cobra, so a failed lint prints no usage text. CLI-level failures do go through cobra, with the shapes `open "<path>": <os error>`, `unknown --output format "<value>" (want: text|json)`, and cobra's own `accepts 1 arg(s), received <n>`. See [Exit codes](#exit-codes) for how they are printed.

---

## amnezigo analyze

Runs heuristic analysis on a **server** config and reports potential weaknesses (`RISK001`–`RISK009`), plus profiles of handshake sizes, junk parameters, header ranges, and regenerated I-packet distributions. See [Validation & Analysis](./validation.md) and [Obfuscation](./obfuscation.md).

### Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--config` | string | `awg0.conf` | Server config file path, **relative to the current directory** — point it at `output/<server>/awg0.conf` or `cd` there first. |
| `--protocol` | string | `random` | Template used to regenerate I-packets: `random`, `quic`, `dns`, `dtls`, `stun`, `sip`, `rtp`. Unknown values are not rejected — they fall through to a random template. |
| `--peer` | string | `""` (all peers) | Analyze only this peer. |
| `--output` | string | `text` | Output format: `text` or `json`. Any other value returns `invalid output format "xml": must be text or json`. |
| `--samples` | int | `0` | Number of samples for distribution analysis. `0` = snapshot only (no distribution block). |
| `--seed` | uint64 | `0` | Accepted but has no effect — see the warning below. |

> **Warning:** `--seed` does **not** make the report reproducible, despite its help text. The CLI builds a seeded `math/rand/v2` PCG reader and assigns it to `AnalyzeOptions.Rand`, but no code path in the analysis package reads that field: I-packet generation always uses `crypto/rand`. Two runs with `--seed 42 --samples 5` produce different distributions. Treat analyze output as a sample, not a deterministic artifact.

> **Note:** The report profiles the `--protocol` template (default `random`), not the protocol stored in the config file. I1–I5 are freshly regenerated, so they rarely match the values in the file — the final `Note:` line says so explicitly.

### Text output

`amnezigo analyze --output text` (the default) prints these sections in order:

1. `=== AmneziaWG Config Analysis ===`
2. `MTU: <n> | Port: <n> | Peers: <n> | Protocol: <value>`
3. `--- Handshake Sizes ---` — one line each for `Init:` (`S1`), `Response:` (`S2`), `Cookie:` (`S3`), and `Transport:` (`S4`), each as `S<n>=<prefix> + <raw> = <padded> bytes`.
4. `--- Junk Packets ---` — `Count: <Jc> (Jc) | Range: [<Jmin>..<Jmax>] | Width: <n> B`.
5. `--- Header Ranges ---` — `H1`–`H4` as `[<Min>..<Max>] (width <n>)`.
6. `--- I-Packets (per peer) ---` — only when the config has peers; per peer a snapshot line and, with `--samples N`, a distribution block.
7. `--- Wire Ordering ---` — the numbered packet sequence.
8. `--- Findings ---` — only when findings exist, formatted `  [<severity>] <code>: <message>` (severity is lowercase here).
9. `Note: <sample_note>` — always last.

Real excerpts:

```text
    Distribution (3 samples):
      i1: min=83 max=133 mean=110.0 median=114
      i2: min=67 max=94 mean=77.3 median=71

--- Findings ---
  [warning] RISK001: junk range [100..150] contains raw WG size 148 — junk packets may be misclassified

Note: I-packet sizes are freshly generated from config parameters and may differ on each run.
```

### JSON output

`amnezigo analyze --output json` prints `FormatJSON(report)` — a single two-space-indented object whose top-level keys are, in order: `peers`, `findings`, `ordering`, `sample_note`, `config`, `handshake`, `headers`, `junk`. Each peer carries `name` and `snapshot`, plus a `distribution` object only when `--samples` is greater than zero. Unlike `validate`, an empty collection serializes as `null` here — e.g. `"peers": null` or `"findings": null` — not as `[]`.

### Examples

```shell
# Default: random-template report on ./awg0.conf
$ amnezigo analyze --config output/server/awg0.conf

# Profile with 100 regenerated samples using the QUIC template
$ amnezigo analyze --config output/server/awg0.conf --protocol quic --samples 100

# JSON report for a single peer
$ amnezigo analyze --config output/server/awg0.conf --peer laptop --output json
```

> **Critical:** `analyze` **always exits `0` on success.** Every finding is informational — `RISK` codes never carry the `error` severity and never affect the exit code. A CI gate must parse `--output json` instead of trusting the exit status. The only non-zero exit paths are a config-load failure (`failed to load server config: <error>`) and an invalid `--output` value.

---

## amnezigo version

Prints the build stamp injected at compile time by `-ldflags -X` (see `internal/buildinfo`), which identifies the exact release a binary came from. There is no root-level `--version` flag; this command is the only way to read the stamp.

### Flags

None — the command declares no flags beyond the automatic `-h, --help`, and it takes no arguments (`NoArgs`).

### Output

```text
amnezigo <Version> (<Commit>)
```

* `<Version>` — a release version with the tag's leading `v` stripped (tag `v0.4.0` → `0.4.0`), a GoReleaser snapshot version, a `git describe` string for `make build`, or `dev` for a plain `go build` / `go install`.
* `<Commit>` — the short commit hash the binary was built from, or `none` for an unstamped build.

`make build` stamps `git describe --tags --always --dirty`, so it keeps the leading `v` and appends `-<n>-g<sha>` (and `-dirty`) for commits that are not exact tags.

```shell
# A release binary carries the tag's version and the commit it was built from
$ amnezigo version
amnezigo 0.4.0 (abc1234)

# A development build shows the defaults
$ go run ./cmd/amnezigo version
amnezigo dev (none)
```

`version --help` prints the command's own help:

```text
Print the version and the commit the binary was built from.

Usage:
  amnezigo version [flags]

Flags:
  -h, --help   help for version
```

---

## Exit codes

| Command | Condition | Exit |
| --- | --- | --- |
| `generate` | Success (including `--dry-run`). | `0` |
| `generate` | Manifest load or generation error. | `1` |
| `validate` | No errors (warnings/info may still be printed). | `0` |
| `validate` | ≥ 1 error. | `1` |
| `validate` | `--strict` set AND ≥ 1 warning. | `1` |
| `validate` | Unknown `--output` value, wrong argument count, or unreadable file. | `1` |
| `analyze` | Success — findings are informational and never affect the exit code. | `0` |
| `analyze` | Config-load failure or invalid `--output` value. | `1` |
| `version` | Success. | `0` |

Exit paths differ between commands:

- A **failed lint** makes `validate` call `exitFn(1)` (`os.Exit`) from inside the command. No usage text is printed, and the summary/findings are already on stdout.
- A **CLI-level error** in `validate` (unreadable file, bad `--output`, wrong argument count) is returned to cobra instead: cobra prints an `Error: …` line followed by the usage block, and `Execute` then writes the same `Error: …` to stderr again before exiting, so the message appears twice.
- `generate` and `analyze` return their errors to cobra, which prints them the same way and exits `1`; `version` has no failure path.

---

## Finding codes

Codes observable from `validate` are `FLD001`, `PSC001`, `PSC002`, `JNK001`, `HPK001`, `TRL001`, `KEY001`, `CPS001`, `PSE001`, and `TRM001` (only its "exactly one bound is zero" case). `PSC003`, `PSC000`, `HDR001`, `HDR002`, and `HPK003` are library-API or defensive codes the CLI does not emit: strict parsing rejects the triggering H ranges and malformed `HeaderProtectionKey` values up front, surfacing `PSE001` instead, and no current `ValidatePacketSizes` path returns an unclassified error.

`generate` runs `ValidateServerConfig` over the in-memory server config and prints the resulting findings (observed: `PSC001`, `JNK001`, `TRL001`) in its `Warnings:` block with no file or line; manifests whose values would trigger `HPK001`/`HPK003`/`TRM001` fail earlier in obfuscation resolution with an `Error:` instead. Severity and threshold details live in [Validation & Analysis](./validation.md).

| Code | Severity | Trigger and CLI reachability |
| --- | --- | --- |
| `FLD001` | error | Required `[Interface]` field missing (`PrivateKey`, `Address`, or `ListenPort`). |
| `PSC001` | error | Two S-padded handshake sizes collide. |
| `PSC002` | error | A junk range `[Jmin..Jmax]` contains a padded or raw WireGuard size. |
| `PSC003` | error | An I-packet length equals a padded size. Unreachable from the CLI today: `validate` passes no I-packet sizes. |
| `PSC000` | error | Unclassified packet-size error fallback — no current path emits it. |
| `JNK001` | error | `Jmin > Jmax`. |
| `HDR001` | error | An H range contains WG type-ids 1..4 while no header-protection key is set. Strict parsing aborts with `PSE001` first, so the CLI does not emit it. |
| `HDR002` | error | An H range has `Max < Min`. Strict parsing aborts with `PSE001` first, so the CLI does not emit it. |
| `HPK001` | error | `HeaderProtectionKey` is present while any S-prefix is below 12. |
| `HPK003` | error | `HeaderProtectionKey` is not 44-character base64 of 32 bytes. The parser rejects malformed values as `PSE001` first. |
| `TRL001` | warning | `RandomTrailers` is enabled while `S1`–`S4` are not all equal. |
| `TRM001` | error | A 3.x uint16 range has `Max < Min` (pre-empted by `PSE001`), or exactly one bound is zero (the CLI-reachable case). |
| `KEY001` | warning | Unknown INI key in `[Interface]`/`[Peer]` (strict parse). |
| `CPS001` | warning | Raw `<c>` tag literal found in the file (strict parse). |
| `PSE001` | error | Structural parse failure; the message is the parser error. Validation rules beyond the parser are skipped (pre-parse `KEY001`/`CPS001` warnings can still accompany it). |

`analyze` uses a separate code space; its findings never change the exit code.

| Code | Severity | Trigger |
| --- | --- | --- |
| `RISK001` | warning | A junk range contains a raw WireGuard size (148/92/64/32). |
| `RISK002` | warning | A peer's I-packet sizes span less than 20 B. |
| `RISK003` | warning | `S4 < 8`. |
| `RISK004` | warning | Two padded sizes differ by less than 5 B. |
| `RISK005` | info | A padded size is within ±4 B of a raw WireGuard size. |
| `RISK006` | warning | Junk range width is less than 32 B. |
| `RISK007` | warning | An H range width is less than 1,000,000 — skipped entirely when a header-protection key is present. |
| `RISK008` | info | No peers in the config. |
| `RISK009` | warning | All S prefixes and junk parameters are zero. |

---

## Global flags

The root command declares **no project-specific global flags** — each subcommand owns its flags (`--config`, for example, belongs to `analyze`). The only flag available on every command is the one cobra registers automatically:

| Flag | Scope | Description |
| --- | --- | --- |
| `-h`, `--help` | root + every subcommand | Show help for the command and exit. |

`--version` is **not** registered (the root command sets no `Version` field); use the [`version`](#amnezigo-version) subcommand instead.

---

## Related

- [Manifest Reference](./manifest-reference.md) — the `amnezigo.json` / `.amnezigo.jsonnet` input consumed by `generate`.
- [Output Format](./output-format.md) — the `awg0.conf` layout that `generate` writes and `validate` reads.
- [Validation & Analysis](./validation.md) — rule catalogue and thresholds behind `validate` and `analyze` findings.
- [Credentials & Key Reuse](./credentials.md) — what `generate` persists and what `--full-reset` discards.
- [Transport Protection (AWG 3.x)](./transport-protection.md) — the version gates and `HPK*`/`TRL001` rules the CLI enforces.
- [Gotchas](./gotchas.md) — pitfalls such as re-drawn obfuscation parameters and the ineffective `--seed`.
