# Agent Guidelines for Amnezigo

## Build/Test Commands

- Build: `go build -o build/amnezigo .`
- Install: `go install github.com/Arsolitt/amnezigo@latest`
- Run all tests: `go test ./...`
- Run single test: `go test -run TestFunctionName ./internal/package`
- Run tests with coverage: `go test -cover ./...`
- Format code: `go fmt ./...`
- Vet code: `go vet ./...`

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

      "github.com/Arsolitt/amnezigo/internal/config"
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

### Comments
- Package comments describe purpose and responsibilities
- Function comments start with what it does, not how
- Comment complex logic, not obvious code
- Exported symbols must have comments

### File Organization
- Main package: `main.go` calls `cli.Execute()`
- Internal packages: `internal/<package>/`
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

### Configuration
- Config files use INI format with [Section] headers
- Store metadata in commented fields with `#_` prefix
- Use atomic writes: write to `.tmp` file, then rename

### Crypto/Security
- Use `crypto/rand` for cryptographic randomness
- WireGuard key clamping: `priv[0] &= 248; priv[31] &= 127; priv[31] |= 64`
- Generate keys with proper error handling (panic only on system failures)

### Obfuscation Patterns
- Store H1-H4 as HeaderRange{Min, Max} for ranges
- I1-I5 CPS strings generated per-client at export time
- Protocol templates in internal/obfuscation/protocols/
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
- `internal/cli/`: Cobra CLI commands (init, add, remove, list, export, edit)
- `internal/config/`: INI parsing/writing and type definitions
- `internal/crypto/`: WireGuard key generation (X25519 curve)
- `internal/obfuscation/`: Obfuscation parameter generation
- `internal/obfuscation/protocols/`: Protocol-specific CPS templates
- `internal/network/`: iptables rule generation

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
