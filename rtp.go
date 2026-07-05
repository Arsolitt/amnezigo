package amnezigo

// RTPTemplate returns an I1I5Template mimicking RTP (Real-time Transport
// Protocol, RFC 3550) media packets.
//
// Wire-format reference: RFC 3550 § 5.1 (RTP fixed header). Verified against
// RFC 3550 (current as of 2026-07-05).
//
// RTP carries real-time audio/video over UDP and is the media layer of every
// VoIP and WebRTC session. It is one of the most common UDP traffic types on
// the internet — VoIP gateways, WebRTC browsers, and IP-PBX systems all emit
// it continuously. DPI almost always whitelists RTP because blocking it would
// break voice/video calls.
//
// The RTP fixed header (12 bytes):
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|V=2|P|X|  CC   |M|     PT      |       sequence number         |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                           timestamp                           |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|           synchronization source (SSRC) identifier            |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// The template uses:
//
//   - V=2 (the only deployed RTP version), P=0 (no padding), X=0 (no extension),
//     CC=0 (no CSRC)  → byte 0 = 0x80.
//   - M=0 (no marker), PT=0 (PCMU / G.711 μ-law — the most universally
//     recognised audio codec)  → byte 1 = 0x00.
//   - <r 2> for the 16-bit sequence number (fresh per packet).
//   - <t> for the 32-bit timestamp (the AWG <t> tag emits a 4-byte uint32 BE
//     Unix timestamp, which fits the 32-bit RTP timestamp field exactly).
//   - <r 4> for the 32-bit SSRC.
//   - <r N> for the media payload, with N varying per interval to simulate
//     different audio frame sizes.
//
// No <rc>/<rd> tags — RTP is a binary protocol, not text. This makes its
// byte-length distribution distinct from SIP (ASCII), DNS (<rc> labels), and
// STUN (<r> txn-ID). Each interval carries exactly one <t>, matching RTP's
// per-packet timestamp semantics.
//
// Byte budgets (template, before MTU clip):
//
//	I1: ~92 B (10 ms G.711 frame — 80 B payload)
//	I2: ~52 B (smaller frame — 40 B payload)
//	I3: ~36 B (comfort-noise frame — 24 B payload)
//	I4: ~20 B (minimal keepalive — 8 B payload)
//	I5: empty (named-template convention).
func RTPTemplate() I1I5Template {
	return I1I5Template{
		// I1 — full RTP audio packet, G.711 μ-law 10 ms frame (~92 B)
		I1: []TagSpec{
			{Type: "bytes", Value: "8000"}, // V=2, P=0, X=0, CC=0 | M=0, PT=0 (PCMU)
			{Type: "random", Value: "2"},   // sequence number (16-bit)
			{Type: "timestamp", Value: ""}, // timestamp (32-bit)
			{Type: "random", Value: "4"},   // SSRC (32-bit)
			{Type: "random", Value: "80"},  // payload (G.711 10 ms frame)
		},

		// I2 — smaller RTP frame (~52 B)
		I2: []TagSpec{
			{Type: "bytes", Value: "8000"},
			{Type: "random", Value: "2"},   // sequence number
			{Type: "timestamp", Value: ""}, // timestamp
			{Type: "random", Value: "4"},   // SSRC
			{Type: "random", Value: "40"},  // payload
		},

		// I3 — comfort-noise / SID frame (~36 B)
		I3: []TagSpec{
			{Type: "bytes", Value: "8000"},
			{Type: "random", Value: "2"},   // sequence number
			{Type: "timestamp", Value: ""}, // timestamp
			{Type: "random", Value: "4"},   // SSRC
			{Type: "random", Value: "24"},  // payload
		},

		// I4 — minimal RTP keepalive (~20 B)
		I4: []TagSpec{
			{Type: "bytes", Value: "8000"},
			{Type: "random", Value: "2"},   // sequence number
			{Type: "timestamp", Value: ""}, // timestamp
			{Type: "random", Value: "4"},   // SSRC
			{Type: "random", Value: "8"},   // payload
		},

		// I5 — empty per named-template convention
		I5: []TagSpec{},
	}
}
