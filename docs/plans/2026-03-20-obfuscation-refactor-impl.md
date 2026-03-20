# Obfuscation Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor Amnezigo to support per-client I1-I5 generation, H1-H4 ranges, configurable interface names, and dynamic client-to-client switching.

**Architecture:** Separate ServerObfuscationConfig and ClientObfuscationConfig to support future mesh topologies. Store metadata as `#_` comments in config to avoid AmneziaWG validation errors.

**Tech Stack:** Go, Cobra CLI, existing codebase patterns

---

## Task 1: Update Data Structures in types.go

**Files:**
- Modify: `internal/config/types.go`

**Step 1: Add HeaderRange type**

Add after imports:
```go
type HeaderRange struct {
	Min, Max uint32
}
```

**Step 2: Add ServerObfuscationConfig**

Replace `ObfuscationConfig` with:
```go
type ServerObfuscationConfig struct {
	Jc, Jmin, Jmax int
	S1, S2, S3, S4 int
	H1, H2, H3, H4 HeaderRange
}

type ClientObfuscationConfig struct {
	ServerObfuscationConfig
	I1, I2, I3, I4, I5 string
}
```

**Step 3: Update InterfaceConfig**

Add new fields:
```go
type InterfaceConfig struct {
	PrivateKey     string
	PublicKey      string
	Address        string
	ListenPort     int
	MTU            int
	PostUp         string
	PostDown       string
	MainIface      string
	TunName        string
	EndpointV4     string
	EndpointV6     string
	ClientToClient bool
}
```

**Step 4: Update PeerConfig**

Add field:
```go
type PeerConfig struct {
	Name              string
	PrivateKey        string
	PublicKey         string
	PresharedKey      string
	AllowedIPs        string
	CreatedAt         time.Time
	ClientObfuscation *ClientObfuscationConfig
}
```

**Step 5: Update ServerConfig**

Change Obfuscation type:
```go
type ServerConfig struct {
	Interface   InterfaceConfig
	Peers       []PeerConfig
	Obfuscation ServerObfuscationConfig
}
```

**Step 6: Update ClientInterfaceConfig**

Change Obfuscation type:
```go
type ClientInterfaceConfig struct {
	PrivateKey  string
	Address     string
	DNS         string
	MTU         int
	Obfuscation ClientObfuscationConfig
}
```

**Step 7: Run tests**

Run: `go test ./internal/config/...`
Expected: Some tests may fail due to type changes

**Step 8: Commit**

```bash
git add internal/config/types.go
git commit -m "refactor: update obfuscation types for server/client separation"
```

---

## Task 2: Implement GenerateHeaderRanges in generator.go

**Files:**
- Modify: `internal/obfuscation/generator.go`
- Test: `internal/obfuscation/generator_test.go`

**Step 1: Write the failing test**

Add to `generator_test.go`:
```go
func TestGenerateHeaderRanges(t *testing.T) {
	ranges := GenerateHeaderRanges()

	// Check all 4 ranges exist
	if len(ranges) != 4 {
		t.Fatalf("expected 4 ranges, got %d", len(ranges))
	}

	// Check minimum range size
	minSize := uint32(10000000)
	for i, r := range ranges {
		size := r.Max - r.Min
		if size < minSize {
			t.Errorf("H%d range too small: %d (min %d)", i+1, size, minSize)
		}
		if r.Min < 5 {
			t.Errorf("H%d Min below 5: %d", i+1, r.Min)
		}
		if r.Max > 2147483647 {
			t.Errorf("H%d Max above 2147483647: %d", i+1, r.Max)
		}
		if r.Min >= r.Max {
			t.Errorf("H%d Min >= Max: %d >= %d", i+1, r.Min, r.Max)
		}
	}

	// Check non-overlapping
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			if ranges[i].Max >= ranges[j].Min && ranges[i].Min <= ranges[j].Max {
				t.Errorf("H%d and H%d overlap: [%d-%d] vs [%d-%d]",
					i+1, j+1, ranges[i].Min, ranges[i].Max, ranges[j].Min, ranges[j].Max)
			}
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/obfuscation/... -run TestGenerateHeaderRanges -v`
Expected: FAIL - function doesn't exist

