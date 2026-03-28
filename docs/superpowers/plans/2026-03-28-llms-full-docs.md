# LLM Documentation (llms-full.txt) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a single-file LLM-friendly documentation (`llms-full.txt`) covering all aspects of the Amnezigo CLI tool and Go library, following the Xray project's `llms-full.txt` format.

**Architecture:** A single `llms-full.txt` file in the project root containing 15 concatenated markdown pages separated by frontmatter. Pages are ordered progressively from beginner tutorial to technical reference. Each page is self-contained with its own frontmatter separator.

**Tech Stack:** Plain text, Markdown formatting

---

## File Structure

- Create: `llms-full.txt` (single file, ~2000-3000 lines estimated)

## Page Format Template

Every page follows this exact format:

```
---
url: /slug
---
# Page Title

Content here...
```

## Markdown Conventions (for all pages)

- Config parameters documented with: `> \`fieldName\`: type`
- Callouts: `::: tip Title`, `::: warning Title`, `::: danger Title`
- CLI commands in ` ```shell ` blocks (no prompt prefix)
- Config examples in ` ```ini ` blocks
- Go code in ` ```go ` blocks
- Flag/command tables with columns: Flag, Default, Required, Description
- H1 for page title, H2 for major sections, H3 for subsections

## Source Files Reference

The subagent implementing each task MUST read these files to extract accurate content:

- `README.md` — feature descriptions, usage examples, config examples
- `AGENTS.md` — library API reference, known gotchas, code style
- `docs/library-usage.md` — library usage with Go code examples
- `types.go` — all struct definitions and field types
- `parser.go` — config parsing behavior, metadata fields
- `writer.go` — config output format, field ordering
- `manager.go` — all Manager methods and their behavior
- `keys.go` — key generation functions
- `generator.go` — obfuscation parameter generation, constants
- `cps.go` — CPS tag types, MTU formula, generation logic
- `helpers.go` — utility functions
- `iptables.go` — rule generation
- `protocols.go` — protocol template dispatcher, "random" behavior
- `quic.go`, `dns.go`, `dtls.go`, `stun.go` — protocol template details
- `internal/cli/init.go` — init command flags and behavior
- `internal/cli/edit.go` — edit command
- `internal/cli/client.go` — client subcommands
- `internal/cli/edge.go` — edge subcommands

---

### Task 1: Write Pages 1-3 (Tutorial: Overview, Installation, Quick Start)

**Files:**
- Read: `README.md`, `Dockerfile`, `go.mod`, `internal/cli/init.go`, `internal/cli/client.go`, `internal/cli/edge.go`
- Write: `/tmp/llms-pages-1-3.md`

- [ ] **Step 1: Write Page 1 — Overview**

Write the `/overview` page with frontmatter `url: /overview`. Content:
- H1: "Overview"
- H2 "What is Amnezigo": Amnezigo is a CLI tool and Go library for generating and managing AmneziaWG v2.0 configurations. It handles key generation, obfuscation parameter creation, client/edge management, and config file I/O.
- H2 "What is AmneziaWG": AmneziaWG is a fork of WireGuard with traffic obfuscation support. It uses the same WireGuard protocol but adds obfuscation layers to disguise VPN traffic.
- H2 "Network Topology": Star topology — server is the hub, clients and edges connect directly to it. Edges can be used for hub-and-spoke relay setups.
- H2 "Features": Bulleted list — X25519 key generation, obfuscation parameter generation (Jc/Jmin/Jmax, S1-S4, H1-H4), CPS (Custom Packet Strings) with protocol templates (QUIC, DNS, DTLS, STUN), client and edge peer management, IP auto-assignment, iptables rule generation, INI config parsing and writing, atomic file saves.
- H2 "CLI vs Library": Two usage modes — CLI binary for server admins, Go library (`github.com/Arsolitt/amnezigo`) for programmatic use.
- H2 "Project Structure": Directory tree showing cmd/, internal/cli/, root package files.

- [ ] **Step 2: Write Page 2 — Installation**

Write the `/installation` page with frontmatter `url: /installation`. Content:
- H1: "Installation"
- H2 "Prerequisites": Go 1.26+
- H2 "Install via go install": `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest`
- H2 "Build from Source": `git clone`, `go build -o build/amnezigo ./cmd/amnezigo/`
- H2 "Docker": Show Dockerfile contents, `docker build -t amnezigo .`, `docker run amnezigo`
- H2 "Verify Installation": `amnezigo --help` showing root command output

- [ ] **Step 3: Write Page 3 — Quick Start**

Write the `/quick-start` page with frontmatter `url: /quick-start`. Content:
- H1: "Quick Start"
- H2 "Step 1: Initialize Server": Show `amnezigo init --ipaddr 10.8.0.1/24` with description of what happens (key generation, obfuscation params, iptables rules, config file creation).
- H2 "Step 2: Add a Client": Show `amnezigo client add myphone` and `amnezigo client add laptop --ipaddr 10.8.0.3` with output descriptions.
- H2 "Step 3: Export Client Config": Show `amnezigo client export myphone --protocol quic` with resulting .conf file content (annotated INI example).
- H2 "Step 4: Connect": Brief note about importing .conf into AmneziaWG client application.
- H2 "Full Server Config Example": Complete annotated `awg0.conf` example with comments explaining each section ([Interface], [Peer], obfuscation fields, metadata comments).

---

### Task 2: Write Pages 4-6 (Management: Server, Clients, Edges)

**Files:**
- Read: `internal/cli/init.go`, `internal/cli/edit.go`, `internal/cli/client.go`, `internal/cli/edge.go`, `manager.go`, `types.go`, `iptables.go`, `writer.go`
- Write: `/tmp/llms-pages-4-6.md`

- [ ] **Step 1: Write Page 4 — Server Setup**

Write the `/server-setup` page with frontmatter `url: /server-setup`. Content:
- H1: "Server Setup"
- H2 "The init Command": Full flag table — `--ipaddr` (required), `--port`, `--mtu` (default 1280), `--dns` (silently ignored), `--keepalive` (silently ignored), `--client-to-client`, `--iface`, `--iface-name` (default "awg0"), `--endpoint-v4`, `--endpoint-v6`, `--config` (default "awg0.conf").
- H2 "Auto-Detection": Interface detection (first non-loopback up interface), port generation (10000-65535), endpoint detection via HTTP to icanhazip.com.
- H2 "What Gets Created": Key pair, obfuscation params (random S1 0-64, random Jc 0-10, header ranges H1-H4, junk params Jmin/Jmax, size prefixes S1-S4), iptables PostUp/PostDown rules, config file saved atomically, `.main.config` symlink/file for CLI to find the config.
- H2 "The edit Command": `--client-to-client true/false` flag, regenerates iptables rules when toggled.
- H2 "iptables Rules": Explain what PostUp/PostDown do — ACCEPT on tunnel, FORWARD rules, NAT MASQUERADE, optional client-to-client forwarding.

- [ ] **Step 2: Write Page 5 — Client Management**

Write the `/client-management` page with frontmatter `url: /client-management`. Content:
- H1: "Client Management"
- H2 "Adding Clients": `client add <name>` with auto-IP, `client add <name> --ipaddr <ip>`. Explain key generation, PresharedKey generation, IP allocation (skips .0 and .1, scans .2-.254). Show output examples.
- H2 "Listing Clients": `client list` output format — tab-separated NAME, IP, CREATED columns.
- H2 "Exporting Clients": `client export <name>` exports single client, `client export` (no args) exports all. `--protocol` flag (default "random"). Endpoint resolution order: config EndpointV4 → EndpointV6 → HTTP to ipv4.icanhazip.com. Output: `<name>.conf` files with 0600 permissions.
- H2 "Removing Clients": `client remove <name>` — removes from config, saves atomically.
- H2 "Constraints": Names must be unique across all clients and edges. Clients and edges share the same IP pool.

- [ ] **Step 3: Write Page 6 — Edge Servers**

Write the `/edge-servers` page with frontmatter `url: /edge-servers`. Content:
- H1: "Edge Servers"
- H2 "What are Edge Servers": Hub-and-spoke topology. Edges connect to the main server (hub) and relay traffic. Useful for multi-location setups.
- H2 "Client vs Edge Differences": Table comparing — Client has DNS ("1.1.1.1, 8.8.8.8") and AllowedIPs ("0.0.0.0/0, ::/0"); Edge has no DNS and AllowedIPs = hub IP only.
- H2 "Edge Commands": `edge add <name>`, `edge list`, `edge export <name>`, `edge remove <name>` — same flags as client commands.
- H2 "Edge Config Example": Show a complete exported edge .conf with annotations highlighting the differences from a client config.
- H2 "Use Cases": Multi-location relay, bypassing ISP restrictions in specific regions, load distribution.

---

### Task 3: Write Pages 7-9 (Obfuscation, CPS, Protocols)

**Files:**
- Read: `generator.go`, `cps.go`, `protocols.go`, `quic.go`, `dns.go`, `dtls.go`, `stun.go`, `types.go`
- Write: `/tmp/llms-pages-7-9.md`

- [ ] **Step 1: Write Page 7 — Obfuscation**

Write the `/obfuscation` page with frontmatter `url: /obfuscation`. Content:
- H1: "Obfuscation"
- H2 "What is Obfuscation": AmneziaWG wraps WireGuard packets in obfuscation layers to disguise VPN traffic as normal HTTPS/QUIC/DNS traffic. This makes the traffic resistant to deep packet inspection.
- H2 "Junk Parameters": Document Jc (junk packet count, 0-10), Jmin (minimum junk size, 64-1024), Jmax (maximum junk size, 64-1024). Explain how junk packets are injected between real packets. Show with `> \`Jc\`: int` pattern.
- H2 "Size Prefixes (S1-S4)": Document S1-S4 (size prefix values). S4: 0-32, S1-S3: 0-64. Constraint: S1+56 != S2. Explain their role in packet size obfuscation.
- H2 "Header Magic Values (H1-H4)": Document H1-H4 as HeaderRange{Min, Max} — non-overlapping ranges in specific uint32 regions. H1: 0x00000005 to 0x3FFFFFFE, H2: starts at 0x40000000, H3: starts at 0x80000000, H4: starts at 0xC0000000. Each range has minimum 10,000,000 span.
- H2 "Point Ranges vs True Ranges": Explain the difference — `GenerateConfig` (for clients) uses point ranges (Min == Max), `GenerateServerConfig` (for server) uses true ranges (Min < Max). Show code examples of both.
- H2 "Server vs Client Obfuscation": Server stores Jc/Jmin/Jmax/S1-S4/H1-H4. Client additionally gets I1-I5 CPS strings generated at export time.

