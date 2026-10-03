# Quick Start

> One complete AWG 3.1 round trip: install, write a manifest, generate configs, inspect them, validate, analyze, and check the version.

## Table of Contents

- [Before You Start](#before-you-start)
- [The Round Trip](#the-round-trip)
- [Step 1: Write the Manifest](#step-1-write-the-manifest)
- [Step 2: Generate the Configs](#step-2-generate-the-configs)
- [Step 3: Inspect the Configs](#step-3-inspect-the-configs)
- [Step 4: Validate the Config](#step-4-validate-the-config)
- [Step 5: Analyze the Config](#step-5-analyze-the-config)
- [Step 6: Check the Version](#step-6-check-the-version)
- [Flag Variations](#flag-variations)
- [Re-running Generate](#re-running-generate)
- [Related](#related)

---

## Before You Start

Two things need to be in place before the first command:

1. **The CLI.** Install it with `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest` (Go 1.26.1 or newer), download a release binary, or run the published container image. The full matrix is in [Installation](./installation.md).
2. **An AmneziaWG 3.x runtime for the configs you generate.** AWG 3.x configs carry transport-protection keys that only 3.x engines understand; older builds reject them (`Line unrecognized`). Keep the generator and the runtime on the same generation — the project's image pins `amneziawg-go` 3.1 (`amneziavpn/amneziawg-go:3.1.20260828`).

> **Note:** amnezigo never starts a tunnel, so everything on this page works offline and writes only config files.

Run the commands below from one project directory that contains the manifest.

## The Round Trip

amnezigo has exactly four commands — `generate`, `validate`, `analyze`, and `version`. The end-to-end flow is:

| Step | Command | Result |
| --- | --- | --- |
| 1. Write the manifest | _(create `amnezigo.json`)_ | Declares the network, one shared obfuscation profile, exactly one server peer, and N client peers. |
| 2. Generate | `$ amnezigo generate` | Reads the manifest, resolves keys and CPS parameters, and writes one `awg0.conf` per peer under `output/`. |
| 3. Inspect | _(open the generated configs)_ | Confirm the AWG 3.1 transport-protection block and the peer sections look right. |
| 4. Validate | `$ amnezigo validate output/server/awg0.conf` | Strict-parses the file and runs every AWG size invariant; errors exit non-zero. |
| 5. Analyze | `$ amnezigo analyze --config output/server/awg0.conf` | Prints heuristic RISK001–RISK009 findings and size/range profiles; findings never change the exit code. |
| 6. Check the version | `$ amnezigo version` | Prints the build stamp to quote in bug reports and upgrade decisions. |

`generate` mutates the filesystem — it writes configs and persists credentials for later runs — while `validate` and `analyze` are read-only. That separation is the whole workflow.

## Step 1: Write the Manifest

Save this as `amnezigo.json` in your project directory. It is the smallest manifest that exercises the AWG 3.1 defaults:

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "awg_version": "3.1"
  },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": {
      "address": "10.0.0.2/32"
    }
  }
}
```

| Field | Purpose |
| --- | --- |
| `version` | Manifest schema version; MUST be `1`. |
| `network.mtu` | MTU written into both configs and the size budget CPS generation works with. |
| `obfuscation.awg_version` | Selects the protocol generation: `"2.0"`, `"3.0"`, or `"3.1"`; defaults to `"3.1"`. Version 3.1 turns on the transport-protection key set you will see below. |
| `peers.server.address` | Server interface address; the `/24` prefix defines the tunnel subnet. |
| `peers.server.endpoint` + `peers.server.listen_port` | A peer with **both** set is the server. A valid manifest has exactly one server peer. |
| `peers.phone.address` | Client interface address; it also becomes the server-side `AllowedIPs` entry for that peer. |

Every other obfuscation field is optional and defaults per version, which is why this short manifest still produces a fully populated 3.1 config.

> **Warning:** Set `network.mtu` explicitly, as above. If you omit it, the emitted `MTU` line still reads `1280`, but the client's `I1`–`I5` collapse to the bare `<t>` placeholder: CPS generation runs with the raw unset value (0) before the 1280 default is applied. The server config is not affected. See [Gotchas](./gotchas.md).

[Manifest Reference](./manifest-reference.md) documents every field, default, and pointer-nil rule.

## Step 2: Generate the Configs

```shell
$ amnezigo generate
```

```text
Generated 2 config(s):
  server/awg0.conf (743 bytes)
  phone/awg0.conf (1027 bytes)
```

By default the command reads `./amnezigo.json` (or `./.amnezigo.jsonnet` — see [Jsonnet](./jsonnet.md)) and writes under `./output`:

```text
my-vpn/
├── amnezigo.json
└── output/
    ├── server/
    │   └── awg0.conf
    └── phone/
        └── awg0.conf
```

There is one directory per peer key from the manifest. The server peer's config gets one `[Peer]` section per client; each client gets a single `[Peer]` section pointing back at the server.

- The listed paths are relative to the output directory; `generate` does not print the directory itself.
- Byte counts are observed examples, not constants. Key material has a fixed size, but obfuscation values are random, so a later `--dry-run` of the same fixture reported `741` and `1025` bytes.
- If the built server config trips a validation rule, a `Warnings: N` block appears after the file list — for example `TRL001` when `RandomTrailers` is on with unequal `S1`–`S4` — and the command still exits `0`. The default 3.1 fixture draws one uniform `S`, so it prints no warnings.

Add `--vpn-links` to also write an AmneziaVPN import file for every client peer:

```shell
$ amnezigo generate --vpn-links
```

```text
Generated 3 config(s):
  server/awg0.conf (743 bytes)
  phone/awg0.conf (1027 bytes)
  phone/amnezigo.vpn (1356 bytes)
```

> **Warning:** the `.vpn` file embeds the client's private key. Treat it exactly like the generated config, and see [VPN Import Links](./vpn-links.md) before sharing it.

## Step 3: Inspect the Configs

`output/server/awg0.conf` from this run (key values abbreviated):

```ini
[Interface]
PrivateKey = <44-char base64>
PublicKey = <44-char base64>
Address = 10.0.0.1/24
ListenPort = 51820
MTU = 1280
Jc = 4
Jmin = 282
Jmax = 871
S1 = 22
S2 = 22
S3 = 22
S4 = 22
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = <44-char base64>
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
#_ClientToClient = false
#_TunName = awg0

[Peer]
#_Name = phone
PublicKey = <44-char base64>
PresharedKey = <44-char base64>
AllowedIPs = 10.0.0.2/32
```

The `<44-char base64>` placeholders stand in for the redacted key values; `HeaderProtectionKey` is base64 that decodes to 32 bytes. `Jc`/`Jmin`/`Jmax` and `S1`–`S4` were drawn randomly for this run and differ on every fresh run.

The AWG 3.1 block sits between the 2.0 obfuscation keys and the `#_` metadata and is emitted in a fixed order. The defaults this run demonstrates:

| Key | Value | Meaning |
| --- | --- | --- |
| `HeaderProtectionKey` | 44-char base64, identical on server and client | ChaCha20 key that encrypts the 4-byte message header. |
| `ContentPaddingAddition` | `2-10` | Content padding added to packets. |
| `RekeyAfterTime` | `120-180` | Jitter around the 120 s rekey interval. |
| `RekeyTimeout` | `5-8` | Jitter around the 5 s rekey timeout. |
| `RejectAfterTime` | `180-240` | Jitter around the 180 s session rejection deadline. |
| `KeepaliveTimeout` | `8-12` | Jitter around the 10 s keepalive timeout. |
| `MaxHandshakeAttempts` | `16-20` | Jitter around the 18-attempt handshake cap. |
| `RandomTrailers` | `on` | Appends random-size trailers; with trailers on, the receiver accepts sizes above the expected size. |
| `DisableCookies` | `on` | Disables the cookie-reply mechanism. |
| `H1`–`H4` | `1-1`, `2-2`, `3-3`, `4-4` | With header protection on and no `h1`–`h4` in the manifest, the plain WireGuard type-ids are used: the header is encrypted, so the type-ids are no longer observable. |
| `S1`–`S4` | One uniform random value `>= 12` (here `22`) | With random trailers on, packets are classified by size, so one `S` is drawn once and reused for all four prefixes. |

The client config carries the same `[Interface]` obfuscation block and transport-protection values, then adds the CPS strings. Because the peers in this manifest set no `protocol`, `generate` used its default QUIC template, which produced `I1`–`I4` in this run:

```ini
I1 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0040><b 0x00><b 0x01><t><r 40>
I2 = <b 0xc0ff><b 0x00000001><b 0x08><d><b 0x00><b 0x00><b 0x0020><b 0x01><t><r 20>
I3 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0010><b 0x01><t><r 10>
I4 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0005><b 0x01><t><r 5>
```

Its `[Peer]` section is the mirror image of the server's:

```ini
[Peer]
PublicKey = <44-char base64>
PresharedKey = <44-char base64>
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 0
```

Client-only details worth knowing:

- `DNS = ` is empty because `network.dns` is unset — the writer always emits the line for client configs. Add `"dns": ["1.1.1.1"]` to `network` to get a usable resolver list.
- `AllowedIPs = 0.0.0.0/0, ::/0` routes all traffic through the tunnel; the client reaches the server at `Endpoint`.
- `PersistentKeepalive = 0` appears when the peer sets no `keepalive` (the server writer omits the key entirely).
- The server config never carries `I1`–`I5`; those strings are generated per client peer.
- `#_`-prefixed lines are metadata comments, not engine keys: the next `generate` run reads `#_Name` back to match persisted credentials. See [Output Format](./output-format.md) and [Credentials](./credentials.md).

## Step 4: Validate the Config

```shell
$ amnezigo validate output/server/awg0.conf
```

```text
✓ output/server/awg0.conf: 0 errors, 0 warnings, 0 info
```

The command strict-parses the file — unknown INI keys and raw `<c>` tags become findings — and runs every invariant. Exit code `0` means no errors; any `error` exits `1`, and `--strict` also fails on warnings. With `--output json` the same summary is emitted as a JSON document with `file`, `findings`, and `summary` (`errors`/`warnings`/`info`) keys.

The rule families:

| Rule family | Findings |
| --- | --- |
| Required fields (`PrivateKey`, `Address`, `ListenPort`) | error · `FLD001` |
| Packet-size collisions across the padded sizes `S1+148`, `S2+92`, `S3+64`, `S4+32` and the junk range | error · `PSC001`/`PSC002`/`PSC003` |
| Junk-range ordering (`Jmin > Jmax`) | error · `JNK001` |
| Header ranges (contain WireGuard type-ids `1..4`, or `Max < Min`) | error · `HDR001`/`HDR002` |
| Header protection (`S1`–`S4 >= 12`; key is 44-char base64 of 32 bytes) | error · `HPK001`/`HPK003` |
| `RandomTrailers` on with unequal `S1`–`S4` | warning · `TRL001` |
| 3.x range structure (`Max < Min`, or exactly one zero bound) | error · `TRM001` |
| Pre-parse checks (unknown INI key; raw `<c>` tag) | warning · `KEY001`/`CPS001` |

> **Note:** `HDR001` is suppressed when a `HeaderProtectionKey` is present, which is why the default `H1 = 1-1` config validates cleanly even though `1..4` are WireGuard type-ids.

All codes, severities, and the JSON shape are in [Validation & Analysis](./validation.md).

## Step 5: Analyze the Config

`analyze` reads a config — by default `./awg0.conf`, so pass `--config` — and prints a profile without modifying anything:

```shell
$ amnezigo analyze --config output/server/awg0.conf
```

```text
=== AmneziaWG Config Analysis ===

MTU: 1280 | Port: 51820 | Peers: 1 | Protocol: random

--- Handshake Sizes ---
  Init:      S1=22 + 148 = 170 bytes
  Response:  S2=22 + 92 = 114 bytes
  Cookie:    S3=22 + 64 = 86 bytes
  Transport: S4=22 + 32 = 54 bytes

--- Junk Packets ---
  Count: 4 (Jc) | Range: [282..871] | Width: 590 B

--- Header Ranges ---
  H1: [1..1] (width 1)
  H2: [2..2] (width 1)
  H3: [3..3] (width 1)
  H4: [4..4] (width 1)

--- I-Packets (per peer) ---
  Peer "phone":
    i1=53  i2=41  i3=86  i4=115  i5=40 bytes

--- Wire Ordering ---
  1. i1 -> i2 -> i3 -> i4 -> i5
  2. junk x 4
  3. Handshake Init

Note: I-packet sizes are freshly generated from config parameters and may differ on each run.
```

The report contains Handshake Sizes, Junk Packets, Header Ranges, I-Packets (per peer), and Wire Ordering sections, plus an optional Findings block with `RISK001`–`RISK009` entries.

| Property | Value |
| --- | --- |
| Default input | `./awg0.conf`; use `--config <path>` for generated configs |
| Protocol | `--protocol` defaults to `random`, and the report **regenerates** I-packets with it |
| Samples | `--samples N` adds min/max/mean/median distributions per I-packet |
| Output | `--output text` (default) or `json` |
| Exit code | `0` even when findings are printed; `1` only for a load failure or an invalid `--output` value |

Two things regularly surprise people:

- The I-packet numbers are freshly generated samples, not the `I1`–`I5` stored in the file. This config was generated with the default QUIC template while `analyze` defaulted to `random`, so the numbers above do not correspond to the config's `I1`–`I4`; pass `--protocol quic` to match the generator's default.
- `--seed N` does **not** make the output reproducible. The CLI assigns a seeded reader to `AnalyzeOptions.Rand`, but no analysis code path reads that field, so runs stay non-deterministic. Do not gate CI on stable `analyze` numbers.

Because findings never fail the command, a CI gate has to parse `--output json` instead of relying on the exit code. The full RISK catalog is in [Validation & Analysis](./validation.md), and the 3.x interplay is in [Transport Protection](./transport-protection.md).

## Step 6: Check the Version

```shell
$ amnezigo version
```

```text
amnezigo dev (none)
```

The format is `amnezigo <version> (<commit>)`, and the stamp depends on how the binary was produced:

| Build | Output | Notes |
| --- | --- | --- |
| `go install` or plain `go build` | `amnezigo dev (none)` | No build stamp. |
| `make build` | `amnezigo v0.4.0 (abc1234)` | The version comes from `git describe`; non-tag commits get a suffix such as `amnezigo v0.4.0-5-gabc1234-dirty (abc1234)`. |
| Release binary | `amnezigo 0.4.0 (abc1234)` | GoReleaser strips the tag's leading `v`. |

Quote the full line, commit included, when reporting a problem, and check it before deploying 3.x configs to an existing runtime. Details: [Installation](./installation.md).

## Flag Variations

The `generate` flags you are most likely to reach for:

| Flag | Effect |
| --- | --- |
| `--dry-run` | Computes and prints the same file list with byte counts, prefixed by `Dry run — no files written`, and writes nothing. |
| `--full-reset` | Discards every persisted credential — server/client key pairs, all peer PSKs, and the `HeaderProtectionKey` — and regenerates them. Every distributed client config stops working against the new server config. |
| `--vpn-links` | Adds one `amnezigo.vpn` import file per client peer. |
| `--peer <name>` | Generates only that client peer; the server config is always regenerated. A name that matches no peer is silently ignored, so `--peer nobody` prints just the server config and still exits `0`. |
| `--project <dir>` / `--output <dir>` | Read the manifest from another directory / write configs elsewhere. This `--output` is a directory, while `validate --output` and `analyze --output` select the output format. |

One `--dry-run` subtlety:

- If `output/` already holds a previous run, the dry run reads those persisted keys and previews the sizes a real run would produce (the random S/J values are still re-drawn, so counts can shift by a byte or two).
- If there is no `output/` yet, the dry run generates fresh key material in memory and throws it away — a following real run writes different keys than the preview showed.

## Re-running Generate

Re-running `generate` in the same project reuses credentials but is not a no-op:

- **Key material is reused.** Server and client `PrivateKey` values, every `PresharedKey`, and the `HeaderProtectionKey` are loaded back from the existing configs under `output/`. Six consecutive runs on this fixture produced byte-identical key material.
- **Unpinned obfuscation parameters are re-drawn.** Every field left unset in the manifest is generated again from `crypto/rand` — `S1`–`S4` and `Jc`/`Jmin`/`Jmax` (the `H1`–`H4` ranges stay `1-1`…`4-4` while header protection is on). In the same six-run check `S` came out as `31, 31, 21, 17, 22, 23` and the junk range changed every run, for example `311-995` and then `545-682`.

The consequence: a regenerated server config no longer matches client configs distributed from an earlier run, even though the keys did not change. Only files produced by the same run are mutually consistent. To make runs byte-stable, pin the parameters in the manifest:

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "awg_version": "3.1",
    "s1": 22,
    "s2": 22,
    "s3": 22,
    "s4": 22,
    "jc": 4,
    "jmin": 282,
    "jmax": 871
  },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": {
      "address": "10.0.0.2/32"
    }
  }
}
```

Two related behaviors:

- `--full-reset` skips reuse entirely and rotates everything, including the `HeaderProtectionKey` — a fleet-wide re-provisioning action, not a local refresh.
- Deleting `output/` destroys the credential store: the next run generates a brand-new identity, and clients provisioned from the old server config stop connecting. The output tree *is* the state, so keep it backed up and treat it as sensitive.

## Related

- [Installation](./installation.md) — every install path and runtime version alignment.
- [Manifest Reference](./manifest-reference.md) — all fields, types, defaults, and pointer-nil semantics.
- [Output Format](./output-format.md) — the full INI key order, `#_` metadata, and server/client differences.
- [Transport Protection](./transport-protection.md) — the AWG 3.x layer behind the defaults shown above.
- [Validation & Analysis](./validation.md) — every finding code, RISK heuristic, and exit-code rule.
- [Credentials](./credentials.md) — how `generate` persists and reuses keys, and what `--full-reset` destroys.
