# Repository Guidelines

Agent guidelines for **amnezigo** — a CLI tool and Go library that generates
AmneziaWG 2.0/3.0/3.1 configurations (default 3.1) from a declarative manifest.
Module: `github.com/Arsolitt/amnezigo`, Go 1.26.1, GPL-3.0.

> The repo completed a **declarative refactor** (plan phase P2): the legacy
> imperative CLI (`init`/`add`/`edit`/`list`/`export`/`remove`) and the
> `Manager` API were **removed entirely** (commit `226e4b8`). Only `generate`,
> `validate`, `analyze`, and `version` remain. Any reference to the old commands
> or `Manager` describes deleted code.

## Project Overview

amnezigo is a **config generator** (not a daemon). It reads a single manifest
(`amnezigo.json` or `.amnezigo.jsonnet`) describing the full network topology,
resolves AWG obfuscation parameters and X25519 crypto, then emits ready-to-deploy
`awg0.conf` files — one per server, one per client peer. Output is INI with `#_`
metadata comments that let later runs **reuse peer credentials** across regenerations.

It also doubles as an importable Go library: every business-logic function lives
in the root package `amnezigo` and is callable without the CLI.

## Architecture & Data Flow

All business logic lives as `.go` files at the **repo root** in package `amnezigo`
(no `internal/` for logic). The CLI is a thin cobra layer that calls into it.

The core flow is **manifest → generate → configs**:

```mermaid
flowchart LR
  M["manifest<br/>(.amnezigo.jsonnet or amnezigo.json)"] --> L[LoadManifest]
  L --> G["Generate(manifest, opts)"]
  G --> RO[resolveObfuscation]
  G --> RC[resolvePeerCredentials<br/>key reuse via #_ metadata]
  G --> BS[buildServerConfig]
  G --> BC["buildClientConfig × N"]
  BS & BC --> W["Write INI + #_ metadata<br/>output/&lt;server&gt;/awg0.conf<br/>output/&lt;peer&gt;/awg0.conf"]
```

1. **`LoadManifest`** (`loader.go:26`) — discovers the manifest. `.amnezigo.jsonnet`
   **takes precedence** over `amnezigo.json`; Jsonnet is evaluated to JSON then
   parsed. Version field MUST equal `1` (`currentManifestVersion`).
2. **`Generate`** (`pipeline.go:423`) — the orchestrator. Two-pass: compute all
   configs in memory, then write. Ordered steps:
   - `resolveObfuscation` (`pipeline.go:39`) — nil pointer fields in the manifest
     signal "generate randomly"; fills S/H/J defaults.
   - `LoadCredentials` + `resolvePeerCredentials` (`pipeline.go:164`) — reuse
     persisted keys unless `--full-reset`; client keys recovered from each peer's
     own client config and the server's `#_PrivateKey` metadata.
   - `buildServerConfig` / `buildClientConfig` — build INI structs.
   - Write each as `FileOutput{RelPath, Content}`. Server at `<serverName>/awg0.conf`,
     each client at `<peerName>/awg0.conf`.
3. **`WriteServerConfig` / `WriteClientConfig`** (`writer.go`) — INI serialization
   with `#_`-prefixed metadata lines.

Key supporting modules:

- `generator.go` — random obfuscation params (S-prefixes, junk, header ranges).
- `cps.go` — CPS (Custom Packet String) grammar and I-packet generation.
- `protocols.go` + `quic.go`/`dns.go`/`dtls.go`/`stun.go`/`sip.go`/`rtp.go` — protocol
  templates that mimic real wire formats; `getTemplate` dispatches by name.
- `validation.go` — AWG 2.0 size-classification invariants and config findings.
- `analysis.go` — `Analyze()` heuristic report (RISK001–009).
- `keys.go` — X25519 keypair + PSK generation with WireGuard clamping.
- `parser.go` — INI + `#_` metadata parser (server configs only).

## Key Directories

```
cmd/amnezigo/main.go   # Entry point: func main() { cli.Execute() }
internal/cli/          # Cobra commands: generate.go, validate.go, analyze.go, version.go, cli.go
internal/buildinfo/    # Version/Commit vars injected at build time via -ldflags -X
hack/noticegen/        # Stdlib-only NOTICE + licenses/ generator (go run ./hack/noticegen)
*.go                   # All business logic (root package `amnezigo`)
testdata/loader/       # Manifest fixtures (valid/, precedence/, invalid-*, …)
docs/                  # llms-full.txt is the source of truth; other guides are hand-written references
docs/plans/            # P0–P3 roadmap plans (PR blueprints)
.github/               # ci.yml (lint, unit, e2e, cross-build, release config, licenses), release.yml
```

## Development Commands

