# Validation & Analysis

> Reference for the `amnezigo validate` and `amnezigo analyze` commands: what each checks, the finding model, output formats, exit-code rules, and the RISK heuristic catalog.

## Table of Contents

- [Two tools, one page](#two-tools-one-page)
- [`amnezigo validate`](#amnezigo-validate)
- [`amnezigo analyze`](#amnezigo-analyze)
- [Validate vs Analyze](#validate-vs-analyze)
- [Library API](#library-api)
- [Related](#related)

---

## Two tools, one page

amnezigo ships two post-generation inspection tools. They have different scopes, different severity models, and different exit-code semantics — do not confuse them.

| Tool | Command | What it checks | Exit code |
|---|---|---|---|
| **validate** | `amnezigo validate <config>` | Lints an existing server config against AWG 2.0 invariants (required fields, S-padding collisions, junk-range hygiene, header-range validity, strict-parse warnings). Findings are correctness defects. | `0` no errors · `1` any error, or any warning under `--strict` |
| **analyze** | `amnezigo analyze` | Heuristic traffic-analysis risk report on a server config (RISK001–RISK009). Findings are advisory; nothing an analyze finding describes is a defect that breaks the config. | `0` always (on successful run) |

Both tools consume a **server** config (`awg0.conf`), not a client peer config. `analyze` loads it via `amnezigo.LoadServerConfig`; `validate` opens the file you pass and parses it with `Strict: true`.

---

## `amnezigo validate`

`validate` reads one server config, parses it strictly, runs every validation rule, and prints findings. See [./cli-reference.md](./cli-reference.md) for the full flag reference.

### Flags

| Flag | Default | Values | Purpose |
|---|---|---|---|
| `--output` | `text` | `text` \| `json` | Output format. |
| `--strict` | `false` | bool | Treat warnings as errors **for the exit code only**. |
| `--quiet` | `false` | bool | Suppress the text summary line. Ignored in JSON mode. |

`validate` takes exactly one positional argument: the config path.

### What it checks

`validate` runs two phases. First it parses with `Strict: true` (unconditionally — see [exit-code rules](#exit-code-rules)); strict-parse warnings become `warning` findings. Then, if parsing succeeded structurally, it runs `ValidateServerConfig` which produces `error` findings for any violated invariant.

| Code | Severity | Phase | Rule | Source |
|---|---|---|---|---|
| `FLD001` | error | validate | Required server field missing — `PrivateKey`, `Address`, or `ListenPort`. | `validateRequiredFields` |
| `PSC001` | error | validate | Two S-padded handshake sizes collide (`S1+148 == S2+92`, etc.). The four padded sizes must be pairwise distinct. | `ValidatePacketSizes` |
| `PSC002` | error | validate | Junk range `[Jmin..Jmax]` contains a padded size or a raw WireGuard size (148/92/64/32). Eight forbidden integers; any inside the inclusive range is a collision. | `ValidatePacketSizes` |
| `PSC003` | error | validate | An I-packet length equals one of the four padded sizes. (`validate` passes `nil` for I-packets, so this code is reachable only via the library API.) | `ValidatePacketSizes` |
| `JNK001` | error | validate | `Jmin > Jmax` — the junk range is empty / inverted. | `validateJunkRange` / `ErrEmptyJunkRange` |
| `HDR001` | error | validate | A header range `H1`–`H4` overlaps the WireGuard message type-ids `[1..4]` (inclusive). Such a range would accept vanilla WireGuard traffic. | `ValidateHeaderRange` |
| `HDR002` | error | validate | A header range has `Max < Min` (structurally invalid). | `ValidateHeaderRange` |
| `PSE001` | error | parse | Structural parse error — the config could not be parsed at all, so validation findings were not run. Emitted as a single fatal finding. | `runValidate` |
| `KEY001` | warning | parse | Unknown INI key in a section (strict parse). | `ParseServerConfigWithOptions` |
| `CPS001` | warning | parse | Raw `<c>` tag literal detected; rejected by `amneziawg-go` and AmneziaVPN clients. | `ParseServerConfigWithOptions` |

`PSC000` is a fallback collision code used only when `findingsFromValidationError` cannot classify the error kind; it is not produced by any current code path but is reserved.

> **Note:** The H-range WG type-id check (`HDR001`) is also enforced **structurally** during strict parsing — a range containing `[1..4]` is rejected as a parse error (`PSE001`) before `ValidateServerConfig` runs. `HDR001` from `ValidateHeaderRange` covers the same invariant at the library-API level.

### The Finding model

Every observation — from both `validate` and `analyze` — is a `Finding`. The same type flows through both commands and both output formats.

| Field | Type | JSON key | Description |
|---|---|---|---|
| `Message` | `string` | `message` | One-line human-readable description. |
| `Detail` | `string` | `detail` (omitted if empty) | Multi-line elaboration. |
| `Code` | `string` | `code` | Stable identifier (`FLD001`, `RISK003`, …). |
| `Severity` | `Severity` | `severity` | One of `error`, `warning`, `info`. |
| `Location.File` | `string` | `file` (in `location`, omitempty) | Config path. |
| `Location.Key` | `string` | `key` (omitempty) | INI key / field name (e.g. `PrivateKey`, `H2`). |
| `Location.Line` | `int` | `line` (omitempty) | 1-indexed line number; `0` = unset. |

`Severity` is a typed string with three constants: `SeverityError` (`"error"`), `SeverityWarning` (`"warning"`), `SeverityInfo` (`"info"`).

### Output formats

**Text** — one `Finding.OneLine()` per finding, followed (unless `--quiet`) by a summary line:

```text
[ERROR FLD001] (key=PrivateKey): required field "PrivateKey" is missing
  server configs require PrivateKey, Address, and ListenPort to function.
[WARNING KEY001] awg0.conf:42 (key=FooBar): unknown INI key "FooBar" in [Interface] section
✗ awg0.conf: 1 errors, 1 warnings, 0 info
```

`OneLine()` format (line and key segments omitted when empty):

```text
[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>
```

The summary marker is `✓` when there are no errors, `✗` otherwise, then the path and `E errors, W warnings, I info` counts.

**JSON** — a single document. `findings` is never `null` (an empty list is emitted):

```json
{
  "file": "awg0.conf",
  "findings": [
    {
      "message": "required field \"PrivateKey\" is missing",
      "detail": "server configs require PrivateKey, Address, and ListenPort to function.",
      "code": "FLD001",
      "severity": "error",
      "location": {
        "key": "PrivateKey"
      }
    }
  ],
  "summary": {
    "errors": 1,
    "warnings": 0,
    "info": 0
  }
}
```

### Exit-code rules

| Condition | Exit code |
|---|---|
| No findings, or only `info`/`warning` findings (without `--strict`) | `0` |
| One or more `error` findings | `1` |
| `--strict` set **and** one or more `warning` findings | `1` |
| Config file cannot be opened, or `--output` is neither `text` nor `json` | command error (non-zero, via cobra) |

Two non-obvious behaviors:

- **`validate` always parses with `Strict: true`.** The `--strict` flag does **not** control whether strict-parse warnings (`KEY001`, `CPS001`) are produced — those appear in every run. `--strict` only changes the **exit code**: with it, warnings count toward failure.
- **`--quiet` is text/summary-only.** It suppresses the `✓/✗ summary` line in text mode and is ignored in JSON mode (findings are always emitted). It never affects the exit code.

---

## `amnezigo analyze`

`analyze` is a heuristic traffic-analysis risk report. It profiles handshake sizes, junk parameters, header ranges, and per-peer I-packet distributions, then runs nine `RISK` checks. Findings are advisory — see [./obfuscation.md](./obfuscation.md) for the obfuscation parameters these checks reason about.

### Flags

| Flag | Default | Values / units | Purpose |
|---|---|---|---|
| `--protocol` | `random` | `random` \| `quic` \| `dns` \| `dtls` \| `stun` \| `sip` \| `rtp` | Protocol template for I-packet generation. Overrides each peer's own protocol for the analysis; `analyze` does not read `peer.Protocol`. |
| `--peer` | `""` | peer name | Analyze only this peer (empty = all peers). |
| `--output` | `text` | `text` \| `json` | Output format. |
| `--samples` | `0` | int | Number of I-packet samples for distribution stats. `0` = single snapshot only. |
| `--seed` | `0` | uint64 | PRNG seed for reproducible I-packet sizes. `0` = `crypto/rand`. |
| `--config` | `awg0.conf` | path | Server config file path. **CWD-relative by default**, not the generator's `output/server/` path. |

> **Note:** I-packet sizes are **freshly generated from the config parameters**, not read from disk. The report carries a `sample_note`: *"I-packet sizes are freshly generated from config parameters and may differ on each run."* Use `--seed N` for reproducible output.

`--seed N` constructs `math/rand/v2`'s PCG generator as `rand.NewPCG(seed, seed)` (both halves set to `N`) and injects it via `AnalyzeOptions.Rand`. This seeding lives only in the CLI; the library `Analyze` call uses `crypto/rand` when `Rand` is `nil`. With `--samples N > 0`, each peer gets min/max/mean/median stats over `N` generated I-packet quintuples; with `--samples 0`, one snapshot quintuple per peer.

### RISK heuristic catalog

All RISK findings are `warning` or `info` — never `error`. Thresholds are package-level constants.

| Code | Severity | Threshold constant | What it flags |
|---|---|---|---|
| `RISK001` | warning | — | Junk range `[Jmin..Jmax]` contains a raw WireGuard size (148/92/64/32). Junk packets may be misclassified. |
| `RISK002` | warning | `iPacketClusterMinWidth = 20` | A peer's I-packet sizes span less than 20 B (`max − min < 20`). Narrow cluster is easier to fingerprint. |
| `RISK003` | warning | `s4MinPadding = 8` | `S4 < 8`. Transport padding under 8 B makes keepalive packets easily distinguishable. |
| `RISK004` | warning | `paddedSizeMinDiff = 5` | Two padded handshake sizes differ by 1–4 B. Close sizes weaken size-class separation. |
| `RISK005` | info | `paddedSizeMinDiff = 5` | A padded size lands within ±4 B of a raw WG size. May confuse naive DPI. |
| `RISK006` | warning | `junkMinWidth = 32` | Junk range width is 1–31 B. Narrow range makes junk packets predictable. |
| `RISK007` | warning | `headerMinWidth = 1_000_000` | An H-range width is under 1 000 000. Narrow header range reduces entropy. |
| `RISK008` | info | — | No peers defined. I-packet analysis is skipped. |
| `RISK009` | warning | — | All S-prefixes and junk parameters are zero (`S1=S2=S3=S4=Jc=Jmin=Jmax=0`). The config behaves like vanilla WireGuard. |

`RISK005` and `RISK008` are the only `info`-level findings; the rest are `warning`. `analyze` emits no `error` findings — `RISK` output is informational regardless of count.

### Exit code

| Condition | Exit code |
|---|---|
| Successful run (any number of RISK findings) | `0` |
| `--output` is neither `text` nor `json`, or server config fails to load | command error (non-zero, via cobra) |

---

## Validate vs Analyze

| Dimension | `validate` | `analyze` |
|---|---|---|
| Scope | Config **correctness** — will AWG accept/parse this? | Traffic-analysis **risk** — how identifiable is the obfuscated stream? |
| Finding model | `error` / `warning` / `info` (correctness defects + strict-parse warnings) | `warning` / `info` only — never `error` |
| Codes | `FLD`, `PSC`, `JNK`, `HDR`, `PSE`, `KEY`, `CPS` | `RISK001`–`RISK009` |
| I-packet source | Reads I-packet sizes from config (none in server config, so `PSC003` is library-only) | Generates fresh I-packets from config params per run |
| Exit on findings | `1` on errors (or warnings under `--strict`) | Always `0` |
| When to use | After every `generate`; in CI before deploying a config; when editing a config by hand. | When tuning obfuscation parameters against a DPI adversary; before deciding a preset is "stealthy enough". |

---

## Library API

All types and functions below live in the root package `amnezigo` ([./library-usage.md](./library-usage.md) for the full API surface).

| Symbol | Signature | Purpose |
|---|---|---|
| `ValidatePacketSizes` | `func ValidatePacketSizes(s1, s2, s3, s4 int, iPacketSizes []int, jmin, jmax int) error` | Enforces the AWG 2.0 size invariant: S-padded sizes pairwise distinct, no I-packet equals a padded size, junk range excludes all padded + raw WG sizes. Returns `nil`, `ErrEmptyJunkRange`, or `*PacketSizeCollisionError`. |
| `ValidateHeaderRange` | `func ValidateHeaderRange(r HeaderRange) error` | Errors if `Max < Min` or the inclusive range overlaps WG type-ids `[1..4]`. |
| `ValidateServerConfig` | `func ValidateServerConfig(cfg *ServerConfig) []Finding` | Runs required-field, S-prefix, junk-range, and header-range checks; returns all findings (empty if clean). Does not re-parse. |
| `Analyze` | `func Analyze(cfg ServerConfig, opts AnalyzeOptions) AnalysisReport` | Produces the heuristic risk report. I-packets are generated, not read. |
| `FormatText` | `func FormatText(report AnalysisReport) string` | Human-readable text report. |
| `FormatJSON` | `func FormatJSON(report AnalysisReport) (string, error)` | Indented JSON report. |
| `Severity` | `type Severity string` | `"error"` \| `"warning"` \| `"info"`. |
| `Location` | `type Location struct { File, Key string; Line int }` | Finding origin within a config file. |
| `Finding` | `type Finding struct { Message, Detail, Code string; Severity Severity; Location Location }` | Single observation; `OneLine()` renders the text form. |
| `AnalyzeOptions` | `type AnalyzeOptions struct { Rand io.Reader; Protocol, PeerName string; Samples int }` | `Rand == nil` → `crypto/rand`. `Protocol == ""` → `random`. `Samples == 0` → snapshot only. |
| `AnalysisReport` | `type AnalysisReport struct { Peers, Findings, Ordering, SampleNote, Config, Handshake, Headers, Junk }` | Top-level analyze result. |

> **Tip:** When calling `ValidateServerConfig` directly, parse the config with `ParseServerConfigWithOptions(r, ParseOptions{Strict: true})` first if you want `KEY001`/`CPS001` warnings; `ValidateServerConfig` itself only runs the four correctness rule groups.

---

## Related

- [./cli-reference.md](./cli-reference.md) — full `generate` / `validate` / `analyze` flag reference.
- [./obfuscation.md](./obfuscation.md) — the S-prefixes, junk range, header ranges, and I-packet parameters these checks reason about.
- [./library-usage.md](./library-usage.md) — Go API surface including the validation and analysis functions.
- [./output-format.md](./output-format.md) — the `awg0.conf` INI structure and `#_` metadata lines that `validate` parses.