- [ ] **Step 2: Write Page 8 — Custom Packet Strings**

Write the `/cps` page with frontmatter `url: /cps`. Content:
- H1: "Custom Packet Strings"
- H2 "What are CPS": I1-I5 are Custom Packet Strings that define how handshake intervals should look. They make the WireGuard handshake mimic other protocols.
- H2 "Tag Syntax": Reference table with columns: Tag, Syntax, Description:
  - `<b 0xHH...>` — fixed bytes (hex)
  - `<r N>` — N random bytes
  - `<rc N>` — N random ASCII characters
  - `<rd N>` — N random digits
  - `<c>` — counter (incrementing)
  - `<t>` — timestamp
- H2 "Building CPS Programmatically": Show Go code example using `BuildCPSTag` and `BuildCPS`.
- H2 "MTU Constraints": Formula: `maxISize = MTU - 49 - 149 - S1`. If CPS exceeds maxISize, tags are progressively removed from the end. Falls back to `<t>` if all tags are too large.
- H2 "Per-Client Generation": CPS strings are generated at export time, not stored in the server config. Each client gets unique I1-I5 values.
- H2 "I1-I5 Progression": I1 is the largest (first handshake), I5 is typically empty. Sizes decrease with each interval.

- [ ] **Step 3: Write Page 9 — Protocol Templates**

