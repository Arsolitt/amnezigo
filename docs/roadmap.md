# AmneziaWG 2.0 Compliance Roadmap

> Working document for iterating on amnezigo generator improvements.
> Each item is structured as an independent unit for brainstorming and PR work.

**Created:** 2026-04-28
**Updated:** 2026-04-30
**Status:** Draft, ready for per-item brainstorm

---

## Context & Scope Decisions

Decisions captured during initial planning session:

1. **Mobile compatibility is mandatory.** Generated configs must work on all platforms (iOS/Android/Windows/macOS), not only Linux kernel module. This drives P0.1 (remove `<c>`).
2. **Provider presets are accepted via PRs to this repository.** Documentation for the contribution flow will be designed when first preset PRs land. Owner decides preset structure and review criteria at that time.
3. **`awg-quick` config format is already the output format.** Generated `awg0.conf` files are valid `awg-quick` input — no format conversion is needed. The old roadmap item P3.2 is moot.
4. **Server-wide vs per-peer parameter split is correct as-is.** S/H ranges are server-wide (must match across peers per AWG 2.0 spec); I-packets are per-peer (regenerated on every export).
5. **Declarative config-driven architecture replaces the imperative CLI.** The old `init → add → edit → remove → export` flow is removed entirely. A single manifest (`amnezigo.json` or `amnezigo.jsonnet`) declares the full network topology; `amnezigo generate` produces per-peer output directories.
6. **Multi-peer topology.** All nodes (servers and clients) are "peers" in the manifest. The generator resolves topology: server peers get all client peers in their config; client peers reference only the server peer.
7. **Jsonnet support.** JSON files are the baseline; `.jsonnet` takes precedence when both exist. `--jpath` for shared library imports. Pattern adopted from cheburbox.
8. **Credential persistence between runs.** Keypairs and PSKs are reused from previous output configs. `--full-reset` regenerates everything. Adopted from cheburbox.
9. **Atomic two-pass generation.** Compute all configs in memory first; write to disk only on success. Adopted from cheburbox.
10. **No backward compatibility shims.** Old imperative commands are deleted, not deprecated.

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
| `<ds>` (base64 data) | yes | yes | no | Deferred |
| `<dz N>` (zero-data BE) | yes | yes | no | Deferred |

---

## P0 — Critical Fixes (DONE)

Configs generated today are broken or non-portable. These must land before any feature work.

### P0.1 — Remove `<c>` (counter) tag ✅

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

### P0.2 — Fix `<t>` size (8 → 4 bytes) ✅

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

### P0.3 — Full pairwise size collision validation ✅

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

### P0.4 — Forbid H1-H4 ranges containing standard WG type-ids ✅

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

### P0.5 — Verify `<rc>` charset matches Go userspace ✅

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

## P1 — Important Improvements (DONE)

Functionality gaps that don't break existing configs but limit the tool's usefulness.

### P1.1 — Add `<d>` (data passthrough) tag ✅

**What:** Support the `<d>` tag in CPS templates and generation. `<d>` does not produce bytes itself; it is a marker the AWG userspace expands at runtime by reusing a value from an earlier I-packet position.

**Why:** `<d>` is the most powerful mimicry primitive in AWG 2.0. A typical use: `i1 = <connection-id>`, then `i2 = <b 0x...><d>` reuses the same connection-id, making `i2` look like a continuation of the same simulated session. Without `<d>` every I-packet looks unrelated, which is itself a fingerprint.

**Where:**

- `cps.go:73-93` — `BuildCPSTag` switch over tag types; `case "d": return tagDataPassthrough`
- `cps.go:191-208` — `mapTagType` maps `"data" → "d"`
- `cps.go:210-245` — `calculateCPSLength` returns 0 bytes for `<d>`
- `types.go:101` — `simpleTag.Type` doc comment includes `"d"`
- `cps_test.go` — `TestBuildCPSTag_Data` covering: zero-byte length, parser round-trip, rejection of stray value
- Protocol templates — QUIC uses `<d>` to chain connection-id across i1 → i2

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

### P1.2 — Expand protocol templates ✅

**What:** Add protocol templates beyond the current four (QUIC, DNS, DTLS, STUN).

