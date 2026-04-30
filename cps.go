package amnezigo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

const (
	reserve       = 49  // IP header (20 or 40) + UDP header 8 (+ 1 just in case)
	handshakeSize = 149 // fixed handshake size (148 + 1 just in case)

	hexBytesPerChar = 2
	// cpsTimestampSize is the byte length of the <t> tag.
	// Source: amneziawg-go device/obf_timestamp.go writes
	// binary.BigEndian.PutUint32(buf, uint32(time.Now().Unix())) — 4 bytes.
	cpsTimestampSize   = 4
	maxByteLen         = 16
	minByteLen         = 4
	maxSize            = 40
	minSize            = 5
	maxTagCount        = 6
	minTagCount        = 3
	tagTerminate       = "<t>"
	tagDataPassthrough = "<d>"

	// cpsRcAlphabet is the canonical alphabet used by amneziawg-go to fill
	// <rc N> tags at packet emission time. 52 ASCII letters, lowercase first
	// then uppercase, sorted alphabetically.
	//
	// Source of truth: amneziawg-go device/obf_randchars.go.
	//
	// IMPORTANT — common pitfall: the Habr article describing AWG 2.0
	// incorrectly states the alphabet is [a-zA-Z0-9]. It is NOT.
	// For mixed letter+digit fields, use tag concatenation:
	//   <rc 4><rd 2>  →  4 letters followed by 2 digits.
	// Do NOT introduce a separate alphanumeric tag.
	//
	// amnezigo itself only emits the tag literal "<rc N>"; the bytes
	// are produced by the AmneziaWG receiver. This constant exists to
	// document the contract and to anchor the regression test.
	cpsRcAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// calculateMaxISize calculates the maximum I packet size based on MTU constraints.
// Formula: maxISize = MTU - reserve - handshakeSize - S1.
func calculateMaxISize(mtu, s1 int) int {
	return mtu - reserve - handshakeSize - s1
}

// BuildCPSTag creates a CPS (Custom Packet String) tag from a tag type and value.
// Supported tag types:
//   - "b" + value → bytes in hex (e.g., "b" + "0xc00000000108" → "<b 0xc00000000108>")
//   - "r" + value → random bytes (e.g., "r" + "20" → "<r 20>")
//   - "rc" + value → tag for N random letters from [a-zA-Z] (52 chars, see cpsRcAlphabet)
//     (e.g., "rc" + "8" → "<rc 8>"; the receiver fills 8 random letters at emit time)
//   - "rd" + value → random digits (e.g., "rd" + "4" → "<rd 4>")
//   - "t" → timestamp (e.g., "t" → "<t>").
//   - "d" → data passthrough (e.g., "d" → "<d>"; AWG userspace expands at emit time
//     by reusing a value from an earlier I-packet position, contributing 0 bytes
//     at generation time. The value parameter is ignored. Requires AWG 2.0
//     userspace; the legacy Linux kernel module rejects "<d>" with "unknown tag".)
//
// The legacy "c" (counter) tag is intentionally NOT supported: it is recognised
// only by amneziawg-linux-kernel-module. amneziawg-go and all AmneziaVPN
// clients (iOS, Android, Windows, macOS) reject "<c>" with "unknown tag",
// breaking generated configs. Passing "c" returns the empty string sentinel
// like any other unknown tag type.
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
	case "t":
		return tagTerminate
	case "d":
		return tagDataPassthrough
	default:
		return ""
	}
}

// BuildCPS concatenates multiple CPS tags into a single CPS string.
func BuildCPS(tags []string) string {
	return strings.Join(tags, "")
}

// generateCPSConfig generates CPS strings for all five intervals based on
// protocol template or random mode with MTU constraints. The forbidden set
// holds packet sizes (typically the four AWG-padded handshake sizes) that
// each I-packet length must avoid.
func generateCPSConfig(protocol string, mtu, s1 int, forbidden [4]int) CPSConfig {
	if protocol == "random" {
		return generateSimpleCPS(mtu, s1, forbidden)
	}
	return generateProtocolCPS(protocol, mtu, s1, forbidden)
}

