package obfuscation

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/Arsolitt/amnezigo/internal/obfuscation/protocols"
)

const (
	reserve       = 28  // IP header 20 + UDP header 8
	handshakeSize = 148 // fixed handshake size
)

// simpleTag represents a CPS tag with type and value
type simpleTag struct {
	Type  string // "b", "r", "rc", "rd", "t", "c"
	Value string // hex for "b", number for "r"/"rc"/"rd", empty for "t"/"c"
}

// calculateMaxISize calculates the maximum I packet size based on MTU constraints.
// Formula: maxISize = (MTU - reserve - handshakeSize - S1) / (5 + jc)
func calculateMaxISize(mtu, s1, jc int) int {
	return (mtu - reserve - handshakeSize - s1) / (5 + jc)
}

// BuildCPSTag creates a CPS (Custom Packet String) tag from a tag type and value.
// Supported tag types:
// - "b" + value → bytes in hex (e.g., "b" + "0xc00000000108" → "<b 0xc00000000108>")
// - "r" + value → random bytes (e.g., "r" + "20" → "<r 20>")
// - "rc" + value → random ASCII chars (e.g., "rc" + "8" → "<rc 8>")
// - "rd" + value → random digits (e.g., "rd" + "4" → "<rd 4>")
// - "c" → counter (e.g., "c" → "<c>")
// - "t" → timestamp (e.g., "t" → "<t>")
func BuildCPSTag(tagType, value string) string {
	switch tagType {
	case "b":
		if !strings.HasPrefix(value, "0x") {
			value = "0x" + value
		}
		return fmt.Sprintf("<b %s>", value)
	case "r":
		return fmt.Sprintf("<r %s>", value)
	case "rc":
		return fmt.Sprintf("<rc %s>", value)
	case "rd":
		return fmt.Sprintf("<rd %s>", value)
	case "c":
		return "<c>"
	case "t":
		return "<t>"
	default:
		return ""
	}
}

// BuildCPS concatenates multiple CPS tags into a single CPS string.
func BuildCPS(tags []string) string {
	return strings.Join(tags, "")
}

// CPSConfig holds the five intervals (I1-I5) of custom packet strings
type CPSConfig struct {
	I1, I2, I3, I4, I5 string
}

// generateCPSConfig generates CPS strings for all five intervals based on protocol template
// or random mode with MTU constraints
func generateCPSConfig(protocol string, mtu, s1, jc int) CPSConfig {
	if protocol == "random" {
		return generateSimpleCPS(mtu, s1, jc)
	}
	return generateProtocolCPS(protocol, mtu, s1, jc)
}

// generateProtocolCPS generates CPS strings from protocol template with MTU validation
func generateProtocolCPS(protocol string, mtu, s1, jc int) CPSConfig {
	tmpl := protocols.GetTemplate(protocol)
	maxI := calculateMaxISize(mtu, s1, jc)

	return CPSConfig{
		I1: buildAndValidateCPS(tmpl.I1, maxI),
		I2: buildAndValidateCPS(tmpl.I2, maxI),
		I3: buildAndValidateCPS(tmpl.I3, maxI),
		I4: buildAndValidateCPS(tmpl.I4, maxI),
		I5: buildAndValidateCPS(tmpl.I5, maxI),
	}
}

// buildCPSFromTemplate converts a template interval to a CPS string
func buildCPSFromTemplate(tags []protocols.TagSpec) string {
	var result []string
	for _, tag := range tags {
		result = append(result, BuildCPSTag(mapTagType(tag.Type), tag.Value))
	}
	return BuildCPS(result)
}

// buildAndValidateCPS builds a CPS from template tags and validates it fits within maxSize.
// If the CPS exceeds maxSize, it attempts to reduce it by removing tags or falls back to minimal CPS.
func buildAndValidateCPS(tags []protocols.TagSpec, maxSize int) string {
	cps := buildCPSFromTemplate(tags)

	// If CPS fits within constraints, return it
	if calculateCPSLength(cps) < maxSize {
		return cps
	}

	// If too large, try progressively smaller versions
	for len(tags) > 0 {
		tags = tags[:len(tags)-1] // Remove one tag at a time
		cps = buildCPSFromTemplate(tags)
		if calculateCPSLength(cps) < maxSize {
			return cps
		}
	}

	// Fallback to minimal valid CPS
	return "<c>" // guaranteed minimal fallback (4 bytes)
}