Write the `/protocols` page with frontmatter `url: /protocols`. Content:
- H1: "Protocol Templates"
- H2 "How Templates Work": Each protocol template defines tag sequences (I1I5Template) that mimic real protocol packets. Used during `GenerateCPS` to produce I1-I5 strings.
- H2 "QUIC": Mimics QUIC Initial packets. Long header (0xC0FF), Version 1, 8-byte DCID, SCID len 0, token len 0, length field, packet number, timestamp, random payload. I1 has ~40 random bytes, I5 is empty.
- H2 "DNS": Mimics DNS query packets. Random transaction ID, standard query flags, label sequences (3+7+3 chars), root label, Type A, Class IN. I1 has full query, I5 is empty.
- H2 "DTLS": Mimics DTLS 1.2 ClientHello. ContentType 0x16, DTLS 1.2 version, epoch 0, sequence 0, cipher suites (ECDHE_ECDSA_AES_256_GCM_SHA384, AES_128_GCM_SHA256, AES_256_CBC_SHA, AES_128_CBC_SHA), null compression. I1 has 4 cipher suites, I3-I4 have fewer, I5 is empty.
- H2 "STUN": Mimics STUN Binding Request (RFC 5389). Message type 0x0001, length 0, magic cookie 0x2112A442, 12 random bytes. I2 adds PADDING attribute. I3-I4 are minimal. I5 is empty.
- H2 "Random": Deterministic selection via `len(protocol) % 4`. Since `len("random") % 4 = 2`, it always selects DTLS. Use explicit protocol names for variety.
- H2 "Choosing a Protocol": `::: tip` recommending QUIC for general use, DNS for environments where DNS traffic is expected, DTLS for TLS-heavy networks.