Install the pinned toolchain once (`mise.toml`: Go 1.26.8, golangci-lint 2.14.0,
goreleaser 2.18.1), then use the `Makefile` targets:

```bash
mise install

make build        # go build -ldflags "…" -o bin/amnezigo ./cmd/amnezigo
make test         # go test ./...
make test-race    # go test -race ./...
make test-e2e     # go test -tags=e2e ./e2e/...  (drives Docker containers; self-skips without Docker)
make lint         # golangci-lint run
make fmt          # golangci-lint fmt
make notice       # go run ./hack/noticegen  (regenerate NOTICE + licenses/)
make snapshot     # goreleaser release --snapshot --clean
make image        # docker build --platform linux/amd64 -t amnezigo .
make clean        # rm -rf bin dist

# Single test (root-package convention: `.`, not ./internal/...)
go test -run TestFunctionName .
```

`make build` and GoReleaser stamp the binary via
`-ldflags -X internal/buildinfo.{Version,Commit}` (`make build` derives the
values from `git describe` / `git rev-parse`). Production binaries use
`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w"` (the release
matrix is amd64-only — see `.goreleaser.yaml`). Pre-merge quality bar: green
`make test-race` + zero `make lint` errors.

## Code Conventions & Common Patterns

### Package layout
- Root package `amnezigo` holds **all** business logic; `internal/cli` is a thin
  cobra wrapper; `cmd/amnezigo` is a one-line entry point. Tests are co-located
  with implementation (`*_test.go` in package `amnezigo`, white-box).

### Imports
stdlib, then external, then internal — blank line between groups:
```go
import (
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/Arsolitt/amnezigo"
)
```
`goimports` local prefix is `github.com/Arsolitt/amnezigo`.

### Error handling
- Wrap with context: `fmt.Errorf("loading manifest: %w", err)`.
- Generator retry loops panic on exhaustion (fail-fast, e.g. `sMaxAttempts`).
- `tryDerivePublicKey` recovers a panic → returns `""` so the pipeline regenerates.
- CLI commands use `RunE` and return wrapped errors (except `validate`, which
  calls `os.Exit(1)` directly via the `exitFn` override seam).

### Crypto & randomness
- **`crypto/rand` everywhere in production** (via `math/big`). `math/rand` is
  **forbidden** in non-test files by the linter — use `math/rand/v2` if needed.
- WireGuard key clamping: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`.
- Keys base64 (`StdEncoding`, 44 chars); `GenerateKeyPair`/`DerivePublicKey`/
  `GeneratePSK` panic only on unrecoverable system failures.

### Determinism
- Peer iteration is **sorted** (`sort.Strings`) in `Generate`, `buildServerConfig`,
  and `PeerNames()`. Never rely on Go map iteration order.

### Config format: INI + `#_` metadata
- Standard INI keys (`[Interface]` / `[Peer]`, `key = value`).
- Lines prefixed `#_` are **persisted metadata** (parsed back); bare `#` are
  ignored comments. Example: `#_PrivateKey`, `#_EndpointV4`, `#_Name`, `#_GenKeyTime`.
- `HeaderRange` serialized as `min-max` (uint32).

### Atomicity & I/O
- `SaveServerConfig` (`writer.go:139`) writes atomically: `.tmp` + `os.Rename`.
- **`Generate` writes via plain `os.WriteFile` (`0600`), NOT the atomic helper** —
  a mid-run failure can leave partial output. (There is no `SaveClientConfig` or
  `ParseClientConfig`; client configs are read only by the lightweight
  `extractClientCredentials` scanner for key recovery.)

### Pointer-nil semantics in the manifest
`ObfuscationManifest` uses `*int` / `*HeaderRange` to distinguish "set to 0"
from "unset" — `nil` drives the random-fallback in `resolveObfuscation`.

### CPS tag grammar (`cps.go`)
Supported tags (the legacy `<c>` counter tag is **deliberately removed** — it is
kernel-module-only and breaks `amneziawg-go` + all AmneziaVPN clients):

| Tag | Meaning | Length |
|-----|---------|--------|
| `<b 0xNN>` | literal bytes (hex) | `len(NN)/2` |
| `<r N>` | N random bytes | N |
| `<rc N>` | N chars from `[a-zA-Z]` (52-char alphabet) | N |
| `<rd N>` | N random digits | N |
| `<t>` | timestamp (uint32 BE) | 4 |
| `<d>` | data passthrough (AWG 2.0 userspace) | 0 |

### CLI conventions
- `spf13/cobra` v1.10.2. Commands built via `New*Command()` factories — **no
  `init()`**; flag setup lives inside each factory.
- Flag names are kebab-case (`--full-reset`, `--dry-run`, `--jpath`).
- `generate`/`analyze` bind closure-local vars; `validate` binds package-level
  vars (not re-entrant). Prefer the closure style for new commands.