**Step 3: Implement GenerateHeaderRanges**

Add to `generator.go`:
```go
func GenerateHeaderRanges() [4]HeaderRange {
	const (
		minValue   = uint32(5)
		maxValue   = uint32(2147483647)
		minRange   = uint32(10000000)
		maxAttempts = 1000
	)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		ranges := [4]HeaderRange{}

		for i := 0; i < 4; i++ {
			minRangeVal := big.NewInt(int64(maxValue - minValue - minRange*3))
			if minRangeVal.Int64() <= 0 {
				minRangeVal = big.NewInt(1)
			}
			minRand, _ := rand.Int(rand.Reader, minRangeVal)
			ranges[i].Min = minValue + uint32(minRand.Uint64())

			maxRangeVal := big.NewInt(int64(maxValue - ranges[i].Min - minRange))
			if maxRangeVal.Int64() < int64(minRange) {
				maxRangeVal = big.NewInt(int64(minRange))
			}
			maxRand, _ := rand.Int(rand.Reader, maxRangeVal)
			ranges[i].Max = ranges[i].Min + minRange + uint32(maxRand.Uint64())

			if ranges[i].Max > maxValue {
				ranges[i].Max = maxValue
			}
		}

		sort.Slice(ranges[:], func(i, j int) bool {
			return ranges[i].Min < ranges[j].Min
		})

		valid := true
		for i := 0; i < 3; i++ {
			if ranges[i].Max >= ranges[i+1].Min {
				valid = false
				break
			}
		}

		if valid {
			return ranges
		}
	}

	panic("failed to generate non-overlapping header ranges after 1000 attempts")
}
```

**Step 4: Add import**

Add `"sort"` to imports in `generator.go`

**Step 5: Run test to verify it passes**

Run: `go test ./internal/obfuscation/... -run TestGenerateHeaderRanges -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/obfuscation/generator.go internal/obfuscation/generator_test.go
git commit -m "feat: add GenerateHeaderRanges for non-overlapping H1-H4 ranges"
```

---

## Task 3: Update parser.go for new config format

**Files:**
- Modify: `internal/config/parser.go`
- Test: `internal/config/parser_test.go`

**Step 1: Read existing parser.go**

First examine the current parser implementation to understand the pattern.

Run: `cat internal/config/parser.go`

**Step 2: Add parseHeaderRange helper**

Add to `parser.go`:
```go
func parseHeaderRange(value string) (HeaderRange, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return HeaderRange{}, fmt.Errorf("invalid header range format: %s", value)
	}
	min, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil {
		return HeaderRange{}, fmt.Errorf("invalid header range min: %w", err)
	}
	max, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
	if err != nil {
		return HeaderRange{}, fmt.Errorf("invalid header range max: %w", err)
	}
	return HeaderRange{Min: uint32(min), Max: uint32(max)}, nil
}
```

**Step 3: Update parseInterfaceSection**

Add parsing for metadata comments and H1-H4 ranges:
```go
// In parseInterfaceSection, add:
case strings.HasPrefix(line, "#_EndpointV4"):
	cfg.Interface.EndpointV4 = strings.TrimSpace(strings.TrimPrefix(line, "#_EndpointV4="))
case strings.HasPrefix(line, "#_EndpointV6"):
	cfg.Interface.EndpointV6 = strings.TrimSpace(strings.TrimPrefix(line, "#_EndpointV6="))
case strings.HasPrefix(line, "#_ClientToClient"):
	val := strings.TrimSpace(strings.TrimPrefix(line, "#_ClientToClient="))
	cfg.Interface.ClientToClient = val == "true"
case strings.HasPrefix(line, "#_TunName"):
	cfg.Interface.TunName = strings.TrimSpace(strings.TrimPrefix(line, "#_TunName="))
case key == "H1":
	hr, err := parseHeaderRange(value)
	if err != nil {
		return err
	}
	cfg.Obfuscation.H1 = hr
// Repeat for H2, H3, H4
```

**Step 4: Update tests for new fields**

Add test case to `parser_test.go`:
```go
func TestParseServerConfigWithMetadata(t *testing.T) {
	config := `[Interface]
