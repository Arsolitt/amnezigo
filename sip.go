package amnezigo

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
			{Type: "bytes", Value: "4f5054494f4e53"}, // "OPTIONS"
			{Type: "bytes", Value: "20"},             // " "
			{Type: "bytes", Value: "7369703a"},       // "sip:"
			{Type: "random_chars", Value: "8"},       // user (8 letters)
			{Type: "bytes", Value: "40"},             // "@"
			{Type: "random_chars", Value: "10"},      // host (10 letters)
			{Type: "bytes", Value: "2e"},             // "."
			{Type: "random_chars", Value: "3"},       // TLD (3 letters)
			{Type: "bytes", Value: "20534950"},       // " SIP"
			{Type: "bytes", Value: "2f322e30"},       // "/2.0"
			{Type: "bytes", Value: "0d0a"},           // CRLF
			// Via header
			{Type: "bytes", Value: "5669613a20"},             // "Via: "
			{Type: "bytes", Value: "5349502f322e302f554450"}, // "SIP/2.0/UDP"
			{Type: "bytes", Value: "20"},                     // " "
			{Type: "random_chars", Value: "10"},              // host token
			{Type: "bytes", Value: "3b6272616e63683d"},       // ";branch="
			{Type: "bytes", Value: "7a39684734624b"},         // "z9hG4bK" (RFC 3261 § 8.1.1.7 magic cookie)
			{Type: "random_chars", Value: "16"},              // branch random token
			{Type: "bytes", Value: "0d0a"},                   // CRLF
			// From header
			{Type: "bytes", Value: "46726f6d3a20"}, // "From: "
			{Type: "bytes", Value: "3c7369703a"},   // "<sip:"
			{Type: "random_chars", Value: "8"},     // user
			{Type: "bytes", Value: "40"},           // "@"
			{Type: "random_chars", Value: "10"},    // host
			{Type: "bytes", Value: "3e"},           // ">"
			{Type: "bytes", Value: "3b7461673d"},   // ";tag="
			{Type: "random_chars", Value: "12"},    // tag random token
			{Type: "bytes", Value: "0d0a"},         // CRLF
			// To header
			{Type: "bytes", Value: "546f3a20"},   // "To: "
			{Type: "bytes", Value: "3c7369703a"}, // "<sip:"
			{Type: "random_chars", Value: "8"},   // user
			{Type: "bytes", Value: "40"},         // "@"
			{Type: "random_chars", Value: "10"},  // host
			{Type: "bytes", Value: "3e"},         // ">"
			{Type: "bytes", Value: "0d0a"},       // CRLF
			// Call-ID
			{Type: "bytes", Value: "43616c6c2d49443a20"}, // "Call-ID: "
			{Type: "random_chars", Value: "20"},          // call-id random
			{Type: "bytes", Value: "40"},                 // "@"
			{Type: "random_chars", Value: "10"},          // host
			{Type: "bytes", Value: "0d0a"},               // CRLF
			// CSeq
			{Type: "bytes", Value: "435365713a20"},     // "CSeq: "
			{Type: "random_digits", Value: "3"},        // sequence number digits
			{Type: "bytes", Value: "204f5054494f4e53"}, // " OPTIONS"
			{Type: "bytes", Value: "0d0a"},             // CRLF
			// Max-Forwards
			{Type: "bytes", Value: "4d61782d466f7277617264733a20"}, // "Max-Forwards: "
			{Type: "bytes", Value: "3730"},                         // "70"
			{Type: "bytes", Value: "0d0a"},                         // CRLF
			// User-Agent
			{Type: "bytes", Value: "557365722d4167656e743a20"}, // "User-Agent: "
			{Type: "random_chars", Value: "12"},                // UA random token
			{Type: "bytes", Value: "0d0a"},                     // CRLF
			// Content-Length
			{Type: "bytes", Value: "436f6e74656e742d4c656e6774683a20"}, // "Content-Length: "
			{Type: "bytes", Value: "30"},                               // "0"
			{Type: "bytes", Value: "0d0a0d0a"},                         // CRLF CRLF (end-of-headers)
		},

		// I2 — drops User-Agent, shortens random tokens (~240 B)
		I2: []TagSpec{
			// Request-Line
			{Type: "bytes", Value: "4f5054494f4e53"}, // "OPTIONS"
			{Type: "bytes", Value: "20"},             // " "
			{Type: "bytes", Value: "7369703a"},       // "sip:"
			{Type: "random_chars", Value: "6"},       // user
			{Type: "bytes", Value: "40"},             // "@"
			{Type: "random_chars", Value: "8"},       // host
			{Type: "bytes", Value: "2e"},             // "."
			{Type: "random_chars", Value: "3"},       // TLD
			{Type: "bytes", Value: "20534950"},       // " SIP"
			{Type: "bytes", Value: "2f322e30"},       // "/2.0"
			{Type: "bytes", Value: "0d0a"},           // CRLF
			// Via header
			{Type: "bytes", Value: "5669613a20"},             // "Via: "
			{Type: "bytes", Value: "5349502f322e302f554450"}, // "SIP/2.0/UDP"
			{Type: "bytes", Value: "20"},                     // " "
			{Type: "random_chars", Value: "8"},               // host token
			{Type: "bytes", Value: "3b6272616e63683d"},       // ";branch="
			{Type: "bytes", Value: "7a39684734624b"},         // "z9hG4bK"
			{Type: "random_chars", Value: "12"},              // branch random token
			{Type: "bytes", Value: "0d0a"},                   // CRLF
			// From header
			{Type: "bytes", Value: "46726f6d3a20"}, // "From: "
			{Type: "bytes", Value: "3c7369703a"},   // "<sip:"
			{Type: "random_chars", Value: "6"},     // user
			{Type: "bytes", Value: "40"},           // "@"
			{Type: "random_chars", Value: "8"},     // host
			{Type: "bytes", Value: "3e"},           // ">"
			{Type: "bytes", Value: "3b7461673d"},   // ";tag="
			{Type: "random_chars", Value: "8"},     // tag
			{Type: "bytes", Value: "0d0a"},         // CRLF
			// Call-ID
			{Type: "bytes", Value: "43616c6c2d49443a20"}, // "Call-ID: "
			{Type: "random_chars", Value: "16"},          // call-id
			{Type: "bytes", Value: "0d0a"},               // CRLF
			// CSeq
			{Type: "bytes", Value: "435365713a20"},     // "CSeq: "
			{Type: "random_digits", Value: "2"},        // sequence number
			{Type: "bytes", Value: "204f5054494f4e53"}, // " OPTIONS"
			{Type: "bytes", Value: "0d0a"},             // CRLF
			// Content-Length
			{Type: "bytes", Value: "436f6e74656e742d4c656e6774683a20"}, // "Content-Length: "
			{Type: "bytes", Value: "30"},                               // "0"
			{Type: "bytes", Value: "0d0a0d0a"},                         // CRLF CRLF
		},

		// I3 — minimal but RFC-conformant (~170 B)
		I3: []TagSpec{
			// Request-Line
			{Type: "bytes", Value: "4f5054494f4e53"}, // "OPTIONS"
			{Type: "bytes", Value: "20"},             // " "
			{Type: "bytes", Value: "7369703a"},       // "sip:"
			{Type: "random_chars", Value: "4"},       // user
			{Type: "bytes", Value: "40"},             // "@"
			{Type: "random_chars", Value: "6"},       // host
			{Type: "bytes", Value: "20534950"},       // " SIP"
			{Type: "bytes", Value: "2f322e30"},       // "/2.0"
			{Type: "bytes", Value: "0d0a"},           // CRLF
			// Via header
			{Type: "bytes", Value: "5669613a20"},             // "Via: "
			{Type: "bytes", Value: "5349502f322e302f554450"}, // "SIP/2.0/UDP"
			{Type: "bytes", Value: "20"},                     // " "
			{Type: "random_chars", Value: "6"},               // host token
			{Type: "bytes", Value: "3b6272616e63683d"},       // ";branch="
			{Type: "bytes", Value: "7a39684734624b"},         // "z9hG4bK"
			{Type: "random_chars", Value: "8"},               // branch token
			{Type: "bytes", Value: "0d0a"},                   // CRLF
			// Call-ID
			{Type: "bytes", Value: "43616c6c2d49443a20"}, // "Call-ID: "
			{Type: "random_chars", Value: "10"},          // call-id
			{Type: "bytes", Value: "0d0a"},               // CRLF
			// CSeq
			{Type: "bytes", Value: "435365713a20"},     // "CSeq: "
			{Type: "random_digits", Value: "1"},        // sequence number
			{Type: "bytes", Value: "204f5054494f4e53"}, // " OPTIONS"
			{Type: "bytes", Value: "0d0a"},             // CRLF
			// Content-Length
			{Type: "bytes", Value: "436f6e74656e742d4c656e6774683a20"}, // "Content-Length: "
			{Type: "bytes", Value: "30"},                               // "0"
			{Type: "bytes", Value: "0d0a0d0a"},                         // CRLF CRLF
		},

		// I4 — request-line + minimal headers only (~120 B)
		I4: []TagSpec{
			// Request-Line
			{Type: "bytes", Value: "4f5054494f4e53"}, // "OPTIONS"
			{Type: "bytes", Value: "20"},             // " "
			{Type: "bytes", Value: "7369703a"},       // "sip:"
			{Type: "random_chars", Value: "4"},       // user
			{Type: "bytes", Value: "20534950"},       // " SIP"
			{Type: "bytes", Value: "2f322e30"},       // "/2.0"
			{Type: "bytes", Value: "0d0a"},           // CRLF
			// Via header (minimal)
			{Type: "bytes", Value: "5669613a20"},             // "Via: "
			{Type: "bytes", Value: "5349502f322e302f554450"}, // "SIP/2.0/UDP"
			{Type: "bytes", Value: "20"},                     // " "
			{Type: "random_chars", Value: "4"},               // host token
			{Type: "bytes", Value: "0d0a"},                   // CRLF
			// Call-ID (minimal)
			{Type: "bytes", Value: "43616c6c2d49443a20"}, // "Call-ID: "
			{Type: "random_chars", Value: "8"},           // call-id
			{Type: "bytes", Value: "0d0a"},               // CRLF
			// CSeq
			{Type: "bytes", Value: "435365713a20"},     // "CSeq: "
			{Type: "random_digits", Value: "1"},        // sequence number
			{Type: "bytes", Value: "204f5054494f4e53"}, // " OPTIONS"
			{Type: "bytes", Value: "0d0a0d0a"},         // CRLF CRLF
		},

		// I5 — empty per named-template convention
		I5: []TagSpec{},
	}
}
