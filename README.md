# Amnezigo

[![Go Reference](https://pkg.go.dev/badge/github.com/Arsolitt/amnezigo.svg)](https://pkg.go.dev/github.com/Arsolitt/amnezigo)
[![Go Report Card](https://goreportcard.com/badge/github.com/Arsolitt/amnezigo)](https://goreportcard.com/report/github.com/Arsolitt/amnezigo)

**Amnezia** + **Go** = **Amnezigo**

A CLI tool and Go library for generating and managing [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0 configurations.

## Features

- Generate AmneziaWG server configurations with obfuscation parameters
- Manage peers (add, remove, list, export, analyze)
- Multiple obfuscation protocols (QUIC, DNS, DTLS, STUN)
- Automatic IP assignment for peers
- iptables rules generation for NAT and forwarding
- Per-peer obfuscation parameters at export time
- Heuristic analysis of obfuscation configs (`analyze` command)
- Dynamic client-to-client switching
- IPv4 and IPv6 endpoint auto-detection
- Config validation against AWG 2.0 invariants (`amnezigo validate`)
- Usable as a Go library

## Quick Start

```bash
# Install
go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest

# Initialize server
amnezigo init --ipaddr 10.8.0.1/24

# Add a peer
amnezigo add laptop

# Export peer config
amnezigo export laptop

# Analyze config for weaknesses
amnezigo analyze
```

## Documentation

| Page | Description |
|------|-------------|
| [Installation & Quick Start](docs/installation.md) | Install methods, Docker, walkthrough |
| [CLI Reference](docs/cli-reference.md) | All commands, flags, defaults, and behavior |
| [Configuration Files](docs/configuration.md) | Server/client config format, metadata fields, iptables |
| [Using as a Go Library](docs/library-usage.md) | Programmatic API: Manager, config I/O, keys, obfuscation |
| [Obfuscation Parameters](docs/obfuscation.md) | Junk packets, size prefixes, header ranges, CPS tags, protocols |

## Using with AI Assistants

It is recommended to copy the following prompt and send it to an AI assistant — this can significantly improve the quality of generated AmneziaWG configurations:

```
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

[MIT](LICENSE)
