# Amnezigo

[![Go Reference](https://pkg.go.dev/badge/github.com/Arsolitt/amnezigo.svg)](https://pkg.go.dev/github.com/Arsolitt/amnezigo)
[![Go Report Card](https://goreportcard.com/badge/github.com/Arsolitt/amnezigo)](https://goreportcard.com/report/github.com/Arsolitt/amnezigo)

**Amnezia** + **Go** = **Amnezigo**

A CLI tool and Go library for generating and managing [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0 configurations.

## Features

- Generate AmneziaWG server configurations with obfuscation parameters
- Manage peers (add, remove, list, export)
- Multiple obfuscation protocols (QUIC, DNS, DTLS, STUN)
- Automatic IP assignment for peers
- iptables rules generation for NAT and forwarding
- Per-peer obfuscation parameters at export time
- Dynamic client-to-client switching
- IPv4 and IPv6 endpoint auto-detection
- Usable as a Go library

## Installation

### go install

```bash
go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest
```

### Build from source

```bash
git clone https://github.com/Arsolitt/amnezigo.git
cd amnezigo
go build -o amnezigo ./cmd/amnezigo/
```

### Docker

```bash
docker build -t amnezigo .
docker run --rm -v $(pwd):/data amnezigo init --ipaddr 10.8.0.1/24
```

## Quick Start

```bash
# Initialize server configuration
amnezigo init --ipaddr 10.8.0.1/24

# Add a peer
amnezigo add laptop

# Export peer config
amnezigo export laptop
```

## Usage

### Initialize Server

```bash
amnezigo init --ipaddr 10.8.0.1/24
```

Options:
- `--ipaddr` - Server IP address with subnet (required)
- `--port` - Listen port (default: random 10000-65535)
- `--mtu` - MTU size (default: 1280)
- `--dns` - DNS servers (default: "1.1.1.1, 8.8.8.8") - not stored in config
- `--keepalive` - Persistent keepalive interval (default: 25) - not stored in config
- `--client-to-client` - Allow client-to-client traffic
- `--iface` - Main network interface for NAT (default: auto-detect)
- `--iface-name` - WireGuard interface name (default: awg0)
- `--endpoint-v4` - IPv4 endpoint address (auto-detect if empty)
- `--endpoint-v6` - IPv6 endpoint address (auto-detect if empty)
- `--config` - Config file path (default: awg0.conf)

Note: The `--dns` and `--keepalive` flags are accepted for compatibility but not stored in the server config. DNS is hardcoded to "1.1.1.1, 8.8.8.8" and keepalive to 25 in exports.

### Peer Commands

#### Add Peer

```bash
# Auto-assign IP
amnezigo add laptop

# Specify IP manually
amnezigo add phone --ipaddr 10.8.0.50
```

Options:
- `--ipaddr` - Peer IP address (auto-assigned if not specified)
- `--config` - Server config file (default: awg0.conf)

#### List Peers

```bash
amnezigo list
```

Options:
- `--config` - Server config file (default: awg0.conf)

#### Export Peer Config

```bash
# Export single peer
amnezigo export laptop

# Export with specific protocol
amnezigo export laptop --protocol quic

# Export all peers
amnezigo export
```

Options:
- `--protocol` - Obfuscation protocol: random, quic, dns, dtls, stun (default: random)
- `--config` - Server config file (default: awg0.conf)

The endpoint is automatically resolved in this order:
1. Stored IPv4 endpoint from server config (`EndpointV4`)
2. Stored IPv6 endpoint from server config (`EndpointV6`)
3. Auto-detected via icanhazip.com HTTP service

#### Remove Peer

```bash
amnezigo remove laptop
```

Options:
- `--config` - Server config file (default: awg0.conf)

### Edit Server Configuration

```bash
# Enable client-to-client traffic
amnezigo edit --client-to-client true

# Disable client-to-client traffic
amnezigo edit --client-to-client false
```

Options:
- `--client-to-client` - Enable/disable client-to-client (true/false)
- `--config` - Server config file (default: awg0.conf)

## Configuration Files

### Server Config (awg0.conf)

```ini
[Interface]
PrivateKey = <server-private-key>
Address = 10.8.0.1/24
ListenPort = 55424
MTU = 1280
PostUp = iptables -t nat ...
PostDown = iptables -t nat ...
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
#_EndpointV4 = 1.2.3.4:51820
#_EndpointV6 = [2001:db8::1]:51820
#_ClientToClient = false
#_TunName = awg0

[Peer]
#_Name = laptop
#_PrivateKey = <peer-private-key>
PublicKey = <peer-public-key>
AllowedIPs = 10.8.0.2/32
```

### Peer Config (laptop.conf)

```ini
[Interface]
PrivateKey = <peer-private-key>
Address = 10.8.0.2/32
DNS = 1.1.1.1, 8.8.8.8
MTU = 1280
Jc = 3
Jmin = 50
Jmax = 1000
S1 = 15
S2 = 16
S3 = 45
S4 = 10
H1 = 191091632-238083235
H2 = 469298095-484308427
H3 = 490129542-1366070158
H4 = 1959094164-1989726207
I1 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0040><b 0x00><b 0x01><t><r 40>
I2 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0020><b 0x01><t><r 20>
I3 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0010><b 0x01><t><r 16>
I4 = <b 0xc0ff><b 0x00000001><b 0x08><r 8><b 0x00><b 0x00><b 0x0005><b 0x01><t><r 5>
I5 = 

[Peer]
PublicKey = <server-public-key>
PresharedKey = <psk>
Endpoint = 1.2.3.4:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```

## Obfuscation Parameters

AmneziaWG uses several parameters to obfuscate WireGuard traffic:

- **Jc, Jmin, Jmax** - Junk packet parameters
- **S1-S4** - Size prefixes
- **H1-H4** - Header value ranges (stored as min-max format)
- **I1-I5** - Custom Packet Strings (CPS) generated per-peer at export time

### CPS Tag Syntax

I1-I5 use a tag-based syntax for constructing byte sequences:

| Tag | Description | Example |
|-----|-------------|---------|
| `<b 0x...>` | Fixed bytes in hex | `<b 0xc0ff>` |
| `<r N>` | Random bytes (N bytes) | `<r 8>` |
| `<t>` | Timestamp (4 bytes) | `<t>` |
| `<c>` | Counter | `<c>` |
| `<rc N>` | Random characters (N bytes) | `<rc 7>` |
| `<rd N>` | Random digits (N bytes) | `<rd 2>` |

### Protocols

Each protocol generates different I1-I5 patterns:

- **quic** - Mimics QUIC Initial packets with long headers, DCID, timestamps
- **dns** - Mimics DNS query packets with transaction IDs and domain structure
- **dtls** - Mimics DTLS 1.2 ClientHello packets with handshake headers
- **stun** - Mimics STUN Binding Request packets with magic cookie
- **random** - Selects a protocol deterministically based on the string length (defaults to DTLS)

## Library Usage

Amnezigo can be used as a Go library. See [docs/library-usage.md](docs/library-usage.md) for detailed documentation.

```go
import "github.com/Arsolitt/amnezigo"

func main() {
    // Generate keypair
    privateKey, publicKey := amnezigo.GenerateKeyPair()
    
    // Create manager for config operations
    mgr := amnezigo.NewManager("awg0.conf")
    
    // Add a peer
    peer, err := mgr.AddPeer("laptop", "")
    if err != nil {
        log.Fatal(err)
    }
    
    // Export peer config
    clientCfg, err := mgr.ExportPeer("laptop", "quic", "1.2.3.4:51820")
    if err != nil {
        log.Fatal(err)
    }
}
```

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
