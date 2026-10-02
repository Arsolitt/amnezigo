package amnezigo

// AWG 3.x device-level transport-protection INI keys.
//
// These strings are the on-wire spelling contract: they are exactly the names
// parsed by amneziawg-tools/src/config.c and dumped by awg-tools/src/show.c.
// The writer emits them, the parser recognises them, and validation findings
// reference them, so all three share one definition instead of repeating
// literals.
const (
	// keyHeaderProtection is the base64 32-byte ChaCha20 header-protection key.
	keyHeaderProtection = "HeaderProtectionKey"
	// keyContentPadding is the ContentPaddingAddition uint16 range.
	keyContentPadding = "ContentPaddingAddition"
	// keyRekeyAfterTime is the rekey-after-time uint16 range.
	keyRekeyAfterTime = "RekeyAfterTime"
	// keyRekeyTimeout is the rekey-timeout uint16 range.
	keyRekeyTimeout = "RekeyTimeout"
	// keyRejectAfterTime is the reject-after-time uint16 range.
	keyRejectAfterTime = "RejectAfterTime"
	// keyKeepaliveTimeout is the keepalive-timeout uint16 range.
	keyKeepaliveTimeout = "KeepaliveTimeout"
	// keyMaxHandshakeAttempts is the max-handshake-attempts uint16 range.
	keyMaxHandshakeAttempts = "MaxHandshakeAttempts"
	// keyRandomTrailers is the random-trailers boolean.
	keyRandomTrailers = "RandomTrailers"
	// keyDisableCookies is the disable-cookies boolean.
	keyDisableCookies = "DisableCookies"
)
