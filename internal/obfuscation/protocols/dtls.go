package protocols

// DTLSTemplate returns an I1I5Template mimicking a DTLS 1.2 ClientHello
// DTLS Record Header format:
// - ContentType (1 byte): 22 = Handshake
// - Version (2 bytes): DTLS 1.2 = {0xfe, 0xfd}
// - Epoch (2 bytes): 0 for initial handshake
// - Sequence Number (6 bytes)
// - Length (2 bytes): length of handshake message
//
// DTLS Handshake Message format:
// - Handshake Type (1 byte): 1 = ClientHello
// - Length (3 bytes): message length
// - Message Sequence (2 bytes)
// - Fragment Offset (3 bytes)
// - Fragment Length (3 bytes)
// - Protocol Version (2 bytes)
// - Random (32 bytes): 4 bytes timestamp + 28 bytes random
// - Session ID (1 byte length + variable)
// - Cookie (1 byte length + variable)
// - Cipher Suites (2 bytes length + variable)
// - Compression Methods (1 byte length + variable)
// - Extensions (variable)
func DTLSTemplate() I1I5Template {
	return I1I5Template{
		// I1: Full DTLS 1.2 ClientHello with timestamp in random field
		I1: []TagSpec{
			// DTLS Record Header
			{Type: "bytes", Value: "16"},           // ContentType: Handshake (22)
			{Type: "bytes", Value: "fefd"},         // Version: DTLS 1.2
			{Type: "bytes", Value: "0000"},         // Epoch: 0 (initial handshake)
			{Type: "bytes", Value: "000000000000"}, // Sequence Number: 0
			{Type: "bytes", Value: "0058"},         // Length: ~88 bytes (example)

			// DTLS Handshake Message: ClientHello
			{Type: "bytes", Value: "01"},     // Handshake Type: ClientHello (1)
			{Type: "bytes", Value: "000054"}, // Length: ~84 bytes
			{Type: "bytes", Value: "0000"},   // Message Sequence: 0
			{Type: "bytes", Value: "000000"}, // Fragment Offset: 0
			{Type: "bytes", Value: "000054"}, // Fragment Length: 84 bytes

			// ClientHello body
			{Type: "bytes", Value: "fefd"}, // Protocol Version: DTLS 1.2
			{Type: "timestamp", Value: ""}, // Random: first 4 bytes are timestamp
			{Type: "random", Value: "28"},  // Random: remaining 28 bytes
			{Type: "bytes", Value: "00"},   // Session ID length: 0
			{Type: "bytes", Value: "00"},   // Cookie length: 0
			{Type: "bytes", Value: "0010"}, // Cipher Suites length: 16 bytes
			{Type: "bytes", Value: "cca8"}, // Cipher Suite 1: TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
			{Type: "bytes", Value: "cca9"}, // Cipher Suite 2: TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
			{Type: "bytes", Value: "cc02"}, // Cipher Suite 3: ECDHE_ECDSA_WITH_AES_256_CBC_SHA
			{Type: "bytes", Value: "cc01"}, // Cipher Suite 4: ECDHE_ECDSA_WITH_AES_128_CBC_SHA
			{Type: "bytes", Value: "01"},   // Compression Methods length: 1
			{Type: "bytes", Value: "00"},   // Compression Method: null
		},

		// I2: Shorter DTLS ClientHello
		I2: []TagSpec{
			// DTLS Record Header
			{Type: "bytes", Value: "16"},           // ContentType: Handshake
			{Type: "bytes", Value: "fefd"},         // Version: DTLS 1.2
			{Type: "bytes", Value: "0000"},         // Epoch: 0
			{Type: "bytes", Value: "000000000000"}, // Sequence Number: 0
			{Type: "bytes", Value: "0038"},         // Length: ~56 bytes

			// DTLS Handshake Message: ClientHello
			{Type: "bytes", Value: "01"},     // Handshake Type: ClientHello
			{Type: "bytes", Value: "000034"}, // Length: ~52 bytes
			{Type: "bytes", Value: "0000"},   // Message Sequence: 0
			{Type: "bytes", Value: "000000"}, // Fragment Offset: 0
			{Type: "bytes", Value: "000034"}, // Fragment Length: 52 bytes

			// ClientHello body
			{Type: "bytes", Value: "fefd"}, // Version: DTLS 1.2
			{Type: "timestamp", Value: ""}, // Random: timestamp
			{Type: "random", Value: "28"},  // Random bytes
			{Type: "bytes", Value: "00"},   // Session ID length: 0
			{Type: "bytes", Value: "00"},   // Cookie length: 0
			{Type: "bytes", Value: "0002"}, // Cipher Suites length: 2 bytes
			{Type: "bytes", Value: "cca8"}, // Cipher Suite: TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
			{Type: "bytes", Value: "01"},   // Compression Methods length: 1
			{Type: "bytes", Value: "00"},   // Compression Method: null
		},

		// I3: Minimal DTLS ClientHello
		I3: []TagSpec{
			// DTLS Record Header
			{Type: "bytes", Value: "16"},           // ContentType: Handshake
			{Type: "bytes", Value: "fefd"},         // Version: DTLS 1.2
			{Type: "bytes", Value: "0000"},         // Epoch: 0
			{Type: "bytes", Value: "000000000000"}, // Sequence Number: 0
			{Type: "bytes", Value: "0028"},         // Length: ~40 bytes

			// DTLS Handshake Message: ClientHello
			{Type: "bytes", Value: "01"},     // Handshake Type: ClientHello
			{Type: "bytes", Value: "000024"}, // Length: ~36 bytes
			{Type: "bytes", Value: "0000"},   // Message Sequence: 0
			{Type: "bytes", Value: "000000"}, // Fragment Offset: 0
			{Type: "bytes", Value: "000024"}, // Fragment Length: 36 bytes

			// ClientHello body
			{Type: "bytes", Value: "fefd"}, // Version: DTLS 1.2
			{Type: "timestamp", Value: ""}, // Random: timestamp
			{Type: "random", Value: "28"},  // Random bytes
			{Type: "bytes", Value: "00"},   // Session ID length: 0
			{Type: "bytes", Value: "00"},   // Cookie length: 0
			{Type: "bytes", Value: "0002"}, // Cipher Suites length: 2
			{Type: "bytes", Value: "c00c"}, // Cipher Suite: TLS_RSA_WITH_AES_128_CBC_SHA
			{Type: "bytes", Value: "01"},   // Compression Methods length: 1
			{Type: "bytes", Value: "00"},   // Compression Method: null
		},

		// I4: Tiny DTLS packet - minimum valid ClientHello
		I4: []TagSpec{
			// DTLS Record Header
			{Type: "bytes", Value: "16"},           // ContentType: Handshake
			{Type: "bytes", Value: "fefd"},         // Version: DTLS 1.2
			{Type: "bytes", Value: "0000"},         // Epoch: 0
			{Type: "bytes", Value: "000000000000"}, // Sequence Number: 0
			{Type: "bytes", Value: "0020"},         // Length: ~32 bytes

			// DTLS Handshake Message: ClientHello
			{Type: "bytes", Value: "01"},     // Handshake Type: ClientHello
			{Type: "bytes", Value: "00001c"}, // Length: ~28 bytes
			{Type: "bytes", Value: "0000"},   // Message Sequence: 0
			{Type: "bytes", Value: "000000"}, // Fragment Offset: 0
			{Type: "bytes", Value: "00001c"}, // Fragment Length: 28 bytes

			// ClientHello body
			{Type: "bytes", Value: "fefd"}, // Version: DTLS 1.2
			{Type: "timestamp", Value: ""}, // Random: timestamp
			{Type: "random", Value: "28"},  // Random bytes
			{Type: "bytes", Value: "00"},   // Session ID length: 0
			{Type: "bytes", Value: "00"},   // Cookie length: 0
			{Type: "bytes", Value: "0002"}, // Cipher Suites length: 2
			{Type: "bytes", Value: "0000"}, // Cipher Suite: SSL_NULL_WITH_NULL_NULL (for simplicity)
			{Type: "bytes", Value: "01"},   // Compression Methods length: 1
			{Type: "bytes", Value: "00"},   // Compression Method: null
		},

		// I5: Empty
		I5: []TagSpec{},
	}
}