PrivateKey = test
Address = 10.8.0.1/24
ListenPort = 51820
#_EndpointV4 = 1.2.3.4:51820
#_EndpointV6 = [::1]:51820
#_ClientToClient = true
#_TunName = wg0
H1 = 100-10000000
H2 = 20000000-30000000
H3 = 40000000-50000000
H4 = 60000000-70000000
`
	// ... assertions
}
```

**Step 5: Run tests**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/config/parser.go internal/config/parser_test.go
git commit -m "feat: parse metadata comments and H1-H4 ranges"
```

---

## Task 4: Update writer.go for new config format

**Files:**
- Modify: `internal/config/writer.go`
- Test: `internal/config/writer_test.go`

**Step 1: Update WriteServerConfig**

Add metadata comments and H1-H4 ranges:
```go
// After writing standard fields, add:
if cfg.Interface.EndpointV4 != "" {
	fmt.Fprintf(w, "#_EndpointV4 = %s\n", cfg.Interface.EndpointV4)
}
if cfg.Interface.EndpointV6 != "" {
	fmt.Fprintf(w, "#_EndpointV6 = %s\n", cfg.Interface.EndpointV6)
}
fmt.Fprintf(w, "#_ClientToClient = %v\n", cfg.Interface.ClientToClient)
if cfg.Interface.TunName != "" {
	fmt.Fprintf(w, "#_TunName = %s\n", cfg.Interface.TunName)
}

