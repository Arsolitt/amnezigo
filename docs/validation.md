# Validation & Analysis

> Reference for the `amnezigo validate` and `amnezigo analyze` commands: what each checks, the finding model, output formats, exit-code rules, and the RISK heuristic catalog.

## Table of Contents

- [Two tools, one page](#two-tools-one-page)
- [`amnezigo validate`](#amnezigo-validate)
- [`amnezigo analyze`](#amnezigo-analyze)
- [Validate vs Analyze](#validate-vs-analyze)
- [Gotchas](#gotchas)
- [Library API](#library-api)
- [Related](#related)

---

## Two tools, one page

amnezigo ships two post-generation inspection tools. They have different scopes, different severity models, and different exit-code semantics — do not confuse them.

| Tool | Command | What it checks | Exit code |
| --- | --- | --- | --- |
| **validate** | `amnezigo validate <config>` | Lints a server config: required fields, S-padding size collisions, junk-range hygiene, header-range validity, AWG 3.x header protection / random trailer / timer rules, and strict-parse warnings. Findings are correctness defects. | `0` no errors · `1` any error, or any warning under `--strict` |
| **analyze** | `amnezigo analyze` | Heuristic traffic-analysis risk report on a server config (`RISK001`–`RISK009`). Findings are advisory — nothing analyze reports breaks the config. | `0` always (on a successful run) |

Both tools consume a **server** config (`awg0.conf`), not a client peer config. `analyze` loads it via `amnezigo.LoadServerConfig`; `validate` opens the file you pass and parses it with `Strict: true`.

---

## `amnezigo validate`

`validate` reads one server config, parses it strictly, runs every validation rule, and prints findings. See [./cli-reference.md](./cli-reference.md) for the complete command reference.

### Flags

| Flag | Default | Values | Purpose |
| --- | --- | --- | --- |
| `--output` | `text` | `text` \| `json` | Output format. |
| `--strict` | `false` | bool | Treat warnings as errors **for the exit code only**. |
| `--quiet` | `false` | bool | Suppress the text summary line. Ignored in JSON mode. |

`validate` takes exactly one positional argument: the config path.

### What it checks

`validate` runs in two phases. First it parses with `Strict: true` — unconditionally. Strict-parse warnings (`KEY001`, `CPS001`) become `warning` findings; a structural parse error aborts with a single `PSE001` finding (see [Parse-time aborts](#parse-time-aborts-pse001)). Then, on a successfully parsed config, `ValidateServerConfig` runs seven rule groups in order and appends their findings:

1. Required fields
2. S-prefixes (`ValidatePacketSizes` with `nil` I-packet sizes)
3. Junk range
4. Header ranges
5. Header protection
6. Random trailers
7. Transport ranges

The rule set produces `error` and `warning` findings only. The `info` severity exists in the shared model but no validate rule emits it — `info` is an analyze-only severity.

| Code | Severity | Rule group | Condition and message |
| --- | --- | --- | --- |
| `FLD001` | error | required fields | One finding per missing field: `PrivateKey` empty, `Address` empty, or `ListenPort` zero. `required field %q is missing`. |
| `PSC001` | error | S-prefixes | Two S-padded sizes are equal (six pairwise checks). `packet size collision (s-pair): S1+148 vs S2+92 at 148 bytes`. |
| `PSC002` | error | S-prefixes | Junk range `[Jmin..Jmax]` (inclusive) contains a padded size or a raw WireGuard size (148/92/64/32) — eight forbidden integers. `packet size collision (junk-range): [Jmin..Jmax] contains 148 at 148 bytes`. |
| `PSC003` | error | S-prefixes | An I-packet length equals one of the four padded sizes. **Unreachable from `ValidateServerConfig`** — it passes `nil` for I-packet sizes; only a direct `ValidatePacketSizes` call can trigger it. |
| `PSC000` | error | S-prefixes | Fallback when a `ValidatePacketSizes` error cannot be classified into a code. Reserved; no current code path produces it. |
| `JNK001` | error | S-prefixes / junk range | `Jmin > Jmax`. Message `junk range Jmin (%d) > Jmax (%d)`, or the sentinel `junk range is empty (jmin > jmax)` from the S-prefix group. Both groups run unconditionally, so one defect can produce two `JNK001` findings. |
| `HDR001` | error | header ranges | An H range intersects WG message type-ids `[1..4]`. **Suppressed when `HeaderProtectionKey` is present** — with header protection the 4-byte type is encrypted and `H1..H4 = 1..4` is the legitimate reference configuration. Message `header range [%d-%d] contains forbidden WG type-id(s) in [%d..%d]`. |
| `HDR002` | error | header ranges | `Max < Min` on any of `H1`–`H4`. Fires even under header protection. `invalid header range: Max (%d) < Min (%d)`. |
| `HPK001` | error | header protection | `HeaderProtectionKey` is non-empty and any of `S1`–`S4` is below 12. `header protection requires S1-S4 >= 12 (got S1=%d, S2=%d, S3=%d, S4=%d)`; the ChaCha20 nonce is the first 12 bytes of the packet. |
| `HPK003` | error | header protection | A non-empty key that is not base64 StdEncoding of exactly 32 bytes. `HeaderProtectionKey is not 44-char base64 of 32 bytes`. |
| `TRL001` | warning | random trailers | `RandomTrailers` is enabled while `S1..S4` are not all equal. `RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification`. |
| `TRM001` | error | transport ranges | One of the six 3.x `U16Range`s (`ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts`) has `Max < Min` (`invalid %s range: max (%d) is below min (%d)`) or exactly one zero bound (`invalid %s range: bounds must both be zero or both non-zero (got %d-%d)`). All-zero `{0,0}` ranges are valid and produce no finding. |
| `PSE001` | error | parse phase | Structural parse error — the config could not be parsed at all, so the rule set was not run. Emitted as a single fatal finding whose message is the parser's error text. |
| `KEY001` | warning | parse phase | Strict parse found a key that is not a known `[Interface]`/`[Peer]` key. `unknown INI key %q in %s section`. |
| `CPS001` | warning | parse phase | Strict parse found a raw `<c>` tag literal — even on a comment line. `raw <c> tag detected; rejected by amneziawg-go and AmneziaVPN clients`. |

> **Note:** Four codes describe conditions the CLI parser rejects before validation runs: `HPK003`, `HDR001`, `HDR002`, and `TRM001`'s `max < min` branch never surface from `amnezigo validate` for such input — the same predicates abort parsing with `PSE001` first. Like `PSC003`, they are reachable only by calling `ValidateServerConfig` directly on a hand-built `ServerConfig`.

> **Note:** `Location.Key` is set for most findings — `PrivateKey`/`Address`/`ListenPort` (`FLD001`), `H1`–`H4` (`HDR001`/`HDR002`), `HeaderProtectionKey` (`HPK001`/`HPK003`), `RandomTrailers` (`TRL001`), and the exact INI key for `TRM001`. `PSC*`, `JNK001`, and `PSE001` carry no key; `KEY001` carries the offending key, `CPS001` the line only.

> **Note:** The same seven rule groups also run inside `generate`: it re-parses its emitted server config and appends `ValidateServerConfig` findings to `GenerateResult.Findings` (only when the config re-parses), printing `Warnings: N` plus one `OneLine()` per finding. Client configs are never re-parsed. Findings from that path have no `Location.File`/`Line`, so their `OneLine()` output has an empty location segment. See [./cli-reference.md](./cli-reference.md) for `generate` output.

### Parse-time aborts (`PSE001`)

Structural problems abort `ParseServerConfigWithOptions`. `validate` reports them as one `PSE001` `error` finding — strict-parse warnings collected before the abort are still included — and does not run `ValidateServerConfig`.

| Cause | Abort message |
| --- | --- |
| Header-protection key not 44-char base64 of 32 bytes | `invalid HeaderProtectionKey %q: must be 44-char base64 of 32 bytes` |
| U16 range value not `N` or `N-M` | `invalid <KEY> %q: expected "N" or "N-M"` |
| U16 range with `max < min` | `invalid <KEY> %q: max (%d) is below min (%d)` |
| Boolean not one of on/off/0/1 | `invalid <KEY> %q: expected on/off/0/1, got %q` |
| H range failing the structural check | `invalid H%d: <ValidateHeaderRange error>` |

> **Note:** The `[1..4]` type-id exclusion is enforced a second time during strict parsing — as a structural abort rather than a finding — but the parser also suppresses the type-id variant when a header-protection key is present, so a 3.1 config with `H1 = 1-1` parses cleanly. `Max < Min` always aborts. Once parsing succeeds, `HDR001`/`HDR002` re-check the same ranges at the library level.

### The Finding model

Every observation — from both `validate` and `analyze` — is a `Finding`. The same type flows through both commands and both output formats.

| Field | Type | JSON key | Description |
| --- | --- | --- | --- |
| `Message` | `string` | `message` | One-line human-readable description. |
| `Detail` | `string` | `detail` (omitted if empty) | Multi-line elaboration. |
| `Code` | `string` | `code` | Stable identifier (`FLD001`, `RISK003`, …). |
| `Severity` | `Severity` | `severity` | One of `error`, `warning`, `info`. |
| `Location` | `Location` | `location` (omitted when all fields are zero) | Where the finding originates. |
| `Location.File` | `string` | `file` (omitted if empty) | Config path. The CLI stamps it with the argument path; the library does not. |
| `Location.Key` | `string` | `key` (omitted if empty) | INI key / field name (e.g. `PrivateKey`, `H2`). |
| `Location.Line` | `int` | `line` (omitted if zero) | 1-indexed line number; `0` = unset. |

`Severity` is a typed string with three constants: `SeverityError` (`"error"`), `SeverityWarning` (`"warning"`), `SeverityInfo` (`"info"`).

### Output formats

**Text** — one `Finding.OneLine()` per finding, followed (unless `--quiet`) by a summary line:

```text
[ERROR FLD001] awg0.conf (key=PrivateKey): required field "PrivateKey" is missing
  server configs require PrivateKey, Address, and ListenPort to function.
[WARNING KEY001] awg0.conf:42 (key=FooBar): unknown INI key "FooBar" in [Interface] section
✗ awg0.conf: 1 errors, 1 warnings, 0 info
```

`OneLine()` format (line and key segments omitted when empty):

```text
[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>
```

The summary marker is a check mark when there are no errors and a cross when there is at least one error, followed by the path and `E errors, W warnings, I info` counts.

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
        "file": "awg0.conf",
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
| --- | --- |
| No findings, or only `info`/`warning` findings (without `--strict`) | `0` |
| One or more `error` findings | `1` |
| `--strict` set **and** one or more `warning` findings | `1` |
| Config file cannot be opened (`open %q: ...`), or `--output` is neither `text` nor `json` (`unknown --output format %q (want: text\|json)`) | command error (non-zero, returned via cobra) |

Two non-obvious behaviors:

- **`validate` always parses with `Strict: true`.** The `--strict` flag does **not** control whether strict-parse warnings (`KEY001`, `CPS001`) are produced — those appear in every run. `--strict` only changes the **exit code**: with it, warnings count toward failure.
- **`--quiet` is text/summary-only.** It suppresses the summary line (marker plus counts) in text mode and is ignored in JSON mode (findings are always emitted). It never affects the exit code.

A failing lint reports the failure by calling `os.Exit(1)` internally, right after printing findings (and the summary, unless `--quiet`); CLI-level failures return a cobra error instead. An invalid `--output` value therefore produces no findings output at all.

---

## `amnezigo analyze`

`analyze` is a heuristic traffic-analysis risk report. It profiles handshake sizes, junk parameters, header ranges, and per-peer I-packet distributions, then runs nine `RISK` checks. Findings are advisory — see [./obfuscation.md](./obfuscation.md) and [./transport-protection.md](./transport-protection.md) for the parameters these checks reason about.

### Flags

| Flag | Default | Values / units | Purpose |
| --- | --- | --- | --- |
| `--protocol` | `random` | `random` \| `quic` \| `dns` \| `dtls` \| `stun` \| `sip` \| `rtp` | I-packet generation mode. `random` runs the random tag-sequence generator (3–6 tags drawn from `b`/`r`/`rc`/`rd`/`t`), not a named template; any other value selects that named template. An unknown value is not rejected — it falls through to a randomly chosen named template. Overrides any per-peer `protocol`; `analyze` never reads `peer.Protocol`. |
| `--peer` | `""` | peer name | Analyze only this peer (empty = all peers). |
| `--output` | `text` | `text` \| `json` | Output format. |
| `--samples` | `0` | int | Number of I-packet samples for distribution stats. `0` = single snapshot only. |
| `--seed` | `0` | uint64 | Ignored in practice — see the warning below. |
| `--config` | `awg0.conf` | path | Server config file path. **CWD-relative by default**, not the generator's `output/server/` path. |

> **Warning:** `--seed` does not currently make the output reproducible. When the seed is non-zero, the CLI builds a `math/rand/v2` PCG reader and assigns it to `AnalyzeOptions.Rand`, but `Analyze` never reads that field — I-packet sizes are drawn from `crypto/rand` regardless. With the default `random` protocol, snapshot and distribution numbers therefore differ between invocations even with the same seed. The flag's help text (`PRNG seed for reproducible output`) describes the intended behavior, not today's.

> **Note:** I-packet sizes are **freshly generated from the config parameters**, not read from disk. The report carries a `sample_note`: *"I-packet sizes are freshly generated from config parameters and may differ on each run."* The config's stored `I1`–`I5` values are ignored entirely.

With `--samples N > 0`, each peer gets min/max/mean/median stats over `N` generated I-packet quintuples; with `--samples 0`, one snapshot quintuple per peer.

### RISK heuristic catalog

All RISK findings are `warning` or `info` — never `error`. Thresholds are package-level constants.

| Code | Severity | Threshold constant | What it flags |
| --- | --- | --- | --- |
| `RISK001` | warning | — | Junk range `[Jmin..Jmax]` contains a raw WireGuard size. Junk packets may be misclassified. |
| `RISK002` | warning | `iPacketClusterMinWidth = 20` | A peer's I-packet sizes span less than 20 B (`max − min < 20`). Narrow cluster is easier to fingerprint. |
| `RISK003` | warning | `s4MinPadding = 8` | `S4 < 8`. Transport padding under 8 B makes keepalive packets easily distinguishable. |
| `RISK004` | warning | `paddedSizeMinDiff = 5` | Two padded handshake sizes differ by 1–4 B (`0 < \|diff\| < 5`). Close sizes weaken size-class separation. |
| `RISK005` | info | `paddedSizeMinDiff = 5` | A padded size is within ±4 B of a raw WG size (`0 < \|diff\| < 5`). May confuse naive DPI. |
| `RISK006` | warning | `junkMinWidth = 32` | Junk range width is 1–31 B (`0 < width < 32`); width 0 (junk disabled) does not fire. Narrow range makes junk packets predictable. |
| `RISK007` | warning | `headerMinWidth = 1_000_000` | An H-range width is under 1,000,000 (`0 < width < 1M`). **The entire check is skipped when a header-protection key is present** — under header protection `H1..H4 = 1..4` is the reference configuration. |
| `RISK008` | info | — | No peers defined. I-packet analysis is skipped. |
| `RISK009` | warning | — | All S-prefixes and junk parameters are zero (`S1=S2=S3=S4=Jc=Jmin=Jmax=0`). The config behaves like vanilla WireGuard. |

`RISK005` and `RISK008` are the only `info`-level findings; the rest are `warning`. `analyze` emits no `error` findings — `RISK` output is informational regardless of count.

> **Note:** `RISK004` and `RISK005` require a strictly positive difference, so equal values are silently accepted: two identical padded sizes produce no `RISK004`, and a padded size exactly equal to a raw WG size (e.g. `S4 = 32` gives padded transport `64 == WGCookieReplySize`) produces no `RISK005`. `RISK005`'s message hardcodes `±4 B` because it formats `paddedSizeMinDiff - 1`; the text assumes the default threshold.

### Exit code

`analyze` never fails on findings: all output is informational.

| Condition | Exit code |
| --- | --- |
| Successful run (any number of RISK findings) | `0` |
| `--output` is neither `text` nor `json` (`invalid output format %q: must be text or json`), or the server config fails to load (`failed to load server config: ...`) | command error (non-zero, via cobra) |

---

## Validate vs Analyze

| Dimension | `validate` | `analyze` |
| --- | --- | --- |
| Scope | Config **correctness** — required fields, size classification, AWG 3.x transport rules | Traffic-analysis **risk** — how identifiable is the obfuscated stream? |
| Finding model | `error` / `warning` (the rules never emit `info`) | `warning` / `info` only — never `error` |
| Codes | `FLD`, `PSC`, `JNK`, `HDR`, `HPK`, `TRL`, `TRM`, `PSE`, `KEY`, `CPS` | `RISK001`–`RISK009` |
| I-packet source | Reads none from a server config, so `PSC003` is library-only | Generates fresh I-packets from config parameters per run |
| Exit on findings | `1` on errors (or warnings under `--strict`) | Always `0` |
| When to use | After every `generate`; in CI before deploying a config; when editing a config by hand. | When tuning obfuscation parameters against a DPI adversary; before deciding a preset is "stealthy enough". |

---

## Gotchas

| Area | Gotcha |
| --- | --- |
| `PSC003` | Unreachable from `ValidateServerConfig` — `validateSPrefixes` passes `nil` I-packet sizes. Only a direct `ValidatePacketSizes` call can detect an I-packet/padded-size collision. |
| `JNK001` | Can be emitted twice for one `Jmin > Jmax` defect: once from the S-prefix group (via `ErrEmptyJunkRange`) and once from the junk-range group. |
| H-range overlap | Never a validator rule. `ValidateServerConfig` checks only type-ids (`HDR001`) and `Max < Min` (`HDR002`); overlapping H ranges are rejected by the engine at `awg setconf` time with `headers must not overlap`. |
| `{0,0}` H range | Passes `ValidateHeaderRange` (the pure-zero range is not flagged), but the generator/engine rejects it later. |
| `HDR001` + header protection | Suppressed whenever `HeaderProtectionKey` is non-empty and `Max >= Min`. The type-id exclusion applies to generation and validation only when header protection is off. |
| `RISK007` + header protection | The whole check returns early when a header-protection key is present, so `H1..H4 = 1..4` with a key produces no `RISK007` while the same ranges without a key do. |
| `RISK004` / `RISK005` equality | Both require `0 < \|diff\| < 5`, so equal values are silent — including the real case of a padded size exactly equal to a raw WG size. |
| `analyze --seed` | Not reproducible: the CLI sets `AnalyzeOptions.Rand`, but `Analyze` never reads it. Every run draws fresh I-packets from `crypto/rand`. |
| `analyze` I-packets | The report does not reflect the config's stored `I1`–`I5`: sizes are regenerated from the protocol/MTU/S1 parameters with an empty forbidden set. |
| `validate` strictness | Always strict-parses, so client-only keys such as `Endpoint` and `I1`–`I5` become `KEY001` warnings. `validate` is designed for the server config; warnings fail the run only under `--strict`. |
| `TRL001` severity | A warning, not an error: `generate` still succeeds and writes configs, printing `Warnings: N`. Four of the seven presets intentionally trigger it — see [./presets.md](./presets.md). |
| Parse abort vs finding | Malformed 3.x keys/ranges/booleans abort with `PSE001`. The semantic problems that reach the CLI on a parsed config are `HPK001`, `TRL001`, `TRM001`'s mixed-zero-bounds branch, `PSC*`, and `JNK*`; `HPK003`, `HDR001`, `HDR002`, and TRM001's `max < min` branch abort parsing first, so `validate` never emits them for such input — they are reachable only through a direct `ValidateServerConfig` call on a hand-built `ServerConfig`. |

---

## Library API

All types and functions below live in the root package `amnezigo` ([./library-usage.md](./library-usage.md) for the full API surface).

| Symbol | Signature | Purpose |
| --- | --- | --- |
| `ValidatePacketSizes` | `func ValidatePacketSizes(s1, s2, s3, s4 int, iPacketSizes []int, jmin, jmax int) error` | Enforces the AWG 2.0 size invariant: S-padded sizes pairwise distinct, no I-packet equals a padded size, junk range excludes all padded + raw WG sizes. Duplicate I-packet sizes are allowed. Returns `nil`, `ErrEmptyJunkRange`, or `*PacketSizeCollisionError`. |
| `ValidateHeaderRange` | `func ValidateHeaderRange(r HeaderRange) error` | Errors if `Max < Min` or the inclusive range intersects WG type-ids `[1..4]`. The primitive itself is not header-protection aware — the type-id suppression lives in `validateHeaderRanges` and the parser. |
| `ValidateServerConfig` | `func ValidateServerConfig(cfg *ServerConfig) []Finding` | Runs the seven rule groups in order (required fields, S-prefixes, junk range, header ranges, header protection, random trailers, transport ranges) and returns all findings (empty if clean). Emits `error` and `warning` findings; never `info`. Does not re-parse. |
| `Analyze` | `func Analyze(cfg ServerConfig, opts AnalyzeOptions) AnalysisReport` | Produces the heuristic risk report. I-packets are generated, not read. |
| `FormatText` | `func FormatText(report AnalysisReport) string` | Human-readable text report. |
| `FormatJSON` | `func FormatJSON(report AnalysisReport) (string, error)` | Indented JSON report. |
| `Severity` | `type Severity string` | `"error"` \| `"warning"` \| `"info"`. |
| `Location` | `type Location struct { File, Key string; Line int }` | Finding origin within a config file. |
| `Finding` | `type Finding struct { Message, Detail, Code string; Severity Severity; Location Location }` | Single observation; `OneLine()` renders the text form. |
| `AnalyzeOptions` | `type AnalyzeOptions struct { Rand io.Reader; Protocol, PeerName string; Samples int }` | `Protocol == ""` → `random`. `Samples == 0` → snapshot only. `Rand` is declared as the randomness source but is currently never read by `Analyze` — output is not reproducible. |
| `AnalysisReport` | `type AnalysisReport struct { Peers, Findings, Ordering, SampleNote, Config, Handshake, Headers, Junk }` | Top-level analyze result. |

> **Tip:** When calling `ValidateServerConfig` directly, parse the config with `ParseServerConfigWithOptions(r, ParseOptions{Strict: true})` first if you want `KEY001`/`CPS001` warnings; `ValidateServerConfig` itself only runs the seven rule groups. The CLI stamps `Location.File` with the config path, the library does not.

---

## Related

- [./cli-reference.md](./cli-reference.md) — full `generate` / `validate` / `analyze` flag reference and exact CLI output.
- [./transport-protection.md](./transport-protection.md) — the AWG 3.x layer that `HPK001`, `HPK003`, `TRL001`, `TRM001`, and `RISK007` reason about.
- [./obfuscation.md](./obfuscation.md) — the S-prefixes, junk range, header ranges, and I-packet parameters these checks reason about.
- [./presets.md](./presets.md) — preset values and which presets intentionally trip `TRL001` or `RISK`.
- [./library-usage.md](./library-usage.md) — Go API surface including the validation and analysis functions.
- [./gotchas.md](./gotchas.md) — project-wide pitfalls around version gating, generation, and runtime alignment.
