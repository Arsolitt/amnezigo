# AmneziaWG 2.0 Compliance Roadmap

> Working document for iterating on amnezigo generator improvements.
> Each item is structured as an independent unit for brainstorming and PR work.

**Created:** 2026-04-28
**Status:** Draft, ready for per-item brainstorm

---

## Context & Scope Decisions

Decisions captured during initial planning session:

1. **Mobile compatibility is mandatory.** Generated configs must work on all platforms (iOS/Android/Windows/macOS), not only Linux kernel module. This drives P0.1 (remove `<c>`).
2. **Provider presets are accepted via PRs to this repository.** Documentation for the contribution flow will be designed when first preset PRs land. Owner decides preset structure and review criteria at that time.
3. **`awg-quick` config format support is in scope.** Currently only AmneziaVPN GUI format is generated; kernel-module/userspace `awg-quick` format support is required (P3.2).
4. **Server-wide vs per-peer parameter split is correct as-is.** S/H ranges are server-wide (must match across peers per AWG 2.0 spec); I-packets are per-peer (regenerated on every export).

---

## Key Findings

Discovered during reverse-engineering of `amneziawg-go` (commit `f6542209` "feat: awg 2.0 (#91)" merged 2025-09-01) and analysis of the current generator:

- **Tag `<c>` is incompatible with userspace.** Present in `amneziawg-linux-kernel-module/src/junk.c` only. Removed from `amneziawg-go` public reference (commit `e7ef4339`, 2026-03-23). Configs containing `<c>` produce `unknown tag <c>` on mobile/Win/Mac.
- **`J1-J3` and `Itime` do not exist in any official AWG 2.0 implementation.** Verified absent from `device/uapi.go`, `device/obf.go`, `src/config.c`. Likely artifacts of unofficial reverse engineering. Do not implement.
- **Tag `<t>` is 4 bytes, not 8.** `device/obf_timestamp.go` writes a `uint32 BigEndian`. Current generator's `cps_test.go` and MTU calculator assume 8 bytes — this miscalculates I-packet sizes, breaks MTU enforcement and collision validation.
- **PR #103 (`amneziawg-go`, merged 2025-12-01)** fixed a bug where transport packets were misinterpreted as init/response/cookie when sizes collided. Pairwise size validation is now mandatory.
- **Tags `<d>`, `<ds>`, `<dz>` exist in Go userspace** for data passthrough. Not used by current generator — missed mimicry primitive.

---

## Capability Matrix

### Configuration Parameters

| Parameter | AWG 2.0 Spec | Current Generator | Action |
| --- | --- | --- | --- |
| `Jc`, `Jmin`, `Jmax` | yes | yes | OK |
| `S1`, `S2` | yes | yes | OK |
| `S3` (cookie reply pad) | yes | yes | OK |
| `S4` (transport pad) | yes | yes | OK |
| `H1-H4` (range form `start-end`) | yes | yes | OK |
| `I1-I5` (CPS, per-peer) | yes | yes | Refine (see P0/P1) |
| `J1-J3` | NOT IN SPEC | not present | Do not add |
| `Itime` | NOT IN SPEC | not present | Do not add |

### CPS Tags

| Tag | Go userspace | Kernel module | Current Generator | Action |
| --- | --- | --- | --- | --- |
| `<b 0xHEX>` | yes | yes | yes | OK |
| `<r N>` | yes | yes | yes | OK |
| `<rc N>` | yes (`a-zA-Z`) | yes | yes — verify charset | Verify (P0.5) |
| `<rd N>` | yes | yes | yes | OK |
| `<t>` (uint32 BE, 4 bytes) | yes | yes | yes — but counted as 8 bytes | **P0.2 fix size** |
| `<c>` (counter) | NO | yes | yes — actively used | **P0.1 remove** |
| `<d>` (data passthrough) | yes | yes | no | **P1.1 add** |
| `<ds>` (base64 data) | yes | yes | no | P2.2 add |
| `<dz N>` (zero-data BE) | yes | yes | no | P2.2 add |

---

## P0 — Critical Fixes

Configs generated today are broken or non-portable. These must land before any feature work.

### P0.1 — Remove `<c>` (counter) tag

**What:** Eliminate `<c>` from generator code, tests, and protocol templates. Replace usages in templates with equivalent `<r N>` or `<rd N>`.

**Why:** Generated configs fail to load on Go-userspace AmneziaWG (mobile clients, Windows, macOS, Android). Only Linux kernel module accepts `<c>`. Mobile compatibility is mandatory (decision 1).

