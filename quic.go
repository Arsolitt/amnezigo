package amnezigo

// quicLongHeaderForm is the two-byte long-header prefix (0xc0 0xff) reused by
// every interval of the QUIC template: 0xc0 sets the header form bit, the
// fixed bit and the Initial packet type.
const quicLongHeaderForm = "c0ff"

// QUICTemplate returns an I1I5Template mimicking a QUIC Initial packet
// QUIC Long Header format:
// - Header Type (Long Header form)
// - Version Number
// - Destination Connection ID (DCID)
// - Source Connection ID (SCID)
// - Token Length
// - Packet Length
// - Packet Number
// - Payload.
func QUICTemplate() I1I5Template {
	return I1I5Template{
		// I1: Long header bytes + random DCID + timestamp + random payload
		I1: []TagSpec{
			{Type: tagTypeBytes, Value: quicLongHeaderForm}, // Long header form with type bits
			{Type: tagTypeBytes, Value: "00000001"},         // Version 1
			{Type: tagTypeBytes, Value: "08"},               // DCID length 8
			{Type: tagTypeRandom, Value: "8"},               // Random DCID (8 bytes)
			{Type: tagTypeBytes, Value: "00"},               // SCID length 0
			{Type: tagTypeBytes, Value: "00"},               // Token length 0
			{Type: tagTypeBytes, Value: "0040"},             // Length (approx 64 bytes)
			{Type: tagTypeBytes, Value: "00"},               // Packet number length
			{Type: tagTypeBytes, Value: "01"},               // Packet number
			{Type: tagTypeTimestamp, Value: ""},             // Timestamp
			{Type: tagTypeRandom, Value: "40"},              // Random payload (40 bytes)
		},

		// I2: Smaller variation - shorter payload. Reuses I1's DCID via <d> so the
		// flow looks like a continuation of the same QUIC session (real QUIC clients
		// keep the DCID stable across Initial → Handshake packets).
		I2: []TagSpec{
			{Type: tagTypeBytes, Value: quicLongHeaderForm}, // Long header form
			{Type: tagTypeBytes, Value: "00000001"},         // Version 1
			{Type: tagTypeBytes, Value: "08"},               // DCID length 8
			{Type: tagTypeData, Value: ""},                  // DCID reused from I1 (<d> passthrough)
			{Type: tagTypeBytes, Value: "00"},               // SCID length 0
			{Type: tagTypeBytes, Value: "00"},               // Token length 0
			{Type: tagTypeBytes, Value: "0020"},             // Length (approx 32 bytes)
			{Type: tagTypeBytes, Value: "01"},               // Packet number
			{Type: tagTypeTimestamp, Value: ""},             // Timestamp
			{Type: tagTypeRandom, Value: "20"},              // Shorter random payload (20 bytes)
		},

		// I3: Even smaller - minimal payload
		I3: []TagSpec{
			{Type: tagTypeBytes, Value: quicLongHeaderForm}, // Long header form
			{Type: tagTypeBytes, Value: "00000001"},         // Version 1
			{Type: tagTypeBytes, Value: "08"},               // DCID length 8
			{Type: tagTypeRandom, Value: "8"},               // Random DCID
			{Type: tagTypeBytes, Value: "00"},               // SCID length 0
			{Type: tagTypeBytes, Value: "00"},               // Token length 0
			{Type: tagTypeBytes, Value: "0010"},             // Length (approx 16 bytes)
			{Type: tagTypeBytes, Value: "01"},               // Packet number
			{Type: tagTypeTimestamp, Value: ""},             // Timestamp
			{Type: tagTypeRandom, Value: "10"},              // Minimal random payload (10 bytes)
		},

		// I4: Very small - just header + minimal data
		I4: []TagSpec{
			{Type: tagTypeBytes, Value: quicLongHeaderForm}, // Long header form
			{Type: tagTypeBytes, Value: "00000001"},         // Version 1
			{Type: tagTypeBytes, Value: "08"},               // DCID length 8
			{Type: tagTypeRandom, Value: "8"},               // Random DCID
			{Type: tagTypeBytes, Value: "00"},               // SCID length 0
			{Type: tagTypeBytes, Value: "00"},               // Token length 0
			{Type: tagTypeBytes, Value: "0005"},             // Length (approx 5 bytes)
			{Type: tagTypeBytes, Value: "01"},               // Packet number
			{Type: tagTypeTimestamp, Value: ""},             // Timestamp
			{Type: tagTypeRandom, Value: "5"},               // Tiny payload (5 bytes)
		},

		// I5: Empty
		I5: []TagSpec{},
	}
}
