# Fix Known Gotchas

## Problem

AGENTS.md documents 4 known gotchas that cause incorrect or misleading behavior.

## Gotcha #1: export has no --endpoint flag

**Current:** `export` always auto-resolves endpoint from server config metadata or HTTP fallback. No manual override.

**Fix:** Add `--endpoint` flag to `NewExportCommand()`. When provided, skip `resolveExportEndpoint()` and use the value directly.

**Files:** `internal/cli/export.go`

## Gotcha #2: --dns and --keepalive on init are silently ignored

**Current:** `--dns` and `--keepalive` flags exist on `init` but are not saved to config. `BuildPeerConfig()` hardcodes DNS to `"1.1.1.1, 8.8.8.8"` and keepalive to `25`.

**Fix:**
1. Add `DNS string` and `PersistentKeepalive int` fields to `InterfaceConfig` in `types.go`
2. Save values from flags in `runInit()` (`internal/cli/init.go`)
3. Parse fields from INI config in `parser.go`
4. Write fields to INI config in `writer.go`
5. Use `serverCfg.Interface.DNS` and `serverCfg.Interface.PersistentKeepalive` in `BuildPeerConfig()` (`manager.go`), falling back to current defaults if empty/zero

**Files:** `types.go`, `internal/cli/init.go`, `parser.go`, `writer.go`, `manager.go`

## Gotcha #3: "random" protocol is deterministic

**Current:** `getTemplate()` uses `len(protocol) % len(protocols)` for random selection. Since `len("random") = 6` and `6 % 4 = 2`, it always selects DTLS.

**Fix:** Replace modulo with `crypto/rand.Int(rand.Reader, big.NewInt(len(protocols)))` for true random selection.

**Files:** `protocols.go`

## Gotcha #4: GenerateConfig uses point ranges for H1-H4

**Current:** `GenerateConfig()` calls `GenerateHeaders()` which returns single values, then stores them as `HeaderRange{Min: h.H1, Max: h.H1}` (point ranges). `GenerateServerConfig()` correctly uses `GenerateHeaderRanges()`.

**Fix:** Replace `GenerateHeaders()` call with `GenerateHeaderRanges()` in `GenerateConfig()`. Remove TODO comments. Remove unused `GenerateHeaders()` function.

**Files:** `generator.go`

## Testing

- Update existing tests for each changed function
- Add test for `--endpoint` flag override in export
- Add test for DNS/keepalive roundtrip (init -> save -> load -> export)
- Add test for random protocol distribution (not always DTLS)
- Update `GenerateConfig` test to verify true ranges