**Where:**

- `cps.go:40` — comment listing supported types
- `cps.go:55` — `case "c":` in `BuildCPSTag`
- `cps.go:135-136` — `"counter" → "c"` mapping in `NormalizeTagType`
- `types.go:101-102` — comments listing supported types
- `cps_test.go:41-61, 115, 154-158` — tests referencing `<c>`
- `quic.go`, `dns.go`, `dtls.go`, `stun.go` — replace any `<c>` usage in templates

**Acceptance criteria:**

- `grep -r '"<c>"\|"c"\|counter' --include="*.go"` returns nothing related to CPS
- All existing tests pass with `<c>` references replaced
- Manual verification: generated config loads in `amneziawg-go` userspace without errors

**Brainstorm questions:**

- Replacement strategy for templates — pure `<r 8>` (random binary) or context-appropriate (`<rd 8>` for digit-like positions)?
- Add a deprecation warning if a user-supplied profile includes `<c>` (for future profile feature)?
- Worth adding a one-line CHANGELOG / release-notes entry calling out the compatibility fix?

---

### P0.2 — Fix `<t>` size (8 → 4 bytes)

**What:** Correct the `<t>` (timestamp) tag size in MTU calculations and tests from 8 to 4 bytes (`uint32 BigEndian`).

**Why:** `device/obf_timestamp.go` in `amneziawg-go` writes `binary.BigEndian.PutUint32(buf, uint32(time.Now().Unix()))` — 4 bytes. Current generator counts 8 bytes, leading to:

- Wrong `maxISize` calculation in `calculateMaxISize`
- Tests asserting `<t>` produces 8 bytes (e.g. `cps_test.go:154` expects total of 20 for `<b 0xdeadbeef><c><t>` = 4 + 8 + 8)
- Potential collision with handshake sizes if real I-packet ends up smaller than expected

**Where:**

- `cps.go` — `calculateMaxISize` and any size accounting that includes `<t>`
- `cps_test.go` — fix expected sizes (after P0.1 also drops `<c>`-related tests)
- `cps_mtu_test.go` — recheck all assertions

**Acceptance criteria:**

- All `<t>` size constants in code = 4
- Tests pass with corrected expected values
- Spot-check: generate I-packet with `<b 0xff><t>` → produced bytes length = 5

**Brainstorm questions:**

- Is `<t>` size 4 a hard property of AWG 2.0 spec or could it change in a future version? If risk exists, define a constant for forward-compat.
- Worth adding a sanity test that compares generator's size accounting against actual byte length of a built CPS string?

---

### P0.3 — Full pairwise size collision validation

**What:** Validate that no two packet types produce equal on-the-wire sizes after padding.

**Why:** PR #103 in `amneziawg-go` (2025-12-01) fixed a bug where transport packets matching handshake sizes were misclassified. Until that fix landed, this was a security/correctness issue. Generator must enforce it.

**Constraints to validate:**

WireGuard message constants (from `device/noise-protocol.go`):

```text
MessageInitiationSize  = 148
MessageResponseSize    = 92
MessageCookieReplySize = 64
MessageTransportSize   = 32  (empty transport / keepalive)
```

After padding, the four padded sizes must all differ:

- `S1 + 148`
- `S2 + 92`
- `S3 + 64`
- `S4 + 32`

Additionally:

- For each `len(I_i)` (per-peer): must not equal any of the four padded sizes
- For the junk range `[Jmin..Jmax]`: must not include any of the four padded sizes
- For the junk range: should not include the un-padded WG sizes either (148, 92, 64, 32) since some receivers may probe before unpacking

**Where:**

- `generator.go` — `GenerateSPrefixes` already checks `S1+56 != S2`; expand to all six pairwise checks plus the four WG constants
- `cps.go` — when generating I-packets, reject sizes that collide; retry with adjusted tag count
- New helper: `ValidatePacketSizes(s1, s2, s3, s4 int, iPacketSizes []int, jmin, jmax int) error`

**Acceptance criteria:**

- All six S-pair collisions are checked
- I-packet generator retries on collision
- Junk range generator rejects ranges that include any of the four padded sizes
- Test cases for each collision type
- Property test: 1000 random generations produce zero collisions

**Brainstorm questions:**

- What's the retry budget before the generator gives up? Current code has `maxAttempts = 1000` for H ranges; same here?
- Should we widen the constraint to also avoid sizes within ±1 of WG constants (in case AWG ever shifts by 1 byte)?
- For the junk range, is excluding 148/92/64/32 sufficient, or should we exclude the padded sizes too (which depend on S1-S4)?

