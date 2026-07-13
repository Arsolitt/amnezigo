# Library Usage

> Go API reference for `github.com/Arsolitt/amnezigo` — the declarative config-generator library that powers the `amnezigo` CLI.

## Table of Contents

- [Import & overview](#import--overview)
- [Generate pipeline](#generate-pipeline)
- [Manifest loading](#manifest-loading)
- [Manifest types & methods](#manifest-types--methods)
- [Crypto / keys](#crypto--keys)
- [Validation](#validation)
- [Analysis](#analysis)
- [Presets](#presets)
- [Credentials](#credentials)
- [Library gotchas](#library-gotchas)

---

## Import & overview

```go
import "github.com/Arsolitt/amnezigo"
```

The root package `amnezigo` exposes every business-logic entry point: manifest loading, the generate pipeline, validation, analysis, presets, credential reuse, and X25519 key utilities. The `amnezigo` binary (see [CLI Reference](./cli-reference.md)) is a thin wrapper over `LoadManifest` → `Generate` → `ValidateServerConfig` → `Analyze`; the same calls are available to any Go program. There is no `Manager` type and no `init`/`add`/`edit`/`remove`/`export`/`list` surface — generation is driven by a single declarative manifest (see [Manifest Reference](./manifest-reference.md)).

## Generate pipeline

`Generate` is the single orchestrator. It resolves obfuscation, loads or reuses credentials, builds one server config plus one client config per client peer, and writes them to disk unless `DryRun` is set or `OutputDir` is empty.

| Symbol | Signature | Purpose |
|---|---|---|
| `GenerateOptions` | `type GenerateOptions struct { ProjectDir string; OutputDir string; JpathDirs []string; PeerFilter []string; DryRun bool; FullReset bool; VPNLinks bool }` | Configures the generate pipeline: project/output dirs, Jsonnet `jpath`, optional peer filter, dry-run, credential reset, and vpn:// import-link generation. |
| `GenerateResult` | `type GenerateResult struct { ServerPeer string; Files []FileOutput; ClientPeers []string; Findings []Finding }` | Output of a generate run: server peer name, all file outputs, client peer names, and any findings. |
| `FileOutput` | `type FileOutput struct { RelPath string; Content []byte }` | A single generated file: path relative to the output dir and its byte content. |
| `Generate` | `func Generate(manifest Manifest, opts GenerateOptions) (GenerateResult, error)` | Orchestrates the full pipeline: resolve obfuscation, load/reuse credentials, build server + client configs, and write them unless `DryRun` or `OutputDir == ""`. |
| `EncodeVPNLink` | `func EncodeVPNLink(clientINI []byte, endpoint string, listenPort int, dns []string) string` | Wraps a client AWG INI config into an AmneziaVPN-app-importable vpn:// link. Returns a `vpn://` URL string (scheme + base64url-encoded payload). See [VPN Import Links](./vpn-links.md). |

### Options detail

| Field | Type | Effect |
|---|---|---|
| `ProjectDir` | `string` | Project root (informational; the manifest is loaded separately via `LoadManifest`). |
| `OutputDir` | `string` | Where configs are written as `<OutputDir>/<peer>/awg0.conf`. Empty → nothing is written (in-memory only). |
| `JpathDirs` | `[]string` | Jsonnet library search paths; forwarded to manifest loading. |
| `PeerFilter` | `[]string` | When non-empty, only the named client peers are built and written; the server is always built. |
| `DryRun` | `bool` | When `true`, configs are computed and returned in `result.Files` but never written. |
| `FullReset` | `bool` | When `true`, persisted keys are discarded and fresh credentials are generated for every peer. |
| `VPNLinks` | `bool` | When `true`, generates an additional `amnezigo.vpn` import-link file per client peer. See [VPN Import Links](./vpn-links.md). |

### End-to-end example

```go
package main

import (
    "fmt"
    "log"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    // 1. Load the manifest (".amnezigo.jsonnet" wins over "amnezigo.json").
    manifest, err := amnezigo.LoadManifest("./my-project", nil)
    if err != nil {
        log.Fatalf("load manifest: %v", err)
    }

    // 2. Generate configs. With DryRun we inspect the result without writing.
    result, err := amnezigo.Generate(manifest, amnezigo.GenerateOptions{
        OutputDir: "./my-project/output",
        DryRun:    true,
    })
    if err != nil {
        log.Fatalf("generate: %v", err)
    }

    fmt.Println("server peer:", result.ServerPeer)
    for _, f := range result.Files {
        fmt.Printf("  %s (%d bytes)\n", f.RelPath, len(f.Content))
    }

    // 3. Generate only selected client peers and write them to disk.
    result, err = amnezigo.Generate(manifest, amnezigo.GenerateOptions{
        OutputDir:  "./my-project/output",
        PeerFilter: []string{"phone", "laptop"},
    })
    if err != nil {
        log.Fatalf("generate: %v", err)
    }
    fmt.Println("clients written:", result.ClientPeers)
}
```

## Manifest loading

| Symbol | Signature | Purpose |
|---|---|---|
| `LoadManifest` | `func LoadManifest(dir string, jpathDirs []string) (Manifest, error)` | Discovers a manifest in `dir`: `.amnezigo.jsonnet` takes precedence, then `amnezigo.json`. Returns an error if neither exists. |
| `LoadManifestFromFile` | `func LoadManifestFromFile(path string, jpathDirs []string) (Manifest, error)` | Loads a manifest from an explicit path; `.jsonnet` extension evaluates via the Jsonnet VM, otherwise plain JSON. |

### Discovery & evaluation rules

| Concern | Behavior |
|---|---|
| Precedence | `.amnezigo.jsonnet` is checked **before** `amnezigo.json`; if both exist, the Jsonnet file wins. |
| `jpathDirs == nil` | Defaults to `[dir/lib]` for `LoadManifest`, `[parentDir/lib]` for `LoadManifestFromFile`. |
| `.jsonnet` evaluation | Jsonnet → JSON string → parsed as `Manifest`; the `version` field is still required. |
| Version check | Both paths reject any `version` other than `1` (missing or zero is also an error). See [Jsonnet](./jsonnet.md). |

## Manifest types & methods

| Symbol | Signature | Purpose |
|---|---|---|
| `Manifest` | `type Manifest struct { Peers map[string]PeerManifest; Obfuscation ObfuscationManifest; Network NetworkConfig; Version int }` | Top-level declarative config. `Version` must be `1`; peer names are the map keys. |
| `NetworkConfig` | `type NetworkConfig struct { DNS []string; MTU int }` | Network-level settings shared by all peers (MTU is per-interface; DNS applies to clients). |
| `ObfuscationManifest` | `type ObfuscationManifest struct { S1, S2, S3, S4 *int; H1, H2, H3, H4 *HeaderRange; Jc, Jmin, Jmax *int; Protocol string }` | Obfuscation profile. Pointer fields distinguish "set to 0" from "unset" (`nil` → random fallback). |
| `PeerManifest` | `type PeerManifest struct { Keepalive *int; Address string; TunName string; MainIface string; Endpoint string; Protocol string; ListenPort int }` | One peer; its name is the `Manifest.Peers` map key. |
| `HeaderRange` | `type HeaderRange struct { Min uint32; Max uint32 }` | Inclusive min-max range for an obfuscation header (H1–H4). |
| `(*ObfuscationManifest).HasAnyValue` | `func (o *ObfuscationManifest) HasAnyValue() bool` | Reports whether any S/H/J field is non-nil (`false` ⇒ fully random generation). |
| `(*ObfuscationManifest).ToSharedObfuscation` | `func (o *ObfuscationManifest) ToSharedObfuscation() ServerObfuscationConfig` | Copies non-nil values into a `ServerObfuscationConfig`; nil fields stay zero (signal for random generation). |
| `(*PeerManifest).IsServer` | `func (p *PeerManifest) IsServer() bool` | `true` when `Endpoint != ""` **and** `ListenPort != 0`. |
| `(*Manifest).ServerPeer` | `func (m *Manifest) ServerPeer() (string, int)` | Returns the server peer name and count. A valid manifest has `count == 1`. |
| `(*Manifest).ServerPeerName` | `func (m *Manifest) ServerPeerName() string` | Returns the sole server peer name. **Panics** if count ≠ 1 — call only after validation. |
| `(*Manifest).PeerNames` | `func (m *Manifest) PeerNames() []string` | All peer names sorted alphabetically (deterministic iteration). |

> **Server-peer detection rule:** a peer is a server peer iff `Endpoint != ""` **and** `ListenPort != 0`. At most one server peer is valid per manifest. `Generate` returns an error (not a panic) when the count is not exactly 1; `ServerPeerName` panics. See [Manifest Reference](./manifest-reference.md).

## Crypto / keys

| Symbol | Signature | Purpose |
|---|---|---|
| `GenerateKeyPair` | `func GenerateKeyPair() (string, string)` | Generates an X25519 key pair with WireGuard clamping; returns `(privateKey, publicKey)` as 44-char base64. |
| `DerivePublicKey` | `func DerivePublicKey(privateKey string) string` | Derives the 44-char base64 public key from a base64 private key (applies clamping). |
| `GeneratePSK` | `func GeneratePSK() string` | Generates a 32-byte preshared key as a 44-char base64 string. |

### Key behavior & gotchas

| Concern | Behavior |
|---|---|
| Panic conditions | `GenerateKeyPair`/`GeneratePSK` panic **only** on `crypto/rand` failure. `DerivePublicKey` panics on invalid base64 or non-32-byte input. |
| WireGuard clamping | Applied before scalar multiplication in both `GenerateKeyPair` and `DerivePublicKey`: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`. A key read back from disk is re-clamped on derivation. |
| Encoding | All keys and PSKs are 44-character standard base64 (32 bytes + padding). |
| Untrusted input | Callers passing untrusted material to `DerivePublicKey` must validate first or `recover` (the unexported `tryDerivePublicKey` helper does the latter, returning `""` on any panic). |

## Validation

> Full rule descriptions and CLI exit-code semantics live in [Validation & Analysis](./validation.md).

| Symbol | Signature | Purpose |
|---|---|---|
| `ValidatePacketSizes` | `func ValidatePacketSizes(s1, s2, s3, s4 int, iPacketSizes []int, jmin, jmax int) error` | Enforces AWG 2.0 size invariants: four S-padded handshake sizes pairwise distinct, no I-packet equals a padded size, junk range excludes all padded + raw WG sizes. |
| `ValidateHeaderRange` | `func ValidateHeaderRange(r HeaderRange) error` | Rejects `Max < Min` or any inclusive range overlapping WG message type-ids `[1..4]`. |
| `ValidateServerConfig` | `func ValidateServerConfig(cfg *ServerConfig) []Finding` | Runs every rule (required fields, S-prefixes, junk range, header ranges); empty slice = clean. Never returns an error. |
| `Finding` | `type Finding struct { Message string; Detail string; Code string; Severity Severity; Location Location }` | One validation/analysis observation (JSON-tagged; reused by `analyze`). |
| `Finding.OneLine` | `func (f Finding) OneLine() string` | Canonical text form `[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>` (line/key omitted when empty). |
| `Severity` | `type Severity string` | Finding impact class. |
| `Severity` constants | `SeverityError`, `SeverityWarning`, `SeverityInfo` (`"error"` / `"warning"` / `"info"`) | The three severity levels. |
| `Location` | `type Location struct { File string; Key string; Line int }` | Where a finding originates within a config file. |
| `PacketSizeCollisionError` | `type PacketSizeCollisionError struct { Kind string; Pair string; Size int }` | One `ValidatePacketSizes` collision; `Kind` ∈ `{"s-pair", "i-packet", "junk-range"}`. |
| `(*PacketSizeCollisionError).Error` | `func (e *PacketSizeCollisionError) Error() string` | Formats as `packet size collision (<kind>): <pair> at <size> bytes`. |
| `ErrEmptyJunkRange` | `var ErrEmptyJunkRange = errors.New("junk range is empty (jmin > jmax)")` | Sentinel returned by `ValidatePacketSizes` when `jmin > jmax`. |
| WG message sizes | `WGInitiationSize`=148, `WGResponseSize`=92, `WGCookieReplySize`=64, `WGTransportSize`=32 | Canonical WireGuard message sizes (bytes), before AWG S-padding. Sourced from amneziawg-go's `device/noise-protocol.go`; used by validation to ensure junk ranges avoid real WG traffic. |

### `ValidatePacketSizes` return contract

| Outcome | Return value | How to detect |
|---|---|---|
| All invariants hold | `nil` | direct check |
| `jmin > jmax` | `ErrEmptyJunkRange` | `errors.Is(err, amnezigo.ErrEmptyJunkRange)` |
| First collision (S-pairs → I-packets → junk range) | `*PacketSizeCollisionError` | `var c *amnezigo.PacketSizeCollisionError; errors.As(err, &c)` |

## Analysis

`Analyze` is informational: it profiles a server config and emits only `warning`/`info` findings (never `error`), so the CLI `analyze` command always exits 0. See [Validation & Analysis](./validation.md).

| Symbol | Signature | Purpose |
|---|---|---|
| `Analyze` | `func Analyze(cfg ServerConfig, opts AnalyzeOptions) AnalysisReport` | Produces an `AnalysisReport`; I-packets are freshly generated from the config (not read from disk). Never returns an error. |
| `AnalyzeOptions` | `type AnalyzeOptions struct { Rand io.Reader; Protocol string; PeerName string; Samples int }` | Configures analysis: randomness source (`nil` ⇒ `crypto/rand`), protocol template, peer filter (empty ⇒ all), sample count. |
| `AnalysisReport` | `type AnalysisReport struct { Peers []PeerProfile; Findings []Finding; Ordering OrderingDesc; SampleNote string; Config ConfigInfo; Handshake HandshakeProfile; Headers HeaderProfile; Junk JunkProfile }` | Top-level output: config metadata, handshake/junk/header profiles, per-peer I-packet analysis, wire ordering, and heuristic findings. |
| `FormatText` | `func FormatText(report AnalysisReport) string` | Renders a report as human-readable multi-section text. |
| `FormatJSON` | `func FormatJSON(report AnalysisReport) (string, error)` | Renders a report as indented JSON (two spaces); wraps marshal errors with `%w`. |
| `ListProtocols` | `func ListProtocols() []string` | Returns alphabetically-sorted protocol template names (`dns`, `dtls`, `quic`, `random`, `rtp`, `sip`, `stun`). Use to validate user-supplied protocol names or enumerate available templates. |

### `AnalyzeOptions` fields

| Field | Default / behavior |
|---|---|
| `Rand` | `nil` ⇒ `crypto/rand`. Supply a deterministic reader for reproducible I-packets. |
| `Protocol` | Empty ⇒ `"random"` (`ProtocolRandom`). Accepted values: `random`, `quic`, `dns`, `dtls`, `stun`, `sip`, `rtp` — see [Protocol constants](#protocol-constants) below. |
| `PeerName` | Empty ⇒ analyze all peers. |
| `Samples` | `0` ⇒ snapshot-only (one `PeerSnapshot` per peer). `> 0` ⇒ distribution mode with `Stats` over N samples. |

### Protocol constants

The protocol template names are available as exported constants. Prefer these over bare string literals when setting `AnalyzeOptions.Protocol` or `PeerManifest.Protocol`.

| Constant | Value |
|---|---|
| `ProtocolQUIC` | `"quic"` |
| `ProtocolDNS` | `"dns"` |
| `ProtocolDTLS` | `"dtls"` |
| `ProtocolSTUN` | `"stun"` |
| `ProtocolSIP` | `"sip"` |
| `ProtocolRTP` | `"rtp"` |
| `ProtocolRandom` | `"random"` |

### Heuristic finding codes

`Analyze` populates `Finding.Code` with the following string values. **They are string codes, not exported Go constants** — match them by string value.

| Code | Severity | Meaning |
|---|---|---|
| `RISK001` | warning | Junk range contains a raw WG size. |
| `RISK002` | warning | A peer's I-packet cluster spans < 20 B. |
| `RISK003` | warning | `S4 < 8`. |
| `RISK004` | warning | Two padded sizes differ by < 5 B. |
| `RISK005` | info | A padded size is within ±4 B of a raw WG size. |
| `RISK006` | warning | Junk range width < 32 B. |
| `RISK007` | warning | An H-range width < 1,000,000. |
| `RISK008` | info | No peers defined. |
| `RISK009` | warning | All S-prefixes and junk params are zero (vanilla-WG shape). |

### Report sub-types

| Type | Purpose |
|---|---|
| `ConfigInfo` | Basic config metadata (`Protocol`, `MTU`, `ListenPort`, `PeerCount`). |
| `HandshakeProfile` | The four S-padded handshake sizes (`Init`, `Response`, `Cookie`, `Transport`). |
| `PaddedSize` | One padded size: `SPrefix`, `RawSize`, `Padded`. |
| `JunkProfile` | `Jc`, `Jmin`, `Jmax`, range `Width`. |
| `HeaderProfile` | The four H-ranges (`H1`–`H4`). |
| `HeaderRangeInfo` | `Min`, `Max`, `Width` for one header range. |
| `PeerProfile` | Per-peer I-packet analysis: `Name`, `Snapshot`, and `Distribution` (when `Samples > 0`). |
| `PeerSnapshot` | One generation of the five I-packet sizes (`I1`–`I5`). |
| `PeerDistrib` | `Stats` over N samples for `I1`–`I5` plus `Samples`. |
| `Stats` | `Mean`, `Min`, `Max`, `Median`. |
| `OrderingDesc` | `Steps` describing packet ordering on the wire per handshake. |

## Presets

> Preset values and selection guidance live in [Presets](./presets.md).

| Symbol | Signature | Purpose |
|---|---|---|
| `Preset` | `type Preset struct { Name string; Description string; DefaultProtocol string; H1, H2, H3, H4 HeaderRange; MTU int; S1, S2, S3, S4 int; Jc, Jmin, Jmax int }` | A named bundle of obfuscation parameters; each yields a config that passes `ValidatePacketSizes`. |
| `Preset.ToServerObfuscation` | `func (p Preset) ToServerObfuscation() ServerObfuscationConfig` | Converts a preset to a `ServerObfuscationConfig`. Carries `Jc`/`Jmin`/`Jmax`, `S1`–`S4`, `H1`–`H4` only — **not** `MTU`, `Description`, or `DefaultProtocol`. |
| `GetPreset` | `func GetPreset(name string) (Preset, error)` | Returns the named preset; on unknown names returns an error listing every available preset. |
| `ListPresets` | `func ListPresets() []Preset` | Returns a defensive copy of all seven built-in presets. |

The seven built-in presets: `lan-conservative`, `home-balanced`, `mobile-aggressive`, `stealth-paranoid`, `standard-1420`, `low-overhead`, `test-minimal`. There is no `preset` field in `Manifest`; to use a preset, copy its values into `ObfuscationManifest` (or evaluate it at the Jsonnet layer).

## Credentials

> Key-reuse model and recovery sources are documented in [Credentials & Key Reuse](./credentials.md).

| Symbol | Signature | Purpose |
|---|---|---|
| `PeerCredentials` | `type PeerCredentials struct { PrivateKey string; PublicKey string; PresharedKey string }` | Cryptographic material for one peer: base64 X25519 private key, derived public key, and a per-connection PSK. |
| `PersistedCredentials` | `type PersistedCredentials struct { Peers map[string]PeerCredentials; Server PeerCredentials }` | All credentials extracted from existing output configs; reused by `Generate` across runs. |
| `EmptyCredentials` | `func EmptyCredentials() *PersistedCredentials` | Returns a `PersistedCredentials` with an initialized `Peers` map and zero-value server credentials (first-run / `FullReset` path). |
| `LoadCredentials` | `func LoadCredentials(outputDir, serverPeerName string) (*PersistedCredentials, error)` | Reads existing output configs from `outputDir` and extracts persisted credentials. |

### `LoadCredentials` behavior

| Situation | Return |
|---|---|
| Output dir or server config absent (first run) | `EmptyCredentials()`, `nil` — **not** an error. |
| Real I/O or parse failure of a present file | `nil`, wrapped error. |
| Server config present | Server keypair from its `[Interface]`; each peer's `PublicKey`/`PresharedKey`/`Name` from its `[Peer]`; each peer's `PrivateKey` recovered from its own `outputDir/<peer>/awg0.conf` (a client secret). |

## Library gotchas

| Concern | Behavior |
|---|---|
| Non-atomic writes | `Generate` builds every config in memory first (all-or-nothing compute), but the disk-write pass uses `os.WriteFile` per file (mode `0600`) — a crash mid-pass can leave partially-written output. See [Gotchas](./gotchas.md). |
| Sorted peer iteration | Client peers are iterated alphabetically; when `PeerFilter` is set, only the named peers are built. Output ordering is deterministic. |
| Pointer nil ⇒ random | `ObfuscationManifest` uses `*int` / `*HeaderRange` so "set to 0" ≠ "unset". `nil` triggers random generation in `resolveObfuscation`. |
| `ServerPeerName` panics | Returns the sole server peer name but **panics** when count ≠ 1. Use `ServerPeer()` first, or call only on validated manifests. |
| Error wrapping | Failures are wrapped with `fmt.Errorf("<context>: %w", err)` (e.g. `resolve obfuscation: %w`, `load credentials: %w`, `build server config: %w`, `write file %s: %w`) — unwrap with `errors.Is` / `errors.As`. |
| `tryDerivePublicKey` | Unexported helper that `recover`s a `DerivePublicKey` panic and returns `""`; used when reloading persisted keys of unknown validity. |
| Version enforced | `LoadManifest` / `LoadManifestFromFile` reject any `version` other than `1`. |
| Protocol constants | `ProtocolQUIC`, `ProtocolDNS`, `ProtocolDTLS`, `ProtocolSTUN`, `ProtocolSIP`, `ProtocolRTP`, `ProtocolRandom` are the exported constants for `AnalyzeOptions.Protocol` and `PeerManifest.Protocol`. Prefer `amnezigo.ProtocolQUIC` etc. over bare string literals; use `ListProtocols()` to enumerate all valid values. See [Obfuscation](./obfuscation.md). |
| Output layout | Configs are written as `<OutputDir>/<peer>/awg0.conf`; the INI/metadata structure is documented in [Output Format](./output-format.md). |
