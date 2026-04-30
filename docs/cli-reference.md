# CLI Reference

> Complete reference for all `amnezigo` CLI commands, flags, and behavior.

## Table of Contents

- [Overview](#overview)
- [Global Flags](#global-flags)
- [amnezigo init](#amnezigo-init)
- [amnezigo add](#amnezigo-add)
- [amnezigo list](#amnezigo-list)
- [amnezigo export](#amnezigo-export)
- [amnezigo remove](#amnezigo-remove)
- [amnezigo edit](#amnezigo-edit)
- [amnezigo validate](#amnezigo-validate)

---

## Overview

`amnezigo` generates AmneziaWG v2.0 configurations for star topology networks. See [installation.md](./installation.md) for install instructions.

All commands accept a `--config` flag to specify the server config file (default: `awg0.conf`). CLI commands are thin wrappers over the `amnezigo` library package — see [configuration.md](./configuration.md) for config file format details.

```
amnezigo [command] [flags]
```

Available commands: `init`, `add`, `list`, `export`, `remove`, `edit`, `validate`.

---

## Global Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--config` | string | `awg0.conf` | Path to the server config file |

---

## amnezigo init

Initialize a new AmneziaWG server configuration. Generates a server keypair, preshared key, obfuscation parameters, iptables rules, and writes the config file.

```
amnezigo init --ipaddr <CIDR> [flags]
```

### Flags

| Flag | Type | Required | Default | Description |
|---|---|---|---|---|
| `--ipaddr` | string | yes | — | Server IP address with subnet (e.g. `10.8.0.1/24`) |
| `--port` | int | no | random (10000–65535) | Listen port |
| `--mtu` | int | no | `1280` | MTU size |
| `--dns` | string | no | `1.1.1.1, 8.8.8.8` | DNS servers (comma-separated) |
| `--keepalive` | int | no | `25` | Persistent keepalive interval in seconds |
| `--client-to-client` | bool | no | `false` | Allow peer-to-peer traffic |
| `--iface` | string | no | auto-detect | Main network interface for NAT |
| `--iface-name` | string | no | `awg0` | Tunnel interface name |
| `--endpoint-v4` | string | no | auto-detect | IPv4 endpoint address |
| `--endpoint-v6` | string | no | auto-detect | IPv6 endpoint address |
| `--config` | string | no | `awg0.conf` | Server config file path |

### Behavior

- Creates the server config file at the `--config` path.
- Creates a `.main.config` file in the current directory (mode 0600) recording the config path.
- Auto-detects the main network interface if `--iface` is not set.
- Auto-detects IPv4 endpoint via `ipv4.icanhazip.com` and IPv6 endpoint via `ipv6.icanhazip.com` (5 s timeout) if `--endpoint-v4` or `--endpoint-v6` are not set. IPv6 endpoints are wrapped in brackets: `[2001:db8::1]:51820`.
- DNS and keepalive values **are** stored in the server config.

### Example

```shell
$ amnezigo init --ipaddr 10.8.0.1/24 --port 51820 --iface eth0
✓ AmneziaWG configuration initialized successfully
  Config: awg0.conf
  Server IP: 10.8.0.1/24
  Listen Port: 51820
  Main Interface: eth0
  IPv4 Endpoint: 203.0.113.1
  IPv6 Endpoint: [2001:db8::1]
```

---

## amnezigo add

Add a new peer to the server configuration. Generates a keypair and preshared key for the peer.

```
amnezigo add <name> [flags]
```

### Arguments

| Arg | Required | Description |
|---|---|---|
| `name` | yes | Peer name (must be unique) |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--ipaddr` | string | auto-assign | Peer IP address (auto-assigned from server subnet if omitted) |
| `--config` | string | `awg0.conf` | Server config file |

### Behavior

- Auto-assigns the next available IP from the server subnet (starts at `.2`, up to `.254`). Skips IPs already in use.
- IPv4 subnets only.
- Writes the updated config atomically (write to `.tmp`, then rename).

### Errors

| Condition | Message |
|---|---|
| Duplicate peer name | `peer with name '<name>' already exists` |
| Config file not found | `failed to load server config: ...` |
| Invalid server address | `invalid server address: ...` |
| No available IPs | `failed to assign IP address: ...` |

### Example

```shell
$ amnezigo add laptop
Peer 'laptop' added successfully
  IP Address: 10.8.0.2/32
  Public Key: aB3cD4eF5gH6iJ7kL8mN9oP0qR1sT2uV3wX4yZ5A6B=

$ amnezigo add phone --ipaddr 10.8.0.10
Peer 'phone' added successfully
  IP Address: 10.8.0.10/32
  Public Key: zY9xW8vU7tS6rQ5pO4nM3lK2jI1hG0fD9eC8bA7Z6Y=
```

---

## amnezigo list

List all configured peers.

```
amnezigo list [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--config` | string | `awg0.conf` | Server config file |

### Output

Displays a tab-separated table with columns: **NAME**, **IP**, **CREATED**. Timestamps use `YYYY-MM-DD HH:MM` format. Prints `No peers configured` when there are no peers.

### Example

```shell
$ amnezigo list
NAME    IP              CREATED
laptop  10.8.0.2/32     2026-04-04 14:30
phone   10.8.0.10/32    2026-04-04 14:31
```

---

## amnezigo export

Export peer configuration(s) as client `.conf` files.

```
amnezigo export [name] [flags]
```

### Arguments

| Arg | Required | Description |
|---|---|---|
| `name` | no | Peer name. If omitted, exports all peers. |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--protocol` | string | `random` | Obfuscation protocol: `random`, `quic`, `dns`, `dtls`, `stun` |
| `--endpoint` | string | auto-resolve | Override endpoint (skips auto-detection) |
| `--config` | string | `awg0.conf` | Server config file |

### Protocol Options

Protocol controls the obfuscation template used in the exported client config. See [obfuscation.md](./obfuscation.md) for details on each protocol.

| Value | Description |
|---|---|
| `random` | Simple random CPS strings |
| `quic` | QUIC protocol template |
| `dns` | DNS protocol template |
| `dtls` | DTLS protocol template |
| `stun` | STUN protocol template |

### Endpoint Resolution

When `--endpoint` is not provided, the endpoint is resolved in this order:

1. `EndpointV4` stored in server config
2. `EndpointV6` stored in server config
3. Auto-detected via `icanhazip.com`
4. Falls back to `YOUR_SERVER_IP:<port>`

### Behavior

- Writes `<peer_name>.conf` files in the **current working directory** (not relative to the server config). File mode: `0600`.
- I1–I5 CPS strings are generated per-peer at export time.
- When exporting all peers, one file is written per peer.

### Example

```shell
$ amnezigo export laptop --protocol quic
Exported peer 'laptop' to laptop.conf

$ amnezigo export --protocol dns
Exported peer 'laptop' to laptop.conf
Exported peer 'phone' to phone.conf
```

---

## amnezigo remove

Remove a peer from the server configuration.

```
amnezigo remove <name> [flags]
```

### Arguments

| Arg | Required | Description |
|---|---|---|
| `name` | yes | Peer name to remove |

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--config` | string | `awg0.conf` | Server config file |

### Behavior

- Removes the peer from the server config with an atomic write.
- The peer's IP address is freed but **not** automatically reassigned to new peers.

### Errors

| Condition | Message |
|---|---|
| Peer not found | `peer '<name>' not found` |

### Example

```shell
$ amnezigo remove phone
Peer 'phone' removed successfully
```

---

## amnezigo edit

Edit server configuration parameters.

```
amnezigo edit [flags]
```

### Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--client-to-client` | string | — | Enable or disable client-to-client traffic (`true`/`false`) |
| `--config` | string | `awg0.conf` | Server config file |

### Behavior

- When `--client-to-client` is changed, PostUp/PostDown iptables rules are regenerated. All other config fields are preserved.
- When disabling client-to-client, the command prints an iptables command to run manually for immediate effect.
- Prints `No changes specified` if no edit flags are provided.
- Saves the config with an atomic write.

### Example

```shell
$ amnezigo edit --client-to-client false
Run this command to disable client-to-client immediately:
  iptables -D FORWARD -i awg0 -o awg0 -j ACCEPT

✓ Configuration updated
  Restart AmneziaWG service to apply changes
```

---

## amnezigo validate

Validate an existing AmneziaWG server config against AWG 2.0 invariants. Runs every check the generator enforces (size collisions, header ranges, required fields, deprecated tags) and reports findings with severity levels.

```text
amnezigo validate <config> [flags]
```

### Flags

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--output` | string | `text` | Output format: `text` or `json` |
| `--strict` | bool | `false` | Treat warnings as errors for exit code |
| `--quiet` | bool | `false` | Suppress summary line; print findings only |

### Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | No errors (warnings and info findings may still be printed) |
| `1` | At least one error, or any warning when `--strict` is set |

### Validation Codes

Each finding carries a stable machine-readable code. Codes are safe to grep in CI pipelines.

| Code | Severity | Description |
| --- | --- | --- |
| `PSC001` | error | Pairwise S-prefix size collision |
| `PSC002` | error | Junk range contains a forbidden padded or raw WG size |
| `HDR001` | error | H1-H4 range overlaps WG message type-ids (1..4) |
| `HDR002` | error | H1-H4 range structurally invalid (Max < Min) |
| `FLD001` | error | Required field missing (PrivateKey, Address, or ListenPort) |
| `JNK001` | error | Junk range structurally invalid (Jmin > Jmax) |
| `PSE001` | error | Config parse aborted due to structural error |
| `CPS001` | warning | Raw `<c>` (counter) tag detected; breaks mobile clients |
| `KEY001` | warning | Unknown INI key |

### Behavior

- Reads the config in strict mode: unknown keys and deprecated `<c>` tags are reported as warnings instead of being silently ignored.
- If the config fails to parse (structural error), a single `PSE001` finding is emitted and validation stops.
- Server configs only in v1. Client config validation is planned for a future release.
- The command is read-only; no files are modified.

### Sample Output (clean config, text)

```text
$ amnezigo validate /etc/amnezia/awg0.conf
✓ /etc/amnezia/awg0.conf: 0 errors, 0 warnings, 0 info
```

### Sample Output (with findings, text)

```text
$ amnezigo validate /tmp/server.conf
[ERROR PSC001] /tmp/server.conf: packet size collision (s-pair): S1+148 vs S2+92 at 148 bytes
[WARNING KEY001] /tmp/server.conf:42 (key=Foobar): unknown INI key
✗ /tmp/server.conf: 1 error, 1 warning, 0 info
```

### Sample Output (clean config, JSON)

```json
{
  "file": "/etc/amnezia/awg0.conf",
  "findings": [],
  "summary": {
    "errors": 0,
    "warnings": 0,
    "info": 0
  }
}
```

### Example

```shell
$ amnezigo validate awg0.conf
✓ awg0.conf: 0 errors, 0 warnings, 0 info

$ amnezigo validate awg0.conf --output json | jq .summary
{
  "errors": 0,
  "warnings": 0,
  "info": 0
}

$ amnezigo validate legacy.conf --strict
[WARNING CPS001] legacy.conf:12: raw <c> tag detected; rejected by amneziawg-go and AmneziaVPN clients
✗ legacy.conf: 0 errors, 1 warning, 0 info
```
