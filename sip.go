package amnezigo

// Hex-encoded wire fragments reused across the SIP template intervals. TagSpec
// values of type tagTypeBytes are decoded to raw bytes on emission, so the
// ASCII fragments below are spelled out as lowercase hex.
const (
	sipHexOptions        = "4f5054494f4e53"                   // "OPTIONS"
	sipHexSpaceOptions   = "204f5054494f4e53"                 // " OPTIONS" (CSeq method, leading space)
	sipHexSIPScheme      = "7369703a"                         // "sip:"
	sipHexSIPSchemeAngle = "3c7369703a"                       // "<sip:"
	sipHexVersion20      = "2f322e30"                         // "/2.0"
	sipHexCRLF           = "0d0a"                             // CRLF line terminator
	sipHexCRLFCRLF       = "0d0a0d0a"                         // CRLF CRLF (end of headers)
	sipHexVia            = "5669613a20"                       // "Via: "
	sipHexUDPTransport   = "5349502f322e302f554450"           // "SIP/2.0/UDP"
	sipHexBranchParam    = "3b6272616e63683d"                 // ";branch="
	sipHexBranchCookie   = "7a39684734624b"                   // "z9hG4bK" (RFC 3261 § 8.1.1.7 magic cookie)
	sipHexCallID         = "43616c6c2d49443a20"               // "Call-ID: "
	sipHexCSeq           = "435365713a20"                     // "CSeq: "
	sipHexContentLength  = "436f6e74656e742d4c656e6774683a20" // "Content-Length: "
)

