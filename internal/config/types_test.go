package config

import (
	"testing"
	"time"
)

// TestConfigExists verifies the config package is properly structured
// This is a placeholder test that will be expanded with actual config tests
func TestConfigExists(t *testing.T) {
	// This test verifies the package compiles
	// More specific tests will be added as config features are implemented
	t.Log("Config package initialized successfully")
}

// TestPeerConfigPresharedKey verifies that PeerConfig has a PresharedKey field
func TestPeerConfigPresharedKey(t *testing.T) {
	peer := PeerConfig{
		Name:         "test-peer",
		PrivateKey:   "private-key-123",
		PublicKey:    "public-key-123",
		PresharedKey: "preshared-key-123",
		AllowedIPs:   "10.0.0.2/32",
		CreatedAt:    time.Now(),
	}

	if peer.PresharedKey != "preshared-key-123" {
		t.Errorf("Expected PresharedKey to be 'preshared-key-123', got '%s'", peer.PresharedKey)
	}
}
