# Jsonnet

> Authoring amnezigo manifests with `.amnezigo.jsonnet` — precedence, the `--jpath`
> flag, and worked examples against the real Manifest schema.

## Table of Contents

- [Why Jsonnet](#why-jsonnet)
- [Discovery & Precedence](#discovery--precedence)
- [The `--jpath` Flag](#the---jpath-flag)
- [Worked Examples](#worked-examples)
- [Gotchas](#gotchas)
- [Related](#related)

---

## Why Jsonnet

A `.amnezigo.jsonnet` manifest is a [Jsonnet](https://jsonnet.org) program that
**evaluates to a JSON document matching the Manifest schema**. It lets you compute
or derive manifest values instead of hard-coding them: factor shared obfuscation
parameters into a library, parametrize peer addressing, or import a preset's
numeric profile from [./presets.md](./presets.md). amnezigo evaluates the program
with `google/go-jsonnet`, JSON-parses the output into a `Manifest`, and runs the
same version validation as a plain `amnezigo.json` — so the deliverable is always
a normal manifest, however it was produced.

## Discovery & Precedence

`LoadManifest(dir, jpathDirs)` probes two well-known filenames in the project
directory. The Jsonnet file is checked **first**; if it exists, the JSON file is
ignored entirely (no merge, no warning).

| File                     | Precedence | Rule                                                              |
|--------------------------|------------|-------------------------------------------------------------------|
| `.amnezigo.jsonnet`      | 1 (wins)   | `os.Stat` succeeds → evaluated as Jsonnet via `loadFromJsonnet`.  |
| `amnezigo.json`          | 2 (fallback) | Tried only if the Jsonnet file is absent → `loadFromJSON`.       |
| *(neither present)*      | —          | Error: `no manifest file found in <dir> (expected .amnezigo.jsonnet or amnezigo.json)`. |

Discovery is by exact filename (constants in `loader.go`): the Jsonnet file is
**`.amnezigo.jsonnet`** — note the leading dot. A file named `amnezigo.jsonnet`
(no dot) is **not** discovered; only `LoadManifestFromFile` with an explicit path
evaluates it.

| `LoadManifest` argument | Type       | Semantics                                                                                      |
|-------------------------|------------|------------------------------------------------------------------------------------------------|
| `dir`                   | `string`   | Project directory scanned for the two manifest filenames.                                      |
| `jpathDirs`             | `[]string` | Jsonnet library search paths. `nil`/empty → defaults to **`[dir/lib]`** via `resolveJpath`.     |

`LoadManifestFromFile(path, jpathDirs)` uses the file **extension** to pick the
evaluator: a `.jsonnet` suffix → Jsonnet VM, anything else → plain JSON. Its
jpath default is `[parentDir/lib]`.

## The `--jpath` Flag

The `generate` command exposes Jsonnet library search paths as a repeatable flag
(`internal/cli/generate.go`). The slice is passed straight through to
`amnezigo.LoadManifest(projectDir, jpathDirs)`.

| Flag      | Type                | Default            | Effect                                                                                    |
|-----------|---------------------|--------------------|-------------------------------------------------------------------------------------------|
| `--jpath` | `[]string` (repeatable, comma-separated) | `nil` | Adds library search dirs for `import` / `importstr`. Empty/omitted → `resolveJpath` returns `[<project>/lib]`. The VM is wired with a `jsonnet.FileImporter{JPaths: …}`. |

```shell
# Default jpath: $PROJECT/lib is searched automatically.
$ amnezigo generate --project ./my-vpn

# Explicit jpath: only the listed dirs are searched (default is overridden).
$ amnezigo generate --project ./my-vpn --jpath ./jsonnet-lib --jpath ./vendor
```

> **Note:** Passing any `--jpath` value **overrides** the default `[<project>/lib]`
> — it does not append. Repeat the flag (or use comma separation) to add multiple
> dirs in one invocation.

## Worked Examples

All three programs evaluate to a JSON document conforming to the `Manifest`
struct (`version: 1` + `network` + `obfuscation` + `peers`). See
[./manifest-reference.md](./manifest-reference.md) for the field-by-field schema.

### (a) Minimal manifest with `local` and a computed field

```jsonnet
local prefix = '10.0.0';

{
  version: 1,
  network: { mtu: 1280 },
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
      address: prefix + '.1/24',
      endpoint: 'vpn.example.com:51820',
      listen_port: 51820,
    },
    laptop: { address: prefix + '.3/32' },
  },
}
```

| Construct                  | Meaning                                                      |
|----------------------------|--------------------------------------------------------------|
| `local NAME = EXPR;`       | Bind a value visible in the following expression.            |
| `+` on strings             | String concatenation; here builds `10.0.0.1/24` from `prefix`. |
| `{ … }` (object literal)   | Evaluates to JSON; field names become JSON keys.            |

### (b) Importing a preset's values into `obfuscation`

Project layout (jpath default `[<project>/lib]` resolves the import):

```text
my-vpn/
├── .amnezigo.jsonnet
└── lib/
    └── preset.libsonnet
```

`lib/preset.libsonnet` — encodes the `home-balanced` numeric profile (see
[./presets.md](./presets.md) for the source values):

```jsonnet
{
  mtu: 1280,
  obfuscation: {
    s1: 30, s2: 35, s3: 20, s4: 12,
    h1: { min: 100, max: 5000000 },
    h2: { min: 10000000, max: 200000000 },
    h3: { min: 400000000, max: 800000000 },
    h4: { min: 1000000000, max: 2100000000 },
    jc: 5, jmin: 250, jmax: 750,
  },
}
```

`.amnezigo.jsonnet`:

```jsonnet
local preset = import 'preset.libsonnet';

{
  version: 1,
  network: { mtu: preset.mtu },
  obfuscation: preset.obfuscation {
    protocol: 'quic',
  },
  peers: {
    server: {
      address: '10.0.0.1/24',
      endpoint: 'vpn.example.com:51820',
      listen_port: 51820,
    },
    phone: { address: '10.0.0.5/32' },
  },
}
```

| Construct                          | Meaning                                                            |
|------------------------------------|--------------------------------------------------------------------|
| `import 'FILE.libsonnet'`          | Loads a Jsonnet module from a `--jpath` entry (here `<project>/lib`). |
| `OBJ_A OBJ_B` (juxtaposition)      | Object merge: right-side fields override left; new fields are added. `preset.obfuscation { protocol: 'quic' }` overlays `protocol` on the imported object. |
| `preset.mtu` / `preset.obfuscation`| Field access into an imported object.                              |

### (c) Generating N client peers from a comprehension

A server plus a templated block of client peers, built with a Jsonnet object
comprehension. Only the server peer carries `endpoint` + `listen_port`; every
other peer is a client by the `IsServer()` rule.

```jsonnet
local endpoint = 'vpn.example.com:51820';
local clientNames = ['laptop', 'phone', 'tablet', 'desktop', 'guest'];

local client(i) = {
  address: '10.0.0.' + std.toString(i + 2) + '/32',
};

{
  version: 1,
  network: { mtu: 1280 },
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
      endpoint: endpoint,
      listen_port: 51820,
    },
  } + {
    [name]: client(i)
    for i, name in clientNames
  },
}
```

| Construct                                | Meaning                                                            |
|------------------------------------------|--------------------------------------------------------------------|
| `local fn(arg) = EXPR;`                  | Bind a parameterized local function.                               |
| `std.toString(n)`                        | Convert a number to its JSON string form.                          |
| `{ [KEY]: VAL for i, x in ARR }`         | Object comprehension: emits one object field per array element; `i` is the index, `x` the value. Computed key in `[ … ]`. |
| `OBJ_A + OBJ_B`                          | Object concatenation (same semantics as juxtaposition).            |

## Gotchas

| Gotcha | Detail |
|---|---|
| **Output must be valid Manifest JSON** | The evaluated JSON is parsed into `Manifest` and version-validated exactly like `amnezigo.json`. `version` must be `1`; any other value is rejected as `unsupported schema version N (expected 1)`. See [./manifest-reference.md](./manifest-reference.md). |
| **The dot prefix is mandatory** | Auto-discovery looks for **`.amnezigo.jsonnet`** (leading dot). A file named `amnezigo.jsonnet` is silently skipped by `LoadManifest`. |
| **Default jpath is `[<dir>/lib]`, not empty** | When `--jpath` is omitted, `resolveJpath` returns `[dir/lib]`. An `import` outside that directory needs an explicit `--jpath`. |
| **`--jpath` overrides, not appends** | Supplying any `--jpath` value replaces the default `[<dir>/lib]`. Repeat the flag (or comma-separate) to list multiple dirs. |
| **Imports resolve relative-first, then jpath** | `import 'x.libsonnet'` resolves relative to the importing file's directory **first**, then against the `FileImporter.JPaths` entries. `[<dir>/lib]` is searched by default, so a sibling `import 'x.libsonnet'` next to the manifest resolves without `--jpath`. |
| **Top-level `obfuscation.protocol` is decorative for generate** | The shared `obfuscation.protocol` is informational; per-peer `peers.<name>.protocol` (default `quic`) drives I-packet generation. See [./obfuscation.md](./obfuscation.md). |
| **Syntax/import errors surface as load errors** | A Jsonnet syntax error is wrapped as `evaluate jsonnet <path>: …` with the go-jsonnet diagnostic (file:line). A failed import names the missing target. |

## Related

- [./manifest-reference.md](./manifest-reference.md) — every `Manifest` field, including `*int` / `*HeaderRange` pointer-nil semantics.
- [./manifest-examples.md](./manifest-examples.md) — equivalent declarative examples in plain JSON.
- [./presets.md](./presets.md) — the 7 preset profiles whose values can be copied into a `.libsonnet`.
- [./cli-reference.md](./cli-reference.md) — full `generate` flag table, including `--jpath`.