// generateProtocolCPS generates CPS strings from a protocol template, honoring
// both MTU and a forbidden-size set.
func generateProtocolCPS(protocol string, mtu, s1 int, forbidden [4]int) CPSConfig {
	tmpl := getTemplate(protocol)
	maxI := calculateMaxISize(mtu, s1)

	return CPSConfig{
		I1: buildAndValidateCPS(tmpl.I1, maxI, forbidden),
		I2: buildAndValidateCPS(tmpl.I2, maxI, forbidden),
		I3: buildAndValidateCPS(tmpl.I3, maxI, forbidden),
		I4: buildAndValidateCPS(tmpl.I4, maxI, forbidden),
		I5: buildAndValidateCPS(tmpl.I5, maxI, forbidden),
	}
}

// buildCPSFromTemplate converts a template interval to a CPS string.
func buildCPSFromTemplate(tags []TagSpec) string {
	var result []string
	for _, tag := range tags {
		result = append(result, BuildCPSTag(mapTagType(tag.Type), tag.Value))
	}
	return BuildCPS(result)
}

// buildAndValidateCPS builds a CPS from template tags and validates it fits
// within maxSize and that its built byte-length is not in forbidden. On a
// forbidden collision after MTU shrinking, it perturbs the output by appending
// `<rd N>` for N in [1..8] before falling back to a minimal CPS.
func buildAndValidateCPS(tags []TagSpec, maxSize int, forbidden [4]int) string {
	cps := buildCPSFromTemplate(tags)
	if cpsAcceptable(cps, maxSize, forbidden) {
		return cps
	}

	// Try progressively smaller versions to fit within maxSize first.
	for len(tags) > 0 {
		tags = tags[:len(tags)-1] // Remove one tag at a time
		cps = buildCPSFromTemplate(tags)
		if cpsAcceptable(cps, maxSize, forbidden) {
			return cps
		}
	}

	// All shrunk versions either exceeded maxSize or collided with a forbidden
	// size. Perturb away with <rd N>: 8 attempts cover any 4-element forbidden
	// set (pigeonhole — eight distinct lengths, at most four forbidden).
	for n := 1; n <= 8; n++ {
		perturbed := tagTerminate + fmt.Sprintf("<rd %d>", n)
		if cpsAcceptable(perturbed, maxSize, forbidden) {
			return perturbed
		}
	}

	return tagTerminate // guaranteed minimal fallback (4 bytes)
}

// cpsAcceptable returns true iff cps fits maxSize AND its calculated length is
// not in forbidden. The MTU bound is strict (`<`) to match prior behavior.
// Zero entries in forbidden are treated as unset (the smallest legal CPS is 4
// bytes, so 0 cannot collide with a real packet).
func cpsAcceptable(cps string, maxSize int, forbidden [4]int) bool {
	n := calculateCPSLength(cps)
	if n >= maxSize {
		return false
	}
	for _, f := range forbidden {
		if f == 0 {
			continue
		}
		if n == f {
			return false
		}
	}
	return true
}

// mapTagType maps protocol tag types to CPS tag types. The legacy "counter"
// type is intentionally not mapped — see BuildCPSTag for rationale; it falls
// through to the empty-string default along with any other unknown type.
// "data" maps to "d" (runtime passthrough, AWG 2.0 only).
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
	case "timestamp":
		return "t"
	case "data":
		return "d"
	default:
		return ""
	}
}

