# Using Amnezigo as a Go Library

This guide explains how to use `github.com/Arsolitt/amnezigo` as a Go library for managing AmneziaWG v2.0 configurations programmatically.

## Installation

```bash
go get github.com/Arsolitt/amnezigo
```

Import the root package in your Go code:

```go
import "github.com/Arsolitt/amnezigo"
```

## Quick Start

Here's a complete example that creates a server configuration, adds a client, and exports the client config:

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    // Create a manager for the server config file
    manager := amnezigo.NewManager("awg0.conf")

    // Create a new server configuration
    cfg := &amnezigo.ServerConfig{
        Interface: amnezigo.InterfaceConfig{
            PrivateKey:    amnezigo.GenerateKeyPair(), // Returns private key
            Address:       "10.8.0.1/24",
            ListenPort:    51820,
            DNS:           "1.1.1.1",
            MTU:           1280,
            Obfuscation:   amnezigo.GenerateServerConfig("quic", 15, 3),
        },
    }

    // Save the initial config
    if err := manager.Save(cfg); err != nil {
        log.Fatalf("Failed to save config: %v", err)
    }

    // Add a client with auto-assigned IP
    peer, err := manager.AddClient("laptop", "")
    if err != nil {
        log.Fatalf("Failed to add client: %v", err)
    }
    fmt.Printf("Added client 'laptop' with IP: %s\n", peer.AllowedIPs)

    // Export client configuration for connection
    clientCfg, err := manager.ExportClient("laptop", "quic", "203.0.113.50:51820")
    if err != nil {
        log.Fatalf("Failed to export client: %v", err)
    }

    // Write client config to file
    clientFile, err := os.Create("laptop.conf")
    if err != nil {
        log.Fatalf("Failed to create client file: %v", err)
    }
    defer clientFile.Close()

    if err := amnezigo.WriteClientConfig(clientFile, clientCfg); err != nil {
        log.Fatalf("Failed to write client config: %v", err)
    }

    fmt.Println("Client configuration exported to laptop.conf")
}
```

---

## Manager API

The `Manager` type provides a high-level CRUD interface for managing server configurations and clients.

### Creating a Manager

```go
manager := amnezigo.NewManager("/path/to/awg0.conf")
```

### Loading Configuration

```go
cfg, err := manager.Load()
if err != nil {
    // Handle error - file may not exist or be malformed
}
```

### Saving Configuration

```go
err := manager.Save(cfg)
if err != nil {
    // Handle error
}
```

### Adding a Client

```go
// Auto-assign IP from subnet
peer, err := manager.AddClient("laptop", "")

// Specify IP manually
peer, err := manager.AddClient("desktop", "10.8.0.50/32")
```

**Returns:** `*PeerConfig` with the new client's configuration.

**Note:** The client name is stored in the config file as a metadata comment (`#_Name = laptop`).

### Removing a Client

```go
err := manager.RemoveClient("laptop")
if err != nil {
    // Client not found or other error
}
```

### Finding a Client

```go
peer, err := manager.FindClient("laptop")
if err != nil {
    // Client not found
}
fmt.Printf("Public Key: %s\n", peer.PublicKey)
fmt.Printf("IP: %s\n", peer.AllowedIPs)
```

### Listing All Clients

```go
peers := manager.ListClients()
for _, peer := range peers {
    fmt.Printf("Name: %s, IP: %s\n", peer.Name, peer.AllowedIPs)
}
```

### Exporting Client Configuration

Generate a complete client configuration file content:

```go
// Protocol options: "quic", "dns", "dtls", "stun", "random"
clientCfg, err := manager.ExportClient("laptop", "quic", "203.0.113.50:51820")
if err != nil {
    // Client not found
}
```

### Building Client Configuration from Peer

If you have a `PeerConfig` already, use `BuildClientConfig`:

```go
peer, _ := manager.FindClient("laptop")
clientCfg, err := manager.BuildClientConfig(peer, "quic", "203.0.113.50:51820")
```

---

## Config Parsing & Writing

### Parsing Server Config from Reader

```go
file, err := os.Open("awg0.conf")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

cfg, err := amnezigo.ParseServerConfig(file)
if err != nil {
    log.Fatal(err)
}
```

### Writing Server Config to Writer