---

### Task 4: Write Pages 10-11 (Config References)

**Files:**
- Read: `types.go`, `writer.go`, `parser.go`, `README.md`
- Write: `/tmp/llms-pages-10-11.md`

- [ ] **Step 1: Write Page 10 — Server Config Reference**

Write the `/server-config-reference` page with frontmatter `url: /server-config-reference`. Content:
- H1: "Server Config Reference"
- H2 "[Interface] Section": Document every field using `> \`FieldName\`: type` pattern:
  - PrivateKey, PublicKey, Address, ListenPort, MTU
  - PostUp, PostDown (iptables rules)
  - Jc, Jmin, Jmax (junk parameters)
  - S1, S2, S3, S4 (size prefixes)
  - H1-H4 (header magic values, written as "min-max" string)
- H2 "Metadata Comments": Document all `#_` prefixed fields:
  - #_Name, #_Role (client/edge), #_PrivateKey, #_GenKeyTime, #_EndpointV4, #_EndpointV6, #_ClientToClient, #_TunName, #_MainIface
- H2 "[Peer] Section": Document fields: PublicKey, PresharedKey, AllowedIPs, and all metadata comments.
- H2 "Full Server Config Example": Complete annotated INI config with inline comments.

- [ ] **Step 2: Write Page 11 — Client Config Reference**

Write the `/client-config-reference` page with frontmatter `url: /client-config-reference`. Content:
- H1: "Client Config Reference"
- H2 "[Interface] Section": Document every field using `> \`FieldName\`: type` pattern:
  - PrivateKey, Address, DNS, MTU
  - Jc, Jmin, Jmax, S1-S4, H1-H4 (obfuscation, same as server)
  - I1, I2, I3, I4, I5 (CPS strings, client-specific)
- H2 "[Peer] Section": Document fields:
  - PublicKey, PresharedKey, Endpoint, AllowedIPs, PersistentKeepalive
- H2 "Client vs Edge Config": Comparison table showing differences in DNS, AllowedIPs, PersistentKeepalive.
- H2 "Full Client Config Example": Annotated INI config for a client.
- H2 "Full Edge Config Example": Annotated INI config for an edge server.

---

### Task 5: Write Page 12 (CLI Reference)

