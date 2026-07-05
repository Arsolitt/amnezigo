package amnezigo

import (
	"crypto/rand"
	"math/big"
)

const (
	protocolQUIC   = "quic"
	protocolDNS    = "dns"
	protocolDTLS   = "dtls"
	protocolSTUN   = "stun"
	protocolSIP    = "sip"
	protocolRTP    = "rtp"
	protocolRandom = "random"
)

// getTemplate returns the I1I5Template for the specified protocol.
// Valid protocols: "quic", "dns", "dtls", "stun", "sip", "rtp", "random" (default).
func getTemplate(protocol string) I1I5Template {
	switch protocol {
	case protocolQUIC:
		return QUICTemplate()
	case protocolDNS:
		return DNSTemplate()
	case protocolDTLS:
		return DTLSTemplate()
	case protocolSTUN:
		return STUNTemplate()
	case protocolSIP:
		return SIPTemplate()
	case protocolRTP:
		return RTPTemplate()
	default:
		protocols := []func() I1I5Template{
			QUICTemplate,
			DNSTemplate,
			DTLSTemplate,
			STUNTemplate,
			SIPTemplate,
			RTPTemplate,
		}
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(protocols))))
		return protocols[n.Int64()]()
	}
}