```go
var buf bytes.Buffer
err := amnezigo.WriteServerConfig(&buf, cfg)
if err != nil {
    log.Fatal(err)
}
fmt.Println(buf.String())
```

### Writing Client Config to Writer

```go
clientFile, err := os.Create("client.conf")
if err != nil {
    log.Fatal(err)
}
defer clientFile.Close()

err = amnezigo.WriteClientConfig(clientFile, clientCfg)
```

### Convenience File Functions

```go
// Load from file path
cfg, err := amnezigo.LoadServerConfig("awg0.conf")

// Save to file path (uses atomic write: writes to .tmp, then renames)
err := amnezigo.SaveServerConfig("awg0.conf", cfg)
```

---

## Key Generation

### Generate Key Pair

Generates a WireGuard-compatible key pair:

```go
privateKey, publicKey := amnezigo.GenerateKeyPair()
fmt.Printf("Private: %s\n", privateKey)
fmt.Printf("Public: %s\n", publicKey)
```

**Note:** Panics if `crypto/rand` fails (treated as unrecoverable system error).

### Derive Public Key from Private

```go
privateKey := "aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890ABCDEF="
publicKey := amnezigo.DerivePublicKey(privateKey)
```

### Generate Preshared Key

```go
psk := amnezigo.GeneratePSK()
```

**Note:** Also panics on `crypto/rand` failure.

---

## Obfuscation

AmneziaWG uses obfuscation parameters to disguise WireGuard traffic as other protocols.

### Generate Client Obfuscation Config

```go
// Parameters: protocol, mtu, junkPacketCount, initPacketJunkSize
clientObf := amnezigo.GenerateConfig("quic", 1280, 15, 3)
```

This generates:
- `I1-I5` CPS strings for the protocol
- `H1-H4` header values (point values, not ranges)
- `S1-S4` prefix values
- `Jc`, `Jmin`, `Jmax` junk parameters

### Generate Server Obfuscation Config

```go
// Parameters: protocol (ignored), junkPacketCount, initPacketJunkSize
serverObf := amnezigo.GenerateServerConfig("quic", 15, 3)
```

**Note:** The protocol parameter is ignored by `GenerateServerConfig`. Server uses true ranges for H1-H4.

### Generate CPS Strings

Generate only the I1-I5 CPS strings:

```go
i1, i2, i3, i4, i5 := amnezigo.GenerateCPS("quic", 1280, 15, 3)
fmt.Printf("I1: %s\n", i1)
```

### Individual Generators

```go
// Header values (H1-H4 as point values)
headers := amnezigo.GenerateHeaders()
fmt.Printf("H1: %d, H2: %d, H3: %d, H4: %d\n", 
    headers.H1, headers.H2, headers.H3, headers.H4)

// SPrefixes (S1-S4)
prefixes := amnezigo.GenerateSPrefixes()
fmt.Printf("S1: %d, S2: %d, S3: %d, S4: %d\n",
    prefixes.S1, prefixes.S2, prefixes.S3, prefixes.S4)

// Junk parameters
junk := amnezigo.GenerateJunkParams()
fmt.Printf("Jc: %d, Jmin: %d, Jmax: %d\n", junk.Jc, junk.Jmin, junk.Jmax)

// Header ranges (for server config)
ranges := amnezigo.GenerateHeaderRanges()
fmt.Printf("H1: %d-%d\n", ranges.H1.Min, ranges.H1.Max)
```

---

## CPS Construction

CPS (Client Packet Size) strings are built from tag specifications.

### Building Tags

```go
// Byte tag: <b 0xc0ff>
tag := amnezigo.BuildCPSTag("b", "0xc0ff")

// Random bytes tag: <r 16>
tag := amnezigo.BuildCPSTag("r", "16")

// Other tag types: "t" (timestamp), "c" (checksum)
```

### Building Complete CPS

```go
tags := []string{
    amnezigo.BuildCPSTag("b", "0xc0ff"),
    amnezigo.BuildCPSTag("r", "16"),
    "<t>",
    "<c>",
}
cps := amnezigo.BuildCPS(tags)
// Result: "<b 0xc0ff><r 16><t><c>"
```

---

## Protocol Templates

Get the I1-I5 tag templates for each protocol:

```go
// QUIC protocol template
quic := amnezigo.QUICTemplate()
fmt.Printf("I1 tags: %+v\n", quic.I1)

// DNS protocol template
dns := amnezigo.DNSTemplate()

// DTLS protocol template
dtls := amnezigo.DTLSTemplate()

// STUN protocol template
stun := amnezigo.STUNTemplate()
```

