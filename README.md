# Amnezigo

**Amnezia** + **Go** = **Amnezigo** 🎮

A CLI tool for generating and managing [AmneziaWG](https://github.com/amnezia-vpn/amneziawg) v2.0 configurations.

## Features

- Generate AmneziaWG server configurations with obfuscation parameters
- Manage clients (add, remove, list, export)
- Multiple obfuscation protocols (QUIC, DNS, DTLS, STUN)
- Automatic IP assignment for clients
- iptables rules generation for NAT and forwarding

## Installation

```bash
go install github.com/Arsolitt/amnezigo@latest
```

Or build from source:

```bash
git clone https://github.com/Arsolitt/amnezigo.git
cd amnezigo
go build -o amnezigo .
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
- `--dns` - DNS servers (default: "1.1.1.1, 8.8.8.8")
- `--keepalive` - Persistent keepalive interval (default: 25)
- `--protocol` - Obfuscation protocol: random, quic, dns, dtls, stun (default: random)
- `--client-to-client` - Allow client-to-client traffic
- `--iface` - Main network interface (default: auto-detect)

### Add Client

```bash
amnezigo add laptop
amnezigo add phone --ipaddr 10.8.0.50
```

### List Clients

```bash
amnezigo list
```

### Export Client Config

```bash
# Export single client
amnezigo export laptop --endpoint 1.2.3.4:55424

# Export all clients
amnezigo export --endpoint 1.2.3.4:55424
```

Options:
- `--endpoint` - Server endpoint (default: auto-detect external IP)

### Remove Client

```bash
amnezigo remove laptop
```

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
H1 = 1827682742
H2 = 742172841
H3 = 1928417263
H4 = 281746291
I1 = <b 0xc0000000><r 16>
I2 = <b 0x40000000><r 12>
I3 = <b 0x80000000><t>
I4 = <b 0xc0000000><c>
I5 = <r 8>

[Peer]
#_Name = laptop
#_PrivateKey = <client-private-key>
PublicKey = <client-public-key>
AllowedIPs = 10.8.0.2/32
```

### Client Config (laptop.conf)

```ini
[Interface]
PrivateKey = <client-private-key>
Address = 10.8.0.2/32
DNS = 1.1.1.1, 8.8.8.8
MTU = 1280
Jc = 3
Jmin = 50
Jmax = 1000
...

[Peer]
PublicKey = <server-public-key>
PresharedKey = <psk>
Endpoint = 1.2.3.4:55424
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
```

## Obfuscation Parameters

AmneziaWG uses several parameters to obfuscate WireGuard traffic:

- **Jc, Jmin, Jmax** - Junk packet parameters
- **S1-S4** - Size prefixes
- **H1-H4** - Header values (non-overlapping uint32 regions)
- **I1-I5** - Custom Packet Strings (CPS) based on protocol template
