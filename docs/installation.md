# Installation & Quick Start

> Get Amnezigo installed and generate your first AmneziaWG server configuration in under a minute.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
  - [go install (recommended)](#go-install-recommended)
  - [Build from Source](#build-from-source)
  - [Docker](#docker)
- [Quick Start](#quick-start)
  - [Step 1: Initialize the Server](#step-1-initialize-the-server)
  - [Step 2: Add a Peer](#step-2-add-a-peer)
  - [Step 3: Export a Client Config](#step-3-export-a-client-config)
  - [What Just Happened](#what-just-happened)
- [Next Steps](#next-steps)

---

## Prerequisites

- **Go 1.26+** — required for building from source or using `go install`
- **AmneziaWG** — the runtime that actually uses the generated configs (not installed by Amnezigo itself)
- For Docker: Docker Engine 17.05+ (multi-stage build support)

> **Note:** Amnezigo is a configuration generator, not a VPN daemon. You still need [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) installed on your server and clients to use the configs it produces.

---

## Installation Methods

### go install (recommended)

The fastest way to install Amnezigo. This fetches the latest tagged release from the Go module proxy and installs the binary to your `GOBIN` (default `$GOPATH/bin`, or `$HOME/go/bin`):

```shell
$ go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
```

Make sure `$GOPATH/bin` (or `$GOBIN`) is in your `PATH`. Verify the installation:

```shell
$ amnezigo --help
AmneziaWG v2.0 Configuration Generator for star topology
```

> **Tip:** The `@latest` suffix always installs the most recent tagged release, not the bleeding-edge `main` branch. If you need an unreleased version, replace `@latest` with `@main`.

### Build from Source

Clone the repository and build manually. This gives you access to the `main` branch and lets you modify the source:

```shell
$ git clone https://github.com/Arsolitt/amnezigo.git
$ cd amnezigo
$ go build -o amnezigo ./cmd/amnezigo/
```

The resulting `amnezigo` binary is in the project root. You can move it anywhere in your `PATH`:

```shell
$ mv amnezigo /usr/local/bin/
```

> **Note:** The production Dockerfile uses `CGO_ENABLED=0` and `-ldflags="-s -w"` to produce a stripped static binary. If you want a smaller binary for distribution, build with those flags:
>
> ```shell
> $ CGO_ENABLED=0 go build -ldflags="-s -w" -o amnezigo ./cmd/amnezigo/
> ```

### Docker

The Docker image is built in two stages: a Go builder stage using `golang:1.26-alpine`, and a runtime stage based on `amneziavpn/amneziawg-go:0.2.16` (which includes the AmneziaWG tools).

```shell
$ docker build -t amnezigo .
```

Run commands by mounting your working directory as a volume. All config files are written into the mounted directory:

```shell
$ docker run --rm -v $(pwd):/data amnezigo init --ipaddr 10.8.0.1/24
```

> **Warning:** The `-v $(pwd):/data` mount is required. Without it, generated config files are written inside the container's filesystem and lost when the container exits (the `--rm` flag cleans up the container).
>
> By default, `init` writes the config to `awg0.conf` inside `/data`, which maps to your current directory. Use `--config` to specify a different filename within `/data`.

---

## Quick Start

This walkthrough covers the three essential commands to go from zero to a working AmneziaWG setup: **init**, **add**, and **export**.

### Step 1: Initialize the Server

Create a new server configuration. The only required flag is `--ipaddr`, which sets the server's internal IP address and VPN subnet:

```shell
$ amnezigo init --ipaddr 10.8.0.1/24
✓ AmneziaWG configuration initialized successfully
  Config: awg0.conf
  Server IP: 10.8.0.1/24
  Listen Port: 42817
  Main Interface: eth0
  IPv4 Endpoint: 203.0.113.42:42817
```

Here's what happened under the hood:

| What | Detail |
|------|--------|
| Keypair generated | X25519 server keypair (private + public) |
| Obfuscation params generated | Jc, Jmin, Jmax, S1-S4, H1-H4 — randomized for each init |
| iptables rules generated | PostUp/PostDown rules for NAT and forwarding |
| Port chosen | Random port between 10000-65535 (unless `--port` is set) |
| Endpoint detected | Auto-detected via `ipv4.icanhazip.com` (5s timeout) |

Useful flags for common scenarios:

```shell
# Use a specific listen port
$ amnezigo init --ipaddr 10.8.0.1/24 --port 51820

# Allow peers to communicate with each other
$ amnezigo init --ipaddr 10.8.0.1/24 --client-to-client

# Specify the tunnel interface name (default: awg0)
$ amnezigo init --ipaddr 10.8.0.1/24 --iface-name wg0

# Set DNS servers for client configs
$ amnezigo init --ipaddr 10.8.0.1/24 --dns "1.1.1.1, 8.8.4.4"
```

> **Tip:** If your server has multiple network interfaces, use `--iface eth1` to specify which one to use for NAT forwarding. By default, Amnezigo tries to auto-detect the main interface.

> **Warning:** Endpoint auto-detection contacts `ipv4.icanhazip.com` and `ipv6.icanhazip.com` over HTTPS. If your server has no internet access at init time, detection fails silently and the endpoint field is left empty. You can set it manually with `--endpoint-v4`.

### Step 2: Add a Peer

Add a client to your VPN by name. Amnezigo generates a keypair, allocates an IP address, and stores the peer in the server config:

```shell
$ amnezigo add laptop
Peer 'laptop' added successfully
  IP Address: 10.8.0.2/32
  Public Key: aB3kX7...
```

The IP address is automatically assigned — it starts at `.2` (the first address after the server's `.1`) and increments for each new peer.

You can also specify an IP manually:

```shell
$ amnezigo add phone --ipaddr 10.8.0.50
Peer 'phone' added successfully
  IP Address: 10.8.0.50/32
  Public Key: cD9mY2...
```

> **Note:** Only IPv4 addresses are supported for auto-allocation. If you need IPv6 peers, assign addresses manually.

### Step 3: Export a Client Config

Export a peer's configuration to a `.conf` file that can be imported directly into AmneziaWG clients:

```shell
$ amnezigo export laptop
Exported peer 'laptop' to laptop.conf
```

The exported file includes per-peer obfuscation parameters (I1-I5 Custom Packet Strings) generated based on the chosen protocol. By default, a random protocol is selected. You can pick one explicitly:

```shell
$ amnezigo export laptop --protocol quic
Exported peer 'laptop' to laptop.conf
```

Available protocols: `quic`, `dns`, `dtls`, `stun`, `random`.

> **Note:** Exported files are written to your **current working directory**, not next to the server config file. Plan your directory layout accordingly.

You can also export all peers at once:

```shell
$ amnezigo export
Exported peer 'laptop' to laptop.conf
Exported peer 'phone' to phone.conf
```

### What Just Happened

After these three commands, you have:

- **`awg0.conf`** — the server configuration file, ready to use with `amnezia-wg`
- **`laptop.conf`** — the client configuration file, ready to import into an AmneziaWG client app

The client config routes all traffic (`0.0.0.0/0, ::/0`) through the VPN by default, with DNS set to `1.1.1.1, 8.8.8.8` and a persistent keepalive of 25 seconds.

> **Note:** Peer private keys are stored in the server config as metadata fields (`#_PrivateKey`). This is necessary so that Amnezigo can export complete client configs on demand. Protect your server config file accordingly.

---

## Next Steps

- [CLI Reference](./cli-reference.md) — full documentation for all commands and flags
- [Configuration](./configuration.md) — server and client config file format, metadata fields, and how parsing works
