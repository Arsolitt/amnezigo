package protocols

// TagSpec defines a single tag with type and value
type TagSpec struct {
	Type  string
	Value string
}

// I1I5Template contains the five intervals (I1-I5) for a protocol template
type I1I5Template struct {
	I1, I2, I3, I4, I5 []TagSpec
}

// GetTemplate returns the I1I5Template for the specified protocol
// Valid protocols: "quic", "dns", "dtls", "stun", "random" (default)
func GetTemplate(protocol string) I1I5Template {
	switch protocol {
	case "quic":
		return QUICTemplate()
	case "dns":
		return DNSTemplate()
	case "dtls":
		return DTLSTemplate()
	case "stun":
		return STUNTemplate()
	default:
		// Random selection for "random" or unknown protocols
		protocols := []func() I1I5Template{
			QUICTemplate,
			DNSTemplate,
			DTLSTemplate,
			STUNTemplate,
		}
		// Simple modulo-based selection
		return protocols[len(protocol)%len(protocols)]()
	}
}
