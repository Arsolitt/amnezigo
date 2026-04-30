# Agent Guidelines for Amnezigo

## Build/Test Commands

- Build: `go build -o build/amnezigo ./cmd/amnezigo/`
- Install: `go install github.com/Arsolitt/amnezigo/cmd/amnezigo@latest`
- Run all tests: `go test ./...`
- Run single test: `go test -run TestFunctionName ./internal/package`
- Run tests with coverage: `go test -cover ./...`
- Lint code: `golangci-lint run`
- Lint and auto-fix: `golangci-lint run --fix`
- Always run `golangci-lint run --fix` first to auto-resolve issues before fixing remaining ones manually

## Project Structure

```
github.com/Arsolitt/amnezigo
|
+-- cmd/amnezigo/main.go            # CLI entry point (package main)
|
+-- internal/cli/                   # Cobra CLI commands (package cli)
|   +-- cli.go                      # Root command + Execute()
|   +-- init.go                     # init command
|   +-- edit.go                     # edit command
|   +-- add.go                      # add command
|   +-- list.go                     # list command
|   +-- export.go                   # export command
|   +-- remove.go                   # remove command
|
+-- [root package: amnezigo]        # All business logic
    +-- types.go                    # All type definitions
    +-- parser.go                   # INI config parser (bufio.Scanner)
    +-- writer.go                   # Config file writers (io.Writer)
    +-- manager.go                  # High-level CRUD Manager
    +-- keys.go                     # X25519 key generation
    +-- generator.go                # Obfuscation param generation
    +-- cps.go                      # Custom Packet String generation
    +-- helpers.go                  # IP, port, interface utilities
    +-- iptables.go                 # iptables rule generation
    +-- protocols.go                # Protocol template dispatcher
    +-- quic.go, dns.go, dtls.go, stun.go  # Protocol templates
```

## Library API

### Manager (manager.go)
High-level CRUD operations for server configs and peers:
- `NewManager(configPath string) *Manager`
- `Load() (ServerConfig, error)`
- `Save(cfg ServerConfig) error`
- `AddPeer(name, ip string) (PeerConfig, error)`
- `RemovePeer(name string) error`
- `FindPeer(name string) (*PeerConfig, error)`
- `ListPeers() []PeerConfig`
- `ExportPeer(name, protocol, endpoint string) (ClientConfig, error)`
- `BuildPeerConfig(peer PeerConfig, protocol, endpoint string) (ClientConfig, error)`

### Config I/O (parser.go, writer.go)
- `ParseServerConfig(r io.Reader) (ServerConfig, error)`
- `WriteServerConfig(w io.Writer, cfg ServerConfig) error`
- `WriteClientConfig(w io.Writer, cfg ClientConfig) error`
- `LoadServerConfig(path string) (ServerConfig, error)`
- `SaveServerConfig(path string, cfg ServerConfig) error`

### Key Generation (keys.go)
- `GenerateKeyPair() (string, string)` — panics on crypto/rand failure
- `DerivePublicKey(privateKey string) string` — panics on invalid base64 or wrong length
- `GeneratePSK() string` — panics on crypto/rand failure

### Obfuscation (generator.go)
- `GenerateConfig(protocol string, mtu, s1, jc int) ClientObfuscationConfig`
- `GenerateServerConfig(_, s1, jc int) ServerObfuscationConfig` — first arg (protocol) ignored
- `GenerateSPrefixes() SPrefixes`
- `GenerateJunkParams() JunkParams`
- `GenerateCPS(protocol string, mtu, s1, _ int) (string, string, string, string, string)` — 4th arg unused
- `GenerateHeaderRanges() [4]HeaderRange`

### CPS Generation (cps.go)
- `BuildCPSTag(tagType, value string) string`
- `BuildCPS(tags []string) string`

### Protocol Templates (protocols.go, quic.go, dns.go, dtls.go, stun.go)
- `QUICTemplate() I1I5Template`
- `DNSTemplate() I1I5Template`
- `DTLSTemplate() I1I5Template`
- `STUNTemplate() I1I5Template`