**Candidates:**

- **SIP** — VoIP, UDP, often whitelisted in corporate networks

**Why:** More protocol diversity = harder for DPI to enumerate "all known AWG mimicry shapes". Each template is a distinct "shape" on the wire.

**Where:**

- `protocols.go:8-33` — `getTemplate()` switch over named protocols (`"quic"`, `"dns"`, `"dtls"`, `"stun"`, `"sip"`, default random); add new `case` clauses
- New file per protocol: `sip.go`. Returns `I1I5Template` (`types.go:116-119`) with five `[]TagSpec` (`types.go:110-114`) intervals
- `internal/cli/export.go:39-41` — `--protocol` flag's allowed values
- `protocols_test.go` — `TestGetTemplate_NamedProtocols` covers new templates

**Acceptance criteria:**

- Each new template file has a `TestTemplate_*` test verifying: (a) returns five non-empty intervals, (b) total CPS length stays under MTU budget, (c) no forbidden tags, (d) charset constraints respected
- `--protocol random` selects uniformly from all templates including new ones
- `getTemplate("sip")` returns the new template; unknown name still falls back to random (existing behavior preserved)
- README and `docs/cli-reference.md` list the new protocols
- One PR per new template (or grouped if cohesive)

**Brainstorm questions:**

- Should templates be data-driven (YAML/JSON) instead of hardcoded Go? Adds parsing complexity but lowers contribution barrier — defer until community pressure justifies it

---

### P1.3 — `validate <config>` command ✅

**What:** CLI subcommand that reads a config file (server or client) and runs all generator validation rules against it.

**Why:** Useful for migration from other AWG generators (sanity-check before adopting), catching `<c>` in legacy configs, pre-flight check before deployment, community education.

**Where:**

- `internal/cli/validate.go:28-51` — `NewValidateCommand` with `--output`, `--strict`, `--quiet` flags
- `parser.go:58-261` — `ParseServerConfigWithOptions` with strict mode collecting warnings
- `validation.go:207-216` — `ValidateServerConfig` running all validation rules
- `validation.go:77-127` — `ValidatePacketSizes` for S-pair, I-packet, and junk-range checks

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

### P1.4 — `analyze <config>` command ✅

**What:** CLI subcommand that takes a config and reports the on-the-wire profile: packet sizes, timing, traffic shape.

**Why:** Helps users understand what their config "looks like" to DPI, and tune accordingly. Complementary to `validate` — `validate` tells you it's correct, `analyze` tells you what it produces.

**Where:**

- `internal/cli/analyze.go:27-66` — `NewAnalyzeCommand` with `--protocol`, `--peer`, `--output`, `--samples`, `--seed` flags
- `analysis.go:142-177` — `Analyze()` producing `AnalysisReport`
- `analysis.go:350-366` — `runHeuristics` applying RISK001-RISK009 checks
- `analysis.go:540-600` — `FormatText` for human-readable output
- `analysis.go:630-636` — `FormatJSON` for machine-parsable output

**Acceptance criteria:**

- Output covers all packet types: handshake init/resp, cookie reply, transport (S-padded), junk range, I-packets (per peer)
- `--peer NAME` selects which peer's I-packets to display; default = all peers
- `--output json` emits machine-parsable form for tooling
- Integration test: a known config produces stable output (snapshot-style)
- "Collision report" section reuses checks from `validate` but presents them as findings rather than errors (no non-zero exit)

**Brainstorm questions:**

- Show a "fingerprint risk" heuristic? (e.g. warn if junk range overlaps within ±10 of any WG size — too close)
- Compare against a reference profile? (`analyze --compare ru-mts.yaml`) — depends on P1.5 landing first
- Plot size distribution graphically (ASCII bar chart in terminal)? Hand-rolled vs new dependency
- Should `analyze` regenerate I-packets multiple times to show a distribution (since I-packets are random per export), or analyze a single instantiation? Default to one snapshot for reproducibility; add `--samples N` for distribution mode

---

### P1.5 — Provider presets (community PR flow) ✅

**What:** Introduce built-in presets with provider-specific bundles (S/H ranges, recommended protocols, optional CPS overrides).

