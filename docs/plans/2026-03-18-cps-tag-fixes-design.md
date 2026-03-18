# CPS Tag Generation Fixes

## Problem

Some I1-I5 obfuscation parameters fail with `Invalid argument` in amneziawg-go userspace.

## Root Causes

1. **Size calculation**: `<t>` and `<c>` counted as 4 bytes, but amneziawg-go generates 8 bytes each
2. **Unique tag constraint**: Random generation can produce multiple `<c>` or `<t>` in same I packet
3. **Protocol templates**: QUIC/DTLS templates have `counter`/`timestamp` with non-empty values, but amneziawg-go rejects any parameter for these tags

## Fixes

### 1. Size Calculation (`internal/obfuscation/cps.go:174`)

```go
// Before
total += len(matches) * 4

// After
total += len(matches) * 8
```

### 2. Unique Tag Constraint (`internal/obfuscation/cps.go:generateRandomTags`)

Track and prevent duplicate `<c>` or `<t>` tags:

```go
func generateRandomTags(minCount, maxCount int) []simpleTag {
    tagTypes := []string{"b", "r", "rc", "rd", "t", "c"}
    usedUnique := make(map[string]bool)
    
    for each tag:
        availableTypes := filter out "t" if usedUnique["t"], "c" if usedUnique["c"]
        if tagType == "t" || tagType == "c" {
            usedUnique[tagType] = true
        }
}
```

### 3. Protocol Templates (`internal/obfuscation/protocols/*.go`)

Remove all values from `counter` and `timestamp` tags:

**quic.go** (lines 26-27, 41-42, 56-57, 71-72):
```go
// Before: {Type: "counter", Value: "2"}, {Type: "timestamp", Value: "ms"}
// After:  {Type: "counter", Value: ""}, {Type: "timestamp", Value: ""}
```

**dtls.go** (lines 44, 75, 103, 131):
```go
// Before: {Type: "timestamp", Value: "sec"}
// After:  {Type: "timestamp", Value: ""}
```

## Files to Modify

- `internal/obfuscation/cps.go` - size calc, unique constraint
- `internal/obfuscation/protocols/quic.go` - template values
- `internal/obfuscation/protocols/dtls.go` - template values

## Testing

- Run existing tests: `go test ./internal/obfuscation/...`
- Verify generated CPS strings don't exceed MTU
- Verify no duplicate `<c>` or `<t>` in generated strings
