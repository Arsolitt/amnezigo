# Gotchas

> Pitfall catalog: every sharp edge in amnezigo's generate/validate/analyze pipeline, grouped by theme so you can CTRL-F your symptom.

## Table of Contents

- [How to read this page](#how-to-read-this-page)
- [Generation & I/O](#generation--io)
- [Manifest semantics](#manifest-semantics)
- [Loader & Jsonnet](#loader--jsonnet)
- [Crypto & credentials](#crypto--credentials)
- [Obfuscation & CPS grammar](#obfuscation--cps-grammar)
- [CLI quirks](#cli-quirks)
- [Validation & analysis](#validation--analysis)
- [Presets & Docker](#presets--docker)
- [Stale docs & removed code](#stale-docs--removed-code)
- [Testing](#testing)

---

## How to read this page

Each table row is one pitfall. Columns:

| Column | Meaning |
|---|---|
| **Gotcha** | What bites you. |
| **Impact** | What you observe (the symptom to CTRL-F). |
| **Where** | Source location `file:area`. |
| **Workaround / fix** | What to do about it. |

Severity is encoded inline as `(high)` / `(med)` / `(low)` / `(info)`. See
[`./manifest-reference.md`](./manifest-reference.md) for field definitions,
[`./output-format.md`](./output-format.md) for the generated INI shape,
[`./credentials.md`](./credentials.md) for the key-reuse model,
[`./obfuscation.md`](./obfuscation.md) for CPS tag grammar, and
[`./validation.md`](./validation.md) for finding codes.

## Generation & I/O

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| `Generate` writes each file with `os.WriteFile` — **not atomic** `(med)` | A crash or power loss mid-run can leave a partially-written `awg0.conf` on disk. The build phase is atomic (see below), but the write phase is not. | `pipeline.go:510-525` (write loop, `os.WriteFile` at `:521`) | Treat `output/` as regenerated, not edited. Re-run `amnezigo generate` after any interruption rather than trusting half-written files. |
| `SaveServerConfig` **is** atomic (`.tmp` + `os.Rename`) but `Generate` does **not** use it `(med)` | The library exposes an atomic write path that the orchestrator bypasses; library consumers who call `SaveServerConfig` directly are safe, CLI users are not. | `writer.go:137-154` (`SaveServerConfig`); `pipeline.go:521` (uses plain `os.WriteFile`) | If you embed amnezigo as a library, prefer `SaveServerConfig`/`SaveClientConfig` over re-implementing writes. See [`./library-usage.md`](./library-usage.md). |
| Build phase **is** atomic: all configs computed in memory before any file is written `(info)` | If any peer's config fails to build, **nothing** is written — you never get a half-generated set. Directories are mode `0750`, files `0600`. | `pipeline.go:414-528` (two-pass `Generate`) | None — this is the intended safety property. A failed `generate` leaves the previous `output/` intact. |
| CPS strings (`I1`–`I5`) are **regenerated every run**; only keys/PSKs are reused `(med)` | Re-running `generate` produces different `I1`–`I5` jitter intervals in client configs even though private keys stay stable. Confusing if you diff outputs expecting determinism. | `pipeline.go:454-504` (creds reused via `resolvePeerCredentials`; CPS rebuilt in `buildServerConfig`/`buildClientConfig`) | Expected behavior — CPS is not persisted. Persist obfuscation params explicitly in the manifest if you need stable CPS shape. See [`./credentials.md`](./credentials.md). |
| Client `AllowedIPs` hardcoded to full tunnel; `PersistentKeepalive = 0` always printed `(med)` | `buildClientConfig` forces `AllowedIPs = 0.0.0.0/0, ::/0` (not configurable). `WriteClientConfig` emits `PersistentKeepalive = 0` even when keepalive is unset. | `pipeline.go:328-412` (`buildClientConfig`); `writer.go:89-135` (`WriteClientConfig`) | For split-tunneling or omitted keepalive, hand-edit the generated client config. |
| No `ParseClientConfig`; client configs read by a minimal line scanner `(low)` | Only server configs are fully parsed. Client `PrivateKey`/`PresharedKey` are recovered via `extractClientCredentials`, a line-by-line INI scanner that skips `#` but reads `#_` metadata. | `writer.go` (no client serializer); `credentials.go` (`extractClientCredentials`) | Don't expect `validate`/`analyze` to deeply inspect client configs. See [`./output-format.md`](./output-format.md). |
| Output ordering is deterministic `(info)` | Peers are sorted alphabetically (`sort.Strings`) for directory creation and server-config `[Peer]` ordering, so diffs between runs are stable (modulo regenerated CPS). | `manifest.go:158-168` (`PeerNames`); `pipeline.go:464-470` | None — useful for reviewing changes. |

## Manifest semantics

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| Numeric obfuscation fields are **pointers** — `null` vs `0` matters `(high)` | Omitting `s1` = `nil` = random generation. Setting `"s1": 0` = fixed at zero. A zero S-prefix is legal input but the generator retries to keep padded sizes distinct. Same rule applies to `h1`–`h4` (`*HeaderRange`), `jc`, `jmin`, `jmax`, `keepalive`. | `manifest.go:22-35` (`ObfuscationManifest` pointer fields); `pipeline.go:36-153` (`resolveObfuscation`, `resolveInt`) | Set a field only when you want it pinned. Leave it out for random. See [`./manifest-reference.md`](./manifest-reference.md). |
| `version` is **required** and must equal `1` (no `omitempty`) `(high)` | `{}` decodes to `version: 0` → `missing or zero version field`. `version: 99` → `unsupported schema version 99 (expected 1)`. | `manifest.go:121-126` (`Manifest.Version` tag `json:"version"`); `loader.go:17-19` (`currentManifestVersion = 1`); `loader.go:120-132` (`validateManifestVersion`) | Always include `"version": 1` at the top level. |
| Exactly **one** server peer required; detection = `endpoint` AND `listen_port` `(high)` | A peer with only `endpoint` or only `listen_port` is a **client**. `Generate` errors `exactly one server peer required, found N` for 0 or >1 servers. `ServerPeerName()` **panics** if count ≠ 1. | `manifest.go:104-108` (`IsServer`); `manifest.go:128-156` (`ServerPeer`, `ServerPeerName`); `pipeline.go:426-430` | Set both `endpoint` and `listen_port` on exactly one peer. |
| Top-level `obfuscation.protocol` is **decorative** — not consumed `(med)` | Setting `obfuscation.protocol` has no effect on generation. Only the per-peer `peers[].protocol` drives `I1`–`I5` generation, defaulting to `quic` when empty. | `manifest.go:22-35` (`Protocol` field exists but pipeline ignores it); `pipeline.go` (`buildClientConfig` reads `peers[].protocol`) | Set `protocol` per client peer. Omit the top-level field or treat it as a comment. See [`./manifest-reference.md`](./manifest-reference.md), [`./obfuscation.md`](./obfuscation.md). |
| Peer names are **map keys**, not a struct field `(med)` | `peers` is `map[string]PeerManifest`. The JSON object key is the peer name and becomes the output directory name. There is no `name` field; an unnamed peer cannot be matched on reload. | `manifest.go:121-126` (`Manifest.Peers` map type); `manifest.go:37-52` (`PeerManifest` has no name field) | Always use meaningful map keys; they ARE the identity. |
| Default protocol is `quic`; default MTU is `1280` `(low)` | A client peer with no `protocol` gets `quic` I-packets. `network.mtu` defaults to `1280` when unset. | `pipeline.go` (protocol fallback); `manifest.go:8-14` (`NetworkConfig.MTU`) | Set `protocol` and `mtu` explicitly if you need non-default behavior. |

## Loader & Jsonnet

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| Jsonnet file must be **dot-prefixed** (`.amnezigo.jsonnet`) for auto-discovery `(high)` | `LoadManifest` looks for `.amnezigo.jsonnet` (leading dot) then `amnezigo.json`. A file named `amnezigo.jsonnet` (no dot) is **not** auto-discovered; only `LoadManifestFromFile` with an explicit path evaluates it. | `loader.go:14-15` (`manifestJsonnet = ".amnezigo.jsonnet"`); `loader.go:26-43` (`LoadManifest` switch) | Name Jsonnet manifests exactly `.amnezigo.jsonnet`, or pass an explicit path. |
| `.jsonnet` takes **precedence** over `.json` when both exist `(med)` | If both `.amnezigo.jsonnet` and `amnezigo.json` are present, only the Jsonnet file is evaluated; the JSON file is silently ignored. | `loader.go:26-43` (`case errJsonnet == nil` first) | Delete or rename the stale file if you intend to switch formats. See [`./jsonnet.md`](./jsonnet.md). |
| Default Jsonnet `jpath` is `[<manifestDir>/lib]` `(med)` | When `--jpath` is empty, imports resolve against `<manifestDir>/lib`. Imports from elsewhere need an explicit `--jpath`. Imports are relative to jpath entries, not the manifest file directory. | `loader.go:58-65` (`resolveJpath`) | Pass `--jpath` for imports outside `<dir>/lib`. |
| `version` is still required in Jsonnet output `(med)` | Jsonnet evaluates to JSON, which is then parsed by the same loader — so the emitted JSON must still contain `"version": 1`. | `loader.go:86-105` (`loadFromJsonnet` → JSON parse → `validateManifestVersion`) | Ensure your `.amnezigo.jsonnet` evaluates to an object with `version: 1`. |

## Crypto & credentials

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| Peer `PrivateKey` recovered from the **client's own** config `(high)` | `generate` never writes a peer's `PrivateKey` to the server config. `LoadCredentials` reads each peer's `PrivateKey` from `output/<peer>/awg0.conf` `[Interface]`; it defensively ignores any server-side `#_PrivateKey` the parser may have read, so keys never rotate on every `generate`. If the client config is missing, `PrivateKey` is empty → fresh keypair next run. | `credentials.go` (`LoadCredentials`, `extractClientCredentials`); `pipeline.go:300-313` (no peer `PrivateKey` in `buildServerConfig`) | Don't delete client configs expecting the server to back-fill keys — it holds none. See [`./credentials.md`](./credentials.md). |
| `--full-reset` regenerates **all** keys; without it, keys are reused `(med)` | Default behavior reuses persisted server+client keys/PSKs from `output/`. `--full-reset` ignores persisted state and generates fresh material for every peer. | `pipeline.go:155-213` (`resolvePeerCredentials`, `fullReset` param); `pipeline.go:455` | Use `--full-reset` only when you intend to rotate all keys. See [`./cli-reference.md`](./cli-reference.md). |
| `DerivePublicKey` **panics** on malformed input `(med)` | `DerivePublicKey` panics on invalid base64 or non-32-byte length. `credentials.go` wraps it in `tryDerivePublicKey` (recovers → `""`), so a hand-edited bad key won't crash `generate`. Calling `DerivePublicKey` directly in library code will crash. | `keys.go:32-54` (`DerivePublicKey` panics); `credentials.go:175-184` (`tryDerivePublicKey` recover) | In library code, either validate input first or wrap calls in `recover`. See [`./library-usage.md`](./library-usage.md). |
| `GenerateKeyPair` / `GeneratePSK` panic only on `crypto/rand` failure `(med)` | These panic with `"crypto: failed to generate random key"` only if the system PRNG is unavailable — effectively never on a healthy host, but unhandled in library code. | `keys.go:12-30` (`GenerateKeyPair`); `keys.go:56-64` (`GeneratePSK`) | Acceptable for CLI use; library callers in long-running services should recover. |
| WireGuard key clamping applied before scalar multiplication `(info)` | `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64` is applied in both `GenerateKeyPair` and `DerivePublicKey`. Raw random bytes are never used directly. | `keys.go:20-23`, `keys.go:45-48` | None — clamping is mandatory for valid Curve25519 keys. |
| Server never gets a `PresharedKey`; each client gets a unique PSK `(med)` | `resolvePeerCredentials`: server gets a keypair only; every client gets a keypair + `GeneratePSK()`. The PSK is per client-to-server connection, recovered from the server `[Peer]` section on reload. | `pipeline.go:155-213` (`resolvePeerCredentials`) | Don't expect a `PresharedKey` line in the server `[Interface]` section. See [`./credentials.md`](./credentials.md). |

## Obfuscation & CPS grammar

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| **No `<c>` tag** — removed; `BuildCPSTag("c", ...)` returns `""` `(high)` | The counter `<c>` tag is kernel-only and rejected by `amneziawg-go` and all AmneziaVPN clients (`unknown tag`). Any literal `<c>` in a config fails `validate` as `CPS001`. | `cps.go` (`BuildCPSTag`, `mapTagType` rejects counter); `validation.go` (`CPS001`) | Never use `<c>`. Use `<rc>`/`<rd>` for random content. See [`./obfuscation.md`](./obfuscation.md). |
| **`<t>` is 4 bytes** (not 8) `(high)` | `<t>` is a `uint32` big-endian timestamp = **4 bytes**. The old docs claimed 8 bytes — that was wrong. At most one `<t>` per interval. | `cps.go` (timestamp tag); `cps_test.go` (`TestCalculateCPSLengthMatchesAccountedSize` pins 4 bytes against `amneziawg-go/device/obf_timestamp.go`) | Size your intervals accounting for 4 bytes per `<t>`. |
| `<rc>` alphabet is **letters only** `[a-zA-Z]` (52 chars), NOT alphanumeric `(med)` | The Habr article is wrong: `<rc>` emits ASCII letters only, no digits. For mixed alphanumeric fields use `<rc 4><rd 2>`. | `cps.go` (`cpsRcAlphabet`, 52 letters); `cps_test.go` (`TestRcAlphabetMatchesReference`) | Use `<rd>` for digits, `<rc>` for letters, combined for mixed. |
| `<d>` is a 0-byte runtime passthrough; requires **AWG 2.0 userspace** `(med)` | `<d>` emits 0 bytes at config time and is expanded by AWG 2.0 userspace. The legacy `amneziawg-linux-kernel-module` rejects it (`unknown tag`). | `cps.go` (`<d>` tag); `Dockerfile` (`amneziavpn/amneziawg-go:0.2.16`) | Run AWG 2.0 userspace (the pinned Docker image does). Avoid `<d>` on kernel-module deployments. |
| `GenerateCPS` ignores its 4th arg (`jc`); `GenerateServerConfig` ignores its 1st `(med)` | Misleading signatures: positional args are discarded. The pipeline still passes them — harmless, but the call sites don't match the signatures' intent. | `generator.go` (`GenerateCPS`, `GenerateServerConfig` parameter lists) | Don't rely on `jc`/protocol reaching CPS generation through these args; values come from the manifest/obfuscation config. |
| Junk range boundary is **inclusive**; duplicate I-sizes allowed `(med)` | `ValidatePacketSizes` treats `[jmin..jmax]` inclusive — `jmin` or `jmax` exactly equal to a forbidden (padded or raw WG) size **is** a collision. Duplicate I-packet sizes are allowed; only I-vs-padded collisions fail. | `validation.go` (`ValidatePacketSizes`) | Keep junk endpoints clear of WG/padded sizes by a margin. See [`./validation.md`](./validation.md). |
| I-packet MTU bound is strict (`<`); zero/negative `maxI` falls back to `<t>` `(low)` | When `maxISize <= 0` (tiny MTU), every interval falls back to the minimal `<t>` (4 bytes). Named-template intervals are shrunk tag-by-tag before fallback. | `cps.go` (`cpsAcceptable`, `calculateMaxISize`) | Use `mtu ≥ 1280` (the default) to avoid degenerate CPS. See [`./obfuscation.md`](./obfuscation.md). |
| Protocol templates have **fixed leading bytes** that must not collide `(info)` | QUIC `0xC0`, DTLS `0x16`, STUN `0x0001`, SIP `OPTIONS ` — new templates must avoid these so `--protocol random` yields distinguishable shapes. DNS and RTP have no fixed prefix. | `protocols_test.go` (`existingTemplatePrefixes`) | None for users; relevant only if extending templates. |

## CLI quirks

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| `validate` **always** parses with `Strict: true`; `--strict` only changes exit code `(low)` | Unknown keys and raw `<c>` tags always appear as `KEY001`/`CPS001` findings regardless of `--strict`. `--strict` only promotes warnings to a non-zero exit code; without it, warnings print but exit 0. | `internal/cli/validate.go` (`ParseOptions{Strict:true}` unconditional) | Don't expect `--strict` to silence findings; it only controls the exit code. See [`./cli-reference.md`](./cli-reference.md), [`./validation.md`](./validation.md). |
| `analyze` **always exits 0**; findings are advisory `(low)` | `analyze` exits 0 on success no matter how many `RISK001`–`RISK009` findings fire — they are `warning`/`info` only. Non-zero exit only on config-load or output-format errors. | `internal/cli/analyze.go` | Don't script `analyze` exit codes for quality gating; parse its output instead. See [`./validation.md`](./validation.md). |
| `--quiet` is ignored in JSON mode `(low)` | `--output json` always emits the full JSON report; `--quiet` only affects text output. | `internal/cli/validate.go`, `internal/cli/analyze.go` | Use `--output json` without `--quiet`, or `--output text --quiet` for terse results. |
| `analyze --config` defaults to `awg0.conf` in **CWD**, not the generated path `(med)` | `analyze` reads `awg0.conf` relative to the current directory by default. To analyze a generated config, pass `--config output/<server>/awg0.conf` explicitly or `cd` into the server dir. | `internal/cli/analyze.go` (`--config` default `awg0.conf`) | Always pass `--config` with the full path for generated configs. |
| `generate --peer` filters **clients only**; server is always generated `(med)` | `PeerFilter` applies to client peers only; the server config is always emitted. A `--peer` name absent from the manifest is **silently ignored**. | `pipeline.go:472-486` (filter applies to `filteredClients`); `pipeline.go:488-492` (server always added) | Verify peer names against the manifest before relying on `--peer`. See [`./cli-reference.md`](./cli-reference.md). |

## Validation & analysis

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| H range containing WG type-id `[1..4]` is a **parse error**, not a finding `(low)` | `ParseServerConfigWithOptions` structurally rejects any `H1`–`H4` range overlapping `[1..4]` (returns an error) **before** `ValidateServerConfig` runs. `validate` surfaces this as `PSE001`. | `parser.go` (`ParseServerConfigWithOptions`); `validation.go` (`PSE001`) | Keep header ranges clear of `[1..4]`. See [`./validation.md`](./validation.md). |
| `Finding` severities are `error \| warning \| info` + a code `(info)` | Text format: `[SEV CODE] file:line (key=..): msg`. JSON format mirrors the struct. These shapes are a public contract pinned by tests. | `validation.go` (`Finding`, `Severity`); test pins in `internal/cli/validate_test.go` | Don't grep for free-form text; match on severity + code. See [`./validation.md`](./validation.md). |

## Presets & Docker

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| **No `preset` manifest field**; presets are copy-paste values `(med)` | `generate` does not resolve presets. Any `--preset` flag or `InitWithPreset` reference in older docs is stale. To use a preset, copy its S/H/J values into `obfuscation.*` (optionally via a Jsonnet lib). | `presets.go` (`GetPreset`, `ListPresets` — library helpers only); `manifest.go:22-35` (no preset field) | Copy preset values into the manifest. See [`./presets.md`](./presets.md), [`./jsonnet.md`](./jsonnet.md). |
| Dockerfile pins `amneziavpn/amneziawg-go:0.2.16` and assumes AWG 2.0 userspace `(low)` | Generated configs (especially `<d>` tags) assume AWG 2.0 userspace; the legacy `amneziawg-linux-kernel-module` rejects `<d>`/`<c>`. | `Dockerfile` (`FROM amneziavpn/amneziawg-go:0.2.16`) | Use the pinned image or another AWG 2.0 userspace runtime. See [`./installation.md`](./installation.md). |

## Stale docs & removed code

> **Note:** This section describes *historical* drift that the **current** doc set (this page included) has corrected. The old standalone guides under `docs/` described a CLI and API that no longer exist; they have been rebuilt. Do not treat the current `docs/*.md` as stale — only pre-rebuild copies and the repo's `README.md`/`AGENTS.md` "Quick Start" sections still carry the removed-command references below.

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| Old standalone guides described **removed** imperative commands `(high)` | Pre-rebuild `docs/installation.md`, `docs/configuration.md`, `docs/cli-reference.md`, `docs/library-usage.md`, `docs/obfuscation.md` documented `amnezigo init/add/edit/remove/export/list` and a `Manager` API — all **deleted** in commit `226e4b8`. The current doc set replaces them. | (historical) commit `226e4b8` | Use the current `docs/*.md`. Only `generate`, `validate`, `analyze` exist. See [`./cli-reference.md`](./cli-reference.md). |
| `README.md` / `AGENTS.md` "Quick Start" still reference removed commands `(high)` | The root `README.md` carries a stale-guides warning; `AGENTS.md` "Quick Start" may still list `init`/`add`/`edit`/`remove`/`export`/`list`, the `Manager` API (`NewManager`/`AddPeer`/`ExportPeer`), and files (`manager.go`, `init.go`, `edit.go`) that no longer exist. `cli_test.go` explicitly asserts the legacy commands are NOT registered. | `README.md`, `AGENTS.md` | Ignore any `init/add/edit/remove/export/list` reference. The `go install` command in those files is still valid; everything else command-related is stale. See [`./overview.md`](./overview.md). |
| Old `obfuscation.md` claimed `<t>` is **8 bytes** `(high)` | The pre-rebuild guide stated `<t>` = 8 bytes. Source confirms it is **4 bytes** (`uint32` big-endian). | (historical) old `docs/obfuscation.md`; corrected in `cps.go` + `cps_test.go` | Trust the current [`./obfuscation.md`](./obfuscation.md) and source. |

## Testing

| Gotcha | Impact | Where | Workaround / fix |
|---|---|---|---|
| **Stdlib `testing` only — no testify** `(info)` | Assertions are manual `if got != want { t.Errorf(...) }`; `t.Fatalf` for setup failures. No `testify` import exists in test files. | `AGENTS.md` (testing convention); `.golangci.yaml` (`testifylint` disabled); all `*_test.go` | When writing tests, follow the manual-assertion style; don't introduce `testify`. |
| Tests are **path-relative** — run `go test ./...` from repo root `(med)` | Tests reference fixtures via `filepath.Join("testdata", "loader", "valid")` etc., which only resolve from the repo root. Running `go test` from a subdirectory fails with missing-testdata errors. | `loader_test.go`, `*_test.go` (testdata paths) | Always run `go test ./...` from `/home/arsolitt/projects/amnezigo` (repo root). |
| **Error substrings are a public contract** `(med)` | Tests assert two-step: `err != nil`, then `strings.Contains(err.Error(), expectedSubstring)`. Changing an error message breaks tests and any tooling that matches on it. | `AGENTS.md` (two-step error contract); `*_test.go` (`strings.Contains` on errors) | Treat error message text as stable; update tests AND callers together if you must change one. |
