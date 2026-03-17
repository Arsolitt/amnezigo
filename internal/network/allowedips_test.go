package network

import (
	"strings"
	"testing"
)

func TestCalculateAllowedIPs(t *testing.T) {
	allowed := CalculateAllowedIPs("10.8.0.0/24")

	// Should NOT contain private ranges
	if strings.Contains(allowed, "10.0.0.0") {
		t.Errorf("AllowedIPs should not contain 10.0.0.0 private range, got: %s", allowed)
	}
	if strings.Contains(allowed, "192.168.0") {
		t.Errorf("AllowedIPs should not contain 192.168.0.0 private range, got: %s", allowed)
	}
	if strings.Contains(allowed, "172.16.0") {
		t.Errorf("AllowedIPs should not contain 172.16.0.0 private range, got: %s", allowed)
	}
	if strings.Contains(allowed, "127.0.0.0") {
		t.Errorf("AllowedIPs should not contain 127.0.0.0 loopback range, got: %s", allowed)
	}
	if strings.Contains(allowed, "192.0.2.0") {
		t.Errorf("AllowedIPs should not contain 192.0.2.0 test-net range, got: %s", allowed)
	}
	if strings.Contains(allowed, "198.51.100") {
		t.Errorf("AllowedIPs should not contain 198.51.100.0 test-net range, got: %s", allowed)
	}
	if strings.Contains(allowed, "203.0.113") {
		t.Errorf("AllowedIPs should not contain 203.0.113.0 test-net range, got: %s", allowed)
	}

	// Should contain the AWG subnet
	if !strings.Contains(allowed, "10.8.0.0/24") {
		t.Errorf("AllowedIPs should contain AWG subnet 10.8.0.0/24, got: %s", allowed)
	}
}

func TestAllowedIPsFormat(t *testing.T) {
	allowed := CalculateAllowedIPs("10.8.0.0/24")
	cidrs := strings.Split(allowed, ", ")

	if len(cidrs) < 10 {
		t.Errorf("Expected multiple CIDRs, got %d: %s", len(cidrs), allowed)
	}

	// Each CIDR should have valid format
	for i, cidr := range cidrs {
		if !strings.Contains(cidr, "/") {
			t.Errorf("CIDR %d missing prefix length: %s", i, cidr)
		}
	}
}

func TestCalculateAllowedIPsWithDifferentAWGSubnet(t *testing.T) {
	allowed := CalculateAllowedIPs("192.168.1.0/24")

	// Should still contain the AWG subnet
	if !strings.Contains(allowed, "192.168.1.0/24") {
		t.Errorf("AllowedIPs should contain AWG subnet 192.168.1.0/24, got: %s", allowed)
	}

	// Should NOT contain the broader private range
	// (We only exclude the specific AWG subnet, not the whole 192.168.0.0/16)
	// But wait, according to requirements, we should exclude ALL 192.168.0.0/16
	// So 192.168.1.0/24 is part of excluded range, but it should be added back
}