**Why:** Different providers/regions have different DPI signatures. A preset captures empirical knowledge ("these S1-S4 values work in Iran/MTS/GFW") and removes guesswork for end users.

**Where:**

- `presets.go:8-16` — `Preset` struct with name, description, protocol, and all obfuscation parameters
- `presets.go:51-120` — `presetRegistry` with four built-in presets (lan-conservative, home-balanced, mobile-aggressive, test-minimal)
- `presets.go:123-134` — `GetPreset()` for preset lookup by name
- `internal/cli/init.go:23-38` — `buildObfuscationConfig()` using `--preset` flag
- `presets_test.go` — load → apply → generate → validate flow

**Acceptance criteria:**

- `amnezigo init --preset ru-mts --ipaddr 1.2.3.4` generates a valid config end-to-end
- Each preset's S-padded sizes are pairwise distinct, junk range excludes all padded and raw WG sizes
- Generator entry points accept overrideable ranges; no hardcoded constants survive into the preset code path
- Embedded presets ship with the binary

**Brainstorm questions:**

- Preset versioning when AWG itself updates — embed `tested_on:` ranges? Should `init` warn if installed AWG falls outside the window?
- How do we curate (anti-spam, quality)? Issue template + maintainer review at PR time; CI must pass `validate` per preset
- Allow presets to override CPS templates per protocol, or only param ranges? Start with param ranges; template overrides are deferred

---

## P2 — Declarative Core

The foundation for the new config-driven architecture. Replaces the old imperative `init → add → edit → remove → export` flow with a single manifest + generate pipeline.

### P2.1 — Config schema & Go types

**What:** Define the Layer 1 schema (user-facing manifest) as Go structs and produce a JSON Schema for editor validation. The manifest describes the full network: global settings, obfuscation profile, and all peers in a flat map.

**Why:** The schema is the contract between user and generator. Everything downstream (loader, credential persistence, generate command) depends on it.

**Manifest structure (draft):**

```jsonnet
{
  version: 1,
  network: {
    mtu: 1280,
    dns: ['1.1.1.1', '8.8.8.8'],
  },
  obfuscation: {
    preset: 'home-balanced',
    // or explicit: { s1: 50, s2: 150, ... h1: '100-200000', ... jc: 4, jmin: 50, jmax: 1000 }
    protocol: 'quic',  // default I-packet protocol
  },
  peers: {
    server: {
      address: '10.0.0.1/24',
      endpoint: 'vpn.example.com:51820',
      listen_port: 51820,
      post_up: 'iptables -A FORWARD ...',
      post_down: 'iptables -D FORWARD ...',
    },
    phone: {
      address: '10.0.0.2/32',
      protocol: 'sip',  // override per-peer
    },
    laptop: {
      address: '10.0.0.3/32',
      // uses default protocol from obfuscation.protocol
    },
  },
}
```

**Where:**

- New file: `manifest.go` — Go structs for `Manifest`, `NetworkConfig`, `ObfuscationConfig`, `PeerConfig` (manifest-layer, distinct from the existing `types.go` output structs)
- `types.go:10-119` — existing output types stay unchanged; manifest types are a separate layer that gets converted to output types during generation
- New file: `manifest_test.go` — test roundtrip JSON/Jsonnet → Go struct → validate → generate
- `presets.go:8-16` — `Preset` struct reused by `ObfuscationConfig.Preset` field

**Acceptance criteria:**

- `Manifest` struct unmarshals from JSON matching the draft schema above
- Peer map preserves insertion order (use `map[string]PeerManifest` with a separate `[]string` for ordered keys, or an ordered-map library)
- Server peer (the one with `endpoint` + `listen_port`) is identifiable programmatically — at most one peer may have `endpoint` set
- `version` field is validated (must be `1`)
- JSON Schema file generated (optional, can be a follow-up)

**Brainstorm questions:**

- Obfuscation: should `preset` and explicit params be mutually exclusive (error if both set), or should explicit params override preset defaults?
- Peer address: require explicit CIDR for all peers, or auto-assign from a pool (like current `AddPeer`)? Explicit is simpler and more declarative
- How to handle peers that need different MTU? Probably not — AWG/WG MTU is per-interface, not per-peer. Document this
- PostUp/PostDown: server-only fields. Validate that client peers don't set them?

