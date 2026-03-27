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
|   +-- client.go                   # client command group (add, list, export, remove)
|   +-- edge.go                     # edge command group (add, list, export, remove)
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

## Known Gotchas

### CLI Behavior
- **export has no --endpoint flag**: Endpoint is auto-resolved from server config's Endpoint field, or via HTTP request if endpoint contains `:http`. There is no manual override.
- **--dns and --keepalive on init are silently ignored**: These flags exist on the init command but do nothing. DNS is hardcoded to "1.1.1.1, 8.8.8.8" and keepalive to 25 in client exports.
- **"random" protocol is deterministic**: Due to `len("random") % 4 = 2`, the random protocol always selects DTLS. Use explicit protocol names if you need variety.
- **#_Role is mandatory on all [Peer] sections**: Every `[Peer]` in the INI config must have `#_Role = client` or `#_Role = edge`. The parser returns an error if missing or invalid.

### Obfuscation Generation
- **GenerateConfig vs GenerateServerConfig**: `GenerateConfig` uses point ranges for H1-H4 (single values masquerading as ranges). `GenerateServerConfig` uses true ranges with different min/max values.
- **CPS strings are per-client**: I1-I5 CPS strings are generated at export time, not stored in the server config. Each client gets unique CPS values.

## Library API

### Manager (manager.go)
High-level CRUD operations for server configs, clients, and edges:
- `NewManager(configPath string) *Manager`
- `Load() (ServerConfig, error)`
- `Save(cfg ServerConfig) error`
- `AddClient(name, ip string) (PeerConfig, error)`
- `RemoveClient(name string) error`
- `FindClient(name string) (*PeerConfig, error)`
- `ListClients() []PeerConfig`
- `ExportClient(name, protocol, endpoint string) (ClientConfig, error)`
- `BuildClientConfig(peer PeerConfig, protocol, endpoint string) (ClientConfig, error)`
- `AddEdge(name, ip string) (PeerConfig, error)`
- `RemoveEdge(name string) error`
- `FindEdge(name string) (*PeerConfig, error)`
- `ListEdges() []PeerConfig`
- `BuildEdgeConfig(name, protocol, endpoint string) (ClientConfig, error)`
- `ExportEdge(name, protocol, endpoint string) ([]byte, error)`

### Config I/O (parser.go, writer.go)
- `ParseServerConfig(r io.Reader) (*ServerConfig, error)`
- `WriteServerConfig(w io.Writer, cfg *ServerConfig) error`
- `WriteClientConfig(w io.Writer, cfg *ClientConfig) error`
- `LoadServerConfig(path string) (*ServerConfig, error)`
- `SaveServerConfig(path string, cfg *ServerConfig) error`

### Key Generation (keys.go)
- `GenerateKeyPair() (privateKey, publicKey string, err error)`
- `DerivePublicKey(privateKey string) (string, error)`
- `GeneratePSK() (string, error)`

### Obfuscation (generator.go)
- `GenerateConfig(protocol string) (*ObfuscationConfig, error)`
- `GenerateServerConfig(protocol string) (*ObfuscationConfig, error)`
- `GenerateHeaders(protocol string) (h1, h2, h3, h4 int, err error)`
- `GenerateSPrefixes() (sPrefix1, sPrefix2 int, err error)`
- `GenerateJunkParams() (junkMin, junkMax int, err error)`
- `GenerateCPS(protocol string) (string, error)`
- `GenerateHeaderRanges(protocol string) (HeaderRange, HeaderRange, HeaderRange, HeaderRange, error)`

### CPS Generation (cps.go)
- `BuildCPSTag(tag string) (string, error)`
- `BuildCPS(tags []string) string`

### Protocol Templates (protocols.go, quic.go, dns.go, dtls.go, stun.go)
- `QUICTemplate() []string`
- `DNSTemplate() []string`
- `DTLSTemplate() []string`
- `STUNTemplate() []string`

### Helpers (helpers.go)
- `IsValidIPAddr(ip string) bool`
- `ExtractSubnet(cidr string) (string, error)`
- `GenerateRandomPort() (int, error)`
- `DetectMainInterface() (string, error)`
- `FindNextAvailableIP(cidr string, existingIPs map[string]bool) (string, error)`

### iptables (iptables.go)
- `GeneratePostUp(interfaceName, port, subnet string) []string`
- `GeneratePostDown(interfaceName, port, subnet string) []string`

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
- Use factory functions: `NewClientCommand()`, `NewClientAddCommand()`, `NewClientListCommand()`, `NewClientExportCommand()`, `NewClientRemoveCommand()`, `NewEdgeCommand()`, `NewEdgeAddCommand()`, `NewEdgeListCommand()`, `NewEdgeExportCommand()`, `NewEdgeRemoveCommand()`, `NewEditCommand()`

### Configuration
- Config files use INI format with [Section] headers
- Store metadata in commented fields with `#_` prefix
- All [Peer] sections require `#_Role = client` or `#_Role = edge`
- `ServerConfig` has `Clients []PeerConfig` and `Edges []PeerConfig` (not `Peers`)
- Use atomic writes: write to `.tmp` file, then rename

### Crypto/Security
- Use `crypto/rand` for cryptographic randomness
- WireGuard key clamping: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`
- Generate keys with proper error handling (panic only on system failures)

### Obfuscation Patterns
- Store H1-H4 as HeaderRange{Min, Max} for ranges
- I1-I5 CPS strings generated per-client at export time
- Protocol templates in root package (quic.go, dns.go, dtls.go, stun.go)
- Use tag-based CPS construction: <b 0x...>, <r N>, <t>, <c>

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
- Include specific identifiers when available: `"client '%s' not found"`
- Use fmt.Errorf for wrapping, not log.Printf
- Return errors from RunE functions, don't log inside
