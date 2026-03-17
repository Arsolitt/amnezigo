package crypto

import (
	"encoding/base64"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	priv, pub := GenerateKeyPair()

	// Verify private key length (44 chars for base64 encoded 32 bytes with padding)
	if len(priv) != 44 {
		t.Errorf("GenerateKeyPair() private key length = %d, want 44", len(priv))
	}

	// Verify public key length (44 chars for base64 encoded 32 bytes with padding)
	if len(pub) != 44 {
		t.Errorf("GenerateKeyPair() public key length = %d, want 44", len(pub))
	}

	// Verify it's valid base64 with proper padding
	_, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		t.Errorf("GenerateKeyPair() private key is not valid base64: %v", err)
	}
	_, err = base64.StdEncoding.DecodeString(pub)
	if err != nil {
		t.Errorf("GenerateKeyPair() public key is not valid base64: %v", err)
	}
}

func TestGeneratePSK(t *testing.T) {
	psk := GeneratePSK()

	// Verify PSK length (44 chars for base64 encoded 32 bytes with padding)
	if len(psk) != 44 {
		t.Errorf("GeneratePSK() length = %d, want 44", len(psk))
	}

	// Verify it's valid base64
	_, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Errorf("GeneratePSK() is not valid base64: %v", err)
	}
}

func TestKeyPairConsistency(t *testing.T) {
	priv1, pub1 := GenerateKeyPair()
	pub2 := DerivePublicKey(priv1)

	// Deriving public key from private should give the same result
	if pub1 != pub2 {
		t.Errorf("DerivePublicKey() = %s, want %s (from GenerateKeyPair)", pub2, pub1)
	}
}

func TestDerivePublicKey(t *testing.T) {
	priv, _ := GenerateKeyPair()
	pub := DerivePublicKey(priv)

	// Verify derived public key is properly formatted
	if len(pub) != 44 {
		t.Errorf("DerivePublicKey() length = %d, want 44", len(pub))
	}

	// Verify it's valid base64
	_, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		t.Errorf("DerivePublicKey() is not valid base64: %v", err)
	}
}

func TestGenerateKeyPairUniqueness(t *testing.T) {
	// Generate multiple key pairs to ensure they're unique
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		priv, _ := GenerateKeyPair()
		if keys[priv] {
			t.Errorf("GenerateKeyPair() produced duplicate private key")
		}
		keys[priv] = true
	}
}

func TestGeneratePSKUniqueness(t *testing.T) {
	// Generate multiple PSKs to ensure they're unique
	psks := make(map[string]bool)
	for i := 0; i < 100; i++ {
		psk := GeneratePSK()
		if psks[psk] {
			t.Errorf("GeneratePSK() produced duplicate PSK")
		}
		psks[psk] = true
	}
}
