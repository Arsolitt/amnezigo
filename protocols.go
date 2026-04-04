package amnezigo

import (
	"crypto/rand"
	"math/big"
)

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
		protocols := []func() I1I5Template{
			QUICTemplate,
			DNSTemplate,
			DTLSTemplate,
			STUNTemplate,
		}
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(protocols))))
		return protocols[n.Int64()]()
	}
}