- All user output goes through `cmd.OutOrStdout()` so tests inject a buffer.

## Important Files

| File | Role |
|------|------|
| `cmd/amnezigo/main.go` | Entry point → `cli.Execute()` |
| `internal/cli/cli.go` | Root command + `Execute()`; registers the 4 subcommands |
| `internal/cli/generate.go` | `generate` — manifest → configs pipeline driver |
| `internal/cli/validate.go` | `validate <config>` — lint a server config against AWG size invariants |
| `internal/cli/analyze.go` | `analyze` — RISK001–009 heuristics + size profiles |
| `internal/cli/version.go` | `version` — prints the build stamp (`amnezigo <Version> (<Commit>)`) |
| `internal/buildinfo/buildinfo.go` | `Version` / `Commit` vars injected at build time via `-ldflags -X` |
| `manifest.go` | User-facing manifest schema (`Manifest`, `PeerManifest`, `ObfuscationManifest`) |
| `loader.go` | Manifest discovery + Jsonnet/JSON precedence + version validation |
| `pipeline.go` | `Generate()` orchestrator — the heart of the system |
| `credentials.go` | Peer key reuse across runs |
| `generator.go` | Random obfuscation generation |
| `cps.go` | CPS grammar + I-packet generation |
| `protocols.go` + `quic/dns/dtls/stun/sip/rtp.go` | Protocol templates + `getTemplate` dispatch |
| `validation.go` | `ValidatePacketSizes`, `ValidateHeaderRange`, `ValidateServerConfig` |
| `analysis.go` | `Analyze()` report + RISK heuristics |
| `keys.go` | X25519 keypair + PSK |
| `parser.go` / `writer.go` | INI + `#_` metadata parse/serialize |
| `presets.go` | Named obfuscation bundles (`lan-conservative`, `home-balanced`, `mobile-aggressive`, `stealth-paranoid`, `standard-1420`, `low-overhead`, `test-minimal`) |
| `testdata/loader/valid/amnezigo.json` | Canonical reference manifest |
| `docs/llms-full.txt` | **Source-of-truth** AI-friendly doc (current architecture) |
| `.goreleaser.yaml` | Release pipeline: amd64-only builds (linux/darwin), raw binaries + checksum + license bundle, GHCR image |
| `Makefile` | Dev targets: `build`, `test`, `test-race`, `test-e2e`, `lint`, `fmt`, `notice`, `snapshot`, `image`, `clean` |
| `mise.toml` | Pinned tool versions: Go 1.26.8, golangci-lint 2.14.0, goreleaser 2.18.1 |
| `hack/noticegen/main.go` | Regenerates `NOTICE` + `licenses/` from the module graph |
| `.github/workflows/ci.yml` | CI: `lint`, `unit_test`, `e2e`, `cross_build`, `release_config`, `licenses` |
| `.github/workflows/release.yml` | Tag-driven GoReleaser release (`v*`) |

## Runtime / Tooling Preferences

- **Go 1.26.1** in `go.mod`; `mise.toml` pins the local toolchain (Go 1.26.8,
  golangci-lint 2.14.0, goreleaser 2.18.1) — run `mise install` once.
- Direct deps: `github.com/google/go-jsonnet` v0.22.0, `github.com/spf13/cobra`
  v1.10.2, `golang.org/x/crypto` v0.57.0 (curve25519). No test-only deps beyond stdlib.
- **Linter**: `golangci-lint` v2.14.0 with a strict "golden config" (~70 linters,
  `.golangci.yaml`). Notable: `depguard` forbids `math/rand` (non-test), `log`
  outside main (use `log/slog`); `mnd` flags magic numbers; `golines` enforces
  **120-char** max line length; `goimports` local prefix
  `github.com/Arsolitt/amnezigo`. `gochecknoglobals`, `gochecknoinits`,
  `paralleltest`, and `testpackage` are deliberately disabled (see the comments
  in `.golangci.yaml`).
- Commands run through `make` (see Development Commands) or the `mise` tools;
  release metadata lives in `.goreleaser.yaml` (tag convention `vX.Y.Z` stable /
  `vX.Y.Z-rc.N` prerelease).
- `.gitignore` excludes `bin`, `build`, `dist`, `*.conf`, `*.config`.

## Testing & QA

- **Stdlib `testing` only — no testify.** Assertions are manual
  `if got != want { t.Errorf(...) }`; `t.Fatalf` for setup failures.
- **Two-step error contract** in failure tests: assert `err != nil`, then
  `strings.Contains(err.Error(), expectedSubstring)`. **Error substrings are part
  of the public contract** — editing a message in `loader.go`/`validation.go`
  silently breaks tests.