### Helpers (helpers.go)
- `IsValidIPAddr(ipaddr string) bool`
- `ExtractSubnet(ipaddr string) string`
- `GenerateRandomPort() (int, error)`
- `DetectMainInterface() string`
- `FindNextAvailableIP(serverAddress string, existingIPs []string) (string, error)`

### iptables (iptables.go)
- `GeneratePostUp(tunName, mainIface, subnet string, clientToClient bool) string`
- `GeneratePostDown(tunName, mainIface, subnet string, clientToClient bool) string`

## Code Style Guidelines

### Imports
- Order: stdlib, external packages, internal packages (each group separated by blank line)
- Use aliases only when necessary to avoid conflicts
- Example:
  ```go
  import (
      "fmt"
      "os"

      "github.com/spf13/cobra"

      "github.com/Arsolitt/amnezigo"
  )
  ```

### Types & Structs
- Define types in dedicated `types.go` files within packages
- Exported fields use CamelCase, unexported fields use camelCase
- Use struct tags only for serialization (JSON, INI, etc.)
- Group related fields with blank lines between sections

### Functions
- Exported functions use PascalCase, private functions use camelCase
- Keep functions focused and under 50 lines when possible
- Use early returns to reduce nesting
- Return errors, don't panic unless unrecoverable

### Error Handling
- Wrap errors with `fmt.Errorf("context: %w", err)` to preserve stack traces
- Check errors immediately after function calls
- Use `t.Fatalf()` for test setup failures, `t.Errorf()` for assertion failures

### Naming Conventions
- CLI flags use snake_case with Var prefix: `var addIPAddr string`
- Variables use descriptive names, avoid abbreviations (e.g., `configPath` not `cfgPath`)
- Constants use PascalCase for exported, camelCase for internal
- Test functions follow `TestFunctionName` pattern
- Factory functions use `New*Command()` pattern for CLI commands

### Comments
- Package comments describe purpose and responsibilities
- Function comments start with what it does, not how
- Comment complex logic, not obvious code
- Exported symbols must have comments

### File Organization
- CLI entry point: `cmd/amnezigo/main.go`
- Root package: `package amnezigo` (all business logic in types.go, parser.go, writer.go, etc.)
- CLI commands: `internal/cli/` (thin wrappers calling amnezigo package functions)
- Tests: `*_test.go` alongside implementation
- Related functions grouped by responsibility (parser.go, writer.go, etc.)

### Testing
- Write tests for all exported functions
- Use table-driven tests for multiple test cases
- Test both success and error paths
- Use `strings.NewReader()` or `bytes.Buffer` for I/O testing

### Cobra CLI Commands
- Define commands with `Use`, `Short`, `Long`, and `Args` fields
- Use `RunE` for error handling, not `Run`
- Store flag variables as package-level vars
- Initialize flags in `init()` function
- Use factory functions: `NewAddCommand()`, `NewListCommand()`, `NewExportCommand()`, `NewRemoveCommand()`, `NewEditCommand()`

### Configuration
- Config files use INI format with [Section] headers
- Store metadata in commented fields with `#_` prefix
- `ServerConfig` has `Peers []PeerConfig`
- Use atomic writes: write to `.tmp` file, then rename