Each template contains `I1`, `I2`, `I3`, `I4`, `I5` fields, each being a `[]TagSpec`.

---

## Network Helpers

### IP Address Validation

```go
valid := amnezigo.IsValidIPAddr("10.8.0.1/24")  // true
valid := amnezigo.IsValidIPAddr("invalid")      // false
```

### Subnet Extraction

```go
subnet := amnezigo.ExtractSubnet("10.8.0.1/24")  // "10.8.0.0/24"
```

### Random Port Generation

```go
port, err := amnezigo.GenerateRandomPort()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Port: %d\n", port)  // e.g., 52847
```

### Main Interface Detection

```go
iface := amnezigo.DetectMainInterface()
fmt.Printf("Main interface: %s\n", iface)  // e.g., "eth0"
```

### Find Next Available IP

```go
existingIPs := map[string]bool{
    "10.8.0.1": true,
    "10.8.0.2": true,
}

ip, err := amnezigo.FindNextAvailableIP("10.8.0.1/24", existingIPs)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Next available: %s\n", ip)  // "10.8.0.3"
```

---

## iptables Rules

Generate PostUp and PostDown commands for NAT/masquerade:

```go
interface := "awg0"
mainIface := "eth0"
subnet := "10.8.0.0/24"
ipv6 := false

postUp := amnezigo.GeneratePostUp(interface, mainIface, subnet, ipv6)
postDown := amnezigo.GeneratePostDown(interface, mainIface, subnet, ipv6)

fmt.Println("PostUp commands:")
fmt.Println(postUp)
fmt.Println("\nPostDown commands:")
fmt.Println(postDown)
```

Example output:
```
PostUp: iptables -A FORWARD -i awg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown: iptables -D FORWARD -i awg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
```

---

## Type Reference

### Configuration Types

```go
// Main server configuration
type ServerConfig struct {
    Interface InterfaceConfig
    Peers     []*PeerConfig
}

// Server interface section
type InterfaceConfig struct {
    PrivateKey    string
    Address       string
    ListenPort    int
    DNS           string
    MTU           int
    Obfuscation   ServerObfuscationConfig
}

// Peer/client configuration (server-side)
type PeerConfig struct {
    Name           string  // Stored as metadata comment
    PublicKey      string
    PresharedKey   string
    AllowedIPs     string
}

// Client configuration (for export)
type ClientConfig struct {
    Interface ClientInterfaceConfig
    Peer      ClientPeerConfig
}

// Client interface section
type ClientInterfaceConfig struct {
    PrivateKey    string
    Address       string
    DNS           string
    MTU           int
    Obfuscation   ClientObfuscationConfig
}

// Client peer section
type ClientPeerConfig struct {
    PublicKey           string
    PresharedKey        string
    Endpoint            string
    AllowedIPs          string
    PersistentKeepalive int
}
```

### Obfuscation Types

```go
// Server-side obfuscation (H1-H4 as ranges)
type ServerObfuscationConfig struct {
    Jc                int
    Jmin              int
    Jmax              int
    H1, H2, H3, H4    HeaderRange
    S1, S2, S3, S4    int
    I1, I2, I3, I4, I5 string
}

// Client-side obfuscation (H1-H4 as point values)
type ClientObfuscationConfig struct {
    Jc                int
    Jmin              int
    Jmax              int
    H1, H2, H3, H4    uint32
    S1, S2, S3, S4    int
    I1, I2, I3, I4, I5 string
}

// Header range (min-max)
type HeaderRange struct {
    Min uint32
    Max uint32
}

// Header point values
type Headers struct {
    H1, H2, H3, H4 uint32
}

// SPrefixes (S1-S4)
type SPrefixes struct {
    S1, S2, S3, S4 int
}

// Junk parameters
type JunkParams struct {
    Jc, Jmin, Jmax int
}

// CPS configuration
type CPSConfig struct {
    I1, I2, I3, I4, I5 string
}

// Tag specification for CPS construction
type TagSpec struct {
    Type  string  // "b", "r", "t", "c"
    Value string
}

// I1-I5 template for a protocol
type I1I5Template struct {
    I1, I2, I3, I4, I5 []TagSpec
}
```

