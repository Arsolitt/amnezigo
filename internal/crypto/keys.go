// Package crypto provides cryptographic functions for AmneziaWG v2.0
package crypto

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/curve25519"
)

// GenerateKeyPair generates an X25519 key pair and returns base64 encoded strings
// Both keys are 44 characters (base64 encoded 32 bytes with padding)
func GenerateKeyPair() (privateKey, publicKey string) {
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		panic("crypto: failed to generate random key: " + err.Error())
	}

	// Apply WireGuard key clamping before scalar multiplication
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)

	return base64.StdEncoding.EncodeToString(priv[:]),
		base64.StdEncoding.EncodeToString(pub[:])
}

// DerivePublicKey derives a public key from a base64 encoded private key
// Returns a 44 character base64 encoded public key
func DerivePublicKey(privateKey string) string {
	priv, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		panic("crypto: invalid base64 private key: " + err.Error())
	}
	if len(priv) != 32 {
		panic("crypto: private key must be 32 bytes")
	}
	var privArr [32]byte
	copy(privArr[:], priv)

	// Apply WireGuard key clamping
	privArr[0] &= 248
	privArr[31] &= 127
	privArr[31] |= 64

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &privArr)

	return base64.StdEncoding.EncodeToString(pub[:])
}

// GeneratePSK generates a preshared key for additional encryption
// Returns a 44 character base64 encoded key
func GeneratePSK() string {
	var psk [32]byte
	if _, err := rand.Read(psk[:]); err != nil {
		panic("crypto: failed to generate random key: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(psk[:])
}