// calculateCPSLength calculates the byte length of a CPS string by parsing its tags.
// It supports:
//   - <b 0xNN>: len(NN)/2 bytes (hex string to bytes)
//   - <r N>, <rc N>, <rd N>: N bytes each
//   - <t>: 4 bytes
//   - <d>: 0 bytes (runtime passthrough; AWG 2.0 userspace expands at emit time
//     by reusing a value from an earlier I-packet position).
//
// Unknown tag literals (including the legacy kernel-only "<c>" tag, which
// AmneziaVPN clients reject) contribute 0 bytes — the receiver would error
// out on them anyway, and accounting for them here would mask the regression.
func calculateCPSLength(cps string) int {
	total := 0

	// Match <b 0x...> tags
	bytesRegex := regexp.MustCompile(`<b\s+0x([0-9a-fA-F]+)>`)
	matches := bytesRegex.FindAllStringSubmatch(cps, -1)
	for _, match := range matches {
		hexValue := match[1]
		total += len(hexValue) / hexBytesPerChar
	}

	// Match <r N>, <rc N>, <rd N> tags
	countRegex := regexp.MustCompile(`<r[c|d]?\s+(\d+)>`)
	matches = countRegex.FindAllStringSubmatch(cps, -1)
	for _, match := range matches {
		count, _ := strconv.Atoi(match[1])
		total += count
	}

	// Match <t> tags (4 bytes each — uint32 BigEndian timestamp)
	tsMatches := regexp.MustCompile(`<t>`).FindAllString(cps, -1)
	total += len(tsMatches) * cpsTimestampSize

	return total
}

// generateRandomTags generates random CPS tags for simple random mode.
// "d" is intentionally excluded: <d> is a runtime-passthrough marker that only
// makes sense in templated multi-interval flows where an earlier interval
// produces the value being reused. Random mode emits standalone intervals with
// no chaining context, so <d> would expand to nothing at runtime.
func generateRandomTags() []simpleTag {
	allTagTypes := []string{"b", "r", "rc", "rd", "t"}
	usedUniqueTag := false

	countRange := max(maxTagCount-minTagCount, 0)
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(countRange+1)))
	count := minTagCount + int(n.Int64())

	tags := make([]simpleTag, count)
	for i := range count {
		// Filter out unique tags if one was already used
		var availableTagTypes []string
		for _, tagType := range allTagTypes {
			if tagType == "t" && usedUniqueTag {
				continue
			}
			availableTagTypes = append(availableTagTypes, tagType)
		}

		// Random tag type from available types
		typeIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(availableTagTypes))))
		tagType := availableTagTypes[typeIdx.Int64()]

		// Mark if unique tag was used
		if tagType == "t" {
			usedUniqueTag = true
		}

		// Generate value based on tag type
		var value string
		switch tagType {
		case "b":
			byteLenRange := maxByteLen - minByteLen
			byteLenRand, _ := rand.Int(rand.Reader, big.NewInt(int64(byteLenRange+1)))
			byteLen := minByteLen + int(byteLenRand.Int64())
			bytes := make([]byte, byteLen)
			rand.Read(bytes)
			value = "0x" + hex.EncodeToString(bytes)
		case "r", "rc", "rd":
			sizeRange := maxSize - minSize
			sizeRand, _ := rand.Int(rand.Reader, big.NewInt(int64(sizeRange+1)))
			size := minSize + int(sizeRand.Int64())
			value = strconv.Itoa(size)
		case "t":
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

func generateSimpleCPS(mtu, s1 int, forbidden [4]int) CPSConfig {
	maxI := calculateMaxISize(mtu, s1)

	return CPSConfig{
		I1: generateSimpleI(maxI, forbidden),
		I2: generateSimpleI(maxI, forbidden),
		I3: generateSimpleI(maxI, forbidden),
		I4: generateSimpleI(maxI, forbidden),
		I5: generateSimpleI(maxI, forbidden),
	}
}

// generateSimpleI generates a random-mode I-packet that fits within maxSize
// and whose length is not in forbidden. On retry-budget exhaustion, it
// perturbs `<t><rd N>` before falling back to bare `<t>`.
func generateSimpleI(maxSize int, forbidden [4]int) string {
	for range iPacketMaxAttempts {
		tags := generateRandomTags()
		cps := tagsToCPS(tags)
		if cpsAcceptable(cps, maxSize, forbidden) {
			return cps
		}
	}
	for n := 1; n <= 8; n++ {
		perturbed := tagTerminate + fmt.Sprintf("<rd %d>", n)
		if cpsAcceptable(perturbed, maxSize, forbidden) {
			return perturbed
		}
	}
	return tagTerminate // guaranteed minimal fallback (4 bytes)
}

func tagsToCPS(tags []simpleTag) string {
	var result []string
	for _, tag := range tags {
		result = append(result, BuildCPSTag(tag.Type, tag.Value))
	}
	return BuildCPS(result)
}