**Files:**
- Read: `internal/cli/cli.go`, `internal/cli/init.go`, `internal/cli/edit.go`, `internal/cli/client.go`, `internal/cli/edge.go`, `README.md`
- Write: `/tmp/llms-pages-12.md`

- [ ] **Step 1: Write Page 12 — CLI Reference**

Write the `/cli-reference` page with frontmatter `url: /cli-reference`. Content:
- H1: "CLI Reference"
- H2 "Global Flags": `--config` (default "awg0.conf") — available on all commands.
- H2 "amnezigo init": Flag table with all 11 flags, defaults, required/optional, descriptions. Include note about `--dns` and `--keepalive` being silently ignored.
- H2 "amnezigo edit": Flag table with `--client-to-client` flag.
- H2 "amnezigo client add": Flag table with `--ipaddr` flag.
- H2 "amnezigo client list": No additional flags.
- H2 "amnezigo client export": Flag table with `--protocol` flag. Note: no `--endpoint` flag, endpoint is auto-resolved. Explain endpoint resolution order.
- H2 "amnezigo client remove": No additional flags.
- H2 "amnezigo edge add": Same as client add.
- H2 "amnezigo edge list": Same as client list.
- H2 "amnezigo edge export": Flag table. Same as client export but exports single only (no bulk export).
- H2 "amnezigo edge remove": Same as client remove.
- H2 "Common Patterns": Example command sequences for typical workflows (new server setup, adding users, rotating configs).

---

### Task 6: Write Pages 13-14 (Library API & Type Reference)

**Files:**
- Read: `manager.go`, `parser.go`, `writer.go`, `keys.go`, `generator.go`, `cps.go`, `helpers.go`, `iptables.go`, `protocols.go`, `quic.go`, `dns.go`, `dtls.go`, `stun.go`, `types.go`, `docs/library-usage.md`
- Write: `/tmp/llms-pages-13-14.md`

- [ ] **Step 1: Write Page 13 — Library API**

Write the `/library-api` page with frontmatter `url: /library-api`. Content:
- H1: "Library API"
- H2 "Installation": `go get github.com/Arsolitt/amnezigo`
- H2 "Manager": Document every method — NewManager, Load, Save, AddClient, RemoveClient, FindClient, ListClients, ExportClient, BuildClientConfig, AddEdge, RemoveEdge, FindEdge, ListEdges, BuildEdgeConfig, ExportEdge. Each with signature, description, and Go code example.
- H2 "Config I/O": ParseServerConfig, WriteServerConfig, WriteClientConfig, LoadServerConfig, SaveServerConfig — each with signature and example.
- H2 "Key Generation": GenerateKeyPair, DerivePublicKey, GeneratePSK — signatures, descriptions, examples. Note panic behavior on crypto/rand failure.
- H2 "Obfuscation Generation": GenerateConfig, GenerateServerConfig, GenerateHeaders, GenerateSPrefixes, GenerateJunkParams, GenerateCPS, GenerateHeaderRanges — signatures, descriptions, examples. Note GenerateServerConfig ignores first arg.
- H2 "CPS Construction": BuildCPSTag, BuildCPS — signatures and examples.
- H2 "Protocol Templates": QUICTemplate, DNSTemplate, DTLSTemplate, STUNTemplate — return I1I5Template.
- H2 "Network Helpers": IsValidIPAddr, ExtractSubnet, GenerateRandomPort, DetectMainInterface, FindNextAvailableIP — signatures and descriptions.
- H2 "iptables Rules": GeneratePostUp, GeneratePostDown — signatures, parameters, examples.

- [ ] **Step 2: Write Page 14 — Type Reference**