---

### P2.2 — JSON + Jsonnet config loader

**What:** Implement file discovery and loading logic: find `amnezigo.json` or `amnezigo.jsonnet` in the working directory (or `--manifest` path), evaluate jsonnet if applicable, unmarshal into `Manifest` struct.

**Why:** The loader is the entry point for all commands. Jsonnet support enables DRY configuration across complex setups and community-shared library imports.

**Key patterns from cheburbox to adopt:**

- `.jsonnet` takes precedence over `.json` when both exist
- `--jpath` flag defaults to `lib/` relative to manifest directory for jsonnet library resolution
- Jsonnet VM setup: `go-jsonnet` library, `--jpath` as import path, native functions not needed initially

**Where:**

- New file: `loader.go` — `LoadManifest(path string, jpathDirs []string) (Manifest, error)` with discovery logic
- New file: `loader_test.go` — test JSON loading, jsonnet loading, precedence, `--jpath` resolution
- `go.mod` — add `github.com/google/go-jsonnet` dependency
- Reference: cheburbox `config/load.go` for the precedence and discovery pattern

**Acceptance criteria:**

- `LoadManifest("")` discovers `amnezigo.jsonnet` or `amnezigo.json` in the current directory
- `LoadManifest("/path/to/custom.jsonnet")` loads from explicit path
- Jsonnet evaluation with `--jpath lib/` resolves imports correctly
- When both `.json` and `.jsonnet` exist, jsonnet takes precedence
- Parse errors produce clear messages with file path and line number
- Returned `Manifest` is fully validated (version check, peer address uniqueness, etc.)

**Brainstorm questions:**

