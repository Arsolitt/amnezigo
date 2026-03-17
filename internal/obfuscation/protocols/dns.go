package protocols

// DNSTemplate returns an I1I5Template mimicking a DNS Query
// DNS Query format:
// - Transaction ID (2 bytes)
// - Flags (2 bytes): recursion desired, etc.
// - Questions count (2 bytes)
// - Answer RRs count (2 bytes)
// - Authority RRs count (2 bytes)
// - Additional RRs count (2 bytes)
// - Query name (variable length)
// - Query type (2 bytes)
// - Query class (2 bytes)
func DNSTemplate() I1I5Template {
	return I1I5Template{
		// I1: Full DNS query with transaction ID and query structure
		I1: []TagSpec{
			{Type: "random", Value: "2"},   // Transaction ID (2 random bytes)
			{Type: "bytes", Value: "0100"}, // Flags: standard query, recursion desired
			{Type: "bytes", Value: "0001"}, // Questions: 1
			{Type: "bytes", Value: "0000"}, // Answer RRs: 0
			{Type: "bytes", Value: "0000"}, // Authority RRs: 0
			{Type: "bytes", Value: "0000"}, // Additional RRs: 0
			{Type: "bytes", Value: "03"},   // First label length: 3
			{Type: "rc", Value: "3"},       // Random chars (3 bytes) - e.g., "www"
			{Type: "bytes", Value: "07"},   // Second label length: 7
			{Type: "rc", Value: "7"},       // Random chars (7 bytes) - e.g., "example"
			{Type: "bytes", Value: "03"},   // Third label length: 3
			{Type: "rc", Value: "3"},       // Random chars (3 bytes) - e.g., "com"
			{Type: "bytes", Value: "00"},   // Root label terminator
			{Type: "bytes", Value: "0001"}, // Query type: A (1)
			{Type: "bytes", Value: "0001"}, // Query class: IN (1)
		},

		// I2: Shorter DNS query with shorter domain
		I2: []TagSpec{
			{Type: "random", Value: "2"},   // Transaction ID
			{Type: "bytes", Value: "0100"}, // Flags
			{Type: "bytes", Value: "0001"}, // Questions: 1
			{Type: "bytes", Value: "0000"}, // Answer RRs: 0
			{Type: "bytes", Value: "0000"}, // Authority RRs: 0
			{Type: "bytes", Value: "0000"}, // Additional RRs: 0
			{Type: "bytes", Value: "04"},   // First label length: 4
			{Type: "rc", Value: "4"},       // Random chars
			{Type: "bytes", Value: "00"},   // Root label
			{Type: "bytes", Value: "0001"}, // Query type: A
			{Type: "bytes", Value: "0001"}, // Query class: IN
		},

		// I3: Minimal DNS query with random digits in domain
		I3: []TagSpec{
			{Type: "rd", Value: "2"},       // Transaction ID (2 random digits)
			{Type: "bytes", Value: "0100"}, // Flags
			{Type: "bytes", Value: "0001"}, // Questions: 1
			{Type: "bytes", Value: "0000"}, // Answer RRs: 0
			{Type: "bytes", Value: "0000"}, // Authority RRs: 0
			{Type: "bytes", Value: "0000"}, // Additional RRs: 0
			{Type: "bytes", Value: "02"},   // Label length: 2
			{Type: "rd", Value: "2"},       // Random digits
			{Type: "bytes", Value: "00"},   // Root label
			{Type: "bytes", Value: "0001"}, // Query type: A
			{Type: "bytes", Value: "0001"}, // Query class: IN
		},

		// I4: Tiny DNS query - shortest possible valid query
		I4: []TagSpec{
			{Type: "random", Value: "2"},   // Transaction ID
			{Type: "bytes", Value: "0100"}, // Flags
			{Type: "bytes", Value: "0001"}, // Questions: 1
			{Type: "bytes", Value: "0000"}, // Answer RRs: 0
			{Type: "bytes", Value: "0000"}, // Authority RRs: 0
			{Type: "bytes", Value: "0000"}, // Additional RRs: 0
			{Type: "bytes", Value: "01"},   // Label length: 1
			{Type: "rc", Value: "1"},       // Single random char
			{Type: "bytes", Value: "00"},   // Root label
			{Type: "bytes", Value: "0001"}, // Query type: A
			{Type: "bytes", Value: "0001"}, // Query class: IN
		},

		// I5: Empty
		I5: []TagSpec{},
	}
}
