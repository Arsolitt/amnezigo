# Credentials & Key Reuse

> How amnezigo generates, stores, and reuses X25519 keys, preshared keys, and the AWG 3.x header-protection key across `generate` runs — and which material is re-randomized every run.

## Table of Contents

- [Key reuse model](#key-reuse-model)
- [Where keys are stored](#where-keys-are-stored)
- [Header-protection key](#header-protection-key)
- [`LoadCredentials`](#loadcredentials)
- [`--full-reset` behavior](#--full-reset-behavior)
- [Key generation API](#key-generation-api)
- [Recovery flow](#recovery-flow)
- [Gotchas](#gotchas)
- [Related](#related)

---

## Key reuse model

A plain `amnezigo generate` reuses the X25519 keys and PSKs produced by previous runs, so key material never forces a client re-import by itself. The rewritten configs still carry freshly resolved unpinned obfuscation values, though, so emitted client configs must be redistributed to stay in sync with the rewritten server config unless you pin those values in the manifest.

| Material | Scope | Reused across runs? | Source on reuse |
| --- | --- | --- | --- |
| Server private key | Server peer | Yes | `output/<server>/awg0.conf` `[Interface] PrivateKey` |
| Server public key | Server peer | Yes — copied verbatim, never re-derived | Same config, `[Interface] PublicKey` |
| Client private key | Per client | Yes | `output/<peer>/awg0.conf` `[Interface] PrivateKey` (the only source) |
| Client public key | Per client | No — always re-derived | `DerivePublicKey(clientPrivateKey)` |
| Preshared key (PSK) | Per client↔server connection | Yes | Server config `[Peer] PresharedKey`; the fallback scan reads it from the client config |
| Header-protection key | Device-wide (AWG 3.0+) | Yes | Server config `[Interface] HeaderProtectionKey` |
| S-prefixes, header ranges, junk (`Jc`/`Jmin`/`Jmax`) | Per config | No — S and junk are drawn fresh from `crypto/rand`; H1–H4 are drawn fresh too, except under header protection with all four unset, where the fixed standard ranges `1-1`/`2-2`/`3-3`/`4-4` are applied — manifest pins override either way | `pipeline.go` generators (`standardHeaderRanges()` for the HP default) |
| AWG 3.x U16 ranges (content padding, timers, handshake attempts) | Per config | Not persisted — re-resolved from the manifest each run | Manifest value, else the fixed version default |
| CPS `I1`–`I5` | Per client | No — not persisted; recomputed per client from its protocol template (deterministic for named templates; `random` draws fresh each run) | `GenerateCPS`; never stored in `PersistedCredentials` |

Two consequences worth internalizing:

- **Keys are reused; configs are not stable.** Unpinned S-prefixes and junk values change on every run — and so do unpinned header ranges, except that with header protection on and no `h1`–`h4` set, H1–H4 stay at the fixed standard ranges `1-1`/`2-2`/`3-3`/`4-4` — even though every private key, PSK, and the header-protection key stay identical. The CPS `I1`–`I5` strings are recomputed but stable: named protocol templates (the default `quic`) render deterministically for a given protocol, MTU, and clipping budget, while `protocol: random` draws a fresh tag sequence each run. The 3.x U16 ranges stay stable only because an omitted manifest field falls back to a fixed default.
- **Reuse is server/client asymmetric.** The client public key is always re-derived from its private key; the server keypair is trusted exactly as recovered (see [Recovery flow](#recovery-flow)).

Server peers get a keypair only and never a PSK. Each client peer gets a keypair plus its own PSK, one per client-to-server connection.

> **Warning:** Credential reuse only engages when the previously recovered private key is non-empty. Deleting `output/`, losing one client config, or renaming a peer in the manifest silently drops the missing material from reuse and rotates it on the next run — every affected client must re-import its config.

## Where keys are stored

There is no separate key store or vault: credentials live in-band in the generated INI configs under `output/`, and that directory doubles as the credential database. See [Output Format](./output-format.md) for the full file layout.

| Credential | Server config (`output/<server>/awg0.conf`) | Client config (`output/<peer>/awg0.conf`) |
| --- | --- | --- |
| Server private key | `[Interface] PrivateKey` | — |
| Server public key | `[Interface] PublicKey` | `[Peer] PublicKey` |
| Client private key | — (never stored server-side) | `[Interface] PrivateKey` (authoritative — sole source) |
| Client public key | `[Peer] PublicKey` | — |
| Preshared key (PSK) | `[Peer] PresharedKey` | `[Peer] PresharedKey` |
| Header-protection key | `[Interface] HeaderProtectionKey` (normal key, plaintext) | `[Interface] HeaderProtectionKey` (same value, plaintext) |
| Peer-name match key | `[Peer] #_Name` | — |

The `#_` metadata matters to recovery:

- `#_Name` on each server `[Peer]` is the match key. The peer's private key is read from `output/<#_Name>/awg0.conf`; a `[Peer]` block without a `#_Name` comment cannot be matched to a manifest entry and is skipped, so its keys regenerate on the next run.
- `#_PrivateKey` and `#_GenKeyTime` are supported by the writer, but `buildServerConfig` never populates them — generated server configs never contain them.
- `extractClientCredentials` ignores bare `#` comment lines; `#_` lines are read like ordinary keys (and ignored unless they are `PrivateKey`/`PresharedKey`).

> **Danger:** The header-protection key is written as plaintext into the server config and into every client config (it is a normal `[Interface]` key, not `#_` metadata). Any client-config reader can decrypt header-protected packet headers, on top of the client private key the file already exposes. Treat every client config — and every [`vpn://` link](./vpn-links.md), which embeds it — as a secret.

## Header-protection key

AWG 3.0 header protection encrypts the packet header (hiding the WireGuard message type-ids) with a device-level **ChaCha20** key. It is a symmetric 32-byte key, not a Curve25519 scalar, so `GenerateHeaderProtectionKey` never applies WireGuard clamping. See [Transport Protection (AWG 3.x)](./transport-protection.md) for the feature as a whole.

Persistence is in-band and parallel to the peer keys:

| Stage | Behavior |
| --- | --- |
| Emission | Written as `HeaderProtectionKey = <44-char base64>` in the server `[Interface]` and in every client `[Interface]`; only when header protection is on and the version is 3.0+. |
| Recovery | `LoadCredentials` reads the server config's line into `PersistedCredentials.HeaderProtectionKey`. |
| Reuse | `resolveObfuscation` keeps the recovered key when `!fullReset && persisted.HeaderProtectionKey != ""`; otherwise it calls `GenerateHeaderProtectionKey()`. |
| Disabled | With `header_protection: false` (or `awg_version` below 3.0), the field stays empty and no line is emitted. |

A worked sequence on the default 3.1 manifest:

```shell
$ amnezigo generate
$ grep HeaderProtectionKey output/server/awg0.conf

$ amnezigo generate
$ grep HeaderProtectionKey output/server/awg0.conf

$ amnezigo generate --full-reset
$ grep HeaderProtectionKey output/server/awg0.conf
```

```text
HeaderProtectionKey = <44-char base64>       # first run — generated
HeaderProtectionKey = <same value>           # second run — reused
HeaderProtectionKey = <new value>            # after --full-reset — every client must re-import
```

The key rotates — and every already-distributed client config stops handshaking — when:

- you run `generate --full-reset`;
- the server config is missing, so `LoadCredentials` takes the fallback scan, which never sets `HeaderProtectionKey`;
- the previous run emitted no key (`awg_version: "2.0"` or `header_protection: false`), leaving nothing to recover the next time header protection is enabled;
- you delete `output/` or the server config.

**Corruption is fatal, not recovered.** The parser validates the line as 44-char base64 decoding to exactly 32 bytes. A hand-edited value aborts credential loading — and therefore the whole `generate` run — with:

```text
load credentials: load server config for credentials: invalid HeaderProtectionKey "notbase64": must be 44-char base64 of 32 bytes
```

There is no regeneration fallback: fix the line or remove the file before the next run.

> **Note:** Under header protection the S-prefix floor rises: S1–S4 must be at least 12 (the ChaCha20 nonce length). The pipeline rejects lower values instead of generating around them. See [Obfuscation](./obfuscation.md).

## `LoadCredentials`

`LoadCredentials` reads the prior `output/` tree and returns everything recoverable. A first run — no output directory at all — returns **empty credentials, not an error**; a missing server config alone still recovers client keys and PSKs via the fallback scan.

| Signature | Behavior |
| --- | --- |
| `LoadCredentials(outputDir, serverPeerName string) (*PersistedCredentials, error)` | Returns `*PersistedCredentials` with `Peers`, `Server`, and `HeaderProtectionKey` populated from the existing configs. Missing output dir → `EmptyCredentials()` with `nil` error (first-run path); missing server config with client configs present → client `PrivateKey`/PSK recovered by the fallback scan (server keypair and header-protection key stay empty). Any other read or parse error → `(nil, err)`. |

Internal behavior:

| Phase | What happens |
| --- | --- |
| Server config exists | Server keypair ← server `[Interface]` (`PrivateKey`, `PublicKey`); header-protection key ← server `[Interface] HeaderProtectionKey`; peers ← `loadPeersFromServer` (PublicKey + PSK from the server `[Peer]`, PrivateKey from each client's own config). |
| Server config missing, dir exists | Fallback: scan every subdirectory except `serverPeerName` and recover PrivateKey + PSK from each client's `awg0.conf` via `extractClientCredentials`. The server keypair and the header-protection key are **not** recovered. |
| Output dir missing | First run — empty credentials, `nil` error. |

Returned types:

| Type | Fields |
| --- | --- |
| `PeerCredentials` | `PrivateKey`, `PublicKey`, `PresharedKey` (all base64 strings) |
| `PersistedCredentials` | `Peers map[string]PeerCredentials` (keyed by peer name); `Server PeerCredentials` (zero-value on first run); `HeaderProtectionKey string` (empty when absent or when the target version predates 3.x) |
| `EmptyCredentials() *PersistedCredentials` | Constructor: initialized `Peers` map, zero-value `Server` and `HeaderProtectionKey`. Used on first run (and when `OutputDir` is empty); `--full-reset` does not call it but suppresses reuse of whatever was loaded. |

> **Warning:** Any parse failure other than "file does not exist" is fatal and wrapped as `load server config for credentials: <err>`. A corrupt `HeaderProtectionKey` (or any other malformed key line) blocks `generate` entirely — there is no fallback to regeneration.

## `--full-reset` behavior

The `generate --full-reset` flag (default `false`) makes the run behave like a first run: persisted credentials are ignored and all key material is regenerated. See [CLI Reference](./cli-reference.md).

| Material | Effect when `--full-reset` is set |
| --- | --- |
| Server keypair | Fresh `GenerateKeyPair()` |
| Every client keypair | Fresh `GenerateKeyPair()` private key |
| Every client PSK | Fresh `GeneratePSK()` |
| Header-protection key | Fresh `GenerateHeaderProtectionKey()` |
| S-prefixes, junk, and header ranges | Unchanged behavior — S/junk already re-drawn every run, header ranges re-resolved (fixed `1-1`/`2-2`/`3-3`/`4-4` under default header protection) |
| CPS `I1`–`I5` | Unchanged behavior — recomputed per client; deterministic for named templates, `random` redraws |
| Addresses, ports, `AllowedIPs`, manifest fields | Untouched |

`resolvePeerCredentials` ignores `*PersistedCredentials` entirely when the flag is set, and `resolveObfuscation` skips its header-protection reuse branch. Because the server key, every PSK, and the device-wide header-protection key all change, `--full-reset` is a fleet-wide re-provisioning action: every client config must be redistributed. It is all-or-nothing — there is no per-peer reset flag.

## Key generation API

All key material is base64 `StdEncoding` (44 characters per 32 bytes). See [Library Usage](./library-usage.md) for the broader exported surface.

| Function | Signature | Returns |
| --- | --- | --- |
| `GenerateKeyPair` | `() (priv, pub string)` | Fresh clamped X25519 private key + its public key, both 44-char base64 |
| `DerivePublicKey` | `(privateKey string) string` | Public key derived from a base64 private key; 44-char base64 |
| `GeneratePSK` | `() string` | 32 random bytes as a 44-char base64 preshared key |
| `GenerateHeaderProtectionKey` | `() string` | 32 random bytes as a 44-char base64 ChaCha20 key; **not clamped** (symmetric key, not a Curve25519 scalar) |

| Property | Detail |
| --- | --- |
| WireGuard clamping | Applied in both `GenerateKeyPair` and `DerivePublicKey`: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`. `GenerateHeaderProtectionKey` deliberately does not clamp. |
| Encoding | `base64.StdEncoding` → 44 chars (32 bytes + padding) |
| Failure mode | `GenerateKeyPair`, `GeneratePSK`, and `GenerateHeaderProtectionKey` panic only on `crypto/rand` failure; `DerivePublicKey` panics on invalid base64 or a length ≠ 32 bytes |
| Panic recovery scope | `tryDerivePublicKey` wraps `DerivePublicKey` in `recover()` and returns `""` on panic — but it is called only from `extractClientCredentials` (the client-config scanner). It does **not** shield the pipeline. |
| Client public keys | `resolvePeerCredentials` always calls raw `DerivePublicKey(clientPrivateKey)`; the persisted client `PublicKey` is never trusted |
| Server public keys | The persisted server pair is reused verbatim — including an empty `PublicKey` — with no re-derivation |

> **Danger:** A hand-edited invalid `PrivateKey` in an existing client config **panics the next `generate`** (`crypto: invalid base64 private key` or `crypto: private key must be 32 bytes`, raised at `pipeline.go:422`). The panic guard in `extractClientCredentials` only protects the scanner; the reuse path re-derives with the raw function, so the pipeline does not regenerate the key — it crashes.

## Recovery flow

For each peer in the manifest, `resolvePeerCredentials` either reuses recovered material or generates fresh material. The decision table below is evaluated per peer.

| Condition (evaluated in order) | Server peer | Client peer |
| --- | --- | --- |
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
   4. The server keypair and `HeaderProtectionKey` stay empty — the next run generates both fresh.

> **Note:** `extractClientCredentials` is a lightweight INI scanner — it reads `PrivateKey` from `[Interface]` and `PresharedKey` from `[Peer]`, skips bare `#` comments, and ignores unparseable lines. It is not a full `ParseClientConfig`.

> **Warning:** Losing the server config's `PublicKey` line is sticky and silent. The reuse condition checks only that the server `PrivateKey` is non-empty, so the empty persisted `PublicKey` is copied verbatim; the next run writes `PublicKey = ` (an empty value) into every client `[Peer]` and nowhere into the server `[Interface]`, then exits 0 with no finding. Only `--full-reset` or deleting the output tree (forcing a fresh server keypair) restores a working public key.

## Gotchas

| Behavior | Detail |
| --- | --- |
| Sorted peer iteration | Client configs are written in sorted peer-name order for deterministic output. |
| Non-atomic writes | `generate` writes via `os.WriteFile` — a mid-run crash can leave a partially written `output/` tree. Re-running reuses whatever complete files exist. See [Gotchas](./gotchas.md). |
| Key reuse ≠ config stability | Unpinned S-prefixes and junk are re-drawn from `crypto/rand`; unpinned H ranges are redrawn too, except under header protection with all four H unset (fixed `1-1`/`2-2`/`3-3`/`4-4`). `I1`–`I5` are not persisted and are recomputed per client, but named protocol templates (the default `quic`) are deterministic for a given protocol, MTU, and clipping budget — only `protocol: random` redraws each run. Keys and the header-protection key stay identical, and the 3.x ranges change only if the manifest changes. |
| `generate --peer <one>` leaves other clients stale | The server config is always rewritten with freshly resolved obfuscation, while peers excluded by the filter keep their previous configs. Those clients no longer match the server and their tunnels break until they are regenerated without the filter. |
| Missing server `PublicKey` breaks every client silently | The empty public key is reused verbatim; `PublicKey = ` lands in every client `[Peer]`, exit code 0, no finding. Only `--full-reset` or deleting the output recovers. |
| Hand-edited invalid client key panics `generate` | `crypto: invalid base64 private key` / `crypto: private key must be 32 bytes`; recovery is not attempted. The scanner's `tryDerivePublicKey` guard does not cover the pipeline's reuse path. |
| Corrupt `HeaderProtectionKey` blocks `generate` | The line must be 44-char base64 of 32 bytes; anything else aborts with `load credentials: load server config for credentials: invalid HeaderProtectionKey ...`. No regeneration fallback. |
| Header-protection key rotation invalidates every client | `--full-reset`, a lost server config, a run that emitted no key (2.0 / `header_protection: false`), or deleting `output/` all force a new device key; every distributed client must re-import. |
| Header-protection key is plaintext | Stored unencrypted in the server `[Interface]` and every client `[Interface]`; a client-config reader can decrypt header-protected headers. |
| Deleting `output/` destroys all key material | Keys and the header-protection key exist only inside the emitted configs; there is no separate store. |
| Fallback scan loses the server keys and HP key | Client keys and PSKs survive; the server keypair and header-protection key are regenerated, so the server identity and all handshakes change. |
| No peer private key in the server config | `generate` never writes a peer's `PrivateKey` to the server config; recovery is solely from each client's own `awg0.conf`. |
| Unnamed peers are skipped | A `[Peer]` without `#_Name` cannot be matched to a manifest entry and is dropped during recovery, forcing fresh keys on the next run. |
| Removed peers leave orphans | Deleting a peer from the manifest does not delete its `output/<peer>/` directory; recovered credentials are simply unused. |
| `--dry-run` discards newly generated keys | A dry run computes keys in memory and writes nothing, so the next real run writes different keys. There is no way to preview the exact keys. |

## Related

- [Output Format](./output-format.md) — where each key is emitted and the full `#_` metadata inventory.
- [Transport Protection (AWG 3.x)](./transport-protection.md) — header protection, version gating, and the 3.x INI keys.
- [Library Usage](./library-usage.md) — `LoadCredentials`, `GenerateKeyPair`, `DerivePublicKey`, `GeneratePSK`, `GenerateHeaderProtectionKey`.
- [CLI Reference](./cli-reference.md) — `generate --full-reset` and `--dry-run`.
- [Validation & Analysis](./validation.md) — HPK001 and the other transport-protection findings.
- [Gotchas](./gotchas.md) — project-wide pitfalls including non-atomic writes.
