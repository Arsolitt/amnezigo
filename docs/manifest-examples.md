# Manifest Examples

> Complete manifests for common setups — from the smallest valid two-peer file to AWG 3.0/3.1 transport-protection profiles and Jsonnet-generated peers. Every example is built from repository fixtures and test manifests, and each states what it exercises and what the generator emits.

## Table of Contents

- [Conventions](#conventions)
- [Minimal Two-Peer Manifest](#minimal-two-peer-manifest)
- [AWG 3.1 Defaults](#awg-31-defaults)
- [Pinned AWG 2.0](#pinned-awg-20)
- [Multi-Peer Manifest with Per-Peer Fields](#multi-peer-manifest-with-per-peer-fields)
- [Fully Random Obfuscation](#fully-random-obfuscation)
- [Explicit Preset Values (home-balanced)](#explicit-preset-values-home-balanced)
- [AWG 3.0 with Explicit Transport Ranges](#awg-30-with-explicit-transport-ranges)
- [AWG 3.1 with Disabled Range and Header Protection](#awg-31-with-disabled-range-and-header-protection)
- [Jsonnet Examples](#jsonnet-examples)
- [Where These Examples Come From](#where-these-examples-come-from)

---

## Conventions

Three rules apply to every example on this page:

1. **`version` MUST be `1`.** It is the only schema version the loader accepts. An omitted or zero value fails with `missing or zero version field`; any other value fails with `unsupported schema version N (expected 1)`. A Jsonnet manifest must evaluate to `version: 1` in its output object, not a Jsonnet-level variable.
2. **An omitted `awg_version` means AmneziaWG 3.1.** The default version is 3.1 whenever the field is absent, so most manifests receive the full transport-protection block.
3. **`.amnezigo.jsonnet` takes precedence over `amnezigo.json`.** When both files exist in the project directory, the JSON file is ignored entirely.

Run an example from its project directory; `--dry-run` computes the full output and prints the warning list without writing files:

```shell
$ amnezigo generate --dry-run
$ amnezigo generate
```

`generate` re-parses the server config it just produced and reports findings after the file list, one line per finding:

```text
Warnings: 1
[WARNING TRL001]  (key=RandomTrailers): RandomTrailers is enabled while S1..S4 differ; the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification
```

TRL001 is the warning these examples trigger most often; it appears whenever `RandomTrailers` is on and `S1`–`S4` are not all equal. All finding codes are catalogued in [./validation.md](./validation.md). Output excerpts below show only `[Interface]` keys relevant to the example — full key order and the `[Peer]` layout are in [./output-format.md](./output-format.md).

## Minimal Two-Peer Manifest

The smallest useful manifest: `network` and `obfuscation` may be omitted entirely, and a client peer needs nothing but `address`. This is a reduced form of the manifest used by `TestGenerate_MinimalManifest`, which additionally sets `network.mtu` and `network.dns` (the test manifest therefore keeps its I-packets intact).

```json
{
  "version": 1,
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

| Field | Value | Why |
| --- | --- | --- |
| `version` | `1` | Schema version; the only accepted value. |
| `peers.server.address` | `10.0.0.1/24` | Server tunnel address with the client pool behind it. |
| `peers.server.endpoint` + `listen_port` | `vpn.example.com:51820`, `51820` | The two markers that identify the server peer. Exactly one peer may set them. |
| `peers.phone.address` | `10.0.0.2/32` | Client host route; no `endpoint`/`listen_port` means "client". |

With `awg_version` omitted the manifest resolves to 3.1 defaults: one generated S value used for all of `S1`–`S4`, `H1`–`H4` fixed to `1-1`/`2-2`/`3-3`/`4-4`, generated `Jc`/`Jmin`/`Jmax`, a generated header-protection key, and the full transport block. The client config looks like this (abridged; generated values shown as placeholders):

```ini
Jc = <generated>
Jmin = <generated>
Jmax = <generated>
S1 = <generated, all four equal>
S2 = <generated>
S3 = <generated>
S4 = <generated>
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = <generated 44-char base64>
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
I1 = <t>
I2 = <t>
I3 = <t>
I4 = <t>
I5 = <t>
```

No TRL001 is reported here: the four S values are generated as one uniform value, which is what the trailer check wants.

> **Warning:** Because this manifest omits `network.mtu`, the emitted file still says `MTU = 1280`, but the CPS generator reads the raw (unset) MTU first and every client I-packet collapses to the bare `<t>` — exactly the `I1`–`I5` lines above, with all protocol mimicry lost. Set `network.mtu` explicitly (as every following example does) until this defect is fixed; see [./gotchas.md](./gotchas.md).

## AWG 3.1 Defaults

This is `testdata/loader/valid/amnezigo-31.json`, the canonical "explicitly target 3.1" manifest and the recommended starting point. It resolves identically to the minimal manifest — 3.1 is already the default — but pins `awg_version` and `network.mtu` so the file is self-describing and the client I-packets stay intact.

Exercises: `network.mtu`, `obfuscation.awg_version`, and the 3.1 defaults.

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

The loader test asserts that this fixture pins no S/H/J value at all (`HasAnyValue() == false`), and that the generated server config contains:

```ini
HeaderProtectionKey = <generated>
ContentPaddingAddition = 2-10
RandomTrailers = on
H1 = 1-1
```

With `MTU = 1280` set, each client's default `quic` template emits `I1`–`I4` (the named templates define no `I5`). No warnings are reported.

## Pinned AWG 2.0

This is `testdata/loader/valid/amnezigo.json`, the project's pre-3.x fixture: it pins `awg_version` to `2.0` and declares every obfuscation value, so none of the AWG 3.x INI keys may appear in its output.

Exercises: `obfuscation.awg_version` pinning, explicit S/H/J values, and the fact that `obfuscation.protocol` is dead configuration.

```json
{
  "version": 1,
  "network": {
    "mtu": 1280
  },
  "obfuscation": {
    "awg_version": "2.0",
    "protocol": "quic",
    "s1": 30, "s2": 35, "s3": 20, "s4": 12,
    "h1": {"min": 100, "max": 5000000},
    "h2": {"min": 10000000, "max": 200000000},
    "h3": {"min": 400000000, "max": 800000000},
    "h4": {"min": 1000000000, "max": 2100000000},
    "jc": 5, "jmin": 250, "jmax": 750
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

The generator uses the declared values verbatim, so the server config contains exactly:

```ini
Jc = 5
Jmin = 250
Jmax = 750
S1 = 30
S2 = 35
S3 = 20
S4 = 12
H1 = 100-5000000
H2 = 10000000-200000000
H3 = 400000000-800000000
H4 = 1000000000-2100000000
#_ClientToClient = false
#_TunName = awg0
```

For `awg_version` 2.0 there is no transport-protection block at all: no `HeaderProtectionKey`, no `ContentPaddingAddition`, none of the timer or handshake ranges, and no `RandomTrailers`/`DisableCookies` lines. Clients still receive generated `I1`–`I4` (the I-packet specs depend on the protocol, MTU, and S-prefix bound — not on `Jc`).

> **Note:** `obfuscation.protocol` is never read by `generate`; only `peers.<name>.protocol` selects the CPS template, and it defaults to `quic`. The fixture keeps the field, but it has no effect on the output. See [./obfuscation.md](./obfuscation.md).

> **Warning:** Manifest fields from a newer generation are hard errors under 2.0, not silently ignored — for example `"header_protection": false` fails with `obfuscation.header_protection requires awg_version 3.0 or later (got "2.0")`. See [./transport-protection.md](./transport-protection.md) for the version gates.

## Multi-Peer Manifest with Per-Peer Fields

One server plus two clients. Exercises: `network.dns`, per-peer `protocol` and `keepalive` and `display_name`, and the server-only `tun_name`/`main_iface` fields.

```json
{
  "version": 1,
  "network": {
    "mtu": 1280,
    "dns": ["1.1.1.1", "8.8.8.8"]
  },
  "obfuscation": {
    "awg_version": "3.1",
    "s1": 30, "s2": 35, "s3": 20, "s4": 12,
    "jc": 5, "jmin": 250, "jmax": 750
  },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "tun_name": "awg0",
      "main_iface": "eth0",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": {
      "address": "10.0.0.2/32",
      "protocol": "sip",
      "keepalive": 25,
      "display_name": "Alice phone"
    },
    "laptop": {
      "address": "10.0.0.3/32"
    }
  }
}
```

What the output looks like:

- The server gains `PostUp`/`PostDown` rules because `main_iface` is set, plus `#_MainIface = eth0` metadata. Each value spans two lines: the IPv4 chain, then the same chain with `ip6tables`, all rules joined with `; `. The `PostUp` value begins with:

```text
iptables -A INPUT -i awg0 -j ACCEPT; iptables -A OUTPUT -o awg0 -j ACCEPT; iptables -A FORWARD -i awg0 -o eth0 -s 10.0.0.0/24 -j ACCEPT; iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT; iptables -A FORWARD -i eth0 -o awg0 -d 10.0.0.0/24 -m state --state ESTABLISHED,RELATED -j ACCEPT; iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
```

- `PostDown` carries the same chains with `-D` instead of `-A`. The `10.0.0.0/24` subnet is derived from `peers.server.address`.
- Both clients receive `DNS = 1.1.1.1, 8.8.8.8` and `AllowedIPs = 0.0.0.0/0, ::/0`.
- `phone` gets `PersistentKeepalive = 25`; `laptop`, which sets no `keepalive`, gets `PersistentKeepalive = 0`.
- `phone`'s I-packets use the `sip` template; `laptop`'s use the default `quic`.
- `display_name` never appears in `awg0.conf` — it is only consumed by `vpn://` import links.
- Server `[Peer]` sections are sorted by peer name (`laptop` before `phone`).

> **Warning:** This manifest reports one `TRL001` warning per run: its explicit S values (`30, 35, 20, 12`) differ while `RandomTrailers` is on by default. Set `"random_trailers": false` or equalize the S values to silence it.

> **Note:** Setting **both** `endpoint` and `listen_port` on a client makes it count as a second server and aborts generation (`exactly one server peer required, found 2`); either marker alone does not. `tun_name`, `main_iface`, and `display_name` have no effect on client configs. Valid template names are `quic`, `dns`, `dtls`, `stun`, `sip`, `rtp`, and `random`; an empty value defaults to `quic`.

## Fully Random Obfuscation

An empty `obfuscation` object (or omitting the key entirely) asks the generator to resolve every obfuscation parameter itself.

Exercises: the default resolution path under AmneziaWG 3.1.

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": {},
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "phone": { "address": "10.0.0.2/32" }
  }
}
```

Because the target version defaults to 3.1, this is not "all values random" in the pre-3.x sense. The resolution is:

- `S1`–`S4` — one uniform generated value (≥ 12) shared by all four, because random trailers are on and all four S fields are unset.
- `H1`–`H4` — the fixed reference windows `1-1`, `2-2`, `3-3`, `4-4` while header protection is on.
- `Jc`/`Jmin`/`Jmax` — generated.
- `HeaderProtectionKey` — generated (44-character base64), identical in the server config and every client config, and persisted for reuse on later runs.
- The rest of the transport block takes defaults: `ContentPaddingAddition = 2-10`, the five other ranges (`RekeyAfterTime`, `RekeyTimeout`, `RejectAfterTime`, `KeepaliveTimeout`, `MaxHandshakeAttempts`), `RandomTrailers = on`, and `DisableCookies = on`.

The emitted config has the same shape as the [AWG 3.1 defaults](#awg-31-defaults) example; only the generated S and J values differ from run to run. No TRL001 is reported, because the generated S values are uniform.

> **Note:** Generated values are re-drawn on every `generate`; only keys and the header-protection key persist in the output tree. Two consecutive runs on this manifest produce different S/J values; the emitted `I1`–`I5` lines are deterministic for a given protocol/MTU and S bound, so they do not change between runs. Pin the fields you care about — pinned values are used verbatim — if you need a stable config.

## Explicit Preset Values (home-balanced)

There is no `preset` field in the manifest schema and no importable preset library, so using a preset means copying its numbers into `obfuscation.*`. This manifest reproduces the `home-balanced` preset including its 3.x fields.

Exercises: explicit `s1`–`s4`, `h1`–`h4`, `jc`/`jmin`/`jmax`, `content_padding`, `random_trailers`, and `disable_cookies`.

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": {
    "awg_version": "3.1",
    "s1": 30, "s2": 35, "s3": 20, "s4": 12,
    "h1": {"min": 100, "max": 5000000},
    "h2": {"min": 10000000, "max": 200000000},
    "h3": {"min": 400000000, "max": 800000000},
    "h4": {"min": 1000000000, "max": 2100000000},
    "jc": 5, "jmin": 250, "jmax": 750,
    "content_padding": {"min": 2, "max": 10},
    "random_trailers": true,
    "disable_cookies": true
  },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "client": { "address": "10.0.0.2/32", "protocol": "quic" }
  }
}
```

The generator emits exactly the declared values and ranges:

```ini
Jc = 5
Jmin = 250
Jmax = 750
S1 = 30
S2 = 35
S3 = 20
S4 = 12
H1 = 100-5000000
H2 = 10000000-200000000
H3 = 400000000-800000000
H4 = 1000000000-2100000000
HeaderProtectionKey = <generated>
ContentPaddingAddition = 2-10
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = on
DisableCookies = on
```

> **Warning:** This manifest reports exactly one `TRL001` warning per run: `home-balanced` has unequal S values while `RandomTrailers` is on. Four of the seven presets (`home-balanced`, `mobile-aggressive`, `stealth-paranoid`, `standard-1420`) behave the same way. Set `"random_trailers": false` or equalize the S values to silence it. All preset values are listed in [./presets.md](./presets.md).

> **Note:** The preset also carries `ContentPadding`, `RandomTrailers`, and `DisableCookies`. A manifest that copies only the S/H/J numbers still generates the same profile, because those 3.x defaults coincide with the preset values — but it still reports TRL001.

## AWG 3.0 with Explicit Transport Ranges

Pinning the target to `3.0` produces the transport-protection block without the 3.1-only booleans.

Exercises: `awg_version: "3.0"`, `content_padding`, `rekey_after_time`, and header protection enabled by default.

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": {
    "awg_version": "3.0",
    "s1": 20, "s2": 21, "s3": 22, "s4": 23,
    "jc": 4, "jmin": 300, "jmax": 700,
    "content_padding": {"min": 4, "max": 12},
    "rekey_after_time": {"min": 120, "max": 150}
  },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "client": { "address": "10.0.0.2/32" }
  }
}
```

Server output (abridged):

```ini
S1 = 20
S2 = 21
S3 = 22
S4 = 23
H1 = 1-1
H2 = 2-2
H3 = 3-3
H4 = 4-4
HeaderProtectionKey = <generated>
ContentPaddingAddition = 4-12
RekeyAfterTime = 120-150
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
```

- Header protection is on by default from 3.0, so the key, the two declared ranges, and the four untouched range defaults are emitted together; `H1`–`H4` keep the reference `1-1`/`2-2`/`3-3`/`4-4` values.
- A 3.0 config contains neither `RandomTrailers` nor `DisableCookies` — those keys are 3.1-only — and no TRL001 can occur even though the S values differ.
- Setting either field under 3.0 is a hard error: `obfuscation.random_trailers requires awg_version 3.1 or later (got "3.0")`.

## AWG 3.1 with Disabled Range and Header Protection

This manifest turns off header protection, disables one transport range, and turns random trailers off.

Exercises: `header_protection: false`, `content_padding` `{0, 0}`, and `random_trailers: false`.

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": {
    "awg_version": "3.1",
    "header_protection": false,
    "content_padding": {"min": 0, "max": 0},
    "random_trailers": false
  },
  "peers": {
    "server": { "address": "10.0.0.1/24", "endpoint": "vpn.example.com:51820", "listen_port": 51820 },
    "phone": { "address": "10.0.0.2/32" }
  }
}
```

Server output (abridged; generated values shown as placeholders):

```ini
S1 = <generated>
S2 = <generated>
S3 = <generated>
S4 = <generated>
H1 = <generated wide range>
H2 = <generated wide range>
H3 = <generated wide range>
H4 = <generated wide range>
RekeyAfterTime = 120-180
RekeyTimeout = 5-8
RejectAfterTime = 180-240
KeepaliveTimeout = 8-12
MaxHandshakeAttempts = 16-20
RandomTrailers = off
DisableCookies = on
```

What changes:

- `header_protection: false` removes only the `HeaderProtectionKey` line. The remaining transport ranges and the 3.1 booleans stay (only the range you disabled disappears), and because header protection is off the `H1`–`H4` windows are generated wide ranges instead of the reference `1-1`/`2-2`/`3-3`/`4-4`.
- `content_padding: {min: 0, max: 0}` disables that key: no `ContentPaddingAddition` line is emitted.
- `random_trailers: false` prints `RandomTrailers = off`; `DisableCookies` remains `on` (its default).
- The S ≥ 12 floor applies only while header protection is on, so with it off the generated S values may be unequal and may fall below 12.
- No TRL001 is reported, because random trailers are off.

> **Note:** `{min: 0, max: 0}` is the disabled state for a range; a half-zero range such as `{"min": 0, "max": 5}` is a hard error (`bounds must both be zero or both non-zero (got 0-5)`), and `{"min": 0}` with no `max` decodes to `{0, 0}` — also disabled.

## Jsonnet Examples

A manifest may be written as `.amnezigo.jsonnet`; the loader evaluates it to JSON and applies the same schema. The default import search path is a `lib/` directory next to the manifest, and `--jpath` replaces it. No Jsonnet library ships with amnezigo — `lib/` is a convention you create. Full evaluation rules are in [./jsonnet.md](./jsonnet.md).

### Shared Network Library

Exercises: a relative import resolved from the default `lib/` path (`testdata/loader/valid-jsonnet-with-lib/`).

```jsonnet
// .amnezigo.jsonnet
local net = import 'network.libsonnet';
{
  version: 1,
  network: net,
  obfuscation: {
    protocol: 'quic',
    s1: 30, s2: 35, s3: 20, s4: 12,
    h1: { min: 100, max: 5000000 },
    h2: { min: 10000000, max: 200000000 },
    h3: { min: 400000000, max: 800000000 },
    h4: { min: 1000000000, max: 2100000000 },
    jc: 5, jmin: 250, jmax: 750,
  },
  peers: {
    server: {
      address: '10.0.0.1/24',
      endpoint: 'vpn.example.com:51820',
      listen_port: 51820,
    },
  },
}
```

```jsonnet
// lib/network.libsonnet
{
  mtu: 1420,
  dns: ['1.1.1.1', '8.8.8.8'],
}
```

The evaluated manifest carries `MTU = 1420` and the two DNS servers from the imported file; the `lib/` import resolves with no `--jpath` needed. With only a server peer defined, `generate` writes a single config. This manifest pins unequal S values, so it reports one TRL001 warning.

> **Note:** Passing `--jpath` replaces the implicit `lib/` path rather than appending to it. Because amnezigo bundles no library, a manifest that wants shared fragments must ship its own `lib/` beside the manifest.

### Peer Comprehension

Exercises: `std.range`, string formatting, an object comprehension, and an object merge that appends generated peers to the server (from `TestLoadManifest_LargeManifest_ManyPeers`).

```jsonnet
// .amnezigo.jsonnet
local peers = {
    ['peer-%03d' % i]: { address: '10.0.%d.%d/32' % [i / 256, i % 256] }
    for i in std.range(1, 100)
};
{
    version: 1,
    network: { mtu: 1280 },
    obfuscation: {},
    peers: { server: { address: '10.0.0.1/24', endpoint: 'vpn:51820', listen_port: 51820 } } + peers,
}
```

The result is 101 peers → 101 configs: the server plus `peer-001` through `peer-100`. The empty `obfuscation` object means uniform generated S values, so no TRL001 is reported.

> **Danger:** Neither the loader nor the pipeline validates peer addresses or names for uniqueness. In this example `peer-001` receives `10.0.0.1/32` — the same address as the server — and generation still succeeds. Keep generated peer names and addresses unique yourself; nothing will reject a collision before the configs are written.

### Jsonnet over JSON Precedence

Exercises: discovery precedence when both manifest files exist (`testdata/loader/precedence/`).

```jsonnet
// .amnezigo.jsonnet
{
  version: 1,
  network: { mtu: 1280 },
  obfuscation: {
    protocol: 'random',
    s1: 5, s2: 7, s3: 7, s4: 7,
    h1: { min: 5, max: 10000 },
    h2: { min: 20000, max: 50000 },
    h3: { min: 100000, max: 500000 },
    h4: { min: 1000000, max: 5000000 },
    jc: 1, jmin: 200, jmax: 250,
  },
  peers: {
    server: { address: '10.0.0.1/24', endpoint: 'from-jsonnet:51820', listen_port: 51820 },
  },
}
```

```json
{
  "version": 1,
  "network": {"mtu": 1280},
  "obfuscation": {
    "protocol": "random",
    "s1": 5, "s2": 7, "s3": 7, "s4": 7,
    "h1": {"min": 5, "max": 10000},
    "h2": {"min": 20000, "max": 50000},
    "h3": {"min": 100000, "max": 500000},
    "h4": {"min": 1000000, "max": 5000000},
    "jc": 1, "jmin": 200, "jmax": 250
  },
  "peers": {"server": {"address": "10.0.0.1/24", "endpoint": "from-json:51820", "listen_port": 51820}}
}
```

The sibling `amnezigo.json` is never read: the loaded endpoint is `from-jsonnet:51820`, not `from-json:51820`.

> **Danger:** This fixture demonstrates loader precedence only — it is not a usable profile. Under the default 3.1 target `generate` fails with `resolve obfuscation: header protection requires S1-S4 >= 12 (got S1=5)`, and its near-equal S values are not a recommended obfuscation profile. Use it to test which file wins, not to route traffic.

## Where These Examples Come From

- There is no committed `examples/` directory; every manifest above comes from repository fixtures or test manifests.
- `testdata/loader/valid/amnezigo-31.json` → [AWG 3.1 defaults](#awg-31-defaults); `testdata/loader/valid/amnezigo.json` → [pinned AWG 2.0](#pinned-awg-20).
- `testdata/loader/valid-jsonnet-with-lib/` → [shared network library](#shared-network-library); `testdata/loader/precedence/` → [Jsonnet over JSON precedence](#jsonnet-over-json-precedence).
- The [minimal](#minimal-two-peer-manifest) example follows `TestGenerate_MinimalManifest` in `pipeline_test.go`; the [multi-peer](#multi-peer-manifest-with-per-peer-fields) layout follows `manifest_test.go`'s three-peer round-trip fixture with `keepalive`/`display_name` added from other tests; the [comprehension](#peer-comprehension) manifest is the exact string embedded in `loader_test.go`.
- The [fully random](#fully-random-obfuscation) and [preset-value](#explicit-preset-values-home-balanced) manifests combine fixture-proof pipeline behavior with the values in `presets.go`.

When adapting an example, remember that unpinned obfuscation values are re-drawn on every run, and that `"s1": 0` does not disable the prefix — a zero S is replaced by a generated value (unlike `"content_padding": {"min": 0, "max": 0}`, which genuinely disables its key).

## Related

- [Manifest Reference](./manifest-reference.md) — every manifest field, pointer-nil semantics, and the loader API.
- [Transport Protection (AWG 3.x)](./transport-protection.md) — the version model, the ten 3.x manifest knobs, and their gates.
- [Presets](./presets.md) — the seven built-in profiles whose values these examples copy by hand.
- [Jsonnet](./jsonnet.md) — evaluation rules, `--jpath`, and import behavior.
- [Output Format](./output-format.md) — the full `awg0.conf` key order and `#_` metadata.
- [Validation & Analysis](./validation.md) — every finding code these examples can print, including TRL001.