- Should we support multiple manifest files (like cheburbox's per-server directories)? No — amnezigo is single-network, not multi-server
- Jsonnet native functions — needed? cheburbox doesn't use them. Defer
- Should the loader also discover and load existing output configs for credential persistence, or is that a separate concern (P2.3)?

---

### P2.3 — Credential persistence

**What:** Between `generate` runs, reuse existing keypairs and PSKs from previously generated output configs rather than regenerating them. Add `--full-reset` flag to force regeneration.

**Why:** Regenerating keys on every run breaks existing peer connections. Users would need to redeploy all configs after every manifest change. Credential persistence is essential for practical use.

**Key patterns from cheburbox to adopt:**

- Read existing `config.json` for each server to extract credentials
- Preserve credentials for peers that still exist in the manifest
- Generate fresh credentials only for new peers or when `--full-reset` is used
- Atomic read/write of persisted keys

**Where:**

- New file: `credentials.go` — `LoadCredentials(outputDir string) (map[string]PeerCredentials, error)` reads existing output configs and extracts keypairs + PSKs
- `parser.go:58-261` — reuse `ParseServerConfigWithOptions` to read existing output configs for credential extraction
- `writer.go:10-67` — reuse `WriteServerConfig` and `WriteClientConfig` for output
- `keys.go:14-30` — `GenerateKeyPair()` used when no persisted credentials exist
- `keys.go:58-64` — `GeneratePSK()` used for new peer preshared keys
- New file: `credentials_test.go` — test persist → reload → verify same keys, test `--full-reset` regenerates

**Acceptance criteria:**

- Run `generate` twice without manifest changes; output configs have identical keypairs and PSKs
- Add a new peer to manifest; run `generate`; existing peers keep their keys, new peer gets fresh keys
- Remove a peer from manifest; run `generate`; removed peer's output dir is NOT deleted (orphan handling deferred), remaining peers keep their keys
- `--full-reset` flag regenerates all keys; output configs differ from previous run
- Server keypair is also persisted (not just peer keypairs)

**Brainstorm questions:**

- Where are credentials stored? Option A: in the output configs themselves (parse `<peer>/awg0.conf`). Option B: in a separate `.credentials.json` file. Option A is simpler and avoids an extra file; option B is cleaner separation of concerns
- Should we warn when a peer is removed from manifest but its output dir still exists? (Yes, but don't delete it — user might want the keys for migration)
- PSK persistence: PSK is per-connection (peer-to-server), not per-peer. When the server key changes (due to `--full-reset`), should PSKs also regenerate? (Yes — PSK protects a specific keypair relationship)

---

### P2.4 — `generate` command

**What:** The main pipeline command. Reads the manifest, validates it, generates or reuses credentials, builds per-peer configs, and atomically writes output directories.

**Pipeline steps:**

1. Load manifest (P2.2)
2. Validate manifest schema + AWG 2.0 invariants (reuse P1.3 validation engine)
3. Load persisted credentials from existing output (P2.3)
4. For each peer: generate or reuse keypair + PSK; generate I-packets (per-peer, per-run unless seeded)
5. Build server config (server peer gets all client peers in `[Peer]` sections)
6. Build client configs (each client peer gets only the server in `[Peer]` section)
7. Atomic write: compute everything in memory, write all files only on success

**Output directory structure:**

```text
output/
  server/
    awg0.conf     # server config with all peers
  phone/
    awg0.conf     # client config for phone
  laptop/
    awg0.conf     # client config for laptop
```

**Where:**

- New file: `pipeline.go` — `Generate(manifest Manifest, opts GenerateOptions) error` orchestrating the full pipeline
- New file: `pipeline_test.go` — test the full flow: manifest → generate → validate output
- `generator.go:174-216` — reuse `GenerateConfig` for per-peer obfuscation parameter generation
- `generator.go:162-168` — reuse `GenerateCPS` for I-packet generation
- `validation.go:207-216` — reuse `ValidateServerConfig` for post-generation validation
- `writer.go:10-67` — reuse `WriteServerConfig` for server output
- `writer.go:89-134` — reuse `WriteClientConfig` for client output
- `manager.go:255-313` — reference `BuildPeerConfig` logic for constructing client configs (to be adapted, not reused directly since it depends on old Manager pattern)
- New file: `internal/cli/generate.go` — CLI command wiring with `--manifest`, `--output`, `--full-reset`, `--jpath`, `--dry-run` flags

**Acceptance criteria:**

- `amnezigo generate` reads `amnezigo.json` (or `.jsonnet`) from current directory, produces output dirs
- `amnezigo generate --manifest /path/to/config.jsonnet --output /path/to/output` uses explicit paths
- Server config contains all client peers as `[Peer]` sections
- Client configs contain only the server peer in `[Peer]` section
- All generated configs pass `amnezigo validate` (use P1.3 engine)
- `--dry-run` prints what would be written without writing files
- Atomic write: if any config fails validation, no files are written
- Credential persistence works across runs (P2.3)

**Brainstorm questions:**

- Output directory naming: use peer names from manifest as directory names? What about name collisions with reserved names (`output`, `.git`, etc.)?
- Should the server config also be written as a client config (so the server operator can import it into AmneziaVPN client on the same machine for testing)?
- Multiple servers in one manifest? Defer to backlog — start with single-server (star topology with one server peer)
- I-packet regeneration: regenerate on every `generate` run (current behavior), or persist and only regenerate with `--full-reset`? Current behavior (regenerate) is correct — I-packets should change over time for fingerprint resistance

---

### P2.5 — Remove legacy imperative CLI

**What:** Delete the old `init`, `add`, `edit`, `remove`, `export`, `list` commands and their backing code. The declarative `generate` command fully replaces them.

**Why:** Maintaining two code paths (imperative and declarative) doubles the surface area for bugs and confuses users. Decision 10: no backward compatibility shims.

**Where (files to delete or gut):**

- `internal/cli/init.go:1-244` — entire file (init command)
- `internal/cli/add.go:1-47` — entire file (add command)
- `internal/cli/edit.go:1-99` — entire file (edit command)
- `internal/cli/remove.go:1-37` — entire file (remove command)
- `internal/cli/export.go:1-145` — entire file (export command)
- `internal/cli/list.go:1-55` — entire file (list command)
- `internal/cli/cli.go:30-38` — remove `AddCommand` calls for deleted commands; add `NewGenerateCommand()`
- `manager.go:1-344` — entire file (`Manager` struct and all its methods: `Init`, `AddPeer`, `RemovePeer`, `FindPeer`, `ListPeers`, `ExportPeer`, `BuildPeerConfig`, `Load`, `Save`)
- `manager_test.go` — entire file

**Where (files to KEEP, adapted by earlier P2 items):**

- `types.go` — output types remain; manifest types added in P2.1
- `generator.go` — generation logic reused by `pipeline.go`
- `validation.go` — validation engine reused by `generate` and `validate` commands
- `analysis.go` — analysis engine reused by `analyze` command
- `parser.go` — INI parser reused for credential persistence (reading existing output configs)
- `writer.go` — config writer reused for output
- `keys.go` — key generation reused
- `presets.go` — preset system reused by manifest's `obfuscation.preset` field
- `cps.go` — CPS generation reused
- `protocols.go` — protocol templates reused

**Acceptance criteria:**

- `amnezigo --help` shows only: `generate`, `validate`, `analyze`
- `amnezigo init` / `amnezigo add` / etc. produce "unknown command" errors
- No dead code: `go vet ./...` and `golangci-lint run` pass with zero issues
- `manager.go` and all `internal/cli/{init,add,edit,remove,export,list}.go` files are deleted
- Test suite passes — all tests referencing deleted code are removed or migrated
- Documentation updated: `docs/cli-reference.md` reflects the new command set

**Brainstorm questions:**

- Should P2.5 be one big PR or split into "add generate command" + "remove old commands"? One PR is cleaner since both sides need to exist at the same time
- Deprecation period? Decision 10 says no — clean break. Version bump (v2.0.0) makes this clear
- What about users who have scripts using the old CLI? Document migration in release notes: "replace `init + add + export` with a single `amnezigo.json` manifest and `amnezigo generate`"

---

## P3 — Validation & Analysis Adaptation

Adapt the existing validation and analysis engines to work with the declarative manifest format.

### P3.1 — Adapt `validate` for declarative manifests

**What:** Extend the `validate` command to accept manifest files (`amnezigo.json` / `amnezigo.jsonnet`) in addition to the existing `awg0.conf` server configs. Validate both the manifest schema and AWG 2.0 invariants in one pass.

**Why:** Users need to validate their manifests before running `generate`. The existing `validate` command only understands INI server configs — it needs to also understand the new JSON/Jsonnet manifest format.

**Where:**

- `internal/cli/validate.go:53-93` — `runValidate` currently opens a file and calls `ParseServerConfigWithOptions`; add format detection (INI vs JSON/Jsonnet) and branch accordingly
- `validation.go:207-216` — `ValidateServerConfig` stays for INI configs; add `ValidateManifest(m *Manifest) []Finding` for manifest validation
- New file: `manifest_validation.go` — manifest-specific checks: version field, peer address uniqueness, server peer existence, obfuscation parameter ranges, preset validity
- `loader.go` (from P2.2) — reuse `LoadManifest` for manifest parsing

**Validation checks (manifest-specific):**

- `version` must be `1`
- Exactly one peer with `endpoint` + `listen_port` (server peer)
- All peer addresses are unique and within the server's subnet
- Obfuscation params (if explicit) pass `ValidatePacketSizes`
- If `preset` is specified, it exists in `presetRegistry`
- No reserved peer names (`output`, `.git`, etc.)

**Acceptance criteria:**

- `amnezigo validate amnezigo.json` validates the manifest and exits 0 on valid
- `amnezigo validate awg0.conf` still works as before (backward compat for INI configs)
- Format auto-detection: `.json`/`.jsonnet` → manifest validation; `.conf` → INI validation
- Manifest findings include peer name and field path in `Location`
- Test: valid manifest → 0 findings; invalid manifest (duplicate addresses, missing server peer) → specific errors

**Brainstorm questions:**

- Should `validate` also run `generate` in dry-run mode to catch generation-time errors (like MTU budget overflow)? Probably yes — "full validate" mode
- Error codes: new `MNF001`-`MNF00N` series for manifest-specific findings, or reuse existing codes where applicable?

---

### P3.2 — Adapt `analyze` for declarative manifests

**What:** Extend the `analyze` command to accept manifest files directly, analyzing the obfuscation profile without needing pre-generated output.

**Why:** Users should be able to analyze their obfuscation profile from the manifest before running `generate`. Currently `analyze` requires an existing INI server config.

**Where:**

- `internal/cli/analyze.go:68-104` — `runAnalyze` currently uses `Manager.Load()` to get a `ServerConfig`; add manifest loading path
- `analysis.go:142-177` — `Analyze()` takes `ServerConfig`; add overload or converter `ManifestToServerConfig(m Manifest) ServerConfig` that resolves presets and builds the analysis-compatible struct
- `loader.go` (from P2.2) — reuse `LoadManifest` for manifest parsing

**Acceptance criteria:**

- `amnezigo analyze --manifest amnezigo.json` analyzes the manifest's obfuscation profile
- `amnezigo analyze --config awg0.conf` still works as before
- Format auto-detection based on file extension
- `--peer NAME` works with manifest peer names
- Output is identical whether analyzing a manifest or its generated output

**Brainstorm questions:**

- Should `analyze` accept `--preset home-balanced` directly (without a manifest file) for quick preset evaluation?
- When analyzing a manifest, should all peers be analyzed or only client peers (since server peer doesn't use I-packets)?

---

### P3.3 — Preset integration with manifests

**What:** Enable presets to be used as jsonnet libraries importable in config files, and integrate the existing preset system with the manifest's `obfuscation.preset` field.

**Why:** Presets are currently used only via `--preset` CLI flag during `init`. In the declarative model, presets should be declarable in the manifest itself and optionally importable as jsonnet fragments for more flexible composition.

**Where:**

- `presets.go:51-120` — `presetRegistry` with built-in presets
- `presets.go:19-33` — `ToServerObfuscation()` converts preset to obfuscation config
- New: `lib/presets.libsonnet` — jsonnet library exposing presets as importable objects
- `pipeline.go` (from P2.4) — resolve `obfuscation.preset` during generation by calling `GetPreset`

**Usage in manifest:**

```jsonnet
local presets = import 'lib/presets.libsonnet';
{
  version: 1,
  obfuscation: presets.homeBalanced,
  // or: obfuscation: presets.homeBalanced + { s1: 40 },  // override S1
  // ...
}
```

**Acceptance criteria:**

- `manifest.obfuscation.preset = "home-balanced"` resolves to correct parameters during generation
- Jsonnet library `lib/presets.libsonnet` is auto-generated from `presetRegistry` (or manually maintained with a test that verifies sync)
- Preset + explicit params: explicit params override preset defaults (not an error)
- Test: manifest with preset → generate → validate → all passing

**Brainstorm questions:**

- Auto-generate `lib/presets.libsonnet` from Go code, or maintain manually? Auto-gen avoids drift but adds a build step
- Should presets be embeddable in the binary (like current `go:embed` pattern) so they work without a `lib/` directory on disk?
- Community presets: should they live in `presets/` as YAML (current plan from P1.5) or as `.libsonnet` files? Both — YAML for the Go registry, auto-gen jsonnet for users who want to import them

---

## Deferred — Backlog

Items from the original roadmap that are not part of the declarative pivot and are deferred to future planning.

### QR code export (was P2.1)

**What:** Add `--qr` flag to export that prints a QR code of the client config.

**Why:** AmneziaVPN mobile apps support QR-code config import. Eliminates manual file transfer.

**Status:** Deferred. Still relevant for the new `generate` command — could be a post-generation step (`amnezigo generate --qr`) or a separate `amnezigo qr <peer>` command.

---

### `<ds>` and `<dz>` tags (was P2.2)

**What:** Support the remaining data tags: `<ds>` (base64-encoded data passthrough), `<dz N>` (zero-data with size N, BigEndian).

**Why:** Completes the CPS tag set. Useful for protocols that expect base64 fields or fixed-zero padding.

**Status:** Deferred. Independent of architecture pivot; can land anytime after P2.

---

### Per-peer DNS (was P2.3)

**What:** Allow different DNS servers per peer rather than using a global default.

**Status:** Deferred. In the declarative model, this becomes a `dns` field on the peer manifest entry. Straightforward to add after P2.1.

---

### Multi-endpoint fallback (was P2.4)

**What:** Support multiple Endpoints in a client config (or DNS-based round-robin).

**Status:** Deferred. Requires investigation into AWG/WG multi-endpoint support.

---

### Rotation reminder (was P2.5)

**What:** When configs are older than a threshold, suggest regeneration.

**Status:** Deferred. In the declarative model, `generate` regenerates I-packets every run by default. The reminder becomes less critical — but could still warn about stale server keys.

---

### Mesh topology (was P3.1)

**What:** Allow peers to communicate directly (mesh), not only via central server (star).

**Status:** Deferred. The declarative manifest could support this by allowing multiple server peers, but the topology logic is significantly more complex. Start with single-server star topology.

---

### Integration tests with real `amneziawg-go` (was P3.3)

**What:** GitHub Action that boots a real `amneziawg-go` container, applies a generated config, and verifies a peer can connect.

**Status:** Deferred. Independent of architecture pivot; valuable at any point.

---

### Benchmark mode (was P3.4)

**What:** `amnezigo bench` runs the generator at load and reports throughput.

**Status:** Deferred. Nice-to-have for regression testing.

---

### Versioned presets (was P3.5)

**What:** Presets carry version metadata; `amnezigo update-presets` pulls newer versions from a remote repo.

**Status:** Deferred. Depends on P1.5 (done) and community traction.

---

### Additional protocol templates

**What:** Add more protocol templates beyond the current five (QUIC, DNS, DTLS, STUN, SIP).

**Candidates:**

- **NTP** — UDP/123, almost always permitted, very small packets
- **WebRTC TURN-Allocate** — extension of STUN, simulates ICE negotiation
- **WireGuard-handshake** — anti-canary: looks like another WG flow

**Status:** Deferred. Independent of architecture pivot; can land anytime. Each template is its own small PR.

---

## Atomic PR Strategy

Recommended PR order, optimized for small reviewable diffs:

**Already merged (P0 + P1):**

1. PR (P0.2): Fix `<t>` size from 8 → 4 bytes
2. PR (P0.1): Remove `<c>` tag
3. PR (P0.5): Verify and lock `<rc>` charset to `[a-zA-Z]`
4. PR (P0.3): Full pairwise size collision validation
5. PR (P0.4): Forbid H1-H4 ranges containing 1-4
6. PR (P1.1): Add `<d>` tag
7. PR (P1.3): `validate` command
8. PR (P1.4): `analyze` command
9. PR (P1.5): Preset infrastructure + built-in presets
10. PR (P1.2): SIP protocol template

**New (P2 — Declarative Core):**

11. **PR (P2.1):** Config schema & Go types — manifest structs, JSON schema draft
12. **PR (P2.2):** JSON + Jsonnet config loader — file discovery, go-jsonnet dependency, `--jpath`
13. **PR (P2.3):** Credential persistence — read existing output configs, reuse keys, `--full-reset`
14. **PR (P2.4):** `generate` command — full pipeline: load → validate → generate → write
15. **PR (P2.5):** Remove legacy imperative CLI — delete old commands and Manager

**New (P3 — Validation & Analysis Adaptation):**

16. **PR (P3.1):** Adapt `validate` for declarative manifests
17. **PR (P3.2):** Adapt `analyze` for declarative manifests
18. **PR (P3.3):** Preset integration with manifests — `lib/presets.libsonnet`

Each PR should include tests, doc updates, and changelog entry. Per project rules: signed commits, conventional commit messages, draft PR by default.

---

## References

### Source code

- [`amneziawg-go`](https://github.com/amnezia-vpn/amneziawg-go) — Go userspace, source of truth for AWG 2.0
- [`amneziawg-tools`](https://github.com/amnezia-vpn/amneziawg-tools) — `awg-quick` and `awg` CLI
- [`amneziawg-linux-kernel-module`](https://github.com/amnezia-vpn/amneziawg-linux-kernel-module) — kernel module (the only place `<c>` works)
- [`cheburbox`](https://github.com/Arsolitt/cheburbox) — sing-box config generator, reference for declarative patterns (jsonnet, credential persistence, atomic generation)

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