---

### P0.4 — Forbid H1-H4 ranges containing standard WG type-ids

**What:** When generating `H1-H4` ranges, ensure none of them include the values 1, 2, 3, 4 (standard WireGuard type-ids).

**Why:** If `H1 = 0-10`, then a vanilla WireGuard packet (`type=1`) would be accepted by AWG-aware peers. This breaks the obfuscation goal — AWG configs should be inert to vanilla WG and vice versa, otherwise a probe-and-fall-back attack is possible.

**Where:**

- `generator.go` — `GenerateHeaderRanges`; add post-condition that `1, 2, 3, 4` ∉ any range
- Add explicit test cases for ranges starting at 0, 1, 2, 3, 4

**Acceptance criteria:**

- No generated range includes any of 1, 2, 3, 4
- Validation also runs when parsing/loading existing configs (so `validate` command catches misconfigured imports)
- Test: generate 10000 configs, verify zero include WG type-ids

**Brainstorm questions:**

- Should we forbid ranges that *cross* the standard type-ids (e.g. `H1 = 0-10` containing 1, 2, 3, 4) or also ranges that *equal* a single type-id?
- Should we also exclude a small buffer around them (0-5 entirely)?

---

### P0.5 — Verify `<rc>` charset matches Go userspace

**What:** Confirm the generator's `<rc N>` tag produces only `[a-zA-Z]` characters, not `[a-zA-Z0-9]`.

**Why:** `amneziawg-go/device/obf_randchars.go` uses letters only (`a-zA-Z`, 52 chars). The Habr article incorrectly says `[A-Za-z0-9]`. If generator follows the article instead of the source, mismatched semantics may produce subtly different traffic patterns and reduce mimicry quality.

**Where:**

- `cps.go` — `BuildCPSTag` for type `"rc"`; charset constant or generator
- `cps_test.go` — assertion on character class

**Acceptance criteria:**

- Generated `<rc N>` output contains only `[a-zA-Z]`
- Test verifies no digits in 1000 samples

**Brainstorm questions:**

- Worth keeping a separate "letters-and-digits" tag if some templates need it? (probably not — `<rd>` covers digits, `<rc>` covers letters, mix can be done with multiple tags)

---

## P1 — Important Improvements

Functionality gaps that don't break existing configs but limit the tool's usefulness.

### P1.1 — Add `<d>` (data passthrough) tag

**What:** Support the `<d>` tag in CPS templates and generation. `<d>` does not produce bytes itself; it is a marker the AWG userspace expands at runtime by reusing a value from an earlier I-packet position.

**Why:** `<d>` is the most powerful mimicry primitive in AWG 2.0. A typical use: `i1 = <connection-id>`, then `i2 = <b 0x...><d>` reuses the same connection-id, making `i2` look like a continuation of the same simulated session. Without `<d>` every I-packet looks unrelated, which is itself a fingerprint.

**Where:**