### Manager Type

```go
type Manager struct {
    ConfigPath string
}
```

---

## Gotchas & Important Notes

### Hardcoded Values

1. **DNS in client exports** is hardcoded to `"1.1.1.1, 8.8.8.8"` in `BuildClientConfig`.

2. **AllowedIPs in client exports** is always `"0.0.0.0/0, ::/0"` (tunnel all traffic).

3. **PersistentKeepalive** is hardcoded to `25` seconds in client exports.

### Protocol Behavior

4. **"random" protocol** deterministically selects DTLS. It's not truly random.

5. **GenerateServerConfig ignores the protocol parameter**. The server config doesn't need protocol-specific I1-I5 values.

### Obfuscation Differences

6. **GenerateConfig vs GenerateServerConfig**:
   - `GenerateConfig`: Uses point values for H1-H4 (for clients)
   - `GenerateServerConfig`: Uses ranges for H1-H4 (for servers)

### Error Handling

7. **Key generation panics**: `GenerateKeyPair()` and `GeneratePSK()` panic if `crypto/rand` fails. This is by design—these are treated as unrecoverable system failures.

### Client Names

8. **Client names are metadata**: The `Name` field in `PeerConfig` is stored as a comment (`#_Name = value`) in the config file, not as a native WireGuard field.

### IP Assignment

9. **Auto IP assignment** scans from `.2` to `.254` in the subnet. The `.1` address is reserved for the server.

### File Writes

10. **Atomic writes**: `SaveServerConfig` and `Save` use atomic writes (write to `.tmp` file, then rename) to prevent corruption from partial writes.

---

## Complete Example: VPN Setup Script

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/Arsolitt/amnezigo"
)

func main() {
    configPath := "/etc/amneziawg/awg0.conf"
    endpoint := "203.0.113.50:51820"
    protocol := "quic"

    // Create manager
    manager := amnezigo.NewManager(configPath)

    // Check if config exists
    cfg, err := manager.Load()
    if err != nil {
        // Create new config
        privKey, _ := amnezigo.GenerateKeyPair()
        port, _ := amnezigo.GenerateRandomPort()
        mainIface := amnezigo.DetectMainInterface()

        cfg = &amnezigo.ServerConfig{
            Interface: amnezigo.InterfaceConfig{
                PrivateKey:  privKey,
                Address:     "10.8.0.1/24",
                ListenPort:  51820,
                DNS:         "1.1.1.1",
                MTU:         1280,
                Obfuscation: amnezigo.GenerateServerConfig(protocol, 15, 3),
            },
        }

        // Generate iptables rules
        postUp := amnezigo.GeneratePostUp("awg0", mainIface, "10.8.0.0/24", false)
        postDown := amnezigo.GeneratePostDown("awg0", mainIface, "10.8.0.0/24", false)

        fmt.Println("PostUp:", postUp)
        fmt.Println("PostDown:", postDown)

        if err := manager.Save(cfg); err != nil {
            log.Fatalf("Failed to save config: %v", err)
        }
        fmt.Println("Created new server configuration")
    }

    // Add clients from command line args
    clients := os.Args[1:]
    for _, name := range clients {
        peer, err := manager.AddClient(name, "")
        if err != nil {
            log.Printf("Failed to add %s: %v", name, err)
            continue
        }

        clientCfg, err := manager.ExportClient(name, protocol, endpoint)
        if err != nil {
            log.Printf("Failed to export %s: %v", name, err)
            continue
        }

        filename := fmt.Sprintf("%s.conf", name)
        file, err := os.Create(filename)
        if err != nil {
            log.Printf("Failed to create %s: %v", filename, err)
            continue
        }

        if err := amnezigo.WriteClientConfig(file, clientCfg); err != nil {
            log.Printf("Failed to write %s: %v", filename, err)
        } else {
            fmt.Printf("Created %s for %s (%s)\n", filename, name, peer.AllowedIPs)
        }
        file.Close()
    }

    // List all clients
    fmt.Println("\nCurrent clients:")
    for _, peer := range manager.ListClients() {
        fmt.Printf("  - %s: %s\n", peer.Name, peer.AllowedIPs)
    }
}
```

---

## See Also

- [AmneziaWG Protocol Documentation](https://docs.amnezia.org/)
- [WireGuard Go Implementation](https://github.com/WireGuard/wireguard-go)