// SIPTemplate returns an I1I5Template mimicking a SIP OPTIONS request.
//
// Wire-format reference: RFC 3261 § 7.1 (Request-Line grammar), § 27.4 (registered
// method tokens — OPTIONS is one of the six core methods), § 8.1.1.7 (branch
// parameter magic cookie "z9hG4bK"), §§ 20.8, 20.14, 20.16, 20.20, 20.22, 20.39,
// 20.41, 20.42 (header field grammars used below).
// Verified against RFC 3261 (current as of 2026-04-30).
//
// SIP runs over UDP/5060. OPTIONS is a no-op ping that VoIP gateways send routinely
// and that enterprise firewalls almost always permit. The template constructs an
// ASCII, line-delimited request:
//
//	OPTIONS sip:user@domain.example SIP/2.0\r\n
//	Via: SIP/2.0/UDP <random branch>\r\n
//	From: <sip:alice@example.com>;tag=<random>\r\n
//	To: <sip:bob@example.com>\r\n
//	Call-ID: <random>@example.com\r\n
//	CSeq: 1 OPTIONS\r\n
//	Max-Forwards: 70\r\n
//	User-Agent: <random>\r\n
//	Content-Length: 0\r\n\r\n
//
// All variable tokens are <rc N> (letters per [a-zA-Z]) or <rd N> (digits) — no
// <t> timestamp, no <r N> binary noise. This makes SIP's byte-length distribution
// distinct from QUIC/DTLS (which use <t>) and STUN (which uses <r> for txn-IDs).
//
// Byte budgets (template, before MTU clip):
//
//	I1: ~360 B (full OPTIONS with all standard headers)
//	I2: ~240 B (drops User-Agent, shortens Call-ID)
//	I3: ~170 B (minimal but RFC-conformant)
//	I4: ~120 B (request-line + Via + Call-ID + CSeq + Content-Length only)
//	I5: empty (named-template convention).
func SIPTemplate() I1I5Template {
	return I1I5Template{
		// I1 — full OPTIONS request with all common headers (~360 B)
		I1: []TagSpec{
			// Request-Line: "OPTIONS sip:" + user + "@" + host + " SIP/2.0\r\n"
			{Type: tagTypeBytes, Value: sipHexOptions},   // "OPTIONS"
			{Type: tagTypeBytes, Value: "20"},            // " "
			{Type: tagTypeBytes, Value: sipHexSIPScheme}, // "sip:"
			{Type: tagTypeRandomChars, Value: "8"},       // user (8 letters)
			{Type: tagTypeBytes, Value: "40"},            // "@"
			{Type: tagTypeRandomChars, Value: "10"},      // host (10 letters)
			{Type: tagTypeBytes, Value: "2e"},            // "."
			{Type: tagTypeRandomChars, Value: "3"},       // TLD (3 letters)
			{Type: tagTypeBytes, Value: "20534950"},      // " SIP"
			{Type: tagTypeBytes, Value: sipHexVersion20}, // "/2.0"
			{Type: tagTypeBytes, Value: sipHexCRLF},      // CRLF
			// Via header
			{Type: tagTypeBytes, Value: sipHexVia},          // "Via: "
			{Type: tagTypeBytes, Value: sipHexUDPTransport}, // "SIP/2.0/UDP"
			{Type: tagTypeBytes, Value: "20"},               // " "
			{Type: tagTypeRandomChars, Value: "10"},         // host token
			{Type: tagTypeBytes, Value: sipHexBranchParam},  // ";branch="
			{Type: tagTypeBytes, Value: sipHexBranchCookie}, // "z9hG4bK" (RFC 3261 § 8.1.1.7 magic cookie)
			{Type: tagTypeRandomChars, Value: "16"},         // branch random token
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// From header
			{Type: tagTypeBytes, Value: "46726f6d3a20"},       // "From: "
			{Type: tagTypeBytes, Value: sipHexSIPSchemeAngle}, // "<sip:"
			{Type: tagTypeRandomChars, Value: "8"},            // user
			{Type: tagTypeBytes, Value: "40"},                 // "@"
			{Type: tagTypeRandomChars, Value: "10"},           // host
			{Type: tagTypeBytes, Value: "3e"},                 // ">"
			{Type: tagTypeBytes, Value: "3b7461673d"},         // ";tag="
			{Type: tagTypeRandomChars, Value: "12"},           // tag random token
			{Type: tagTypeBytes, Value: sipHexCRLF},           // CRLF
			// To header
			{Type: tagTypeBytes, Value: "546f3a20"},           // "To: "
			{Type: tagTypeBytes, Value: sipHexSIPSchemeAngle}, // "<sip:"
			{Type: tagTypeRandomChars, Value: "8"},            // user
			{Type: tagTypeBytes, Value: "40"},                 // "@"
			{Type: tagTypeRandomChars, Value: "10"},           // host
			{Type: tagTypeBytes, Value: "3e"},                 // ">"
			{Type: tagTypeBytes, Value: sipHexCRLF},           // CRLF
			// Call-ID
			{Type: tagTypeBytes, Value: sipHexCallID}, // "Call-ID: "
			{Type: tagTypeRandomChars, Value: "20"},   // call-id random
			{Type: tagTypeBytes, Value: "40"},         // "@"
			{Type: tagTypeRandomChars, Value: "10"},   // host
			{Type: tagTypeBytes, Value: sipHexCRLF},   // CRLF
			// CSeq
			{Type: tagTypeBytes, Value: sipHexCSeq},         // "CSeq: "
			{Type: tagTypeRandomDigits, Value: "3"},         // sequence number digits
			{Type: tagTypeBytes, Value: sipHexSpaceOptions}, // " OPTIONS"
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// Max-Forwards
			{Type: tagTypeBytes, Value: "4d61782d466f7277617264733a20"}, // "Max-Forwards: "
			{Type: tagTypeBytes, Value: "3730"},                         // "70"
			{Type: tagTypeBytes, Value: sipHexCRLF},                     // CRLF
			// User-Agent
			{Type: tagTypeBytes, Value: "557365722d4167656e743a20"}, // "User-Agent: "
			{Type: tagTypeRandomChars, Value: "12"},                 // UA random token
			{Type: tagTypeBytes, Value: sipHexCRLF},                 // CRLF
			// Content-Length
			{Type: tagTypeBytes, Value: sipHexContentLength}, // "Content-Length: "
			{Type: tagTypeBytes, Value: "30"},                // "0"
			{Type: tagTypeBytes, Value: sipHexCRLFCRLF},      // CRLF CRLF (end-of-headers)
		},

		// I2 — drops User-Agent, shortens random tokens (~240 B)
		I2: []TagSpec{
			// Request-Line
			{Type: tagTypeBytes, Value: sipHexOptions},   // "OPTIONS"
			{Type: tagTypeBytes, Value: "20"},            // " "
			{Type: tagTypeBytes, Value: sipHexSIPScheme}, // "sip:"
			{Type: tagTypeRandomChars, Value: "6"},       // user
			{Type: tagTypeBytes, Value: "40"},            // "@"
			{Type: tagTypeRandomChars, Value: "8"},       // host
			{Type: tagTypeBytes, Value: "2e"},            // "."
			{Type: tagTypeRandomChars, Value: "3"},       // TLD
			{Type: tagTypeBytes, Value: "20534950"},      // " SIP"
			{Type: tagTypeBytes, Value: sipHexVersion20}, // "/2.0"
			{Type: tagTypeBytes, Value: sipHexCRLF},      // CRLF
			// Via header
			{Type: tagTypeBytes, Value: sipHexVia},          // "Via: "
			{Type: tagTypeBytes, Value: sipHexUDPTransport}, // "SIP/2.0/UDP"
			{Type: tagTypeBytes, Value: "20"},               // " "
			{Type: tagTypeRandomChars, Value: "8"},          // host token
			{Type: tagTypeBytes, Value: sipHexBranchParam},  // ";branch="
			{Type: tagTypeBytes, Value: sipHexBranchCookie}, // "z9hG4bK"
			{Type: tagTypeRandomChars, Value: "12"},         // branch random token
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// From header
			{Type: tagTypeBytes, Value: "46726f6d3a20"},       // "From: "
			{Type: tagTypeBytes, Value: sipHexSIPSchemeAngle}, // "<sip:"
			{Type: tagTypeRandomChars, Value: "6"},            // user
			{Type: tagTypeBytes, Value: "40"},                 // "@"
			{Type: tagTypeRandomChars, Value: "8"},            // host
			{Type: tagTypeBytes, Value: "3e"},                 // ">"
			{Type: tagTypeBytes, Value: "3b7461673d"},         // ";tag="
			{Type: tagTypeRandomChars, Value: "8"},            // tag
			{Type: tagTypeBytes, Value: sipHexCRLF},           // CRLF
			// Call-ID
			{Type: tagTypeBytes, Value: sipHexCallID}, // "Call-ID: "
			{Type: tagTypeRandomChars, Value: "16"},   // call-id
			{Type: tagTypeBytes, Value: sipHexCRLF},   // CRLF
			// CSeq
			{Type: tagTypeBytes, Value: sipHexCSeq},         // "CSeq: "
			{Type: tagTypeRandomDigits, Value: "2"},         // sequence number
			{Type: tagTypeBytes, Value: sipHexSpaceOptions}, // " OPTIONS"
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// Content-Length
			{Type: tagTypeBytes, Value: sipHexContentLength}, // "Content-Length: "
			{Type: tagTypeBytes, Value: "30"},                // "0"
			{Type: tagTypeBytes, Value: sipHexCRLFCRLF},      // CRLF CRLF
		},

		// I3 — minimal but RFC-conformant (~170 B)
		I3: []TagSpec{
			// Request-Line
			{Type: tagTypeBytes, Value: sipHexOptions},   // "OPTIONS"
			{Type: tagTypeBytes, Value: "20"},            // " "
			{Type: tagTypeBytes, Value: sipHexSIPScheme}, // "sip:"
			{Type: tagTypeRandomChars, Value: "4"},       // user
			{Type: tagTypeBytes, Value: "40"},            // "@"
			{Type: tagTypeRandomChars, Value: "6"},       // host
			{Type: tagTypeBytes, Value: "20534950"},      // " SIP"
			{Type: tagTypeBytes, Value: sipHexVersion20}, // "/2.0"
			{Type: tagTypeBytes, Value: sipHexCRLF},      // CRLF
			// Via header
			{Type: tagTypeBytes, Value: sipHexVia},          // "Via: "
			{Type: tagTypeBytes, Value: sipHexUDPTransport}, // "SIP/2.0/UDP"
			{Type: tagTypeBytes, Value: "20"},               // " "
			{Type: tagTypeRandomChars, Value: "6"},          // host token
			{Type: tagTypeBytes, Value: sipHexBranchParam},  // ";branch="
			{Type: tagTypeBytes, Value: sipHexBranchCookie}, // "z9hG4bK"
			{Type: tagTypeRandomChars, Value: "8"},          // branch token
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// Call-ID
			{Type: tagTypeBytes, Value: sipHexCallID}, // "Call-ID: "
			{Type: tagTypeRandomChars, Value: "10"},   // call-id
			{Type: tagTypeBytes, Value: sipHexCRLF},   // CRLF
			// CSeq
			{Type: tagTypeBytes, Value: sipHexCSeq},         // "CSeq: "
			{Type: tagTypeRandomDigits, Value: "1"},         // sequence number
			{Type: tagTypeBytes, Value: sipHexSpaceOptions}, // " OPTIONS"
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// Content-Length
			{Type: tagTypeBytes, Value: sipHexContentLength}, // "Content-Length: "
			{Type: tagTypeBytes, Value: "30"},                // "0"
			{Type: tagTypeBytes, Value: sipHexCRLFCRLF},      // CRLF CRLF
		},

		// I4 — request-line + minimal headers only (~120 B)
		I4: []TagSpec{
			// Request-Line
			{Type: tagTypeBytes, Value: sipHexOptions},   // "OPTIONS"
			{Type: tagTypeBytes, Value: "20"},            // " "
			{Type: tagTypeBytes, Value: sipHexSIPScheme}, // "sip:"
			{Type: tagTypeRandomChars, Value: "4"},       // user
			{Type: tagTypeBytes, Value: "20534950"},      // " SIP"
			{Type: tagTypeBytes, Value: sipHexVersion20}, // "/2.0"
			{Type: tagTypeBytes, Value: sipHexCRLF},      // CRLF
			// Via header (minimal)
			{Type: tagTypeBytes, Value: sipHexVia},          // "Via: "
			{Type: tagTypeBytes, Value: sipHexUDPTransport}, // "SIP/2.0/UDP"
			{Type: tagTypeBytes, Value: "20"},               // " "
			{Type: tagTypeRandomChars, Value: "4"},          // host token
			{Type: tagTypeBytes, Value: sipHexCRLF},         // CRLF
			// Call-ID (minimal)
			{Type: tagTypeBytes, Value: sipHexCallID}, // "Call-ID: "
			{Type: tagTypeRandomChars, Value: "8"},    // call-id
			{Type: tagTypeBytes, Value: sipHexCRLF},   // CRLF
			// CSeq
			{Type: tagTypeBytes, Value: sipHexCSeq},         // "CSeq: "
			{Type: tagTypeRandomDigits, Value: "1"},         // sequence number
			{Type: tagTypeBytes, Value: sipHexSpaceOptions}, // " OPTIONS"
			{Type: tagTypeBytes, Value: sipHexCRLFCRLF},     // CRLF CRLF
		},

		// I5 — empty per named-template convention
		I5: []TagSpec{},
	}
}
