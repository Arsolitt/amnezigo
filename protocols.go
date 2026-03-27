package amnezigo

// getTemplate returns the I1I5Template for the specified protocol.
// Valid protocols: "quic", "dns", "dtls", "stun", "random" (default).
func getTemplate(protocol string) I1I5Template {
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
