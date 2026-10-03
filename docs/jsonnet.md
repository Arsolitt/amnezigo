# Jsonnet

> Authoring amnezigo manifests in Jsonnet — `.amnezigo.jsonnet` discovery and precedence, `--jpath` resolution, worked examples, and the deliberate limits of the integration.

## Table of Contents

- [Why Jsonnet](#why-jsonnet)
- [Discovery & Precedence](#discovery--precedence)
- [The `--jpath` Flag](#the---jpath-flag)
- [Import Resolution Order](#import-resolution-order)
- [Worked Examples](#worked-examples)
- [Limitations](#limitations)
- [Gotchas](#gotchas)
- [Related](#related)

---

## Why Jsonnet

A `.amnezigo.jsonnet` manifest is a [Jsonnet](https://jsonnet.org) program that **evaluates to a JSON document matching the Manifest schema**. amnezigo evaluates the program with `google/go-jsonnet`, parses the manifested JSON string into `Manifest`, and runs the exact same schema-version check as for a plain `amnezigo.json` — the deliverable is always an ordinary manifest, however it was produced.

Use Jsonnet when you want computation instead of hard-coded values: bind shared snippets with `local`, factor obfuscation parameters into a library you maintain, generate peer blocks from a list, or compose objects with the `+` operator. Jsonnet also allows comments and trailing commas, which the JSON form does not.

Two things do **not** ship with amnezigo:

- **No Jsonnet libraries.** There is no `lib/` directory at the project root and no `presets.libsonnet`; the only Jsonnet files in the repository are test fixtures under `testdata/loader/`. A `lib/` search path exists only when *you* create one.
- **No preset import.** Presets are implemented in Go (`presets.go`), not Jsonnet, so a manifest cannot `import` a preset by name. Copy a preset's numbers into your own Jsonnet file or library by hand — see [Presets](./presets.md) for the profiles.

## Discovery & Precedence

`LoadManifest(dir, jpathDirs)` probes two well-known filenames in the project directory. The Jsonnet file is checked **first**; if it exists, the JSON file is ignored entirely (no merge, no warning).

| File | Precedence | Rule |
| --- | --- | --- |
| `.amnezigo.jsonnet` | 1 (wins) | `os.Stat` succeeds → evaluated as Jsonnet via `loadFromJsonnet`. |
| `amnezigo.json` | 2 (fallback) | Tried only if the Jsonnet file is absent → `loadFromJSON`. |
| *(neither present)* | — | Error: `no manifest file found in <dir> (expected .amnezigo.jsonnet or amnezigo.json)`. |

Discovery is by exact filename (constants in `loader.go`): the Jsonnet file is **`.amnezigo.jsonnet`** — note the leading dot. A file named `amnezigo.jsonnet` (no dot) is **not** discovered; the explicit-path loader `LoadManifestFromFile` evaluates it only when called directly through the library API — see [Library Usage](./library-usage.md).

`LoadManifestFromFile(path, jpathDirs)` routes on a **case-sensitive** string suffix: a path ending in exactly `.jsonnet` goes through the Jsonnet VM, everything else is parsed as JSON. `Custom.Jsonnet` and `manifest.jsonnet.bak` therefore fail as JSON.

| `LoadManifest` argument | Type | Semantics |
| --- | --- | --- |
| `dir` | `string` | Project directory scanned for the two manifest filenames. |
| `jpathDirs` | `[]string` | Jsonnet library search paths. `nil` or empty → defaults to **`[dir/lib]`** via `resolveJpath`; any non-empty value is used verbatim, replacing the default. |

`LoadManifestFromFile` defaults `jpathDirs` to `[parentDir/lib]` in the same way.

Both loaders funnel into the same `validateManifestVersion`, so a Jsonnet manifest must manifest `version: 1` exactly like JSON: `version: 0` reports `missing or zero version field` and any other value reports `unsupported schema version <N> (expected 1)`.

## The `--jpath` Flag

The `generate` command exposes Jsonnet library search paths as a repeatable flag; the slice is passed straight to `amnezigo.LoadManifest(projectDir, jpathDirs)`. The full flag table lives in [CLI Reference](./cli-reference.md).

| Flag | Type | Default | Effect |
| --- | --- | --- | --- |
| `--jpath` | `[]string` (repeatable or comma-separated) | `nil` | Library search dirs for `import` / `importstr`. Omitted → `resolveJpath` returns `[<project>/lib]`. Any value → the user list is used as given. |

```shell
# Default: the manifest's imports resolve through <project>/lib.
$ amnezigo generate --project ./my-vpn

# Explicit: only the listed dirs are searched — <project>/lib is gone.
$ amnezigo generate --project ./my-vpn --jpath ./jsonnet-lib --jpath ./vendor
```

> **Warning:** Any `--jpath` value **replaces** the default `[<project>/lib]`; it is never appended. If you pass `--jpath` while your manifest still uses a bare `import 'network.libsonnet'`, list `<project>/lib` too, or the import fails with `couldn't open import "network.libsonnet": no match locally or in the Jsonnet library paths`.

## Import Resolution Order

For every `import` or `importstr`, go-jsonnet's `FileImporter` probes in this order and stops at the first hit:

1. The directory of the file that contains the import.
2. Each `--jpath` directory, iterated **last to first** — so the last `--jpath` on the command line wins.

Consequences:

- `import 'lib/network.libsonnet'` works with no `--jpath` at all, because the importing file's own directory is searched first.
- A bare `import 'network.libsonnet'` works when a `lib/` sits next to the manifest, because the default jpath is `<project>/lib`.
- When two `--jpath` directories contain the same filename, the one you listed **last** is used:

```shell
$ amnezigo generate --project . --jpath /a --jpath /b   # /b/name.libsonnet wins
$ amnezigo generate --project . --jpath /b --jpath /a   # /a/name.libsonnet wins
```

`--jpath` strings are used as given, so a relative entry like `--jpath lib` resolves against the **process working directory**, not the project directory:

```shell
# Reads <cwd>/lib/network.libsonnet — NOT ./my-vpn/lib/network.libsonnet.
$ amnezigo generate --project ./my-vpn --jpath lib
```

With `--project` omitted the project directory *is* the current working directory, so the two happen to line up; in scripts, pass absolute paths.

> **Note:** Imports are **not sandboxed**. `FileImporter` documents that absolute import paths and paths that traverse up the directory hierarchy (`../`) are both allowed, so a manifest can read any file the process can read. Treat a `.amnezigo.jsonnet` from an untrusted source like any other executable input.

Within a single evaluation, go-jsonnet caches file reads (including misses) per absolute path and memoises evaluated imports, so importing the same library twice reads and evaluates it once. A fresh VM is built for every load, so edits to a library are picked up by the next `generate` run.

## Worked Examples

All three programs evaluate to a JSON document conforming to the `Manifest` structure (`version: 1` + `network` + `obfuscation` + `peers`). See [Manifest Reference](./manifest-reference.md) for the field-by-field schema.

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

| Construct | Meaning |
| --- | --- |
| `local NAME = EXPR;` | Bind a value visible in the following expression. |
| `+` on strings | String concatenation; here builds `10.0.0.1/24` from `prefix`. |
| `{ … }` (object literal) | Evaluates to JSON; field names become JSON keys. |

### (b) Importing shared values from `lib/`

amnezigo ships no libraries, so this `lib/preset.libsonnet` is a file **you** create; it encodes the `home-balanced` numeric profile copied from [Presets](./presets.md). Project layout (the default jpath `[<project>/lib]` resolves the bare import):

```text
my-vpn/
|-- .amnezigo.jsonnet
`-- lib/
    `-- preset.libsonnet
```

`lib/preset.libsonnet`:

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

| Construct | Meaning |
| --- | --- |
| `import 'FILE.libsonnet'` | Loads a Jsonnet module; resolved against the importing file's directory first, then the jpath entries (see [Import Resolution Order](#import-resolution-order)). |
| `OBJ_A OBJ_B` (juxtaposition) | Object merge: right-side fields override the left on conflict; new fields are added. `preset.obfuscation { protocol: 'quic' }` overlays `protocol`. |
| `preset.mtu` / `preset.obfuscation` | Field access into an imported object. |

### (c) Generating N client peers from a comprehension

A server plus a templated block of client peers, built with a Jsonnet object comprehension. Only the server peer carries `endpoint` + `listen_port`; every other peer is a client by the `IsServer()` rule.

```jsonnet
local endpoint = 'vpn.example.com:51820';
local clientNames = ['laptop', 'phone', 'tablet', 'desktop', 'guest'];

local client(index) = {
  address: '10.0.0.' + std.toString(index + 2) + '/32',
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
    [clientNames[i]]: client(i)
    for i in std.range(0, std.length(clientNames) - 1)
  },
}
```

| Construct | Meaning |
| --- | --- |
| `local fn(arg) = EXPR;` | Bind a parameterized local function. |
| `std.toString(n)` | Convert a number to its JSON string form. |
| `{ [KEY]: VAL for x in ARR }` | Object comprehension: emits one object field per array element, with a computed key. `for` binds a **single** variable — `for i, name in arr` is a Jsonnet syntax error; iterate indices and index the array when you need both. |
| `std.range(a, b)` | Inclusive integer range (`std.range(1, 3)` is `[1, 2, 3]`). |
| `OBJ_A + OBJ_B` | Object concatenation (same semantics as juxtaposition). |

> **Note:** Examples (a)–(c) pin unequal `S1`–`S4` values. Under the default AWG 3.1 profile (`obfuscation.awg_version` unset, `random_trailers` on) that combination makes `generate` print one `TRL001` warning per run. Set `obfuscation.awg_version: '2.0'` (which predates trailers), `obfuscation.random_trailers: false`, or equal `S` values to silence it — see [Validation & Analysis](./validation.md).

## Limitations

The Jsonnet integration is deliberately thin: one filename convention, one `--jpath` knob, no VM extras. The manifest contract stays in `Manifest`, and both loaders validate it identically.

| Capability | Status | Evidence |
| --- | --- | --- |
| External variables (`std.extVar`) | Not supported | No ext vars are registered and there are no `--ext-str`/`--ext-code` flags; using one fails with `RUNTIME ERROR: Undefined external variable: mtu`. |
| Top-level arguments (TLAs) | Not supported | No `--tla-str`/`--tla-code` flags exist and no TLAs are registered on the VM; the only Jsonnet knob is `--jpath`. |
| Native functions (`std.native`) | Not supported | No native functions are registered, so `std.native('x')` evaluates to `null` and calling it fails with `Unexpected type null, expected function`. |
| Import sandboxing | Not supported | `FileImporter` allows absolute paths and `../` traversal. |
| Bundled libraries | None | No `lib/` at the project root, no `presets.libsonnet`; `lib/` is a user-created convention. |
| Preset import | Not supported | Presets live in Go (`presets.go`); copy their numbers into your Jsonnet by hand. |
| Non-object top level | Rejected | An array or string manifests, then fails as `parse jsonnet output from <path>: json: cannot unmarshal array into Go value of type amnezigo.Manifest`; a top-level function fails earlier at evaluation with `RUNTIME ERROR: Missing argument: <name>` plus a `Top-level function call` frame. |

## Gotchas

Every load failure is wrapped by the loader and then by the CLI:

- Evaluation failures: `evaluate jsonnet <path>: <go-jsonnet error>` — the go-jsonnet diagnostic includes file, line, and column, and runtime failures carry its multi-line `RUNTIME ERROR` trace.
- Manifestation failures: `parse jsonnet output from <path>: <encoding/json error>`.
- The CLI adds a `loading manifest: ` prefix, so `generate` prints `Error: loading manifest: evaluate jsonnet …`.

| Gotcha | Detail |
| --- | --- |
| **Jsonnet silently shadows JSON** | When both `.amnezigo.jsonnet` and `amnezigo.json` exist, only the Jsonnet file is read. A stale Jsonnet file makes edits to the JSON file invisible. See [Gotchas](./gotchas.md). |
| **The dot prefix is mandatory** | Auto-discovery looks for **`.amnezigo.jsonnet`**. A file named `amnezigo.jsonnet` is skipped by `LoadManifest`; only the explicit-path API evaluates it. |
| **`.jsonnet` detection is case-sensitive** | `LoadManifestFromFile` routes on `strings.HasSuffix(path, ".jsonnet")`, so `Manifest.Jsonnet` and `manifest.jsonnet.bak` are parsed as JSON and fail. |
| **Version is still required** | The manifested object must contain `version: 1`; anything else is rejected exactly as for JSON. |
| **Default jpath is `[<dir>/lib]` — but `lib/` is yours to create** | amnezigo ships no library there. An import outside your own `lib/` needs an explicit `--jpath`. |
| **`--jpath` replaces, never appends** | Supplying any value drops the default `[<dir>/lib]`. List the default explicitly if you need both. |
| **The last `--jpath` wins for duplicate filenames** | `FileImporter` probes jpaths in reverse order, so a later override directory takes precedence over an earlier base directory. |
| **Relative `--jpath` is resolved against the CWD** | `--jpath lib` means `<cwd>/lib`, not `<project>/lib`; pass absolute paths in scripts. |
| **No ext vars, TLAs, or native functions** | `std.extVar('x')` fails with `Undefined external variable: x`; there are no CLI flags to provide values. |
| **Examples warn under AWG 3.1 defaults** | Unequal `S` values plus the default `random_trailers` produce one `TRL001` per `generate` run; see the note under [Worked Examples](#worked-examples). |
| **Comprehensions can emit duplicate addresses** | Neither the loader nor the pipeline validates peer addresses for uniqueness, so a comprehension (or an object merge) can produce two peers with the same `address` and nothing rejects them before configs are written. |

## Related

- [Manifest Reference](./manifest-reference.md) — every `Manifest` field, including `*int` / `*HeaderRange` pointer-nil semantics.
- [Manifest Examples](./manifest-examples.md) — complete JSON and Jsonnet manifests with field-by-field explainers.
- [CLI Reference](./cli-reference.md) — full `generate` flag table, including `--jpath` and `--project`.
- [Presets](./presets.md) — the 7 preset profiles whose values you can copy into a `.libsonnet`.
- [Validation & Analysis](./validation.md) — finding codes such as `TRL001` and what triggers them.
- [Gotchas](./gotchas.md) — project-wide pitfalls, including loader and Jsonnet traps.