- **Naming**: `TestFunctionName` for unit tests; `TestFunctionName_Scenario` for
  variants. Table-driven (`t.Run`) for fan-out (protocols, presets, MTU, boundary
  values); one-test-per-case for targeted regressions.
- **Fixtures**: real files under `testdata/loader/<sub>/` accessed via
  `filepath.Join("testdata", "loader", ...)`. **Tests are path-relative and only
  pass when `go test ./...` runs from the repo root** (where `package amnezigo`
  lives).
- **No `t.Parallel()`** in root tests (the `tparallel` linter is on but
  `paralleltest` is off). Adding it is allowed but unprecedented.
- In-memory I/O via `strings.NewReader` / `bytes.Buffer`; filesystem via
  `t.TempDir()`. No golden-file pattern.
- Randomness in tests relies on statistical variety over many iterations (e.g.
  `TestGenerateKeyPairUniqueness` loops 100×), not seeded RNG. The `analyze --seed`
  flag seeds an injected `io.Reader` (`math/rand/v2` PCG) only inside the CLI.
- All test helpers call `t.Helper()`.

### Reference manifest
`testdata/loader/valid/amnezigo.json` is the canonical manifest all loader/
generator tests build on: `version=1`, `network.mtu=1280`, obfuscation
(`protocol:"quic"`, `s1:30`/`s2:35`/`s3:20`/`s4:12`, full H1–H4, `jc:5`/
`jmin:250`/`jmax:750`), 2 peers (`server` + `phone`).

## Adding a New Protocol Template

Every new protocol template MUST satisfy this contract. Reviewers reject PRs that
miss any item. (sip.go + sip_test.go is the reference implementation.)

**Required**
- File `<protocol>.go` at the repo root with constructor `XxxTemplate() I1I5Template`
  (pure data, no I/O/globals).
- Co-located `<protocol>_test.go`.
- A `case` in `protocols.go:getTemplate` switch, append the constructor to the
  random-fallback slice, a row in `TestGetTemplate_NamedProtocols`
  (`protocols_test.go`), and the `--protocol` flag helptext in `internal/cli/analyze.go`.

**Tag rules**
- No `<c>` tag (removed in P0.1). For pseudo-monotonic bytes use `<rd N>` / `<r N>`.
- `<t>` is 4 bytes; at most one `<t>` per interval. `<rc>` is `[a-zA-Z]` only.

**Byte budget**
- Each interval ≥ 16 B (avoid raw-WG size collisions) and ≤ `MTU - 49 - 149 - S1`.
- Recommended ceiling ≤ 700 B per interval; `I5` always empty; `I1 ≥ I2 ≥ I3 ≥ I4`.
- Leading bytes must not collide with any prefix in the `existingTemplatePrefixes`
  slice in `protocols_test.go` — new fixed prefixes MUST be appended there in the
  same PR.

**Required per-template tests**
`TestXxxTemplate_AllIntervalsNonEmpty_I1ToI4`, `_I5Empty`, `_NoForbiddenTags`,
`_NoCounterLiteral`, `_FitsMTU`, `_ByteBudgetUnderCeiling`,
`_AtMostOneTimestampPerInterval`, `_AvoidsExistingPrefixes` (calls
`assertTemplateAvoidsExistingPrefixes`).

## Gotchas

- **`generate` is not atomic on disk** — it writes via `os.WriteFile`, not
  `SaveServerConfig`. A mid-run crash can leave a partial `output/` tree.
- **Per-client I1–I5 are not stored in the server config** — they live only in
  each client's own `awg0.conf`. CPS strings are regenerated every run; only
  crypto keys are reused.
- **Each client's PrivateKey is stored as `#_PrivateKey` in the server config's
  `[Peer]`** — this is the key-reuse recovery source, alongside the client config.
- **Docs split**: `docs/llms-full.txt` is the source of truth. The standalone
  guides under `docs/` are hand-written references for the current declarative
  CLI; pages describing the removed imperative commands are historical only.
- **The release matrix is amd64-only** (`.goreleaser.yaml`: linux/amd64 and
  darwin/amd64 raw binaries, `checksums.txt`, plus a
  `amnezigo-licenses_<version>.tar.gz` bundle). The GHCR images are built from
  the same single platform.
- **The container images are amd64-only** — both the from-source `Dockerfile`
  and the published `ghcr.io/arsolitt/amnezigo` image. On arm64 hosts pass
  `--platform linux/amd64` to `docker build`; `make image` already pins it.
  `mise exec -- goreleaser check` validates the release config locally.
- **`validate` always parses with `Strict:true`** regardless of `--strict`;
  `--strict` only affects the exit code (warnings → exit 1). `--quiet` is
  text/summary-only (ignored in JSON mode).
- **`analyze` always exits 0** — findings are informational.
- Default protocol is `quic`; default MTU is `1280` when unset in the manifest.