Write the `/type-reference` page with frontmatter `url: /type-reference`. Content:
- H1: "Type Reference"
- H2 "ServerConfig": All fields — Clients []PeerConfig, Edges []PeerConfig, Interface InterfaceConfig, Obfuscation ServerObfuscationConfig.
- H2 "InterfaceConfig": All fields — PrivateKey, PublicKey, Address, PostUp, PostDown, MainIface, TunName, EndpointV4, EndpointV6, ListenPort int, MTU int, ClientToClient bool.
- H2 "PeerConfig": All fields — CreatedAt time.Time, ClientObfuscation *ClientObfuscationConfig, Name, Role, PrivateKey, PublicKey, PresharedKey, AllowedIPs.
- H2 "ServerObfuscationConfig": Jc, Jmin, Jmax int; S1, S2, S3, S4 int; H1, H2, H3, H4 HeaderRange.
- H2 "ClientObfuscationConfig": Embeds ServerObfuscationConfig + I1, I2, I3, I4, I5 string.
- H2 "ClientConfig": Peer ClientPeerConfig, Interface ClientInterfaceConfig.
- H2 "ClientInterfaceConfig": PrivateKey, Address, DNS, Obfuscation ClientObfuscationConfig, MTU int.
- H2 "ClientPeerConfig": PublicKey, PresharedKey, Endpoint, AllowedIPs, PersistentKeepalive int.
- H2 "HeaderRange": Min, Max uint32.
- H2 "Headers": H1, H2, H3, H4 uint32.
- H2 "SPrefixes": S1, S2, S3, S4 int.
- H2 "JunkParams": Jc, Jmin, Jmax int.
- H2 "CPSConfig": I1, I2, I3, I4, I5 string.
- H2 "TagSpec": Type string, Value string.
- H2 "I1I5Template": I1, I2, I3, I4, I5 []TagSpec.
- H2 "Manager": ConfigPath string.

---

### Task 7: Write Page 15 (Gotchas & FAQ) and Assemble Final File

**Files:**
- Read: `AGENTS.md`, `README.md`, `manager.go`, `generator.go`, `protocols.go`, `parser.go`
- Write: `/tmp/llms-pages-15.md`, then assemble `llms-full.txt`

- [ ] **Step 1: Write Page 15 — Gotchas & FAQ**

Write the `/gotchas` page with frontmatter `url: /gotchas`. Content:
- H1: "Gotchas & FAQ"
- Numbered list (1-13) of all known gotchas, each as an H2 or H3 with explanation:
  1. `--dns` and `--keepalive` on init are silently ignored — DNS hardcoded to "1.1.1.1, 8.8.8.8", keepalive to 25 in exports.
  2. "random" protocol always selects DTLS — `len("random") % 4 = 2` maps to DTLS index. Use explicit protocol names.
  3. Export has no `--endpoint` flag — Endpoint auto-resolved from server config's EndpointV4/V6 fields, or via HTTP to icanhazip.com.
  4. `#_Role` is mandatory on all [Peer] sections — Parser returns error if missing or invalid.
  5. GenerateConfig uses point ranges, GenerateServerConfig uses true ranges — Client H1-H4 have Min==Max, server has Min<Max.
  6. CPS strings are per-client, generated at export time — Not stored in server config.
  7. Key generation functions panic on crypto/rand failure — GenerateKeyPair, DerivePublicKey, GeneratePSK all panic on system crypto failure.
  8. Peer names must be globally unique across clients and edges — A client and edge cannot share the same name.
  9. Clients and edges share the same IP pool — IP auto-assignment considers both when finding next available.
  10. GenerateServerConfig ignores its first argument (protocol) — Only uses s1 and jc parameters.
  11. Edge exports return []byte, client exports return ClientConfig — Different return types.
  12. IP allocation skips .0 and .1 — Only assigns from .2 to .254.
  13. Atomic writes — Config saved to .tmp file first, then renamed.

- [ ] **Step 2: Assemble final llms-full.txt**

Concatenate all temporary page files in order (1-3, 4-6, 7-9, 10-11, 12, 13-14, 15) into `llms-full.txt` in the project root. Verify:
- Each page starts with `---\nurl: /slug\n---`
- Pages are separated by `---` between them
- No duplicate frontmatter or missing separators
- File ends with final page content (no trailing `---`)

- [ ] **Step 3: Commit**

```bash
git add llms-full.txt
git commit -m "docs: add llms-full.txt LLM documentation"
```