- `cps.go:68-86` — `BuildCPSTag` switch over tag types; add `case "d": return "<d>"` (no value, no bytes)
- `cps.go:183-198` — `mapTagType` (user-name → shorthand); add `"data" → "d"`
- `cps.go:209+` — `calculateCPSLength` must return 0 bytes for `<d>` so size accounting and `ValidatePacketSizes` stay correct
- `types.go:111-114` — `TagSpec{Type: "data", Value: ""}`; no struct change needed
- `cps_test.go` — new `TestBuildCPSTag_Data` covering: zero-byte length, parser round-trip, rejection of stray value (`<d foo>` should fail)
- `quic.go:13-75` — exercise `<d>` in at least one template (QUIC's connection-id is the natural candidate to chain across i1 → i2)
- `validation.go:76+` — verify `ValidatePacketSizes` still works when an interval contains only `<d>` (expected: zero-byte interval, no collision contribution)

**Acceptance criteria:**

- `<d>` parses and round-trips through `ParseCPS` / `BuildCPSTag`
- `calculateCPSLength` returns identical totals whether `<d>` is present or absent
- At least one bundled protocol template uses `<d>` to chain a value across intervals
- `ValidatePacketSizes` ignores `<d>` and continues to detect collisions correctly
- Docs (`docs/obfuscation.md` or `docs/cli-reference.md`) state: "`<d>` defers to runtime, contributes 0 bytes, requires AWG 2.0 userspace"

**Brainstorm questions:**

- Does our generator need to track `<d>` "scope" (which earlier value it refers to) or is it implicit by position in `device/obf.go`'s state machine? (Initial reading: implicit by position — confirm in source)
- Which templates benefit most? (HTTP-like with session tokens — strong; STUN/NTP — weak because messages are typed and short)
- MTU accounting under `<d>`: track minimum (zero) and let runtime add the actual passthrough bytes, or precompute a worst-case upper bound from the source-interval value?
- Kernel-module support: does `amneziawg-linux-kernel-module` honor `<d>`? If kernel-only deploys exist, decide between "skip if kernel" (feature gate) vs "always emit, document caveat"

---

### P1.2 — Expand protocol templates

**What:** Add protocol templates beyond the current four (QUIC, DNS, DTLS, STUN).

**Candidates:**

- **SIP** — VoIP, UDP, often whitelisted in corporate networks
- **NTP** — UDP/123, almost always permitted, very small packets (challenging for I-packet sizing)
- **WebRTC TURN-Allocate** — extension of STUN, simulates ICE negotiation
- **MQTT-over-UDP** (rare but exists) — IoT mimicry
- **WireGuard-handshake** — anti-canary: looks like another WG flow, defeats naive WG fingerprinting

**Why:** More protocol diversity = harder for DPI to enumerate "all known AWG mimicry shapes". Each template is a distinct "shape" on the wire.

**Where:**

- `protocols.go:8-30` — `getTemplate()` switch over named protocols (`"quic"`, `"dns"`, `"dtls"`, `"stun"`, default random); add new `case` clauses
- New file per protocol: `sip.go`, `ntp.go`, `webrtc.go`, `wg_handshake.go`. Each returns `I1I5Template` (`types.go:117-119`) with five `[]TagSpec` (`types.go:111-114`) intervals
- `internal/cli/export.go:35` — extend `--protocol` flag's allowed values; today: `random`, `quic`, `dns`, `dtls`, `stun`
- `protocols_test.go:7-51` — extend `TestGetTemplate_NamedProtocols` to cover new templates (mirror existing pattern)
- New per-protocol tests: `TestTemplate_<Name>` validating tag mix, total byte budget, no `<c>`, charset constraints for `<rc>`/`<rd>`
- `docs/cli-reference.md:186, 197-200` — update `--protocol` allowed-values list and per-protocol descriptions
- `docs/obfuscation.md` — extended explanation per protocol (when to pick which)

**Acceptance criteria:**

- Each new template file has a `TestTemplate_*` test verifying: (a) returns five non-empty intervals, (b) total CPS length stays under MTU budget, (c) no forbidden tags, (d) charset constraints respected
- `--protocol random` selects uniformly from all templates including new ones
- `getTemplate("sip")` etc. returns the new template; unknown name still falls back to random (existing behavior preserved)
- README and `docs/cli-reference.md` list the new protocols
- One PR per new template (or grouped if cohesive: STUN + WebRTC TURN can ship together since they share the magic-cookie family)

**Brainstorm questions:**

- How realistic should SIP look — full INVITE with SDP body, or just OPTIONS ping? OPTIONS is smaller and fits MTU more reliably; INVITE is more convincing but eats budget
- WebRTC TURN-Allocate shares the `0x00 0x01` STUN-class prefix — how do we keep `random` selection producing visibly distinct shapes? (Differentiate via attribute set, not magic prefix)
- WG-handshake template — vanilla WG type-ids (1-4) appear inside CPS body, but P0.4 forbids them in H ranges. The two are different surfaces; clarify in a doc note that "header-range forbid" ≠ "CPS-body forbid"
- Should templates be data-driven (YAML/JSON) instead of hardcoded Go? Adds parsing complexity but lowers contribution barrier — defer until community pressure justifies it
- NTP is 48 bytes per packet — five intervals × 48 = 240 bytes is tight for full I1-I5; do we relax to "approximate NTP shape" or skip the template when MTU < threshold?

---

### P1.3 — `validate <config>` command

**What:** New CLI subcommand that reads a config file (server or client) and runs all generator validation rules against it.

**Why:** Useful for:

- Migration from other AWG generators (sanity-check before adopting)
- Catching `<c>` in legacy configs (P0.1 cleanup) — currently `parser.go:46` silently `continue`s on unknown keys, masking deprecation
- Pre-flight check before deployment
- Community education — users see what makes a "good" config

**Where:**

- New file: `internal/cli/validate.go`, mirroring existing subcommands (`init.go`, `export.go`, `add.go`, `edit.go`, `list.go`, `remove.go`)
- `parser.go:20` — `ParseServerConfig` is already public and accepts `io.Reader`; today line 46 silently `continue`s on unknown keys. Add an opt-in strict mode (extra parameter or `ParseServerConfigStrict`) that surfaces unknown keys as warnings without failing parse
- `parser.go:199+` — reuse `validateHeaderRange` (already exists)
- `validation.go:76+` — reuse `ValidatePacketSizes` (delivered in P0.3)
- `generator.go:205-216` — reference for how validation is wired post-generation today
- New consolidated entry point (likely `validation.go` or `internal/validate/`) that runs every check in sequence and returns structured findings: `[]Finding{Severity, Location, Message}`
- `cps.go:209+` — `calculateCPSLength` for I-packet size validation

**Validation checks (consolidated list):**

- All P0.1-P0.5 constraints (no `<c>`, `<t>` size 4, no S/I/junk size collisions, no WG type-ids in H1-H4, `<rc>` charset)
- Range non-overlap for H1-H4
- S-pair collisions via `ValidatePacketSizes`
- I-packet syntax + size + per-peer collisions
- Junk range vs WG sizes
- Required fields present
- Unknown keys flagged (warning, not error)

**Acceptance criteria:**

- `amnezigo validate server.conf` exits 0 on valid, non-zero on errors; warnings do not fail
- Each finding has: severity (`error`/`warning`/`info`), location (line/key when available), short message
- Both server and client config formats supported (or document client format as out of scope for v1, with a tracking item)
- A config containing `<c>` produces a clear error referencing P0.1
- Integration test: generate a fresh config, run `validate`, expect 0 findings; mutate the file (inject `<c>`), expect specific error

**Brainstorm questions:**

- Output format — human-readable text by default, `--output json` for tooling integration?
- Severity levels — error / warning / info; expose `--max-severity warning` flag so CI can fail-on-warnings?
- Should `validate` mutate (auto-fix) or stay read-only? (Default read-only; `--fix` for mechanical replacements like `<c>` → `<r 1>` once safe-replacement logic is settled)
- Where does strict-mode toggle live — CLI flag (`--strict`), always-on for `validate` since strictness is the entire point, or library-level option for downstream tools?

---

### P1.4 — `analyze <config>` command

**What:** New CLI subcommand that takes a config and reports the on-the-wire profile: packet sizes, timing, traffic shape.

**Why:** Helps users understand what their config "looks like" to DPI, and tune accordingly. Complementary to `validate` — `validate` tells you it's correct, `analyze` tells you what it produces.

**Where:**

- New file: `internal/cli/analyze.go` (parallel to `validate.go`)
- New file: `analysis.go` (or `internal/analyze/analyze.go`) reusing helpers below
- `cps.go:50-52` — `calculateMaxISize` (formula: `MTU - reserve(49) - handshakeSize(149) - S1`) for budget reporting
- `cps.go:209+` — `calculateCPSLength` for actual per-interval byte counts
- `validation.go:8-14` — WG message-size constants (`wgInitiationSize=148`, `wgResponseSize=92`, `wgCookieReplySize=64`, `wgTransportSize=32`)
- `validation.go:56-62` — `paddedSizes` helper for S-padded totals
- `generator.go:162-168` — `GenerateCPS` exposes per-peer I1-I5 strings
- `cps.go:97+` — `generateCPSConfig` for per-peer generation reference

**Output structure:**

```text
Handshake Init:    50 + 148 = 198 bytes (S1 padding)
Handshake Resp:   149 +  92 = 241 bytes (S2 padding)
Cookie Reply:      32 +  64 =  96 bytes (S3 padding)
Transport (empty): 16 +  32 =  48 bytes (S4 padding)

Junk packets: 4 (Jc), size range [50..1000]
I-packets (per-peer, sample):
  i1: 167 bytes (QUIC-like)
  i2: 132 bytes
  i3: ...

Order on the wire (per handshake):
  1. i1 → i2 → i3 → i4 → i5
  2. junk × 4
  3. Handshake Init
```

**Acceptance criteria:**

- Output covers all packet types: handshake init/resp, cookie reply, transport (S-padded), junk range, I-packets (per peer)
- `--peer NAME` selects which peer's I-packets to display; default = first peer in config
- `--output json` emits machine-parsable form for tooling
- Integration test: a known config produces stable output (snapshot-style)
- "Collision report" section reuses checks from `validate` but presents them as findings rather than errors (no non-zero exit)

**Brainstorm questions:**

- Show a "fingerprint risk" heuristic? (e.g. warn if junk range overlaps within ±10 of any WG size — too close)
- Compare against a reference profile? (`analyze --compare ru-mts.yaml`) — depends on P1.5 landing first
- Plot size distribution graphically (ASCII bar chart in terminal)? Hand-rolled vs new dependency
- Should `analyze` regenerate I-packets multiple times to show a distribution (since I-packets are random per export), or analyze a single instantiation? Default to one snapshot for reproducibility; add `--samples N` for distribution mode

---

### P1.5 — Provider presets (community PR flow)

**What:** Introduce a `presets/` directory with provider-specific bundles (S/H ranges, recommended protocols, optional CPS overrides). Accept community contributions via PR.

**Why:** Different providers/regions have different DPI signatures. A preset captures empirical knowledge ("these S1-S4 values work in Iran/MTS/GFW") and removes guesswork for end users.

**Out of scope for this iteration:** the contribution doc/process — will be designed when first PRs land (decision 2).

**Where:**

- New: `presets/` directory at repo root with YAML files (e.g. `presets/ru-mts.yaml`); does not exist yet
- `internal/cli/init.go:55-76` — current init flags (`--ipaddr`, `--port`, `--mtu`, `--dns`, `--keepalive`, `--client-to-client`, `--iface`, `--iface-name`, `--endpoint-v4`, `--endpoint-v6`, `--config`); add `--preset NAME` flag
- New file: `presets.go` (or `internal/presets/presets.go`) for parsing preset YAML and applying onto defaults
- Embed presets via `go:embed presets/*.yaml`. No existing `go:embed` usage in the repo — this PR establishes the pattern
- `generator.go:10-23` — current default constants (`junkMinValue=64`, `junkRangeSize=961`, `sPrefixRangeMax=65`, `s4RangeMax=33`, `jcRangeMax=11`); refactor entry points to accept a `GeneratorOptions` (or similar) so preset values flow through instead of being hardcoded
- `types.go:46-51` — `ServerObfuscationConfig` already holds final values; presets influence generation, not storage, so no schema change here
- New: `presets_test.go` covering load → apply → generate → `validate` flow

**Preset schema (draft):**

```yaml
name: ru-mts
description: Empirical defaults for MTS Russia mobile network (2026-Q1)
version: 1
maintained_by: <github username>
tested_on:
  - awg-go: v0.2.17
  - amneziawg-tools: v1.0.20260223

params:
  jc: { min: 8, max: 12 }
  jmin: { min: 50, max: 100 }
  jmax: { min: 800, max: 1200 }
  s1:   { min: 50, max: 90 }
  s2:   { min: 100, max: 200 }
  s3:   { min: 16, max: 64 }
  s4:   { min: 8, max: 32 }
  h_range_size:
    min: 100
    max: 100000000

protocols: [quic, dns]   # which I-packet templates work well here
notes: |
  Free-form notes for users.
```

**Acceptance criteria:**

- `amnezigo init --preset ru-mts --ipaddr 1.2.3.4` generates a valid config end-to-end
- Preset YAML format documented with one canonical example (`presets/example.yaml` and a `docs/presets.md` page)
- Each preset is tested in CI: `for f in presets/*.yaml; do amnezigo init --preset ${f%.yaml} ...; amnezigo validate ...; done` — depends on P1.3 landing
- Generator entry points accept overrideable ranges; no hardcoded constants survive into the preset code path
- Embedded presets ship with the binary (no runtime file lookup unless `--preset-file path.yaml` is added later as an escape hatch)

**Brainstorm questions:**

- Preset versioning when AWG itself updates — embed `tested_on:` ranges? Should `init` warn if installed AWG falls outside the window?
- How do we curate (anti-spam, quality)? Issue template + maintainer review at PR time; CI must pass `validate` per preset
- Should presets carry test fixtures (`expected_output.conf` for snapshot testing)? Increases maintenance but locks behavior across refactors
- Allow presets to override CPS templates per protocol, or only param ranges? Start with param ranges; template overrides are P3.5 territory
- Naming convention — region-prefix (`ru-mts`, `ir-irancell`)? Codify in CONTRIBUTING when first community PR lands

---

## P2 — Quality of Life

### P2.1 — QR code export

**What:** Add `--qr` flag to `export` subcommand that prints a QR code of the client config.

**Why:** AmneziaVPN mobile apps support QR-code config import. Eliminates manual file transfer.

**Where:**

- `internal/cli/export.go`
- New dependency: `github.com/skip2/go-qrcode` or similar (terminal output)

**Acceptance criteria:**

- `amnezigo export peer1 --qr` prints QR to stdout
- Optional `--qr-png path.png` writes to file
- Handles config size limit (QR codes have max bytes; warn or use chunked-QR if exceeded)

**Brainstorm questions:**

- Terminal QR uses block characters — issues with light/dark themes?
- Encrypted QR (with passphrase) for secure transfer over insecure channels?

---

### P2.2 — `<ds>` and `<dz>` tags

**What:** Support the remaining data tags: `<ds>` (base64-encoded data passthrough), `<dz N>` (zero-data with size N, BigEndian).

**Why:** Completes the CPS tag set. Useful for protocols that expect base64 fields (some auth flows) or fixed-zero padding.

**Where:**

- `cps.go` — add cases
- Templates can opt in

**Acceptance criteria:**

- Both tags parse and produce expected output
- Tests cover edge cases (empty data, large N)

**Brainstorm questions:**

- `<dz N>` with `N=0` — valid or warning?
- Where do these fit in templates — separate template files, or as variants?

---

### P2.3 — Per-peer DNS

**What:** Allow `export` to set DNS servers per peer rather than using a global default.

**Where:**

- `internal/cli/export.go` — `--dns 1.1.1.1,8.8.8.8` flag
- `manager.go` — extend `BuildPeerConfig` signature

**Brainstorm questions:**

- Should DNS be stored in server config (per-peer record) or specified at export time?
- Default behavior when not specified — fall back to existing server-wide DNS?

---

### P2.4 — Multi-endpoint fallback

**What:** Support multiple Endpoints in a client config (or DNS-based round-robin).

**Why:** If primary server gets blocked, client falls back automatically. Useful for users with multiple geographically diverse servers.

**Where:**

- `manager.go`, server config schema
- AWG/WG protocol may not support multi-endpoint natively — investigate

**Brainstorm questions:**

- Is multi-endpoint a wireguard-userspace concept or just multiple `[Peer]` blocks?
- Does Amnezia client honor multiple Endpoints?
- Or is this better solved by DNS A-record round-robin, no config change needed?

---

### P2.5 — Rotation reminder

**What:** When loading a server config, if its `created_at` (new field) is older than threshold (default 30 days), print a warning suggesting to re-export peers (regenerates I-packets).

**Why:** I-packets are per-peer and per-export. Static I-packets across months become a fingerprint themselves. Reminders nudge good hygiene.

**Where:**

- `types.go` — add `CreatedAt` to server config
- `parser.go`, `writer.go` — handle the field
- All commands — print reminder when threshold exceeded

**Brainstorm questions:**

- Threshold configurable per-server? Per-CLI flag?
- Should `init` write `created_at`, but `edit` not bump it (only `init` resets)?
- Emit a structured event so external tooling can pick it up?

---

## P3 — Strategic

### P3.1 — Mesh topology

**What:** Allow peers to communicate directly (mesh), not only via central server (star).

**Why:** Resilience, latency reduction, p2p use cases. Currently only star topology is supported.

**Where:** Major architectural change — touches `types.go`, `manager.go`, all CLI commands.

**Brainstorm questions:**

- AWG/WG primitives support mesh natively (each peer just has multiple `[Peer]` entries) — main work is in tooling
- Key distribution model — does the server still issue keys, or each peer generates its own?
- Conflict with star — flag-gated or separate command tree?

---

### P3.2 — `awg-quick` config format support

**What:** Generate configs in `awg-quick` (kernel module / userspace systemd) format, not only AmneziaVPN GUI format.

**Why:** Self-hosted Linux deployments use `awg-quick` (similar to `wg-quick`). Without this, server admins manually convert.

**Where:**

- New: `awg_quick.go` writer
- `internal/cli/export.go` — `--format awg-quick` flag

**Reference:** `amneziawg-tools/src/config.c` for format spec.

**Acceptance criteria:**

- `amnezigo export peer1 --format awg-quick` produces a config that `awg-quick up <file>` accepts
- Both server-side `[Interface]` and client-side `[Peer]` blocks supported

**Brainstorm questions:**

- Format diff vs AmneziaVPN GUI format — likely just key casing and a few extra fields? Need detailed comparison.
- Default format flag — keep AmneziaVPN GUI as default (current behavior)?

---

### P3.3 — Integration tests with real `amneziawg-go`

**What:** GitHub Action that boots a real `amneziawg-go` container, applies a generated config, and verifies a peer can connect.

**Why:** Today we test the generator in isolation. End-to-end test would catch real bugs (like the `<c>` issue we found by reading source) before users hit them.

**Where:**

- `.github/workflows/integration.yml`
- `test/integration/` directory with Go test driver

**Brainstorm questions:**

- Pin `amneziawg-go` to a specific tag (e.g. v0.2.17), or test against multiple recent versions in matrix?
- How fast can the test be? Bringing up two awg-go instances + handshake = seconds, OK for PR CI.
- Test matrix — Linux only, or also test mobile-format configs through some kernel-module emulation?

---

### P3.4 — Benchmark mode

**What:** `amnezigo bench` runs the generator at load and reports throughput.

**Why:** Useful for regression testing as the generator grows. Less essential — only matters if generation becomes a bottleneck (it likely won't).

**Brainstorm questions:**

- Realistic — actual blocker is the H-range generation retry loop (1000 max attempts). Benchmark would help tune retry strategy.

---

### P3.5 — Versioned presets

**What:** Presets carry version metadata; `amnezigo update-presets` pulls newer versions from a remote repo.

**Why:** Provider DPI signatures change over time. Without updates, presets stale.

**Where:** Depends on P1.5 landing first.

**Brainstorm questions:**

- Update mechanism — git pull, separate registry, embedded with releases?
- Trust model — who signs presets?
- Offline use — must continue working without network.

---

## Atomic PR Strategy

Recommended PR order, optimized for small reviewable diffs (per `git.md` rules):

1. **PR 1 (P0.2):** Fix `<t>` size from 8 → 4 bytes. Smaller diff, prerequisite for accurate size validation.
2. **PR 2 (P0.1):** Remove `<c>` tag. Templates updated, tests fixed.
3. **PR 3 (P0.5):** Verify and lock `<rc>` charset to `[a-zA-Z]`.
4. **PR 4 (P0.3):** Full pairwise size collision validation.
5. **PR 5 (P0.4):** Forbid H1-H4 ranges containing 1-4.
6. **PR 6 (P1.1):** Add `<d>` tag.
7. **PR 7 (P1.3):** `validate` command (uses everything from P0).
8. **PR 8 (P1.4):** `analyze` command (sibling to validate).
9. **PR 9 (P1.5):** Preset infrastructure + 1 example preset.
10. **PR 10 (P1.2):** New protocol templates (one per PR ideally).
11. **PR 11+ (P2/P3):** As prioritized.

Each PR should include tests, doc updates, and changelog entry. Per project rules: signed commits, conventional commit messages, draft PR by default.

---

## References

### Source code

- [`amneziawg-go`](https://github.com/amnezia-vpn/amneziawg-go) — Go userspace, source of truth for AWG 2.0
- [`amneziawg-tools`](https://github.com/amnezia-vpn/amneziawg-tools) — `awg-quick` and `awg` CLI
- [`amneziawg-linux-kernel-module`](https://github.com/amnezia-vpn/amneziawg-linux-kernel-module) — kernel module (the only place `<c>` works)

### Key files in `amneziawg-go`

- `device/uapi.go` — UAPI parameter parsing
- `device/obf.go` — CPS chain parser
- `device/obf_*.go` — individual tag implementations
- `device/magic-header.go` — H1-H4 range parsing
- `device/noise-protocol.go` — message size constants
- `device/send.go` — packet ordering on the wire

### Critical commits / PRs

- [PR #91 "feat: awg 2.0"](https://github.com/amnezia-vpn/amneziawg-go/pull/91) — main AWG 2.0 merge (commit `f6542209`, 2025-09-01)
- [PR #103 "fix: refactor processing of junk packets"](https://github.com/amnezia-vpn/amneziawg-go/pull/103) — size-collision fix (commit `0361c54d`, 2025-12-01)
- Commit `e7ef4339` (2026-03-23) — removed `<c>` from public Go reference
- Commit `12a01220` (2026-03-31) — H1-H4 documented as string type, not uint32

### Articles

- [AmneziaWG 2.0: от маскировки трафика к мимикрии](https://habr.com/ru/companies/amnezia/articles/1014636/) — Habr, 2026-03-25, by AmneziaLover
- [Original AmneziaWG 1.0 announcement](https://habr.com/ru/companies/amnezia/articles/769992/) — Habr, 2023

### Documentation

- [AmneziaWG self-hosted setup](https://docs.amnezia.org/ru/documentation/instructions/new-amneziawg-selfhosted)
