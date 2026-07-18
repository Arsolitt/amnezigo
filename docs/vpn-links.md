# vpn:// Import Links

> AmneziaVPN-app-importable `vpn://` links — one URL per client peer that the app imports in a single tap.

## Table of Contents

- [What vpn:// Import Links Are](#what-vpn-import-links-are)
- [The `amnezigo.vpn` Output File](#the-amnezigovpn-output-file)
- [Envelope Format](#envelope-format)
- [Encoding Pipeline](#encoding-pipeline)
- [The `last_config` Structured Fields](#the-last_config-structured-fields)
- [DNS Mapping](#dns-mapping)
- [Display Name (`description`)](#display-name-description)
- [Library API: `EncodeVPNLink`](#library-api-encodevpnlink)
- [CLI Usage](#cli-usage)
- [Compatibility Caveat](#compatibility-caveat)
- [Related](#related)

---

## What vpn:// Import Links Are

The `--vpn-links` flag (library: `GenerateOptions.VPNLinks`) tells `amnezigo generate`
to produce, in addition to the standard `awg0.conf` files, a **`vpn://` import link**
for every **client** peer. Each link is a self-contained URL that the **AmneziaVPN
app** (Android, iOS, desktop) can import in a single tap or click — no need to copy
INI files, scan QR codes from a terminal, or manually enter server details.

The link encodes the client's full AWG 2.0 configuration — keys, endpoint, DNS,
obfuscation parameters, and CPS tags — into a compact, URL-safe string prefixed with
`vpn://`. The AmneziaVPN app recognises this scheme, decodes the embedded JSON
envelope, and configures the tunnel automatically.

> **Key distinction:** `vpn://` links target the **AmneziaVPN app**, not the
> standalone AmneziaWG client or kernel module. The envelope format is
> AmneziaVPN-specific. See [Compatibility Caveat](#compatibility-caveat).

### Why use vpn:// links?

| Scenario | Without `--vpn-links` | With `--vpn-links` |
|---|---|---|
| Onboard a mobile user | Copy `awg0.conf` → transfer file → import manually | Share one `vpn://` URL → tap to import |
| Provision a desktop client | Copy-paste INI into config editor | Paste `vpn://` link into AmneziaVPN app |
| Audit what the app receives | Inspect `awg0.conf` (INI format) | Decode `vpn://` link (structured JSON envelope) |

The `awg0.conf` file is always generated regardless; `--vpn-links` adds the
`amnezigo.vpn` file alongside it. The two encode the same client configuration in
different formats for different consumers.

---

## The `amnezigo.vpn` Output File

When `--vpn-links` (or `GenerateOptions.VPNLinks: true`) is set, `generate` emits
an additional file named `amnezigo.vpn` in **each client peer's** output directory,
alongside the existing `awg0.conf`.

```text
output/
├── server/
│   └── awg0.conf              # server config — NO .vpn file
├── alice/
│   ├── awg0.conf              # standard client INI (always present)
│   └── amnezigo.vpn           # vpn:// link (only with --vpn-links)
└── bob/
    ├── awg0.conf
    └── amnezigo.vpn
```

| Property | Value | Source |
|---|---|---|
| File name | `amnezigo.vpn` (constant `outputVPNLinkName`) | `vpnlink.go:16` |
| Path | `<OutputDir>/<peerName>/amnezigo.vpn` | `vpnlink.go:217` (`appendVPNLink`) |
| File content | The raw `vpn://` URL string (no wrapper, no formatting) | `vpnlink.go:217` |
| Generated for | Client peers only | `pipeline.go:500-502` |
| Server peer | **No** `.vpn` file — server configs are not importable links | `pipeline.go:500-502` |
| Gated on | `--vpn-links` flag / `GenerateOptions.VPNLinks` | `pipeline.go:500` |

The file content **is** the URL — a single line starting with `vpn://` followed by
the base64url-encoded payload. No JSON wrapper, no trailing newline, no metadata.

> **Note:** The server peer never gets a `.vpn` file. A server config represents
> the listening side of the tunnel, not a client connection that an app would
> import. See [Output Format](./output-format.md) for the full output layout.

---

## Envelope Format

The `vpn://` link encodes a JSON envelope through a multi-stage pipeline. The
format is **AmneziaVPN-specific** — it was reverse-engineered from the app's
`containersModel`/`importController` and `Wireguard.kt:parseConfigData()`. The
standalone AmneziaWG client does not understand it.

### Envelope structure

The decoded payload is a JSON object with four nested levels:

```text
vpnEnvelope
├── hostName           string         — server endpoint host
├── description        string?        — displayed server name (omitempty; app falls back to hostName)
├── defaultContainer   string         — always "amnezia-awg"
├── dns1               string?        — first DNS server (omitempty)
├── dns2               string?        — second DNS server (omitempty)
└── containers[]       [1]vpnContainer
    └── [0]
        ├── container      string         — always "amnezia-awg"
        └── awg            vpnAwgConfig
            ├── last_config         string   — JSON-encoded vpnLastConfig (double-serialized!)
            ├── port                string   — server port as decimal string (e.g. "51820")
            ├── transport_proto     string   — always "udp"
            └── isThirdPartyConfig  bool     — always true
```

The `containers` array always has exactly one element. The container type is
always `amnezia-awg` — this is the AmneziaVPN internal identifier for an AWG 2.0
container, distinct from plain WireGuard (`amnezia-wireguard`) or OpenVPN
containers.

> **Note:** The envelope contains **no SSH credential fields** (`userName`,
> `password`). AmneziaVPN's `vpn://` format supports SSH-based containers with
> credentials, but amnezigo only emits `amnezia-awg` containers with no SSH data.
> This is by design — the link carries tunnel configuration, not server access
> credentials.

### Constants

| Constant | Value | Source |
|---|---|---|
| `outputVPNLinkName` | `"amnezigo.vpn"` | `vpnlink.go:16` |
| `vpnLinkScheme` | `"vpn://"` | `vpnlink.go:17` |
| `vpnContainerAWG` | `"amnezia-awg"` | `vpnlink.go:18` |
| `qCompressHeaderSize` | `4` | `vpnlink.go:20` |

---

## Encoding Pipeline

The `vpn://` link is built through a 7-step pipeline. Each step transforms the
data toward the final URL-safe string:

```text
Client INI ──► vpnLastConfig struct
                    │
                    ▼  json.Marshal (inner JSON)
              lastCfgJSON bytes
                    │
                    ▼  embed as string in vpnAwgConfig.LastConfig
              vpnEnvelope struct
                    │
                    ▼  json.Marshal (outer JSON)
              envelopeJSON bytes
                    │
                    ▼  qCompress (4-byte BE length header + zlib)
              compressed bytes
                    │
                    ▼  base64.RawURLEncoding (URL-safe, no padding)
              encoded string
                    │
                    ▼  prepend "vpn://"
              vpn://<payload>
```

| Step | Operation | Detail |
|---|---|---|
| 1 | Build `vpnLastConfig` | Structured fields parsed from client INI + verbatim INI in `Config` field |
| 2 | Marshal `lastConfig` → JSON | `json.Marshal(lastCfg)` → inner JSON bytes |
| 3 | Build `vpnEnvelope` | Envelope: `hostName`, `description`, `defaultContainer`, `dns1`/`dns2`, `containers[0].awg.last_config` |
| 4 | Marshal envelope → JSON | `json.Marshal(envelope)` → outer JSON bytes |
| 5 | `qCompress` | Prepend 4-byte big-endian uncompressed-length header, then zlib-compress |
| 6 | Base64URL encode | `base64.RawURLEncoding` — URL-safe alphabet, **no padding** (`=` chars) |
| 7 | Prepend scheme | `"vpn://" + encoded payload` |

### Why qCompress?

The `qCompress` function (`vpnlink.go:246`) replicates Qt's `qCompress` format: a
4-byte big-endian length header followed by zlib-compressed data. AmneziaVPN is a
Qt application — its decompressor (`QByteArray::qUncompress`) expects this format
so it can pre-allocate the output buffer from the length header before inflating.

### Double serialization of `last_config`

The `last_config` field is serialized **twice**:

1. The `vpnLastConfig` struct is marshaled to a JSON **string** (inner JSON).
2. That string is embedded as a field value in `vpnAwgConfig.LastConfig`.
3. The entire envelope (containing the string) is marshaled to JSON again (outer
   JSON).

When the app imports the link, it JSON-decodes the envelope, then JSON-decodes
`last_config` **again** to recover the structured fields. This double encoding is
required by AmneziaVPN's import path — `last_config` is typed as a JSON string in
the envelope, not a nested object.

> **Warning:** Never hand-craft a `vpn://` link. The double serialization of
> `last_config` is easy to get wrong — omitting it causes `JSONException` →
> error 1000 in the app. Always use [`EncodeVPNLink`](#library-api-encodevpnlink).

---

## The `last_config` Structured Fields

The `vpnLastConfig` struct (`vpnlink.go:63-91`) carries the actual tunnel
configuration. The app's connect path reads these structured fields via
`configWireguard()` and `configExtensionParameters()` — the `config` field
(verbatim INI) is for **display only**.

| JSON key | Go field | Source (INI key) | Type | Notes |
|---|---|---|---|---|
| `config` | `Config` | (entire INI) | `string` | Verbatim client INI, display only |
| `hostName` | `HostName` | server endpoint | `string` | Server host |
| `client_priv_key` | `ClientPrivKey` | `PrivateKey` | `string` | Client private key |
| `client_ip` | `ClientIP` | `Address` | `string` | Client tunnel IP |
| `server_pub_key` | `ServerPubKey` | `PublicKey` | `string` | Server public key |
| `psk_key` | `PSKKey` | `PresharedKey` | `string?` | omitempty |
| `allowed_ips` | `AllowedIPs` | `AllowedIPs` | `[]string` | **JSON array**, split on comma |
| `mtu` | `MTU` | `MTU` | `string?` | omitempty |
| `persistent_keep_alive` | `PersistentKeepAlive` | `PersistentKeepalive` | `string?` | omitempty |
| `Jc` | `Jc` | `Jc` | `string?` | AWG junk count |
| `Jmin` | `Jmin` | `Jmin` | `string?` | |
| `Jmax` | `Jmax` | `Jmax` | `string?` | |
| `S1`–`S4` | `S1`–`S4` | `S1`–`S4` | `string?` | |
| `H1`–`H4` | `H1`–`H4` | `H1`–`H4` | `string?` | |
| `I1`–`I5` | `I1`–`I5` | `I1`–`I5` | `string?` | CPS tag strings, verbatim |
| `port` | `Port` | server port | `int` | JSON number (not string) |
| `isObfuscationEnabled` | `IsObfuscationEnabled` | derived | `bool` | `true` when `Jc` is present |

### Critical field behaviors

**`allowed_ips` is a JSON array, not a string.** The app reads it via
`getJSONArray()`. `EncodeVPNLink` splits the INI `AllowedIPs` value on comma into
`[]string`. For example, `AllowedIPs = 0.0.0.0/0, ::/0` becomes
`["0.0.0.0/0","::/0"]` in the JSON.

**`isObfuscationEnabled` gates AWG fields.** Set to `true` when `Jc` is present in
the INI (`vpnlink.go:117`). When `true`, all AWG obfuscation fields (`Jc`, `Jmin`,
`Jmax`, `S1`–`S4`, `H1`–`H4`, `I1`–`I5`) are included. When `false` (no `Jc`),
they are omitted and the app treats the config as vanilla WireGuard.

**I1–I5 CPS tags are preserved verbatim.** The CPS tag strings (e.g.
`<b 0x12><r 16>`) are stored as-is, not expanded to hex bytes. The
`amneziawg-go` UAPI handler fully parses CPS tags (`<r>`, `<rc>`, `<rd>`, `<b>`,
`<t>`, `<d>`) at connect time. Expanding them in the link would freeze random
bytes and reduce obfuscation quality.

**All AWG fields are strings.** `Jc`, `Jmin`, `Jmax`, `S1`–`S4`, `H1`–`H4`,
`I1`–`I5` are stored as strings (matching the INI key names), not numbers. The app
reads them via `getString()`. The sole exception is `port`, which is a JSON number
(`int`).

---

## DNS Mapping

DNS servers from the manifest's `Network.DNS` field feed into the envelope's
`dns1` and `dns2` slots. The mapping is positional and limited to two entries.

| Manifest field | Envelope field | Condition |
|---|---|---|
| `Network.DNS[0]` | `dns1` | Set when `DNS` has ≥ 1 entry |
| `Network.DNS[1]` | `dns2` | Set when `DNS` has ≥ 2 entries |
| `Network.DNS[2+]` | *(ignored)* | Only 2 DNS slots exist in the envelope |
| *(empty slice)* | both omitted | `omitempty` on both fields |

**Source:** `vpnlink.go:179-184`, manifest field `NetworkConfig.DNS` (`[]string`,
json tag `"dns,omitempty"`) at `manifest.go:12`.

### Example

Given a manifest with:

```json
{
  "network": {
    "dns": ["1.1.1.1", "1.0.0.1"]
  }
}
```

The envelope will contain:

```json
{
  "dns1": "1.1.1.1",
  "dns2": "1.0.0.1"
}
```

If `dns` is `["8.8.8.8"]` (one entry), only `dns1` is set and `dns2` is omitted.
If `dns` is empty or absent, both fields are omitted from the JSON entirely.

> **Note:** The AmneziaVPN envelope format only has two DNS slots. If your manifest
> configures three or more DNS servers, only the first two appear in the `vpn://`
> link. See [Manifest Reference](./manifest-reference.md) for the `network.dns`
> field specification.

---

## Display Name (`description`)

The envelope's optional `description` field names the imported server in the
AmneziaVPN app. On import, the app reads `description` for the displayed server name;
when it is empty or absent the app falls back to `hostName` (the server endpoint host).
The envelope serializes `description` with `json:"description,omitempty"`, so the field
is dropped entirely when empty — preserving the hostName fallback for every existing
manifest.

| Envelope field | Source | Effect |
|---|---|---|
| `description` | `peers.<client>.display_name` (manifest) | Displayed server name in the app's server list |
| `hostName` (fallback) | server peer `endpoint` | Used when `description` is empty |

Set it via `display_name` on the **client** peer in the manifest:

```json
{
  "peers": {
    "phone": { "address": "10.0.0.2/32", "display_name": "Alice's phone" }
  }
}
```

Only client peers carry a `vpn://` link (the server peer never does), so `display_name`
is meaningful on client peers only. Setting it on the server peer is accepted but
ignored. See [Manifest Reference](./manifest-reference.md) for the field spec.

> **Caveat:** The app shows `description` as the server-list **title**, but the
> collapsed server row's **subtitle** still displays the `hostName`. This is expected
> app behavior; amnezigo cannot change the subtitle.

---

## Library API: `EncodeVPNLink`

The `EncodeVPNLink` function is the public entry point for generating `vpn://`
links programmatically. It is a pure function — no I/O, no manifest lookups. It
takes raw client INI bytes and returns the complete `vpn://` URL string.

### Signature

```go
func EncodeVPNLink(clientINI []byte, endpoint string, listenPort int, dns []string, description string) string
```

| Parameter | Type | Description |
|---|---|---|
| `clientINI` | `[]byte` | Raw client AWG INI config bytes (the `awg0.conf` content) |
| `endpoint` | `string` | Server endpoint. May be `host:port` or bare `host` |
| `listenPort` | `int` | Fallback port when `endpoint` lacks a port suffix |
| `dns` | `[]string` | DNS servers. `dns[0]` → `dns1`, `dns[1]` → `dns2`; beyond 2 ignored |
| `description` | `string` | Displayed server name in the AmneziaVPN app; empty `""` ⇒ app falls back to `hostName`. Serialized as envelope `description` (`omitempty`). |

**Returns:** A `vpn://` URL — the literal scheme `vpn://` followed by a
base64url-encoded payload with no padding.

### Example

```go
package main

import (
    "fmt"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    // Client AWG config (typically read from awg0.conf or built by Generate)
    clientINI := []byte(`[Interface]
PrivateKey = wLw4m8fG6q3N7sBp1xVzYk2tR9dE5aC0oU+iJl7TmHk=
Address = 10.0.0.2/32
DNS = 1.1.1.1
MTU = 1280
Jc = 5
Jmin = 250
Jmax = 750
S1 = 30
S2 = 35
S3 = 20
S4 = 12
H1 = 100-200
H2 = 150-250
H3 = 200-300
H4 = 250-350
I1 = <b 0x12><r 16>
I2 = <b 0x34><r 16>
I3 = <b 0x56><r 16>
I4 = <b 0x78><r 16>

[Peer]
PublicKey = SERVER_PUB_KEY_EXAMPLE_VALUE_HERE=
PresharedKey = PSK_EXAMPLE_VALUE_HERE=
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25`)

    // Generate the vpn:// import link
    link := amnezigo.EncodeVPNLink(
        clientINI,                        // client config bytes
        "vpn.example.com:51820",          // server endpoint
        51820,                            // fallback listen port
        []string{"1.1.1.1", "1.0.0.1"},   // DNS servers
        "Alice's phone",                  // displayed server name ("" → hostName fallback)
    )

    fmt.Println(link)
    // Output: vpn://eJx1kM1OwzAMhF9F9...  (base64url-encoded payload)
}
```

### How endpoint resolution works

The `endpoint` parameter may or may not include a port:

- `"vpn.example.com:51820"` → host=`vpn.example.com`, port=`51820`
- `"vpn.example.com"` → host=`vpn.example.com`, port=`listenPort` (fallback)

When called from the generate pipeline (`appendVPNLink`), the endpoint and
listen port come from the server peer's manifest fields (`Endpoint` and
`ListenPort`). See [Manifest Reference](./manifest-reference.md).

### Pipeline integration

Inside the `Generate` pipeline, the internal `appendVPNLink` function
(`vpnlink.go:217`) wraps `EncodeVPNLink` and appends the result as a
`FileOutput` to `GenerateResult.Files`. It is called once per client peer when
`opts.VPNLinks` is `true`:

```go
// Simplified from pipeline.go:500-502
if opts.VPNLinks {
    for _, clientName := range filteredClients {
        appendVPNLink(result, clientName, clientBytes, manifest, serverName)
    }
}
```

For the full `Generate` API including `GenerateOptions.VPNLinks`, see
[Library Usage](./library-usage.md).

---

## CLI Usage

Enable `vpn://` link generation with the `--vpn-links` flag on the `generate`
command:

```shell
$ amnezigo generate --vpn-links
```

This produces the standard output tree plus an `amnezigo.vpn` file in each client
peer directory:

```text
output/
├── server/
│   └── awg0.conf
├── alice/
│   ├── awg0.conf
│   └── amnezigo.vpn        ← new: vpn:// import link
└── bob/
    ├── awg0.conf
    └── amnezigo.vpn        ← new: vpn:// import link
```

Combine with other flags as needed:

```shell
# Generate configs + vpn:// links for a specific peer only
$ amnezigo generate --vpn-links --peer alice

# Preview without writing files
$ amnezigo generate --vpn-links --dry-run
```

### Reading a generated link

The `amnezigo.vpn` file contains a single `vpn://` URL:

```shell
$ cat output/alice/amnezigo.vpn
vpn://eJx1kM1OwzAMhF9F9lG3qZ2N7sBp1xVzYk2tR9dE5aC0oU-iJl7TmHk...
```

Share this URL with the AmneziaVPN app user — they paste it into the app's import
dialog or scan it as a QR code.

### Flag reference

| Flag | Type | Default | Description |
|---|---|---|---|
| `--vpn-links` | `bool` | `false` | Generate AmneziaVPN `vpn://` import links for each client peer |

**Source:** `internal/cli/generate.go:46-49`

When set, `Generate` is called with `GenerateOptions.VPNLinks = true`. See
[CLI Reference](./cli-reference.md) for the complete `generate` flag list.

---

## Compatibility Caveat

> **Warning:** `vpn://` import links work **only** with the AmneziaVPN app
> (Android, iOS, desktop). They do **not** work with the standalone AmneziaWG
> client, `amneziawg-go`, or the kernel module.

The envelope format — `vpnEnvelope`/`vpnContainer`/`vpnAwgConfig`/`vpnLastConfig`
JSON, `qCompress` encoding, base64url — is AmneziaVPN-specific. It was
reverse-engineered from the app's `containersModel`/`importController` and
`Wireguard.kt:parseConfigData()`. No other client implements this import format.

| Consumer | `vpn://` link | `awg0.conf` |
|---|---|---|
| AmneziaVPN app (mobile/desktop) | Import directly | Import via file |
| Standalone AmneziaWG client | Does not work | Use directly |
| `amneziawg-go` userspace | Does not work | Use directly |
| Kernel module (`amneziawg-linux-kernel-module`) | Does not work | Use directly |

**If you need standalone AmneziaWG or kernel module deployment**, use the
generated `awg0.conf` file directly. See [Output Format](./output-format.md) for
the INI layout.

For more compatibility pitfalls, see [Gotchas](./gotchas.md).

---

## Related

- [CLI Reference](./cli-reference.md) — `--vpn-links` flag and all `generate` options
- [Library Usage](./library-usage.md) — `EncodeVPNLink` API and `GenerateOptions.VPNLinks`
- [Output Format](./output-format.md) — `awg0.conf` INI layout and output directory structure
- [Manifest Reference](./manifest-reference.md) — `network.dns` field specification
- [Gotchas](./gotchas.md) — compatibility pitfalls and encoding warnings