// Write H1-H4 as ranges
fmt.Fprintf(w, "H1 = %d-%d\n", cfg.Obfuscation.H1.Min, cfg.Obfuscation.H1.Max)
fmt.Fprintf(w, "H2 = %d-%d\n", cfg.Obfuscation.H2.Min, cfg.Obfuscation.H2.Max)
fmt.Fprintf(w, "H3 = %d-%d\n", cfg.Obfuscation.H3.Min, cfg.Obfuscation.H3.Max)
fmt.Fprintf(w, "H4 = %d-%d\n", cfg.Obfuscation.H4.Min, cfg.Obfuscation.H4.Max)
```

**Step 2: Update WriteClientConfig**

Change to use ClientObfuscationConfig:
```go
func WriteClientConfig(w io.Writer, cfg ClientConfig) error {
	// ... existing code ...
	
	// Write obfuscation including I1-I5
	fmt.Fprintf(w, "Jc = %d\n", cfg.Interface.Obfuscation.Jc)
	// ... other fields ...
	fmt.Fprintf(w, "H1 = %d\n", randInRange(cfg.Interface.Obfuscation.H1.Min, cfg.Interface.Obfuscation.H1.Max))
	fmt.Fprintf(w, "H2 = %d\n", randInRange(cfg.Interface.Obfuscation.H2.Min, cfg.Interface.Obfuscation.H2.Max))
	fmt.Fprintf(w, "H3 = %d\n", randInRange(cfg.Interface.Obfuscation.H3.Min, cfg.Interface.Obfuscation.H3.Max))
	fmt.Fprintf(w, "H4 = %d\n", randInRange(cfg.Interface.Obfuscation.H4.Min, cfg.Interface.Obfuscation.H4.Max))
	fmt.Fprintf(w, "I1 = %s\n", cfg.Interface.Obfuscation.I1)
	fmt.Fprintf(w, "I2 = %s\n", cfg.Interface.Obfuscation.I2)
	fmt.Fprintf(w, "I3 = %s\n", cfg.Interface.Obfuscation.I3)
	fmt.Fprintf(w, "I4 = %s\n", cfg.Interface.Obfuscation.I4)
	fmt.Fprintf(w, "I5 = %s\n", cfg.Interface.Obfuscation.I5)
	
	return nil
}
```

**Step 3: Add randInRange helper**

```go
func randInRange(min, max uint32) uint32 {
	if min >= max {
		return min
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return min + uint32(n.Uint64())
}
```

**Step 4: Run tests**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/writer.go internal/config/writer_test.go
git commit -m "feat: write metadata comments and H1-H4 ranges"
```

---

## Task 5: Update init.go CLI command

**Files:**
- Modify: `internal/cli/init.go`
- Test: `internal/cli/init_test.go`

**Step 1: Add new flags**

Add to var block:
```go
var (
	// ... existing ...
	initIfaceName   string
	initEndpointV4  string
	initEndpointV6  string
	initConfigPath  string
)
```

**Step 2: Register new flags in init()**

```go
initCmd.Flags().StringVar(&initIfaceName, "iface-name", "awg0", "Tunnel interface name")
initCmd.Flags().StringVar(&initEndpointV4, "endpoint-v4", "", "IPv4 endpoint (auto-detect if empty)")
initCmd.Flags().StringVar(&initEndpointV6, "endpoint-v6", "", "IPv6 endpoint (optional)")
initCmd.Flags().StringVar(&initConfigPath, "config", "awg0.conf", "Server config file path")

// Remove --protocol flag
```

**Step 3: Add endpoint detection function**

```go
func getEndpointV4(port int) string {
	ip, err := getExternalIP()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

func getEndpointV6(port int) string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ipv6.icanhazip.com")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return fmt.Sprintf("[%s]:%d", ip, port)
}
```

**Step 4: Update runInit**

```go
func runInit(cmd *cobra.Command, args []string) error {
	// ... existing validation ...

	// Determine endpoints
	endpointV4 := initEndpointV4
	if endpointV4 == "" {
		endpointV4 = getEndpointV4(initPort)
	}
	
	endpointV6 := initEndpointV6
	if endpointV6 == "" {
		endpointV6 = getEndpointV6(initPort)
	}

	// Generate obfuscation config (without I1-I5)
	obfConfig := obfuscation.GenerateServerConfig(initMTU, s1, jc)

	// Create server config
	serverCfg := config.ServerConfig{
		Interface: config.InterfaceConfig{
			// ... existing ...
			TunName:        initIfaceName,
			EndpointV4:     endpointV4,
			EndpointV6:     endpointV6,
			ClientToClient: initClientToClient,
		},
		Obfuscation: obfConfig,
	}

	// Write to specified path
	if err := writeConfigFile(initConfigPath, serverCfg); err != nil {
		return err
	}

	// ... output ...
}
```

**Step 5: Update obfuscation generator**

Add to `generator.go`:
```go
func GenerateServerConfig(mtu, s1, jc int) config.ServerObfuscationConfig {
	h := GenerateHeaderRanges()
	s := GenerateSPrefixes()
	j := GenerateJunkParams()

	return config.ServerObfuscationConfig{
		Jc:   jc,
		Jmin: j.Jmin,
		Jmax: j.Jmax,
		S1:   s1,
		S2:   s.S2,
		S3:   s.S3,
		S4:   s.S4,
		H1:   h[0],
		H2:   h[1],
		H3:   h[2],
		H4:   h[3],
	}
}
```

**Step 6: Run tests**

Run: `go test ./internal/cli/... -v`
Expected: Some tests may need updates

**Step 7: Commit**

```bash
git add internal/cli/init.go internal/cli/init_test.go internal/obfuscation/generator.go
git commit -m "feat: add endpoint detection and new init flags"
```

---

## Task 6: Update export.go CLI command

**Files:**
- Modify: `internal/cli/export.go`
- Test: `internal/cli/export_test.go`

**Step 1: Add protocol flag**

```go
var (
	exportProtocol string
)

func init() {
	exportCmd.Flags().StringVar(&exportProtocol, "protocol", "random", "Obfuscation protocol")
	// Remove --endpoint flag
}
```

**Step 2: Update exportClient function**

```go
func exportClient(client config.PeerConfig, serverCfg config.ServerConfig, protocol string) error {
	// Determine endpoint from server config
	endpoint := serverCfg.Interface.EndpointV4
	if endpoint == "" {
		endpoint = serverCfg.Interface.EndpointV6
	}
	if endpoint == "" {
		// Fallback to auto-detection
		ip, err := getExternalIP()
		if err != nil {
			return fmt.Errorf("no endpoint available. Use init with --endpoint-v4")
		}
		endpoint = fmt.Sprintf("%s:%d", ip, serverCfg.Interface.ListenPort)
	}

	// Generate client obfuscation with I1-I5
	clientObf := obfuscation.GenerateClientConfig(
		protocol,
		serverCfg.Interface.MTU,
		serverCfg.Obfuscation,
	)

	// Build client config
	clientConfig := config.ClientConfig{
		Interface: config.ClientInterfaceConfig{
			PrivateKey:  client.PrivateKey,
			Address:     clientIP + "/32",
			DNS:         "1.1.1.1, 8.8.8.8",
			MTU:         serverCfg.Interface.MTU,
			Obfuscation: clientObf,
		},
		// ...
	}
	
	return config.WriteClientConfig(file, clientConfig)
}
```

**Step 3: Add GenerateClientConfig to generator.go**

```go
func GenerateClientConfig(protocol string, mtu int, serverCfg config.ServerObfuscationConfig) config.ClientObfuscationConfig {
	cps := generateCPSConfig(protocol, mtu, serverCfg.S1)
	
	return config.ClientObfuscationConfig{
		ServerObfuscationConfig: serverCfg,
		I1: cps.I1,
		I2: cps.I2,
		I3: cps.I3,
		I4: cps.I4,
		I5: cps.I5,
	}
}
```

**Step 4: Update runExport**

```go
func runExport(cmd *cobra.Command, args []string) error {
	// ... load config ...
	
	for _, client := range clientsToExport {
		if err := exportClient(client, serverCfg, exportProtocol); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 5: Run tests**

Run: `go test ./internal/cli/... -run TestExport -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cli/export.go internal/cli/export_test.go internal/obfuscation/generator.go
git commit -m "feat: move protocol to export, generate I1-I5 per client"
```

---

## Task 7: Create edit.go CLI command

**Files:**
- Create: `internal/cli/edit.go`
- Create: `internal/cli/edit_test.go`

**Step 1: Create edit.go**

```go
package cli

import (
	"fmt"

	"github.com/Arsolitt/amnezigo/internal/config"
	"github.com/Arsolitt/amnezigo/internal/network"
	"github.com/spf13/cobra"
)

var (
	editClientToClient string
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit server configuration",
	Long:  `Edit server configuration parameters.`,
	RunE:  runEdit,
}

func init() {
	editCmd.Flags().StringVar(&editClientToClient, "client-to-client", "", "Enable/disable client-to-client (true/false)")
	editCmd.Flags().StringVar(&cfgFile, "config", "awg0.conf", "Server config file")
}

func runEdit(cmd *cobra.Command, args []string) error {
	serverCfg, err := loadServerConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	changed := false

	if editClientToClient != "" {
		newValue := editClientToClient == "true"
		if serverCfg.Interface.ClientToClient && !newValue {
			// Disabling - print iptables command
			tunName := serverCfg.Interface.TunName
			if tunName == "" {
				tunName = "awg0"
			}
			fmt.Printf("Run this command to disable client-to-client immediately:\n")
			fmt.Printf("  iptables -D FORWARD -i %s -o %s -j ACCEPT\n\n", tunName, tunName)
		}
		serverCfg.Interface.ClientToClient = newValue
		changed = true
	}

	if !changed {
		fmt.Println("No changes specified")
		return nil
	}

	// Regenerate iptables rules
	subnet := extractSubnet(serverCfg.Interface.Address)
	tunName := serverCfg.Interface.TunName
	if tunName == "" {
		tunName = "awg0"
	}
	serverCfg.Interface.PostUp = network.GeneratePostUp(tunName, serverCfg.Interface.MainIface, subnet, serverCfg.Interface.ClientToClient)
	serverCfg.Interface.PostDown = network.GeneratePostDown(tunName, serverCfg.Interface.MainIface, subnet, serverCfg.Interface.ClientToClient)

	// Save config
	if err := writeConfigFile(cfgFile, serverCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Configuration updated")
	fmt.Println("  Restart AmneziaWG service to apply changes")
	return nil
}
```

**Step 2: Register command in cli.go**

Add to `init()` in `cli.go`:
```go
rootCmd.AddCommand(editCmd)
```

**Step 3: Create edit_test.go**

```go
package cli

import (
	"os"
	"testing"
)

func TestEditClientToClient(t *testing.T) {
	// Create temp config
	tmpFile := "test-edit.conf"
	defer os.Remove(tmpFile)

	// Run init first
	// Run edit
	// Verify change
}
```

**Step 4: Run tests**

Run: `go test ./internal/cli/... -run TestEdit -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/edit.go internal/cli/edit_test.go internal/cli/cli.go
git commit -m "feat: add edit command for client-to-client switching"
```

---

## Task 8: Update existing tests for new types

**Files:**
- Modify: `internal/config/types_test.go`
- Modify: `internal/obfuscation/generator_test.go`

**Step 1: Fix type assertions in tests**

Update any tests that use `ObfuscationConfig` to use `ServerObfuscationConfig` or `ClientObfuscationConfig`.

**Step 2: Update generator tests**

Remove tests for old `GenerateConfig`, update to test `GenerateServerConfig` and `GenerateClientConfig`:
```go
func TestGenerateServerConfig(t *testing.T) {
	cfg := GenerateServerConfig(1280, 15, 3)
	
	if cfg.Jc != 3 {
		t.Errorf("Jc = %d, want 3", cfg.Jc)
	}
	if cfg.S1 != 15 {
		t.Errorf("S1 = %d, want 15", cfg.S1)
	}
	// H1-H4 should be ranges
	if cfg.H1.Min >= cfg.H1.Max {
		t.Error("H1 Min >= Max")
	}
}

func TestGenerateClientConfig(t *testing.T) {
	serverCfg := GenerateServerConfig(1280, 15, 3)
	clientCfg := GenerateClientConfig("random", 1280, serverCfg)
	
	if clientCfg.I1 == "" {
		t.Error("I1 is empty")
	}
	// Should inherit server fields
	if clientCfg.Jc != serverCfg.Jc {
		t.Error("Client Jc != Server Jc")
	}
}
```

**Step 3: Run all tests**

Run: `go test ./...`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/config/types_test.go internal/obfuscation/generator_test.go
git commit -m "test: update tests for new obfuscation types"
```

---

## Task 9: Update documentation

**Files:**
- Modify: `README.md`
- Modify: `README.ru.md`

**Step 1: Update init command docs**

Remove `--protocol`, add new flags:
```markdown
### Инициализация сервера

amnezigo init --ipaddr 10.8.0.1/24 \
    [--config awg0.conf] \
    [--iface-name awg0] \
    [--endpoint-v4 1.2.3.4:51820] \
    [--endpoint-v6 [::1]:51820] \
    [--client-to-client]
```

**Step 2: Update export command docs**

Add `--protocol`:
```markdown
### Экспорт конфигурации клиента

amnezigo export laptop --protocol quic
```

**Step 3: Add edit command docs**

```markdown
### Изменение настроек сервера

amnezigo edit --client-to-client true|false
```

**Step 4: Commit**

```bash
git add README.md README.ru.md
git commit -m "docs: update for new CLI flags and commands"
```

---

## Task 10: Integration testing

**Step 1: Build binary**

Run: `go build -o amnezigo .`

**Step 2: Test init with new flags**

```bash
./amnezigo init --ipaddr 10.8.0.1/24 --iface-name test0 --config test.conf
cat test.conf
```

Expected: Config with `#_TunName = test0`

**Step 3: Test export with protocol**

```bash
./amnezigo add client1 --config test.conf
./amnezigo export client1 --protocol quic --config test.conf
cat client1.conf
```

Expected: Client config with I1-I5

**Step 4: Test edit command**

```bash
./amnezigo edit --client-to-client true --config test.conf
cat test.conf
```

Expected: `#_ClientToClient = true`

**Step 5: Cleanup**

```bash
rm -f test.conf client1.conf .main.config
```

**Step 6: Final commit**

```bash
git add -A
git commit -m "feat: complete obfuscation refactor with per-client I1-I5 and H1-H4 ranges"
```

---

## Summary

This plan implements:

1. **Data structure refactoring** - Separate server/client obfuscation configs
2. **H1-H4 ranges** - Non-overlapping ranges with min 10M size
3. **CLI init changes** - New flags for endpoint, iface-name, config path
4. **CLI export changes** - Protocol flag moved from init to export
5. **CLI edit command** - Dynamic client-to-client switching
6. **Config format** - Metadata stored as `#_` comments

**Execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks
2. **Parallel Session (separate)** - Open new session with executing-plans skill

Which approach?
