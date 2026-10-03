# Library Usage

> Go API reference for the `amnezigo` package — every exported symbol for embedding the declarative AmneziaWG generator in your own tooling.

## Table of Contents

- [Import & overview](#import--overview)
- [Generate pipeline](#generate-pipeline)
- [Manifest loading](#manifest-loading)
- [Manifest types & methods](#manifest-types--methods)
- [AWG version model](#awg-version-model)
- [U16Range](#u16range)
- [Crypto & keys](#crypto--keys)
- [Obfuscation generators](#obfuscation-generators)
- [CPS helpers](#cps-helpers)
- [Protocol constants & templates](#protocol-constants--templates)
- [Config types & writer/parser API](#config-types--writerparser-api)
- [Validation](#validation)
- [Analysis](#analysis)
- [Presets](#presets)
- [Credentials](#credentials)
- [VPN import links](#vpn-import-links)
- [Helpers & iptables](#helpers--iptables)
- [Library gotchas](#library-gotchas)
- [Related](#related)

---

## Import & overview

```go
import "github.com/Arsolitt/amnezigo"
```

The root package is named `amnezigo` (module `github.com/Arsolitt/amnezigo`, GPL-3.0-only, Go 1.26.1). It holds the entire business logic — manifest loading, the generate pipeline, validation, analysis, presets, credential reuse, key utilities, and the INI writer/parser. The `amnezigo` binary (see [CLI Reference](./cli-reference.md)) is a thin wrapper over `LoadManifest` → `Generate` → `ValidateServerConfig` → `Analyze`; the same calls are available to any Go program.

> **Note:** The package has no package-level doc comment and ships no `doc.go`, so `go doc github.com/Arsolitt/amnezigo` starts at the constant blocks. This page is the intended package overview.

There is no `Manager` type and no `init`/`add`/`edit`/`remove`/`export`/`list` surface — generation is driven by a single declarative manifest (see [Manifest Reference](./manifest-reference.md)). The root package has no dependency on the CLI (`cobra` is used only under `internal/cli`); its direct dependencies are `go-jsonnet` (manifest loading) and `golang.org/x/crypto` (key derivation), so it can be vendored into other tooling.

The canonical flow is two calls:

```go
manifest, err := amnezigo.LoadManifest("./my-project", nil)
result, err := amnezigo.Generate(manifest, amnezigo.GenerateOptions{OutputDir: "./my-project/output"})
```

## Generate pipeline

| Symbol | Signature | Since |
| --- | --- | --- |
| `GenerateOptions` | `type GenerateOptions struct { ProjectDir string; OutputDir string; JpathDirs []string; PeerFilter []string; DryRun bool; FullReset bool; VPNLinks bool }` | — |
| `GenerateResult` | `type GenerateResult struct { ServerPeer string; Files []FileOutput; ClientPeers []string; Findings []Finding }` | — |
| `FileOutput` | `type FileOutput struct { RelPath string; Content []byte }` | — |
| `Generate` | `func Generate(manifest Manifest, opts GenerateOptions) (GenerateResult, error)` | — |

`Generate` runs a nine-step pipeline: identify the server peer, load persisted credentials, resolve obfuscation, resolve peer credentials, build the server config, validate it into `Findings`, build client configs (alphabetical order, `PeerFilter` applied), collect `FileOutput`, and optionally write to disk.

`Generate` is a two-pass function: every config is computed in memory first, so a build failure leaves no files behind. The write pass itself is not atomic across files — each `<peer>/awg0.conf` (and optional `<peer>/amnezigo.vpn`) is written with `os.WriteFile` mode `0600` after `os.MkdirAll` mode `0750`, so a crash mid-pass can leave a partial output tree.

### `GenerateOptions` fields

| Field | Type | Effect |
| --- | --- | --- |
| `ProjectDir` | `string` | Stored but **never read** inside `Generate` — the manifest is loaded separately via `LoadManifest`. |
| `OutputDir` | `string` | Root of the output tree. Empty → `Generate` computes files in memory and writes nothing, even when `DryRun` is `false`. |
| `JpathDirs` | `[]string` | Stored but **never read** inside `Generate`; it belongs to manifest loading (`LoadManifest`). |
| `PeerFilter` | `[]string` | When non-empty, only the named client peers are built; the server is always built. Names that match no peer are silently ignored, and persisted credentials are still loaded for every peer. |
| `DryRun` | `bool` | When `true`, files are returned in `result.Files` but never written. |
| `FullReset` | `bool` | When `true`, persisted key material is discarded and fresh credentials are generated (including a new header-protection key). |
| `VPNLinks` | `bool` | When `true`, adds one `<peer>/amnezigo.vpn` file per client peer. See [VPN Import Links](./vpn-links.md). |

### `GenerateResult` fields

| Field | Type | Meaning |
| --- | --- | --- |
| `ServerPeer` | `string` | Name of the sole server peer. |
| `Files` | `[]FileOutput` | Planned files: the server config first, then client configs alphabetically; a `.vpn` file follows its client config when `VPNLinks` is set. `RelPath` is always `<peer>/awg0.conf` or `<peer>/amnezigo.vpn`, slash-joined relative to `OutputDir`. |
| `ClientPeers` | `[]string` | The client peer names that were built (after `PeerFilter`). |
| `Findings` | `[]Finding` | Findings from validating the generated server config in memory (`Location.File`/`Line` are empty and `Location.Key` may be set — see [gotchas](#library-gotchas)). |

### Error contract

`Generate` returns plain wrapped errors; phase failures are prefixed so `errors.Is`/`errors.As` and prefix matching both work.

| Error string | Raised when |
| --- | --- |
| `exactly one server peer required, found %d` | `Manifest.ServerPeer()` reports a count other than 1. |
| `load credentials: %w` | Reading a present output tree fails with an error other than "does not exist". |
| `resolve obfuscation: %w` | Version gates, malformed `U16Range` bounds, or the header-protection S floor fail. |
| `build server config: %w` | Server config construction fails. |
| `build client config for <peer>: %w` | A per-peer client build fails. |
| `create directory <dir>: %w` | Output directory creation fails mid-write. |
| `write file <path>: %w` | A file write fails mid-write. |

`Generate` never validates `Manifest.Version` (see [gotchas](#library-gotchas)) and never constructs a manifest itself.

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

    // 2. Generate configs in memory. DryRun keeps the tree untouched.
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
    for _, f := range result.Findings {
        fmt.Println(f.OneLine())
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

| Symbol | Signature | Since |
| --- | --- | --- |
| `LoadManifest` | `func LoadManifest(dir string, jpathDirs []string) (Manifest, error)` | — |
| `LoadManifestFromFile` | `func LoadManifestFromFile(path string, jpathDirs []string) (Manifest, error)` | — |

| Concern | Behavior |
| --- | --- |
| Discovery | `LoadManifest` checks `.amnezigo.jsonnet` **before** `amnezigo.json`; when both exist, the Jsonnet file wins. Neither present → error. |
| Explicit path | `LoadManifestFromFile` reads the given path; a `.jsonnet` suffix evaluates through the Jsonnet VM, anything else is parsed as plain JSON. |
| `jpathDirs == nil` | Defaults to `[dir/lib]` for `LoadManifest` and `[parentDir/lib]` for `LoadManifestFromFile`. |
| Jsonnet evaluation | Jsonnet → JSON string → parsed as `Manifest`; the `version` field is still required. See [Jsonnet](./jsonnet.md). |
| Schema version | Both loaders reject a `version` other than `1`: missing or zero → `<path>: missing or zero version field`, otherwise `<path>: unsupported schema version %d (expected 1)`. |
| Missing file | Unlike `LoadCredentials`, a missing manifest file is an error — there is no empty-manifest fallback. |

> **Warning:** The schema-version check lives only in the loaders. A `Manifest` you construct in Go with `Version: 0` is accepted by `Generate`; only file loading enforces version 1.

## Manifest types & methods

| Symbol | Signature | Purpose |
| --- | --- | --- |
| `Manifest` | `type Manifest struct { Peers map[string]PeerManifest; Obfuscation ObfuscationManifest; Network NetworkConfig; Version int }` | Top-level declarative config; peer names are map keys. |
| `NetworkConfig` | `type NetworkConfig struct { DNS []string; MTU int }` | Global MTU and DNS applied to every peer. |
| `ObfuscationManifest` | see below | Obfuscation profile, including the AWG 3.x pointer fields. |
| `PeerManifest` | `type PeerManifest struct { Keepalive *int; Address string; TunName string; MainIface string; Endpoint string; Protocol string; ListenPort int; DisplayName string }` | One peer; `DisplayName` feeds the `vpn://` envelope description. |
| `HeaderRange` | `type HeaderRange struct { Min uint32; Max uint32 }` | Inclusive H1–H4 range (JSON `min`/`max`). |
| `(*Manifest) PeerNames` | `func (m *Manifest) PeerNames() []string` | All peer names sorted alphabetically. |
| `(*Manifest) ServerPeer` | `func (m *Manifest) ServerPeer() (string, int)` | Server peer name and count; check the count before using the name (with count > 1 the name is arbitrary map-order). |
| `(*Manifest) ServerPeerName` | `func (m *Manifest) ServerPeerName() string` | Sole server peer name; **panics** unless count == 1. Use `ServerPeer()` in validation paths. |
| `(*ObfuscationManifest) HasAnyValue` | `func (o *ObfuscationManifest) HasAnyValue() bool` | `true` when any S/H/J pointer is non-nil; `false` means fully random generation. |
| `(*ObfuscationManifest) ToSharedObfuscation` | `func (o *ObfuscationManifest) ToSharedObfuscation() ServerObfuscationConfig` | Copies explicit values; nil fields stay zero as the signal for random generation. |
| `(*PeerManifest) IsServer` | `func (p *PeerManifest) IsServer() bool` | `true` when `Endpoint != ""` **and** `ListenPort != 0`. |

```go
type ObfuscationManifest struct {
    S1, S2, S3, S4 *int
    H1, H2, H3, H4 *HeaderRange
    Jc, Jmin, Jmax *int
    Protocol       string

    // AWG 3.x additions.
    AWGVersion           string    // JSON: awg_version
    HeaderProtection     *bool
    ContentPadding       *U16Range
    RekeyAfterTime       *U16Range
    RekeyTimeout         *U16Range
    RejectAfterTime      *U16Range
    KeepaliveTimeout     *U16Range
    MaxHandshakeAttempts *U16Range
    RandomTrailers       *bool
    DisableCookies       *bool
}
```

Pointer semantics distinguish "explicitly set" from "unset": a nil field selects the version default or triggers generation, while an explicit value is used verbatim (with two documented exceptions — a pinned `s1: 0` is replaced by a generated value, and 3.x range fields use `U16Range{0, 0}` to disable their key). `AWGVersion` accepts the manifest strings `"2.0"` / `"3.0"` / `"3.1"`; the Go enum and its parser are described in [AWG version model](#awg-version-model).

## AWG version model

Added with the AWG 3.x work (commit `df0485a`).

```go
type AWGVersion int

const (
    AWG20 AWGVersion = iota + 1 // AmneziaWG 2.0
    AWG30                       // adds HeaderProtectionKey, ContentPaddingAddition, five timer ranges
    AWG31                       // adds RandomTrailers, DisableCookies
)

const DefaultAWGVersion = AWG31
```

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `ParseAWGVersion` | `func ParseAWGVersion(s string) (AWGVersion, error)` | `""` (unset) → `DefaultAWGVersion`; `"2.0"`/`"3.0"`/`"3.1"` → the enum; anything else → `unsupported awg_version %q (expected "2.0", "3.0", or "3.1")`. |
| `(AWGVersion) String` | `func (v AWGVersion) String() string` | `"2.0"`, `"3.0"`, `"3.1"`; unknown values render `AWGVersion(%d)`. |

Each generation is a superset of the previous one, and an engine build rejects INI keys it does not know, so a key must never be emitted for a lower target version. `Generate` enforces this: a manifest that sets a field newer than its `awg_version` fails with `obfuscation.<field> requires awg_version <version> or later (got "<got>")`. See [Transport Protection](./transport-protection.md).

## U16Range

Added with the AWG 3.x work.

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `U16Range` | `type U16Range struct { Min uint16; Max uint16 }` (JSON `min` / `max`) | Inclusive uint16 range for `ContentPaddingAddition` and the five timer parameters. |
| `(U16Range) IsZero` | `func (r U16Range) IsZero() bool` | `true` iff `Min == 0 && Max == 0`. |
| `(U16Range) String` | `func (r U16Range) String() string` | `"N"` when `Min == Max`, otherwise `"Min-Max"` — the engine's INI notation. |

The zero value `{0, 0}` means unset/disabled: the key is omitted from the emitted config and the engine keeps its default. That is distinct from an omitted manifest pointer, which selects the version default. See [Manifest Reference](./manifest-reference.md) for the field defaults.

## Crypto & keys

| Symbol | Signature | Purpose | Since |
| --- | --- | --- | --- |
| `GenerateKeyPair` | `func GenerateKeyPair() (string, string)` | X25519 key pair `(privateKey, publicKey)`, both 44-char base64 (32 bytes + padding). | — |
| `DerivePublicKey` | `func DerivePublicKey(privateKey string) string` | Re-derives and clamps the 44-char base64 public key from a base64 private key. | — |
| `GeneratePSK` | `func GeneratePSK() string` | 32-byte preshared key as a 44-char base64 string. | — |
| `GenerateHeaderProtectionKey` | `func GenerateHeaderProtectionKey() string` | 32 `crypto/rand` bytes as a 44-char base64 string. | 3.x |

| Concern | Behavior |
| --- | --- |
| Panic conditions | `GenerateKeyPair` and `GeneratePSK` panic only on `crypto/rand` failure. `DerivePublicKey` panics on malformed base64 or a wrong-length key — validate untrusted input first. |
| WireGuard clamping | Applied before scalar multiplication in `GenerateKeyPair` and `DerivePublicKey`: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`. |
| Header-protection key | A symmetric ChaCha20 key (`device.HeaderCipherKey`), **not** a Curve25519 scalar — it must not be clamped. |
| Encoding | All four helpers return standard 44-character base64 (32 bytes + padding). |

See [Credentials & Key Reuse](./credentials.md) for how these keys are persisted and recovered.

## Obfuscation generators

| Symbol | Signature | Behavior | Since |
| --- | --- | --- | --- |
| `GenerateSPrefixes` | `func GenerateSPrefixes(minS int) SPrefixes` | Draws S1–S3 in `[minS, 64]` and S4 in `[minS, 32]`, retrying until the four padded sizes are pairwise distinct; **panics** after 1000 attempts. `minS` is new — the old no-argument signature is a source-level breaking change. | 3.x |
| `GenerateSPrefixesWithS1` | `func GenerateSPrefixesWithS1(minS, fixedS1 int) SPrefixes` | Keeps a caller-supplied `fixedS1` and draws S2–S4 like above, requiring all six padded pairs to be distinct; panics after 1000 attempts. The fixed S1 is **not** validated. | 3.x |
| `GenerateUniformSPrefixes` | `func GenerateUniformSPrefixes(minS int) SPrefixes` | Draws one value in `[minS, 32]` and returns it as `S1 == S2 == S3 == S4`. No retry loop is needed because the padded sizes stay pairwise distinct for any S. Use under AWG 3.1 random trailers, where the receiver classifies packets by size. | 3.x |
| `SPrefixes` | `type SPrefixes struct { S1, S2, S3, S4 int }` | S-prefix bundle returned by the three generators. | — |
| `GenerateJunkParams` | `func GenerateJunkParams() JunkParams` | Legacy generator without collision avoidance: `Jc` in `[0, 10]`, `Jmin`/`Jmax` in `[64, 1024]` with `Jmin < Jmax`. Prefer the collision-aware variant. | — |
| `GenerateJunkParamsWithForbidden` | `func GenerateJunkParamsWithForbidden(forbiddenSizes [4]int) (JunkParams, error)` | Draws junk parameters whose range excludes every given size (typically the four padded handshake sizes) plus the four raw WG message sizes (148, 92, 64, 32, always checked); returns an error when the retry budget is exhausted. | — |
| `JunkParams` | `type JunkParams struct { Jc, Jmin, Jmax int }` | Junk parameter bundle. | — |
| `GenerateHeaderRanges` | `func GenerateHeaderRanges() [4]HeaderRange` | Four sorted, non-overlapping H1–H4 ranges generated above the WG type-id window `[1..4]`; panics after 1000 attempts. | — |
| `GenerateConfig` | `func GenerateConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig` | Full client obfuscation bundle including I1–I5, satisfying the size-classification invariant; **panics** if the generated sizes violate `ValidatePacketSizes`. | — |
| `GenerateServerConfig` | `func GenerateServerConfig(_, s1, jc int) ServerObfuscationConfig` | Server obfuscation without I1–I5. The first parameter is unused (historically the MTU); panics when junk-range generation exhausts its retry budget. | — |
| `GenerateCPS` | `func GenerateCPS(protocol string, mtu, s1, _ int) (string, string, string, string, string)` | Returns I1–I5 for a protocol template. The fourth parameter is unused and the forbidden set is empty, so there is **no** size-collision avoidance — prefer `GenerateConfig` when collisions matter. | — |

`minS` is the inclusive lower bound for every generated S value: use `0` for the AWG 2.0 path and `12` (the ChaCha20 header-cipher nonce size) whenever header protection is active, because the engine rejects `S < 12` in that mode. The pipeline applies the same floor via `checkSPrefixFloor`, and `GenerateSPrefixesWithS1` does not check a caller-supplied `fixedS1` against it.

Under header protection with every H field unset, the pipeline writes the constant `1-1`/`2-2`/`3-3`/`4-4` ranges instead of calling `GenerateHeaderRanges`, because the 4-byte message type is encrypted. Uniform S is selected only when random trailers are on **and** all four S values are unset in the manifest.

## CPS helpers

| Symbol | Signature | Since |
| --- | --- | --- |
| `BuildCPSTag` | `func BuildCPSTag(tagType, value string) string` | — |
| `BuildCPS` | `func BuildCPS(tags []string) string` | — |
| `CPSLength` | `func CPSLength(cps string) int` | — |
| `CPSConfig` | `type CPSConfig struct { I1, I2, I3, I4, I5 string }` | — |
| `TagSpec` | `type TagSpec struct { Type, Value string }` | — |
| `I1I5Template` | `type I1I5Template struct { I1, I2, I3, I4, I5 []TagSpec }` | — |

### Supported `BuildCPSTag` types

| `tagType` | Result | Notes |
| --- | --- | --- |
| `"b"` | `<b 0x...>` | Hex bytes; a missing `0x` prefix is added automatically. |
| `"r"` | `<r N>` | N random bytes. |
| `"rc"` | `<rc N>` | N random letters from `[a-zA-Z]` (52 characters, no digits). |
| `"rd"` | `<rd N>` | N random digits. |
| `"t"` | `<t>` | 4-byte big-endian timestamp; `value` is ignored. |
| `"d"` | `<d>` | Passthrough marker; `value` is ignored; contributes 0 bytes at generation time and requires AWG 2.0 userspace. |

Any other type — including the legacy kernel-only `"c"` counter tag — returns the empty string sentinel. `BuildCPS` simply concatenates the tag strings it is given.

`CPSLength` returns the on-wire byte length with the same accounting as generation-time validation: hex length / 2 for `<b>`, N for `<r>`/`<rc>`/`<rd>`, 4 bytes per `<t>`, and 0 for `<d>` and unknown tags.

### Padded sizes & WG constants

| Symbol | Signature / value | Behavior |
| --- | --- | --- |
| `PaddedSizes` | `func PaddedSizes(s1, s2, s3, s4 int) [4]int` | `[init, response, cookie, transport]` = `[s1+148, s2+92, s3+64, s4+32]` — note the order is packet classes, not S1–S4 semantics. |
| `WGInitiationSize` | `= 148` | Raw WireGuard initiation size before S-padding. |
| `WGResponseSize` | `= 92` | Raw response size. |
| `WGCookieReplySize` | `= 64` | Raw cookie-reply size. |
| `WGTransportSize` | `= 32` | Raw transport size. |

The four constants come from `amneziawg-go`'s `device/noise-protocol.go`. See [Obfuscation](./obfuscation.md) for the CPS grammar and the size-classification invariant.

## Protocol constants & templates

| Constant | Value |
| --- | --- |
| `ProtocolQUIC` | `"quic"` |
| `ProtocolDNS` | `"dns"` |
| `ProtocolDTLS` | `"dtls"` |
| `ProtocolSTUN` | `"stun"` |
| `ProtocolSIP` | `"sip"` |
| `ProtocolRTP` | `"rtp"` |
| `ProtocolRandom` | `"random"` |

`ListProtocols()` (`func ListProtocols() []string`) returns all seven names sorted alphabetically: `dns`, `dtls`, `quic`, `random`, `rtp`, `sip`, `stun`. Use it to validate user-supplied protocol names. Note that `Analyze` treats an unknown name as a named template (it never rejects it), and that an empty protocol means QUIC during `Generate` but random during `Analyze`.

| Constructor | Mimics |
| --- | --- |
| `QUICTemplate()` | QUIC Initial long-header packet. |
| `DNSTemplate()` | DNS query. |
| `DTLSTemplate()` | DTLS 1.2 ClientHello. |
| `STUNTemplate()` | STUN binding request. |
| `SIPTemplate()` | SIP OPTIONS request (ASCII; literal fragments are `<b>` hex, variable tokens use `<rc>`/`<rd>` only). |
| `RTPTemplate()` | RTP media packet (RFC 3550 fixed header). |

Each constructor returns an `I1I5Template` whose intervals are `[]TagSpec` type/value pairs; I5 is empty for the named templates.

## Config types & writer/parser API

| Type | Definition | Purpose |
| --- | --- | --- |
| `ServerConfig` | `struct { Peers []PeerConfig; Interface InterfaceConfig; Obfuscation ServerObfuscationConfig }` | Full server config. |
| `InterfaceConfig` | `struct { TunName, DNS, Address, PostUp, PostDown, MainIface, EndpointV6, PrivateKey, PublicKey, EndpointV4 string; MTU, ListenPort, PersistentKeepalive int; ClientToClient bool }` | `[Interface]` section. |
| `PeerConfig` | `struct { CreatedAt time.Time; ClientObfuscation *ClientObfuscationConfig; Name, PrivateKey, PublicKey, PresharedKey, AllowedIPs string }` | `[Peer]` section; `ClientObfuscation` is nil in parsed server configs. |
| `ServerObfuscationConfig` | `struct { Jc, Jmin, Jmax, S1..S4 int; H1..H4 HeaderRange; Version AWGVersion; HeaderProtectionKey string; ContentPadding, RekeyAfterTime, RekeyTimeout, RejectAfterTime, KeepaliveTimeout, MaxHandshakeAttempts U16Range; RandomTrailers, DisableCookies bool }` | Server obfuscation; a **zero** `Version` disables emission of the whole 3.x block. |
| `ClientObfuscationConfig` | `struct { I1, I2, I3, I4, I5 string; ServerObfuscationConfig }` | Client obfuscation = server block plus I1–I5. |
| `ClientConfig` | `struct { Peer ClientPeerConfig; Interface ClientInterfaceConfig }` | Full client config. |
| `ClientInterfaceConfig` | `struct { PrivateKey, Address, DNS string; Obfuscation ClientObfuscationConfig; MTU int }` | Client `[Interface]`. |
| `ClientPeerConfig` | `struct { PublicKey, PresharedKey, Endpoint, AllowedIPs string; PersistentKeepalive int }` | Client `[Peer]`. |

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `WriteServerConfig` | `func WriteServerConfig(w io.Writer, cfg ServerConfig) error` | Writes the server INI in canonical key order, including version-gated 3.x keys and `#_` metadata. |
| `WriteClientConfig` | `func WriteClientConfig(w io.Writer, cfg ClientConfig) error` | Writes the client INI including I1–I5 and an always-present `PersistentKeepalive` line. |
| `SaveServerConfig` | `func SaveServerConfig(path string, cfg ServerConfig) error` | Atomic write: `path + ".tmp"` followed by a rename. |
| `ParseServerConfig` | `func ParseServerConfig(r io.Reader) (ServerConfig, error)` | Back-compat shim over `ParseServerConfigWithOptions` with default options; never returns warnings. |
| `ParseServerConfigWithOptions` | `func ParseServerConfigWithOptions(r io.Reader, opts ParseOptions) (ServerConfig, []ParseWarning, error)` | Full parser. In non-strict mode the warning slice is always nil; malformed 3.x values (bad `HeaderProtectionKey`, inverted H range, malformed uint16 range, bad bool) are errors. |
| `ParseOptions` | `type ParseOptions struct { Strict bool }` | `Strict` collects non-fatal anomalies — unknown INI keys (`KEY001`) and raw `<c>` literals (`CPS001`) — as warnings; structural H-range checks stay errors. |
| `ParseWarning` | `type ParseWarning struct { Message, Key, Code string; Line int }` | One non-fatal strict-parse observation. |
| `LoadServerConfig` | `func LoadServerConfig(path string) (ServerConfig, error)` | Opens a file and parses it non-strictly. |

> **Warning:** A hand-built `ServerObfuscationConfig` with the zero `AWGVersion` emits exactly the AWG 2.0 key set, because the writer returns early below `AWG30`. Set `Version` explicitly when writing 3.x configs by hand. See [Output Format](./output-format.md).

## Validation

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `ValidatePacketSizes` | `func ValidatePacketSizes(s1, s2, s3, s4 int, iPacketSizes []int, jmin, jmax int) error` | Enforces the size-classification invariant: four S-padded handshake sizes pairwise distinct, no I-packet size equal to a padded size, and no padded or raw WG size inside `[jmin..jmax]`. Checks run in that order. |
| `ValidateHeaderRange` | `func ValidateHeaderRange(r HeaderRange) error` | Rejects `Max < Min` and any inclusive range intersecting the WG type-ids `[1..4]`. |
| `ValidateServerConfig` | `func ValidateServerConfig(cfg *ServerConfig) []Finding` | Runs every rule and returns all findings; an empty slice means the config is clean. Never returns an error. |
| `ErrEmptyJunkRange` | `var ErrEmptyJunkRange = errors.New("junk range is empty (jmin > jmax)")` | Structural input error, distinct from a collision. |
| `PacketSizeCollisionError` | `type PacketSizeCollisionError struct { Kind, Pair string; Size int }` | One collision; `Kind` ∈ `{"s-pair", "i-packet", "junk-range"}`. |
| `(*PacketSizeCollisionError) Error` | `func (e *PacketSizeCollisionError) Error() string` | `packet size collision (<kind>): <pair> at <size> bytes`. |

### `ValidatePacketSizes` return contract

| Outcome | Return value | How to detect |
| --- | --- | --- |
| All invariants hold | `nil` | direct check |
| `jmin > jmax` | `ErrEmptyJunkRange` | `errors.Is(err, amnezigo.ErrEmptyJunkRange)` |
| First collision (S-pairs → I-packets → junk range) | `*PacketSizeCollisionError` | `var c *amnezigo.PacketSizeCollisionError; errors.As(err, &c)` |

### Finding types

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `Finding` | `type Finding struct { Message string; Detail string; Code string; Severity Severity; Location Location }` | One observation; JSON tags `message`, `detail` (`omitempty`), `code`, `severity`, `location` (`omitzero`). Reused by `analyze`. |
| `Finding.OneLine` | `func (f Finding) OneLine() string` | `[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>`; line and key segments are omitted when empty. |
| `Severity` | `type Severity string` | `SeverityError` `"error"`, `SeverityWarning` `"warning"`, `SeverityInfo` `"info"`. |
| `Location` | `type Location struct { File string; Key string; Line int }` | Where a finding originates; JSON omits empty values. |

The finding codes produced by `ValidateServerConfig` — including the AWG 3.x codes `HPK001`, `HPK003`, `TRL001`, and `TRM001` — are catalogued in [Validation & Analysis](./validation.md). `ValidateServerConfig` calls `ValidatePacketSizes` with a nil I-packet slice, so an I-packet collision (`PSC003`) is reachable only by calling `ValidatePacketSizes` directly.

## Analysis

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `Analyze` | `func Analyze(cfg ServerConfig, opts AnalyzeOptions) AnalysisReport` | Profiles a server config and returns heuristic `RISK001`–`RISK009` findings. I-packets are freshly generated from the config parameters, not read from disk, so the numbers may differ from the config's stored I1–I5. Never returns an error and only emits `warning`/`info` findings. |
| `AnalyzeOptions` | `type AnalyzeOptions struct { Rand io.Reader; Protocol string; PeerName string; Samples int }` | See below. |
| `AnalysisReport` | `type AnalysisReport struct { Peers []PeerProfile; Findings []Finding; Ordering OrderingDesc; SampleNote string; Config ConfigInfo; Handshake HandshakeProfile; Headers HeaderProfile; Junk JunkProfile }` | Top-level report; JSON keys `peers`, `findings`, `ordering`, `sample_note`, `config`, `handshake`, `headers`, `junk`. |
| `FormatText` | `func FormatText(report AnalysisReport) string` | Human-readable multi-section text report. |
| `FormatJSON` | `func FormatJSON(report AnalysisReport) (string, error)` | Two-space-indented JSON; wraps marshal errors. |

| `AnalyzeOptions` field | Behavior |
| --- | --- |
| `Rand` | Declared as "the randomness source for CPS generation", but **never read** by the package: I-packet generation always uses `crypto/rand`. Passing a seeded reader changes nothing and analysis output stays non-deterministic. |
| `Protocol` | Empty → `"random"`. Unknown names are not rejected; they fall through to a randomly chosen named template. |
| `PeerName` | Empty → all peers. |
| `Samples` | `0` → snapshot only (one `PeerSnapshot` per peer); `> 0` → distribution mode with `Stats` over N samples. |

### Report sub-types

| Type | Purpose |
| --- | --- |
| `ConfigInfo` | Protocol, MTU, listen port, peer count. |
| `HandshakeProfile` | The four padded handshake sizes as `PaddedSize` values. |
| `PaddedSize` | `SPrefix`, `RawSize`, `Padded` for one packet class. |
| `JunkProfile` | `Jc`, `Jmin`, `Jmax`, range `Width`. |
| `HeaderProfile` | `H1`–`H4` `HeaderRangeInfo` values. |
| `HeaderRangeInfo` | `Min`, `Max`, `Width`. |
| `PeerProfile` | `Name`, `Snapshot`, and optional `Distribution`. |
| `PeerSnapshot` | One generation of I1–I5 sizes. |
| `PeerDistrib` | Per-interval `Stats` plus `Samples`. |
| `Stats` | `Mean`, `Min`, `Max`, `Median`. |
| `OrderingDesc` | `Steps` describing packet ordering per handshake. |

## Presets

| Symbol | Signature | Since |
| --- | --- | --- |
| `Preset` | see below | — |
| `Preset.ToServerObfuscation` | `func (p Preset) ToServerObfuscation() ServerObfuscationConfig` | — |
| `GetPreset` | `func GetPreset(name string) (Preset, error)` | — |
| `ListPresets` | `func ListPresets() []Preset` | — |

```go
type Preset struct {
    Name            string
    Description     string
    DefaultProtocol string
    H1, H2, H3, H4  HeaderRange
    MTU             int
    S1, S2, S3, S4  int
    Jc, Jmin, Jmax  int

    // AWG 3.x additions.
    ContentPadding U16Range
    RandomTrailers bool
    DisableCookies bool
}
```

`ToServerObfuscation` copies `Jc`/`Jmin`/`Jmax`, `S1`–`S4`, `H1`–`H4`, and the three AWG 3.x fields, and hardcodes `Version: AWG31`. It does not carry `Name`, `Description`, `DefaultProtocol`, or `MTU`, and it leaves `HeaderProtectionKey` empty on purpose because the pipeline supplies the key separately. The writer omits an empty header-protection key, so a `ServerObfuscationConfig` obtained from a preset and written directly has no `HeaderProtectionKey` line.

`GetPreset` returns `unknown preset %q; available presets: %v` (the registry slice is printed Go-style) for unknown names; `ListPresets` returns a copy of the registry. The seven built-in presets are `lan-conservative`, `home-balanced`, `mobile-aggressive`, `stealth-paranoid`, `standard-1420`, `low-overhead`, and `test-minimal`. There is no `preset` field in `Manifest` and no `Preset.Version` field — presets are AWG 3.1 data by construction, so resolve them at the Jsonnet layer or copy their values into `ObfuscationManifest`. See [Presets](./presets.md).

## Credentials

| Symbol | Signature | Since |
| --- | --- | --- |
| `PeerCredentials` | `type PeerCredentials struct { PrivateKey string; PublicKey string; PresharedKey string }` | — |
| `PersistedCredentials` | `type PersistedCredentials struct { Peers map[string]PeerCredentials; Server PeerCredentials; HeaderProtectionKey string }` | `HeaderProtectionKey` 3.x |
| `EmptyCredentials` | `func EmptyCredentials() *PersistedCredentials` | — |
| `LoadCredentials` | `func LoadCredentials(outputDir, serverPeerName string) (*PersistedCredentials, error)` | — |

| Situation | `LoadCredentials` return |
| --- | --- |
| Output directory or server config absent (first run) | Empty credentials, `nil` error — not an error. |
| Real I/O or parse failure of a present file | `nil`, wrapped error. |
| Server config present | Server key pair **and** `HeaderProtectionKey` from its `[Interface]`; each peer's public key, PSK, and name from its `[Peer]` sections; each peer's private key from its own `<outputDir>/<peer>/awg0.conf`, because the server config never carries a client's private key. |
| Server config missing, client directories present | Peer material is scanned from the client directories, but `HeaderProtectionKey` is not recovered — the next `Generate` generates a fresh one, and existing clients stop handshaking. |

The output tree is the credential store: keys exist only inside the generated INIs, so deleting `output/` silently rotates everything on the next run. `Generate` loads credentials from `opts.OutputDir`; when `OutputDir` is empty it always starts from `EmptyCredentials()`, which means keys are regenerated on every call. Credential reuse is asymmetric — persisted keys survive, but every obfuscation parameter the manifest leaves nil is re-drawn from `crypto/rand` on each call. See [Credentials & Key Reuse](./credentials.md).

## VPN import links

| Symbol | Signature |
| --- | --- |
| `EncodeVPNLink` | `func EncodeVPNLink(clientINI []byte, endpoint string, listenPort int, dns []string, description string) string` |

`EncodeVPNLink` wraps a client AWG INI config into an AmneziaVPN-app-importable `vpn://` link: a zlib-compressed JSON envelope (Qt-style `qCompress` with a 4-byte big-endian length prefix) encoded as base64url without padding. The `last_config` JSON carries the verbatim INI in `config` plus the structured fields the app's `configWireguard()` connect path reads.

- `description` is copied verbatim into the envelope's `description` field (`json:"description,omitempty"`) — there is **no** hostname fallback. The pipeline passes the peer's `display_name`.
- Under AWG 3.x, `HeaderProtectionKey`, `ContentPaddingAddition`, `RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts`, `RandomTrailers`, and `DisableCookies` are copied into `last_config` under their INI key names; pre-3.x links simply omit them.
- The link is importable by the AmneziaVPN app, not by the standalone AmneziaWG app.

See [VPN Import Links](./vpn-links.md) for the envelope layout.

## Helpers & iptables

| Symbol | Signature | Behavior |
| --- | --- | --- |
| `IsValidIPAddr` | `func IsValidIPAddr(ipaddr string) bool` | `true` for a parseable CIDR string. |
| `ExtractSubnet` | `func ExtractSubnet(ipaddr string) string` | Network address plus prefix length; returns the input unchanged when it is not CIDR. |
| `GenerateRandomPort` | `func GenerateRandomPort() (int, error)` | `crypto/rand` port in `[10000, 65535]`. |
| `DetectMainInterface` | `func DetectMainInterface() string` | First non-loopback, up interface with at least one address; `""` when none. |
| `FindNextAvailableIP` | `func FindNextAvailableIP(serverAddress string, existingIPs []string) (string, error)` | Scans host parts `.2`–`.254`, skipping `existingIPs`; returns `""` with a nil error when nothing is free. |
| `GeneratePostUp` | `func GeneratePostUp(tunName, mainIface, subnet string, clientToClient bool) string` | IPv4 `iptables` setup rules joined with `"; "`. |
| `GeneratePostDown` | `func GeneratePostDown(tunName, mainIface, subnet string, clientToClient bool) string` | The same IPv4 rules with `-D` instead of `-A`. |
| `GeneratePostUp6` | `func GeneratePostUp6(tunName, mainIface, subnet string, clientToClient bool) string` | IPv6 `ip6tables` setup rules. |
| `GeneratePostDown6` | `func GeneratePostDown6(tunName, mainIface, subnet string, clientToClient bool) string` | IPv6 teardown rules with `-D`. |

## Library gotchas

| Concern | Behavior |
| --- | --- |
| Asymmetric credential reuse | `Generate` reuses key pairs, PSKs, and the header-protection key from `OutputDir`, but re-draws every obfuscation parameter left nil (S1–S4, H ranges without header protection, Jc/Jmin/Jmax) from `crypto/rand` on each call. Pin the fields you care about for stable output. |
| `AnalyzeOptions.Rand` is dead | The field is declared but never read; analysis always uses `crypto/rand`, so seeded readers do not make reports reproducible. |
| `minS` is a breaking change | `GenerateSPrefixes` and `GenerateSPrefixesWithS1` gained a leading `minS` parameter in the 3.x work — callers of the old signatures must be updated. Pass `0` for 2.0 and `12` under header protection. |
| `DerivePublicKey` panics | Malformed base64 or a wrong-length key panics; validate untrusted material first or wrap the call in `recover`. |
| `GenerateSPrefixesWithS1` trusts its S1 | A `fixedS1` below the header-protection floor passes generation and is only rejected later by the pipeline or `ValidateServerConfig`. |
| Preset → config loses the HP key | `Preset.ToServerObfuscation` leaves `HeaderProtectionKey` empty, and the writer omits an empty key, so writing that config directly yields a 3.1 config without header protection. |
| Zero `Version` = AWG 2.0 output | A hand-built `ServerObfuscationConfig` with the zero `AWGVersion` emits exactly the 2.0 key set. Set `Version` to emit 3.x keys. |
| No writes without `OutputDir` | `Generate` only writes when `DryRun == false` **and** `OutputDir != ""`; with an empty `OutputDir` it silently returns the files in memory — and never reuses credentials. |
| `ProjectDir` / `JpathDirs` are inert | `GenerateOptions` stores both, but `Generate` never reads them; manifest discovery and `jpath` belong to `LoadManifest`. |
| `PaddedSizes` ordering | It takes four separate ints (not an `SPrefixes` value) and returns init/response/cookie/transport — `148+S1`, `92+S2`, `64+S3`, `32+S4`. |
| `GenerateCPS` has no collision avoidance | It forwards an empty forbidden set and ignores its fourth parameter, so I-packet sizes may collide with padded handshake sizes; use `GenerateConfig` when that matters. |
| Missing manifest vs missing output | `LoadManifest` errors when no manifest file exists, while `LoadCredentials` returns empty credentials for a missing output directory — the two "first run" paths differ. |
| `ServerPeerName` panics | Prefer `ServerPeer()` in validation code and reserve `ServerPeerName()` for paths where the manifest has already been checked. |
| `Manifest.Version` is loader-only | Only `LoadManifest`/`LoadManifestFromFile` enforce version 1; `Generate` accepts a hand-built `Manifest` with any `Version`. |
| Generate findings have no location | Findings added to `GenerateResult.Findings` come from validating an in-memory config, so `Location.File`/`Line` are empty and `Finding.OneLine()` renders a double space. `validate` re-stamps the file path. |
| Manual writes are not atomic | `Generate`'s write pass is per-file `os.WriteFile` (mode `0600`); use `SaveServerConfig` for atomic single-file writes. |
| No package doc comment | `godoc` opens at the constants; treat this page as the package overview. |

See [Gotchas](./gotchas.md) for project-wide pitfalls that also apply to library callers.

## Related

- [Overview](./overview.md) — what amnezigo is and how the declarative model fits together.
- [Manifest Reference](./manifest-reference.md) — every manifest field, including the AWG 3.x knobs used by `ObfuscationManifest`.
- [Credentials & Key Reuse](./credentials.md) — how `Generate` persists and recovers keys between runs.
- [Validation & Analysis](./validation.md) — finding codes and the rules behind `ValidateServerConfig` and `Analyze`.
- [Presets](./presets.md) — the seven built-in preset bundles and their AWG 3.1 values.
- [VPN Import Links](./vpn-links.md) — the `vpn://` envelope produced by `EncodeVPNLink`.
