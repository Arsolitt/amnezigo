package amnezigo

import (
	"crypto/rand"
	"math/big"
)

const (
	ProtocolQUIC   = "quic"
	ProtocolDNS    = "dns"
	ProtocolDTLS   = "dtls"
	ProtocolSTUN   = "stun"
	ProtocolSIP    = "sip"
	ProtocolRTP    = "rtp"
	ProtocolRandom = "random"
)

// ListProtocols returns the names of all supported protocol templates,
// including ProtocolRandom. The slice is sorted alphabetically for
// deterministic iteration. Use it to validate user-supplied protocol
// names or to enumerate available templates.
func ListProtocols() []string {
	return []string{
		ProtocolDNS,
		ProtocolDTLS,
		ProtocolQUIC,
		ProtocolRandom,
		ProtocolRTP,
		ProtocolSIP,
		ProtocolSTUN,
	}
}

// getTemplate returns the I1I5Template for the specified protocol.
// Valid protocols: "quic", "dns", "dtls", "stun", "sip", "rtp", "random" (default).
func getTemplate(protocol string) I1I5Template {
	switch protocol {
	case ProtocolQUIC:
		return QUICTemplate()
	case ProtocolDNS:
		return DNSTemplate()
	case ProtocolDTLS:
		return DTLSTemplate()
	case ProtocolSTUN:
		return STUNTemplate()
	case ProtocolSIP:
		return SIPTemplate()
	case ProtocolRTP:
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