// mapTagType maps protocol tag types to CPS tag types
func mapTagType(tagType string) string {
	switch tagType {
	case "bytes":
		return "b"
	case "random":
		return "r"
	case "random_chars":
		return "rc"
	case "random_digits":
		return "rd"
	case "counter":
		return "c"
	case "timestamp":
		return "t"
	default:
		return ""
	}
}

// calculateCPSLength calculates the byte length of a CPS string by parsing its tags.
// It supports:
// - <b 0xNN>: len(NN)/2 bytes (hex string to bytes)
// - <r N>, <rc N>, <rd N>: N bytes each
// - <t>, <c>: 8 bytes each
func calculateCPSLength(cps string) int {
	total := 0

	// Match <b 0x...> tags
	bytesRegex := regexp.MustCompile(`<b\s+0x([0-9a-fA-F]+)>`)
	matches := bytesRegex.FindAllStringSubmatch(cps, -1)
	for _, match := range matches {
		hexValue := match[1]
		total += len(hexValue) / 2
	}

	// Match <r N>, <rc N>, <rd N> tags
	countRegex := regexp.MustCompile(`<r[c|d]?\s+(\d+)>`)
	matches = countRegex.FindAllStringSubmatch(cps, -1)
	for _, match := range matches {
		count, _ := strconv.Atoi(match[1])
		total += count
	}

	// Match <t> and <c> tags (8 bytes each)
	fixedRegex := regexp.MustCompile(`<[tc]>`)
	matches = fixedRegex.FindAllStringSubmatch(cps, -1)
	total += len(matches) * 8

	return total
}

// generateRandomTags generates random CPS tags for simple random mode
func generateRandomTags(minCount, maxCount int) []simpleTag {
	allTagTypes := []string{"b", "r", "rc", "rd", "t", "c"}
	usedUnique := make(map[string]bool)

	// Generate random count between minCount and maxCount
	countRange := maxCount - minCount
	if countRange < 0 {
		countRange = 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(countRange+1)))
	count := minCount + int(n.Int64())

	tags := make([]simpleTag, count)
	for i := 0; i < count; i++ {
		// Filter out already-used unique tags
		var availableTagTypes []string
		for _, tagType := range allTagTypes {
			if (tagType == "t" || tagType == "c") && usedUnique[tagType] {
				continue // skip if unique tag already used
			}
			availableTagTypes = append(availableTagTypes, tagType)
		}

		// Random tag type from available types
		typeIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(availableTagTypes))))
		tagType := availableTagTypes[typeIdx.Int64()]

		// Mark unique tags as used
		if tagType == "t" || tagType == "c" {
			usedUnique[tagType] = true
		}

		// Generate value based on tag type
		var value string
		switch tagType {
		case "b":
			// Random hex 4-16 bytes (8-32 hex chars with 0x prefix)
			byteLenRange := 16 - 4 // 12 possible values (4-16)
			byteLenRand, _ := rand.Int(rand.Reader, big.NewInt(int64(byteLenRange+1)))
			byteLen := 4 + int(byteLenRand.Int64())
			bytes := make([]byte, byteLen)
			rand.Read(bytes)
			value = "0x" + hex.EncodeToString(bytes)
		case "r", "rc", "rd":
			// Random 5-40 bytes
			sizeRange := 40 - 5
			sizeRand, _ := rand.Int(rand.Reader, big.NewInt(int64(sizeRange+1)))
			size := 5 + int(sizeRand.Int64())
			value = fmt.Sprintf("%d", size)
		case "t", "c":
			// No value
			value = ""
		}

		tags[i] = simpleTag{
			Type:  tagType,
			Value: value,
		}
	}

	return tags
}

func generateSimpleCPS(mtu, s1, jc int) CPSConfig {
	maxI := calculateMaxISize(mtu, s1, jc)

	return CPSConfig{
		I1: generateSimpleI(maxI),
		I2: generateSimpleI(maxI),
		I3: generateSimpleI(maxI),
		I4: generateSimpleI(maxI),
		I5: generateSimpleI(maxI),
	}
}

func generateSimpleI(maxSize int) string {
	for attempt := 0; attempt < 100; attempt++ {
		tags := generateRandomTags(3, 6)
		cps := tagsToCPS(tags)
		if calculateCPSLength(cps) < maxSize {
			return cps
		}
	}
	return "<c>" // guaranteed minimal fallback (4 bytes)
}

func tagsToCPS(tags []simpleTag) string {
	var result []string
	for _, tag := range tags {
		result = append(result, BuildCPSTag(tag.Type, tag.Value))
	}
	return BuildCPS(result)
}
