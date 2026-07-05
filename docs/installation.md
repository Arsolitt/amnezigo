# Installation

> How to install or build the `amnezigo` CLI — a declarative AmneziaWG v2.0 configuration generator (not a daemon).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Install Methods](#install-methods)
  - [go install](#go-install)
  - [Build from Source](#build-from-source)
  - [Docker](#docker)
- [Verify the Install](#verify-the-install)
- [Build Entry Point](#build-entry-point)
- [Next Steps](#next-steps)

---

## Prerequisites

| Requirement | Detail |
|---|---|
| Go toolchain | **1.26.1 or newer** (pinned as `go 1.26.1` in `go.mod`). Required for `go install` and source builds. Not needed when using the Docker image. |
| Git | Required to clone the repository for source and Docker builds. |
| Docker | Required only for the multi-stage image build. |
| AmneziaWG runtime | **To actually run** a generated config, you need AmneziaWG userspace (`amneziawg-go` ≥ 2.0) or the kernel module. See note below. |
| Operating system | Any Go-supported OS. The Dockerfile production build targets `GOOS=linux`; the binary is otherwise cross-compilable via standard `GOOS`/`GOARCH` flags. |

> **Warning:** amnezigo **only generates** `awg0.conf` files. It does **not** install, enable, or manage the AmneziaWG runtime, set up network interfaces, or bring tunnels up/down. The Docker runtime stage pins `amneziavpn/amneziawg-go:0.2.16` so a generated `<d>` passthrough tag (which requires AWG 2.0 userspace) works out of the box — see [Obfuscation](./obfuscation.md).

## Install Methods

| Method | Command | Notes |
|---|---|---|
| `go install` | `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest` | Installs the binary to `$GOPATH/bin` (or `$GOBIN`). Go 1.26+ required. |
| Build from source (dev) | `go build -o build/amnezigo ./cmd/amnezigo/` | Produces `./build/amnezigo` from a working-tree clone. |
| Build from source (production) | `CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o build/amnezigo ./cmd/amnezigo/` | Static, stripped binary; mirrors the Dockerfile build stage. |
| Docker | `docker build -t amnezigo .` | Multi-stage image; run via `docker run --rm amnezigo <command>`. |

### go install

```shell
$ go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
$ amnezigo --help
```

### Build from source

```shell
$ git clone https://github.com/Arsolitt/amnezigo.git
$ cd amnezigo

# Development build
$ go build -o build/amnezigo ./cmd/amnezigo/

# Production build (static + stripped, matches the Dockerfile)
$ CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o build/amnezigo ./cmd/amnezigo/
```

### Docker

The `Dockerfile` is multi-stage:

| Stage | Base image | Purpose |
|---|---|---|
| `builder` | `golang:1.26-alpine` | Compiles a static binary: `CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./build/amnezigo ./cmd/amnezigo/`. |
| runtime | `amneziavpn/amneziawg-go:0.2.16` | AmneziaWG userspace base image. Installs `ca-certificates` and `bash`, copies the binary to `/usr/local/bin/amnezigo`. |

```shell
$ docker build -t amnezigo .
$ docker run --rm amnezigo generate --help
```

## Verify the Install

`amnezigo --help` prints the root command and its available subcommands. amnezigo registers **three** project commands — `generate`, `validate`, `analyze` (cobra additionally exposes its built-in `completion` and `help` commands):

```text
$ amnezigo --help
Declarative AmneziaWG v2.0 configuration generator.

Usage:
  amnezigo [command]

Available Commands:
  analyze     Analyze obfuscation config for potential weaknesses
  completion  Generate the autocompletion script for the specified shell
  generate    Generate AmneziaWG configs from manifest
  help        Help about any command
  validate    Validate an AmneziaWG server config against AWG 2.0 invariants

Flags:
  -h, --help   help for amnezigo

Use "amnezigo [command] --help" for more information about a command.
```

> **Note:** Any reference to `init`, `add`, `edit`, `remove`, `export`, or `list` commands is **stale** — that imperative CLI was removed in the declarative refactor. Only `generate`, `validate`, and `analyze` exist. See [CLI Reference](./cli-reference.md) for full flag tables.

## Build Entry Point

The binary entry point is intentionally thin. `cmd/amnezigo/main.go` is the entire `main` package:

```go
package main

import "github.com/Arsolitt/amnezigo/internal/cli"

func main() {
	cli.Execute()
}
```

`cli.Execute()` (in `internal/cli/cli.go`) constructs the cobra root command — `Use: "amnezigo"`, `Short: "AmneziaWG v2.0 Configuration Generator"` — and registers the three subcommands (`generate`, `validate`, `analyze`). All command logic, flag definitions, and help text live in `internal/cli/`; `main` does nothing beyond delegating.

## Next Steps

- [Quick Start](./quick-start.md) — generate your first configs from a manifest.
- [CLI Reference](./cli-reference.md) — full flag tables for `generate`, `validate`, and `analyze`.
- [Manifest Reference](./manifest-reference.md) — every manifest field with semantics and defaults.
