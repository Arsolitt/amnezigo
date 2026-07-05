# Credentials & Key Reuse

> How amnezigo generates, stores, and reuses X25519 keys and preshared keys across `generate` runs, and what is regenerated every run.

## Table of Contents

- [Key reuse model](#key-reuse-model)
- [Where keys are stored](#where-keys-are-stored)
- [`LoadCredentials`](#loadcredentials)
- [`--full-reset` behavior](#--full-reset-behavior)
- [Key generation API](#key-generation-api)
- [Recovery flow](#recovery-flow)
- [Gotchas](#gotchas)
- [Related](#related)

---

## Key reuse model

amnezigo is idempotent across runs: a plain `amnezigo generate` reuses the cryptographic keys produced by the previous run so client configs do not churn. Only the per-client AmneziaWG obfuscation strings (CPS `I1`–`I5`) are regenerated every run — they are not credentials and are never persisted as key material.

| Material | Scope | Reused across runs? | Source on reuse |
|---|---|---|---|
| Private key | Per peer (server + each client) | Yes | Recovered from prior output (see [Recovery flow](#recovery-flow)) |
| Public key | Per peer | Yes (re-derived, never trusted from storage) | `DerivePublicKey(privateKey)` |
| Preshared key | Per client↔server connection | Yes | Recovered from prior output |
| CPS strings `I1`–`I5` | Per client | **No — regenerated every run** | `generator.go` (not in `PeerCredentials`) |

Recovery reads the prior `output/` tree via `#_` metadata plus each client's own `awg0.conf`; the server peer gets a keypair only (no PSK), each client gets a keypair plus a unique PSK. Keys are reused unless `--full-reset` is set or the prior key is missing/invalid.

## Where keys are stored

Credentials live in the generated INI configs under `output/`. The two files per client peer serve different roles; see [Output Format](./output-format.md) for the full INI structure.

| Credential | Server config (`output/<server>/awg0.conf`) | Client config (`output/<peer>/awg0.conf`) |
|---|---|---|
| Server private key | `[Interface] PrivateKey` | — |
| Server public key | `[Interface] PublicKey` | `[Peer] PublicKey` |
| Client private key | — (never stored server-side) | `[Interface] PrivateKey` (authoritative — sole source) |
| Client public key | `[Peer] PublicKey` | (re-derived, not stored) |
| Preshared key (PSK) | `[Peer] PresharedKey` | `[Peer] PresharedKey` |
| Peer name | `[Peer] #_Name` | — |

> **Note:** `generate` does **not** write a peer's private key or key-generation timestamp into the server config — each server `[Peer]` carries only `#_Name`, `PublicKey`, `PresharedKey`, and `AllowedIPs`. A client's private key is recovered **solely** from its own `output/<peer>/awg0.conf` `[Interface]` section. (The writer *can* emit `#_PrivateKey` for hand-authored or library-built `PeerConfig`, but `buildServerConfig` never sets it, so generated configs never contain it.) Bare `#` comment lines are ignored by the parser; only `#_`-prefixed lines are metadata.

## `LoadCredentials`

`LoadCredentials` reads the prior `output/` tree and returns everything recoverable. A first run (no output directory or no server config) returns **empty credentials, not an error**.

| Signature | Behavior |
|---|---|
| `LoadCredentials(outputDir, serverPeerName string) (*PersistedCredentials, error)` | Returns `*PersistedCredentials{Peers: map, Server: PeerCredentials}`. On a missing output dir or missing server config → returns `EmptyCredentials()` with `nil` error (first-run path). On any other read/parse error → returns `(nil, err)`. |

Internal behavior:

| Phase | What happens |
|---|---|
| Server config exists | Server keypair ← server config `[Interface]` (`PrivateKey`, `PublicKey`); peers ← `loadPeersFromServer` (PublicKey + PSK from server `[Peer]`, PrivateKey from each client's own config). |
| Server config missing, dir exists | Fallback: scan every subdirectory except `serverPeerName`, recover PrivateKey + PSK directly from each client's `awg0.conf` via `extractClientCredentials`. |
| Output dir missing | First run — return empty credentials, `nil` error. |

Returned types:

| Type | Fields |
|---|---|
| `PeerCredentials` | `PrivateKey`, `PublicKey`, `PresharedKey` (all base64 strings) |
| `PersistedCredentials` | `Peers map[string]PeerCredentials` (keyed by peer name); `Server PeerCredentials` (zero-value on first run) |
| `EmptyCredentials() *PersistedCredentials` | Constructor: initialized `Peers` map, zero-value `Server`. Used on first run and under `--full-reset`. |

## `--full-reset` behavior

The `generate --full-reset` flag (default `false`) makes the run behave like a first run: persisted credentials are ignored and every key is regenerated. See [CLI Reference](./cli-reference.md).

| Flag | Default | Effect when `true` |
|---|---|---|
| `--full-reset` | `false` | `resolvePeerCredentials` ignores `*PersistedCredentials` entirely: server gets a fresh `GenerateKeyPair()`; each client gets a fresh `GenerateKeyPair()` private key + fresh `GeneratePSK()`. Public keys are still derived. Equivalent to a first run on an empty `output/`. |

Without the flag, the pipeline calls `LoadCredentials(opts.OutputDir, serverName)` and reuses any recovered key whose `PrivateKey` is non-empty (see [Recovery flow](#recovery-flow)).

## Key generation API

All material is X25519 via `golang.org/x/crypto/curve25519`, base64 `StdEncoding` (44 characters per 32-byte value). See [Library Usage](./library-usage.md) for the broader API surface.

| Function | Signature | Returns |
|---|---|---|
| `GenerateKeyPair` | `() (priv, pub string)` | Fresh clamped X25519 private key + its public key, both 44-char base64 |
| `DerivePublicKey` | `(privateKey string) string` | Public key derived from a base64 private key; 44-char base64 |
| `GeneratePSK` | `() string` | 32 random bytes as a 44-char base64 preshared key |

| Property | Detail |
|---|---|
| WireGuard clamping | Applied in both `GenerateKeyPair` and `DerivePublicKey`: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64` |
| Encoding | `base64.StdEncoding` → 44 chars (32 bytes + padding) |
| Failure mode | `GenerateKeyPair` and `GeneratePSK` **panic** only on `crypto/rand` failure; `DerivePublicKey` **panics** on invalid base64 or a length ≠ 32 bytes |
| Panic safety | `tryDerivePublicKey(privateKey) string` wraps `DerivePublicKey` in `recover()`, returning `""` on panic so the pipeline regenerates the key instead of crashing — guards against hand-edited invalid configs |
| Public-key consistency | Client `PublicKey` is **always** re-derived from `PrivateKey` (`resolvePeerCredentials`); the persisted `PublicKey` is never trusted |

## Recovery flow

For each peer in the manifest, `resolvePeerCredentials` either reuses a recovered key or generates a fresh one. The decision table below is evaluated per peer.

| Condition (evaluated in order) | Server peer | Client peer |
|---|---|---|
| `!fullReset && persisted.Server.PrivateKey != ""` / `!fullReset && hasPersisted && persistedClient.PrivateKey != ""` | Reuse `PrivateKey` + `PublicKey` from `persisted.Server` | Reuse `PrivateKey` + `PresharedKey` from `persisted.Peers[name]` |
| Otherwise (full reset, no entry, or empty key) | Fresh `GenerateKeyPair()` | Fresh `GenerateKeyPair()` private key + fresh `GeneratePSK()` |
| `PublicKey` | Reused verbatim from storage | **Always** `DerivePublicKey(PrivateKey)` — never trusted from storage |
| `PresharedKey` | Never (server has none) | Reused if recovered, else freshly generated |

How a **client private key** is recovered into `persisted.Peers[name]` — the client's own `awg0.conf` is the **only** source (the server config stores no peer private keys):

1. **Primary path — server config present** (`LoadCredentials` → `loadPeersFromServer`):
   1. For each `[Peer]` in the server config, skip if `#_Name` is empty (unnamed peers cannot be matched to manifest entries).
   2. Take `PublicKey` and `PresharedKey` from the server `[Peer]` section.
   3. Recover `PrivateKey` from the client's own `output/<peer>/awg0.conf` `[Interface]` via `extractClientCredentials`. The server config carries no peer private key to read.
   4. If the client config is unreadable or its `PrivateKey` is empty, the peer is recorded with an empty `PrivateKey` → `resolvePeerCredentials` generates a fresh keypair on the next run.
2. **Fallback path — server config missing** (`LoadCredentials` scans subdirectories):
   1. For each subdirectory of `output/` except `serverPeerName`, run `extractClientCredentials` on `<dir>/awg0.conf`.
   2. Recover `PrivateKey` from `[Interface]` and `PresharedKey` from `[Peer]`; derive `PublicKey` via `tryDerivePublicKey`.
   3. Skip unreadable configs; only store entries with a non-empty `PrivateKey`.

> **Note:** `extractClientCredentials` is a lightweight INI scanner — it reads `PrivateKey` from `[Interface]` and `PresharedKey` from `[Peer]`, skips bare `#` comments, and ignores unparseable lines. It is not a full `ParseClientConfig`.

## Gotchas

| Behavior | Detail |
|---|---|
| Sorted peer iteration | Client configs are written in sorted peer-name order for deterministic output. |
| Non-atomic writes | `generate` writes via `os.WriteFile` — a mid-run crash can leave a partially written `output/` tree. Re-running reuses whatever complete files exist. See [Gotchas](./gotchas.md). |
| CPS strings are never reused | `I1`–`I5` are regenerated every run; only the X25519 keys and PSK are reused. |
| No peer private key in the server config | `generate` never writes a peer's `PrivateKey` to the server config; recovery is solely from each client's own `awg0.conf`. |
| Unnamed peers are skipped | A `[Peer]` without `#_Name` cannot be matched to a manifest entry and is dropped during recovery. |
| Removed peers leave orphans | Deleting a peer from the manifest does not delete its `output/<peer>/` directory; recovered creds are simply unused. |
| `DerivePublicKey` panics on bad input | Hand-edited invalid keys panic the raw call; callers must use `tryDerivePublicKey` (the pipeline does). |

## Related

- [Output Format](./output-format.md) — full INI structure and `#_` metadata inventory.
- [Library Usage](./library-usage.md) — `Generate`, `GenerateKeyPair`, `DerivePublicKey`, `GeneratePSK` API.
- [CLI Reference](./cli-reference.md) — `generate --full-reset` and `--dry-run` flags.
- [Gotchas](./gotchas.md) — project-wide pitfalls including non-atomic writes.