### Crypto/Security
- Use `crypto/rand` for cryptographic randomness
- WireGuard key clamping: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`
- Generate keys with proper error handling (panic only on system failures)

### Obfuscation Patterns

- Store H1-H4 as HeaderRange{Min, Max} for ranges
- I1-I5 CPS strings generated per-peer at export time
- Protocol templates in root package (quic.go, dns.go, dtls.go, stun.go, sip.go)
- Use tag-based CPS construction: `<b 0x...>`, `<r N>`, `<rc N>`, `<rd N>`, `<t>`, `<d>`

### Adding a New Protocol Template (P1.2 framework)

Every new protocol template MUST satisfy this contract. Reviewers reject PRs that miss any item.

**Required interface**

- File `<protocol>.go` at the repository root.
- Constructor `XxxTemplate() I1I5Template` — pure data construction, no I/O, no globals.
- Test file `<protocol>_test.go` co-located.
- Switch case key, `--protocol` flag value, and doc table row all match the file's lowercase short-name.

**Tag mix rules**

- No `<c>` tag (removed in P0.1). For pseudo-monotonic bytes use `<rd N>` or `<r N>`.
- `<t>` is 4 bytes (post-P0.2).
- `<rc>` is `[a-zA-Z]` only (post-P0.5). For mixed letter+digit fields, concatenate `<rc 4><rd 2>`.
- At most one `<t>` per interval.
- No new tag types — propose those in P1.1, not in a template PR.

**Byte budget**

- Each interval >= 16 B (avoid raw-WG size collisions).
- Each interval <= `MTU - 49 - 149 - S1` (worst case MTU=1280, S1=64 -> 1018 B).
- Recommended ceiling <= 700 B per interval; raise only with reviewer agreement.
- I5 always empty (`[]TagSpec{}` literal) for named templates.
- I1 >= I2 >= I3 >= I4 in byte budget (STUN's I2 > I1 is a known exception).
- Leading bytes should not collide with any prefix in the centralized `existingTemplatePrefixes` slice in `protocols_test.go`. New templates that introduce a fixed prefix MUST append it to that slice in the same PR.

**Required tests**

- `TestXxxTemplate_AllIntervalsNonEmpty_I1ToI4`
- `TestXxxTemplate_I5Empty`
- `TestXxxTemplate_NoForbiddenTags`
- `TestXxxTemplate_NoCounterLiteral` — scans the rendered CPS string for `<c>` substring
- `TestXxxTemplate_FitsMTU`
- `TestXxxTemplate_ByteBudgetUnderCeiling`
- `TestXxxTemplate_AtMostOneTimestampPerInterval`
- `TestXxxTemplate_AvoidsExistingPrefixes` (recommended if leading bytes are fixed; calls the shared `assertTemplateAvoidsExistingPrefixes` helper)
- One row added to `TestGetTemplate_NamedProtocols` in `protocols_test.go`.

**Wiring checklist (each template PR ticks every box)**

- [ ] `protocols.go:getTemplate` switch — new `case`
- [ ] `protocols.go:getTemplate` random-fallback slice — append constructor
- [ ] `protocols.go:getTemplate` doc-comment — append protocol name
- [ ] `protocols_test.go:TestGetTemplate_NamedProtocols` — append row
- [ ] `internal/cli/export.go` — extend `--protocol` helptext
- [ ] `docs/cli-reference.md` — extend allowed-values list and protocol table
- [ ] `docs/obfuscation.md` — extend Available Protocols table

**Quality bar**

`go test ./... -race` green. `gofmt -l .` empty. `go vet ./...` clean. `golangci-lint run` zero errors. PR description ticks every wiring-checklist box explicitly.

### Config File Parsing
- Use bufio.Scanner for line-by-line INI parsing
- Skip empty lines and regular comments (# prefix)
- Parse commented metadata fields with #_ prefix
- Handle sections with [SectionName] headers
- Parse H1-H4 ranges as "min-max" string format

### File I/O Patterns
- Load configs: `os.Open()` → `ParseServerConfig(file)`
- Save configs: `Create(.tmp)` → `Write()` → `Rename(tmp, path)`
- Always defer file.Close() after opening
- Use atomic writes to prevent corruption

### IP Address Handling
- Parse CIDR with `net.ParseCIDR()` to get subnet
- Use `ipnet.Contains()` to check subnet membership
- Convert IP to bytes with `ip.To4()` for IPv4
- Find next available IP by iterating subnet (.2 to .254)

### Package Organization
- `package amnezigo` (root): All business logic (config, crypto, obfuscation, network)
- `cmd/amnezigo/`: CLI entry point (package main)
- `internal/cli/`: Cobra CLI commands (package cli, thin wrappers over root package)

### Common Patterns
- Use string slices for collecting config data
- Use maps for quick lookup (existing IPs, etc.)
- Generate random values with `rand.Int(rand.Reader, range)`
- Hex encoding with `encoding/hex.EncodeToString()`
- Time parsing with `time.Parse(time.RFC3339, value)`
- Switch statements for key/value parsing
- Use `continue` in loops to skip invalid entries

### Error Message Format
- Prefix with context: `"failed to load config: %w"`
- Include specific identifiers when available: `"peer '%s' not found"`
- Use fmt.Errorf for wrapping, not log.Printf
- Return errors from RunE functions, don't log inside
