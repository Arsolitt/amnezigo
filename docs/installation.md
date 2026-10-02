# Installation

> How to install or build the `amnezigo` CLI — a declarative AmneziaWG configuration generator targeting AWG 2.0/3.0/3.1 (default 3.1; not a daemon).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Install Methods](#install-methods)
  - [go install](#go-install)
  - [Prebuilt release binaries](#prebuilt-release-binaries)
  - [Build from source](#build-from-source)
  - [Docker](#docker)
- [Verify the Install](#verify-the-install)
- [Build Entry Point](#build-entry-point)
- [Development Builds](#development-builds)
- [Next Steps](#next-steps)

---

## Prerequisites

| Requirement | Detail |
|---|---|
| Go toolchain | **1.26.1 or newer** (pinned as `go 1.26.1` in `go.mod`; `mise.toml` pins Go 1.26.8 for local development). Required for `go install` and source builds. Not needed when using a release binary or the Docker image. |
| Git | Required to clone the repository for source and Docker builds. |
| Docker | Required only for the image build; both the runtime base image and the release images are amd64-only (see [Docker](#docker)). |
| AmneziaWG runtime | **To actually run** a generated config, you need the AmneziaWG userspace (`amneziawg-go`) or the kernel module **matching the manifest's `obfuscation.awg_version`** — `"2.0"`, `"3.0"`, or `"3.1"` (unset defaults to `"3.1"`). See note below. |
| Operating system | Any Go-supported OS. The release matrix is linux/amd64 and darwin/amd64; the container images are linux/amd64 only. |

> **Warning:** amnezigo **only generates** `awg0.conf` files. It does **not** install, enable, or manage the AmneziaWG runtime, set up network interfaces, or bring tunnels up/down. The runtime stage of both images pins `amneziavpn/amneziawg-go:3.1.20260828` (Alpine 3.19, amd64-only), which already ships the `awg` tools — so a generated `<d>` passthrough tag (which requires AWG 2.0 userspace and is rejected by the legacy kernel module) works out of the box. See [Obfuscation](./obfuscation.md).

## Install Methods

| Method | Command | Notes |
|---|---|---|
| `go install` | `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest` | Installs the binary to `$GOPATH/bin` (or `$GOBIN`). Go 1.26+ required. |
| Release binaries | Download from the [releases page](https://github.com/Arsolitt/amnezigo/releases) | Raw `amnezigo-linux-amd64` / `amnezigo-darwin-amd64` binaries, `checksums.txt`, and `amnezigo-licenses_<version>.tar.gz`. amd64 only. |
| Build from source (dev) | `make build` | Produces `bin/amnezigo` from a working-tree clone, with the version/commit stamp injected via `-ldflags`. |
| Build from source (production) | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/amnezigo ./cmd/amnezigo/` | Static, stripped binary; mirrors the container build stage. |
| Docker (from source) | `docker build --platform linux/amd64 -t amnezigo .` | Multi-stage image; run via `docker run --rm amnezigo <command>`. |
| Docker (published) | `docker run --rm ghcr.io/arsolitt/amnezigo:<version> --help` | Release image built by GoReleaser; `amd64` only. |

### go install

```shell
$ go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
$ amnezigo version
```

`amnezigo version` prints `amnezigo <Version> (<Commit>)`. A binary installed from a tagged release shows the tag, for example `amnezigo v1.0.0 (abc1234)`; a plain `go build` shows `amnezigo dev (none)`.

### Prebuilt release binaries

Every tagged release attaches raw, statically linked executables — there is no archive to extract:

| Asset | Contents |
|---|---|
| `amnezigo-linux-amd64` | Linux x86-64 executable. |
| `amnezigo-darwin-amd64` | macOS x86-64 executable (runs on Apple Silicon through Rosetta 2). |
| `checksums.txt` | SHA-256 checksums for every asset. |
| `amnezigo-licenses_<version>.tar.gz` | The `LICENSE`, `NOTICE`, and `licenses/` bundle for the release (GPL-3.0 and third-party attribution). |

> **Note:** the release matrix is **amd64-only** — there are no arm64 binaries. Fetch them from the [releases page](https://github.com/Arsolitt/amnezigo/releases) and verify with `sha256sum -c checksums.txt`.

### Build from source

```shell
$ git clone https://github.com/Arsolitt/amnezigo.git
$ cd amnezigo

# Development build (stamps the version/commit via git describe)
$ make build              # -> bin/amnezigo

# Production build (static + stripped, mirrors the release build)
$ CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/amnezigo ./cmd/amnezigo/
```

### Docker

The root `Dockerfile` is multi-stage:

| Stage | Base image | Purpose |
|---|---|---|
| `builder` | `golang:1.26-alpine` | Compiles a static binary: `CGO_ENABLED=0 GOOS=linux go build -ldflags="..." -o ./build/amnezigo ./cmd/amnezigo/`. The version stamp defaults to `dev`/`none`; override it with `--build-arg VERSION=vX.Y.Z --build-arg COMMIT=$(git rev-parse --short HEAD)`. |
| runtime | `amneziavpn/amneziawg-go:3.1.20260828` | AmneziaWG userspace base image (Alpine 3.19, amd64-only). It already ships bash, the ca-certificates bundle and the `awg` tools, so the build adds **no package layer**. Copies the binary to `/usr/local/bin/amnezigo` and sets `ENTRYPOINT ["/usr/local/bin/amnezigo"]`. |

Because both the base image and the published images are amd64-only, the build must request the platform explicitly on arm64 hosts (Docker then runs it through Rosetta/containerd emulation):

```shell
$ docker build --platform linux/amd64 -t amnezigo .
$ docker run --rm amnezigo generate --help
```

The published image `ghcr.io/arsolitt/amnezigo:<version>` is built by GoReleaser from `cmd/amnezigo/Dockerfile`, sets the same `ENTRYPOINT ["/usr/local/bin/amnezigo"]`, and ships `NOTICE`, `LICENSE`, and `licenses/` under `/usr/share/doc/amnezigo/`:

```shell
$ docker run --rm ghcr.io/arsolitt/amnezigo:<version> --help

# Real runs mount the project directory as the working directory:
$ docker run --rm -v "$(pwd):/work" -w /work ghcr.io/arsolitt/amnezigo:<version> generate
```

> **Note:** the entrypoint is the binary itself, so subcommands are passed directly (`docker run … amnezigo generate`). Use `--entrypoint /bin/sh` if you need a shell inside the container.

## Verify the Install

`amnezigo --help` prints the root command and its available subcommands. The project registers **four** commands — `generate`, `validate`, `analyze`, `version` (cobra additionally exposes its built-in `completion` and `help` commands):

```text
$ amnezigo --help
Declarative AmneziaWG v3.1 configuration generator.

Usage:
  amnezigo [command]

Available Commands:
  analyze     Analyze obfuscation config for potential weaknesses
  completion  Generate the autocompletion script for the specified shell
  generate    Generate AmneziaWG configs from manifest
  help        Help about any command
  validate    Validate an AmneziaWG server config against AWG size invariants
  version     Print the amnezigo version and commit

Flags:
  -h, --help   help for amnezigo

Use "amnezigo [command] --help" for more information about a command.
```

The exact environment shown above is from an unstamped development build (`go run ./cmd/amnezigo --help`); release binaries show the same command surface. `amnezigo version` identifies the build:

```text
$ amnezigo version
amnezigo dev (none)
```

> **Note:** Any reference to `init`, `add`, `edit`, `remove`, `export`, or `list` commands is **stale** — that imperative CLI was removed in the declarative refactor. Only `generate`, `validate`, `analyze`, and `version` exist. See [CLI Reference](./cli-reference.md) for full flag tables.

## Build Entry Point

The binary entry point is intentionally thin. `cmd/amnezigo/main.go` is the entire `main` package:

```go
package main

import "github.com/Arsolitt/amnezigo/internal/cli"

func main() {
	cli.Execute()
}
```

`cli.Execute()` (in `internal/cli/cli.go`) constructs the cobra root command — `Use: "amnezigo"`, `Short: "AmneziaWG v3.1 Configuration Generator"`, `Long: "Declarative AmneziaWG v3.1 configuration generator."` — and registers the four subcommands (`generate`, `validate`, `analyze`, `version`). All command logic, flag definitions, and help text live in `internal/cli/`; `main` does nothing beyond delegating.

## Development Builds

`mise.toml` pins the toolchain used by the project: Go 1.26.8, golangci-lint 2.14.0, and goreleaser 2.18.1.

```shell
$ mise install
```

The `Makefile` wraps the common tasks:

| Target | Command | Purpose |
|---|---|---|
| `make build` | `go build -ldflags "…" -o bin/amnezigo ./cmd/amnezigo` | Build the CLI with the version stamp. |
| `make test` | `go test ./...` | Run the unit tests. |
| `make test-race` | `go test -race ./...` | Race-enabled tests (pre-merge gate). |
| `make test-e2e` | `go test -tags=e2e ./e2e/...` | End-to-end suite; it drives real `amneziawg-go` containers and self-skips when Docker is unavailable. |
| `make lint` | `golangci-lint run` | Lint with the golden config. |
| `make fmt` | `golangci-lint fmt` | Format the tree. |
| `make notice` | `go run ./hack/noticegen` | Regenerate `NOTICE` and `licenses/` from the module graph. |
| `make snapshot` | `goreleaser release --snapshot --clean` | Build all release artifacts locally without publishing. |
| `make image` | `docker build --platform linux/amd64 -t amnezigo .` | Build the from-source image; the explicit platform is already pinned. |
| `make clean` | `rm -rf bin dist` | Remove build outputs. |

Validate the release configuration (and the tag convention `vX.Y.Z` for stable releases, `vX.Y.Z-rc.N` for prereleases) with:

```shell
$ mise exec -- goreleaser check
```

## Next Steps

- [Quick Start](./quick-start.md) — generate your first configs from a manifest.
- [CLI Reference](./cli-reference.md) — full flag tables for `generate`, `validate`, `analyze`, and `version`.
- [Manifest Reference](./manifest-reference.md) — every manifest field with semantics and defaults.
