# LLM Documentation (llms-full.txt) Design

## Goal

Create a single-file LLM-friendly documentation for the Amnezigo project, following the format established by the Xray project's `llms-full.txt` (https://xtls.github.io/llms-full.txt).

## Deliverable

A single file `llms-full.txt` in the project root containing concatenated markdown pages covering all aspects of the Amnezigo CLI tool and Go library.

## Format

**File:** `llms-full.txt` in project root

**Format:** Single plain text file with concatenated markdown pages, separated by frontmatter:

```
---
url: /page-slug
---
# Page Title
...content...
---
```

**Language:** English with technical terms as-is

**Markdown conventions:**
- `> \`fieldName\`: type` blockquote pattern for documenting config parameters
- `::: tip`, `::: warning`, `::: danger` containers for callouts
- `shell` fenced blocks for CLI commands (no prompt prefix)
- `ini` fenced blocks for config file examples
- `go` fenced blocks for code examples
- Tables for command/flag summaries
- H1/H2/H3 heading hierarchy

## Page Structure

15 pages ordered progressively from beginner to reference:

| # | URL Slug | Title | Depth |
|---|----------|-------|-------|
| 1 | `/overview` | Overview | Tutorial-style, verbose |
| 2 | `/installation` | Installation | Tutorial-style, verbose |
| 3 | `/quick-start` | Quick Start | Tutorial-style, verbose, step-by-step with annotated examples |
| 4 | `/server-setup` | Server Setup | Moderate, workflow-oriented with config examples |
| 5 | `/client-management` | Client Management | Moderate, workflow-oriented with config examples |
| 6 | `/edge-servers` | Edge Servers | Moderate, workflow-oriented with config examples |
| 7 | `/obfuscation` | Obfuscation | Conceptual explanations + reference tables |
| 8 | `/cps` | Custom Packet Strings | Conceptual explanations + reference tables |
| 9 | `/protocols` | Protocol Templates | Conceptual explanations + reference tables |
| 10 | `/server-config-reference` | Server Config Reference | Terse parameter-by-parameter reference |
| 11 | `/client-config-reference` | Client Config Reference | Terse parameter-by-parameter reference |
| 12 | `/cli-reference` | CLI Reference | Tables of commands/flags with descriptions |
| 13 | `/library-api` | Library API | Go code examples for each exported function |
| 14 | `/type-reference` | Type Reference | Struct definitions with field-by-field docs |
| 15 | `/gotchas` | Gotchas & FAQ | Numbered list with explanations |

## Page Content Details

### 1. Overview
- What is Amnezigo (Amnezia + Go)
- What is AmneziaWG v2.0
- Star topology explanation
- Feature list (obfuscation, client/edge management, IP auto-assignment, iptables, protocol templates, CPS)
- CLI vs Library usage modes

### 2. Installation
- Prerequisites (Go 1.26+)
- `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest`
- Build from source (`go build -o build/amnezigo ./cmd/amnezigo/`)
- Docker (`docker build -t amnezigo .`, `docker run amnezigo`)
- Verifying installation

### 3. Quick Start
- Step-by-step init workflow with full command output
- Adding a client with output
- Exporting a client config with resulting .conf content
- Connecting with AmneziaWG client
- Full annotated config examples

### 4. Server Setup
- `init` command deep dive: all flags with descriptions
- Auto-detection behavior (interface, port, endpoints)
- Generated server config explained field-by-field
- iptables rules explanation
- `.main.config` file
- `edit` command (client-to-client toggle)

### 5. Client Management
- `client add` with auto-IP and explicit IP
- `client list` output format
- `client export` single and all clients
- Endpoint resolution order (EndpointV4 -> EndpointV6 -> HTTP fallback)
- Protocol flag and CPS generation
- `client remove`
- Name uniqueness constraint
- IP pool sharing with edges

### 6. Edge Servers
- Hub-and-spoke topology concept
- Edge vs client differences (no DNS, AllowedIPs=hub only)
- `edge add/list/export/remove` commands
- Edge config structure
- Use cases

### 7. Obfuscation
- What obfuscation is and why it exists
- Jc (junk count), Jmin, Jmax parameters
- S1-S4 (size prefixes)
- H1-H4 (header magic values) as min-max ranges
- Point ranges (GenerateConfig) vs true ranges (GenerateServerConfig)
- How parameters relate to packet construction

