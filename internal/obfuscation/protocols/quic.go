package protocols

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
			{Type: "bytes", Value: "c0ff"},     // Long header form with type bits
			{Type: "bytes", Value: "00000001"}, // Version 1
			{Type: "bytes", Value: "08"},       // DCID length 8
			{Type: "random", Value: "8"},       // Random DCID (8 bytes)
			{Type: "bytes", Value: "00"},       // SCID length 0
			{Type: "bytes", Value: "00"},       // Token length 0
			{Type: "bytes", Value: "0040"},     // Length (approx 64 bytes)
			{Type: "bytes", Value: "00"},       // Packet number length
			{Type: "bytes", Value: "01"},       // Packet number
			{Type: "timestamp", Value: ""},     // Timestamp
			{Type: "random", Value: "40"},      // Random payload (40 bytes)
		},

		// I2: Smaller variation - shorter payload
		I2: []TagSpec{
			{Type: "bytes", Value: "c0ff"},     // Long header form
			{Type: "bytes", Value: "00000001"}, // Version 1
			{Type: "bytes", Value: "08"},       // DCID length 8
			{Type: "random", Value: "8"},       // Random DCID
			{Type: "bytes", Value: "00"},       // SCID length 0
			{Type: "bytes", Value: "00"},       // Token length 0
			{Type: "bytes", Value: "0020"},     // Length (approx 32 bytes)
			{Type: "bytes", Value: "01"},       // Packet number
			{Type: "timestamp", Value: ""},     // Timestamp
			{Type: "random", Value: "20"},      // Shorter random payload (20 bytes)
		},

		// I3: Even smaller - minimal payload
		I3: []TagSpec{
			{Type: "bytes", Value: "c0ff"},     // Long header form
			{Type: "bytes", Value: "00000001"}, // Version 1
			{Type: "bytes", Value: "08"},       // DCID length 8
			{Type: "random", Value: "8"},       // Random DCID
			{Type: "bytes", Value: "00"},       // SCID length 0
			{Type: "bytes", Value: "00"},       // Token length 0
			{Type: "bytes", Value: "0010"},     // Length (approx 16 bytes)
			{Type: "bytes", Value: "01"},       // Packet number
			{Type: "timestamp", Value: ""},     // Timestamp
			{Type: "random", Value: "10"},      // Minimal random payload (10 bytes)
		},

		// I4: Very small - just header + minimal data
		I4: []TagSpec{
			{Type: "bytes", Value: "c0ff"},     // Long header form
			{Type: "bytes", Value: "00000001"}, // Version 1
			{Type: "bytes", Value: "08"},       // DCID length 8
			{Type: "random", Value: "8"},       // Random DCID
			{Type: "bytes", Value: "00"},       // SCID length 0
			{Type: "bytes", Value: "00"},       // Token length 0
			{Type: "bytes", Value: "0005"},     // Length (approx 5 bytes)
			{Type: "bytes", Value: "01"},       // Packet number
			{Type: "timestamp", Value: ""},     // Timestamp
			{Type: "random", Value: "5"},       // Tiny payload (5 bytes)
		},

		// I5: Empty
		I5: []TagSpec{},
	}
}
