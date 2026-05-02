package amnezigo

import (
	"crypto/rand"
	"math/big"
)

const protocolSIP = "sip"

// getTemplate returns the I1I5Template for the specified protocol.
// Valid protocols: "quic", "dns", "dtls", "stun", "sip", "random" (default).
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
	case protocolSIP:
		return SIPTemplate()
	default:
		protocols := []func() I1I5Template{
			QUICTemplate,
			DNSTemplate,
			DTLSTemplate,
			STUNTemplate,
			SIPTemplate,
		}
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(protocols))))
		return protocols[n.Int64()]()
	}
}
