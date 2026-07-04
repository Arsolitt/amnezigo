<p align="center">
  <img src="assets/amnezigo-logo.png" alt="Amnezigo" width="256">
</p>

<h1 align="center">Amnezigo</h1>

<p align="center">
  <strong>Amnezia</strong> + <strong>Go</strong> = <strong>Amnezigo</strong><br>
  A CLI tool and Go library for generating <a href="https://github.com/amnezia-vpn/amneziawg">AmneziaWG</a> v2.0 configurations from a declarative manifest.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/Arsolitt/amnezigo"><img src="https://pkg.go.dev/badge/github.com/Arsolitt/amnezigo.svg" alt="Go Reference"></a>
  <a href="https://github.com/Arsolitt/amnezigo/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0-blue" alt="License: GPL-3.0"></a>
  <a href="https://github.com/Arsolitt/amnezigo"><img src="https://img.shields.io/github/go-mod/go-version/Arsolitt/amnezigo?logo=go&logoColor=white" alt="Go Version"></a>
</p>

---

## Features

- **Declarative manifests** — describe the full network topology in `amnezigo.json` or `.amnezigo.jsonnet`
- **One-shot generation** — `amnezigo generate` builds the server plus every client config in a single atomic run
- **Credential reuse** — keys are recovered from existing output, so re-running `generate` keeps peers stable
- **AmneziaWG obfuscation** — S1–S4 size prefixes, H1–H4 header ranges, junk packets, and per-client I1–I5 custom packet strings
- **Protocol templates** — QUIC, DNS, DTLS, STUN, and SIP handshake shapes
- **Built-in presets** — tuned parameter sets for LAN, home, mobile, and CI environments
- **iptables rules** — PostUp/PostDown NAT and forwarding generated when `main_iface` is set
- **Validation** — `amnezigo validate` checks configs against AWG 2.0 invariants
- **Heuristic analysis** — `amnezigo analyze` inspects obfuscation strength
- **IPv4 & IPv6** endpoint auto-detection
- Usable as a Go library

## Quick Start

Install the CLI:

```shell
go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
```

Or build with Docker:

```shell
docker build -t amnezigo .
```

Declare your network in `amnezigo.json` — one server peer (sets both `endpoint` and `listen_port`) plus any number of client peers:

```json
{
  "version": 1,
  "network": { "mtu": 1280 },
  "obfuscation": { "protocol": "quic" },
  "peers": {
    "server": {
      "address": "10.0.0.1/24",
      "endpoint": "vpn.example.com:51820",
      "listen_port": 51820
    },
    "phone": { "address": "10.0.0.2/32" }
  }
}
```

Generate the server and client configs:

```shell
# Writes output/server/awg0.conf and output/<peer>/awg0.conf
amnezigo generate

# Check a generated config against AWG 2.0 invariants
amnezigo validate output/server/awg0.conf

# Inspect obfuscation strength
amnezigo analyze
```

See [llms-full.txt](docs/llms-full.txt) for the full manifest field reference.

## Presets

Built-in presets provide tuned obfuscation parameters for common network environments. There is no `preset` field — copy a preset's values into the `obfuscation` block of your manifest (or a Jsonnet library):

| Preset | Description |
| --- | --- |
| `lan-conservative` | Small S values, narrow junk range for corporate LANs with minimal DPI |
| `home-balanced` | Moderate parameters for home internet connections |
| `mobile-aggressive` | Maximum entropy for carrier networks with heavy DPI |
| `test-minimal` | Smallest valid set for integration testing and CI |

Use `amnezigo.GetPreset(name)` from Go code, or copy the values from the [Presets reference](docs/llms-full.txt).

## Documentation

The complete, AI-friendly documentation lives in a single file:

| Page | Description |
|------|-------------|
| [**llms-full.txt**](docs/llms-full.txt) | Overview, quick start, manifest reference, examples, output format, CLI reference, credentials, Jsonnet, obfuscation, presets, validation, gotchas |

> **Note:** The standalone guides under `docs/` (`installation.md`, `cli-reference.md`, `configuration.md`, `library-usage.md`, `obfuscation.md`) describe the **pre-declarative** imperative CLI (`init` / `add` / `export`), which was removed in the declarative refactor. Treat `llms-full.txt` as the source of truth.

## Using with AI Assistants

It is recommended to copy the following prompt and send it to an AI assistant — this can significantly improve the quality of generated AmneziaWG configurations:

```text
https://raw.githubusercontent.com/Arsolitt/amnezigo/refs/heads/main/docs/llms-full.txt This link is the full documentation of Amnezigo.

【Role Setting】
You are an expert proficient in network protocols and AmneziaWG configuration.

【Task Requirements】
1. Knowledge Base: Please read and deeply understand the content of this link, and use it as the sole basis for answering questions and writing configurations.
2. No Hallucinations: Absolutely do not fabricate fields that do not exist in the documentation. If the documentation does not mention it, please tell me directly "Documentation does not mention".
3. Default Format: Output INI format configuration by default (unless I explicitly request a different format), and add key comments.
4. Exception Handling: If you cannot access this link, please inform me clearly and prompt me to manually download the documentation and upload it to you.
```

## License

[GPL-3.0](LICENSE)
