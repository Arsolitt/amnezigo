package obfuscation

import (
	"fmt"
	"strings"

	"github.com/Arsolitt/amnezigo/internal/obfuscation/protocols"
)

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
func generateCPSConfig(protocol string) CPSConfig {
	tmpl := protocols.GetTemplate(protocol)

	return CPSConfig{
		I1: buildCPSFromTemplate(tmpl.I1),
		I2: buildCPSFromTemplate(tmpl.I2),
		I3: buildCPSFromTemplate(tmpl.I3),
		I4: buildCPSFromTemplate(tmpl.I4),
		I5: buildCPSFromTemplate(tmpl.I5),
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

// mapTagType maps protocol tag types to CPS tag types
func mapTagType(tagType string) string {
	switch tagType {
	case "bytes":
		return "b"
	case "random":
		return "r"
	case "counter":
		return "c"
	case "timestamp":
		return "t"
	default:
		return ""
	}
}