### 8. Custom Packet Strings
- What CPS (I1-I5) are
- Tag syntax reference table: `<b 0x..>`, `<r N>`, `<rc N>`, `<rd N>`, `<c>`, `<t>`
- MTU constraint formula: maxISize = MTU - 49 - 149 - S1
- Progressive tag reduction when CPS exceeds max size
- Fallback to `<t>`
- Per-client generation (not stored in server config)

### 9. Protocol Templates
- QUIC: mimics QUIC Initial packets
- DNS: mimics DNS query packets
- DTLS: mimics DTLS 1.2 ClientHello
- STUN: mimics STUN Binding Request (RFC 5389)
- Random: deterministic selection (always DTLS due to `len("random") % 4 = 2`)
- When to use each protocol
- I1-I5 size progression (I1 largest, I5 empty)

### 10. Server Config Reference
- [Interface] section: PrivateKey, PublicKey, Address, ListenPort, MTU, PostUp, PostDown
- Obfuscation fields: Jc, Jmin, Jmax, S1-S4, H1-H4
- Metadata comments: #_Name, #_Role, #_PrivateKey, #_GenKeyTime, #_EndpointV4, #_EndpointV6, #_ClientToClient, #_TunName, #_MainIface
- [Peer] section fields
- Full annotated server config example

### 11. Client Config Reference
- [Interface] section: PrivateKey, Address, DNS, MTU
- Client obfuscation: Jc, Jmin, Jmax, S1-S4, H1-H4, I1-I5
- [Peer] section: PublicKey, PresharedKey, Endpoint, AllowedIPs, PersistentKeepalive
- Client config vs Edge config differences
- Full annotated client and edge config examples

### 12. CLI Reference
- All commands in table format with flags, defaults, required/optional
- `init` command and all flags
- `edit` command and all flags
- `client add/list/export/remove` and all flags
- `edge add/list/export/remove` and all flags
- Global `--config` flag
- Common usage patterns

### 13. Library API
- Manager (NewManager, Load, Save, AddClient, RemoveClient, FindClient, ListClients, ExportClient, BuildClientConfig, AddEdge, RemoveEdge, FindEdge, ListEdges, BuildEdgeConfig, ExportEdge)
- Config I/O (ParseServerConfig, WriteServerConfig, WriteClientConfig, LoadServerConfig, SaveServerConfig)
- Key Generation (GenerateKeyPair, DerivePublicKey, GeneratePSK)
- Obfuscation (GenerateConfig, GenerateServerConfig, GenerateHeaders, GenerateSPrefixes, GenerateJunkParams, GenerateCPS, GenerateHeaderRanges)
- CPS (BuildCPSTag, BuildCPS)
- Protocol Templates (QUICTemplate, DNSTemplate, DTLSTemplate, STUNTemplate)
- Helpers (IsValidIPAddr, ExtractSubnet, GenerateRandomPort, DetectMainInterface, FindNextAvailableIP)
- iptables (GeneratePostUp, GeneratePostDown)
- Each function with Go code example and description

### 14. Type Reference
- ServerConfig, InterfaceConfig, PeerConfig
- ServerObfuscationConfig, ClientObfuscationConfig
- ClientConfig, ClientInterfaceConfig, ClientPeerConfig
- HeaderRange, Headers, SPrefixes, JunkParams
- CPSConfig, TagSpec, I1I5Template
- Manager
- Each struct with field names, types, and descriptions

### 15. Gotchas & FAQ
- `--dns` and `--keepalive` on init are silently ignored
- "random" protocol always selects DTLS
- Export has no `--endpoint` flag; endpoint auto-resolved
- `#_Role` is mandatory on all [Peer] sections
- GenerateConfig uses point ranges, GenerateServerConfig uses true ranges
- CPS strings are per-client, generated at export time
- Key generation functions panic on crypto/rand failure
- Peer names must be globally unique across clients and edges
- Clients and edges share the same IP pool
- GenerateServerConfig ignores its first argument (protocol)
- Edge exports return []byte, client exports return ClientConfig
- IP allocation skips .0 and .1
- Atomic writes (write to .tmp, then rename)

## Content Sources

- README.md, README.ru.md
- docs/library-usage.md, docs/library-usage.ru.md
- AGENTS.md
- Source code: types.go, parser.go, writer.go, manager.go, keys.go, generator.go, cps.go, helpers.go, iptables.go, protocols.go, quic.go, dns.go, dtls.go, stun.go
- CLI source: cli.go, init.go, edit.go, client.go, edge.go
- Test files for edge case documentation

## Out of Scope

- Build/test internals (covered by AGENTS.md)
- Non-English content
- Screenshots or images (text-only)
- Changelog or version history
