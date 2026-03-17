package protocols

// STUNTemplate returns an I1I5Template mimicking a STUN Binding Request
// STUN Message format (RFC 5389):
// - Message Type (2 bytes)
// - Message Length (2 bytes): length of attributes (not including header)
// - Magic Cookie (4 bytes): 0x2112A442
// - Transaction ID (12 bytes): random/unique
// - Attributes (variable): optional, include PADDING attribute if needed for padding to multiple of 4
//
// Message Types (Big Endian):
// - Binding Request: 0x0001
// - Binding Response: 0x0101
// - Binding Error Response: 0x0111
func STUNTemplate() I1I5Template {
	return I1I5Template{
		// I1: Full STUN Binding Request with standard header
		I1: []TagSpec{
			{Type: "bytes", Value: "0001"},     // Message Type: Binding Request (0x0001)
			{Type: "bytes", Value: "0000"},     // Message Length: 0 (no attributes)
			{Type: "bytes", Value: "2112a442"}, // Magic Cookie: 0x2112A442
			{Type: "random", Value: "12"},      // Transaction ID: 12 random bytes
		},

		// I2: STUN Binding Request with minimal padding
		I2: []TagSpec{
			{Type: "bytes", Value: "0001"},     // Message Type: Binding Request
			{Type: "bytes", Value: "0004"},     // Message Length: 4 bytes (PADDING attribute)
			{Type: "bytes", Value: "2112a442"}, // Magic Cookie
			{Type: "random", Value: "12"},      // Transaction ID
			// PADDING attribute (RFC 5389, 15.6)
			{Type: "bytes", Value: "0020"}, // Attribute Type: PADDING (0x0020)
			{Type: "bytes", Value: "0000"}, // Attribute Length: 0 bytes
		},

		// I3: STUN Binding Request with minimal header
		I3: []TagSpec{
			{Type: "bytes", Value: "0001"},     // Message Type: Binding Request
			{Type: "bytes", Value: "0000"},     // Message Length: 0
			{Type: "bytes", Value: "2112a442"}, // Magic Cookie
			{Type: "random", Value: "12"},      // Transaction ID
		},

		// I4: Minimal STUN packet - shortest possible valid Binding Request
		I4: []TagSpec{
			{Type: "bytes", Value: "0001"},     // Message Type: Binding Request
			{Type: "bytes", Value: "0000"},     // Message Length: 0
			{Type: "bytes", Value: "2112a442"}, // Magic Cookie
			{Type: "random", Value: "12"},      // Transaction ID
		},

		// I5: Empty
		I5: []TagSpec{},
	}
}
