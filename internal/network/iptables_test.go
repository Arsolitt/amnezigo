package network

import (
	"strings"
	"testing"
)

func TestGeneratePostUp(t *testing.T) {
	rules := GeneratePostUp("awg0", "eth0", "10.8.0.0/24", false)

	// Verify all expected rules are present
	if !strings.Contains(rules, "iptables -A INPUT -i awg0 -j ACCEPT") {
		t.Errorf("Missing INPUT rule")
	}
	if !strings.Contains(rules, "iptables -A OUTPUT -o awg0 -j ACCEPT") {
		t.Errorf("Missing OUTPUT rule")
	}
	// Verify permissive FORWARD rule is NOT present (client-to-client should be blocked)
	if strings.Contains(rules, "iptables -A FORWARD -i awg0 -j ACCEPT") {
		t.Errorf("Permissive FORWARD rule should not be present - it allows client-to-client")
	}
	if !strings.Contains(rules, "iptables -A FORWARD -i awg0 -o eth0 -s 10.8.0.0/24 -j ACCEPT") {
		t.Errorf("Missing FORWARD tunnel to main interface rule")
	}
	if !strings.Contains(rules, "iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT") {
		t.Errorf("Missing ESTABLISHED,RELATED rule")
	}
	if !strings.Contains(rules, "iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE") {
		t.Errorf("Missing MASQUERADE rule")
	}

	// Verify clientToClient rule is NOT present when false
	if strings.Contains(rules, "-i awg0 -o awg0") {
		t.Errorf("Client-to-client rule should not be present when clientToClient is false")
	}

	// Verify rules are joined with "; "
	rulesArray := strings.Split(rules, "; ")
	if len(rulesArray) != 6 {
		t.Errorf("Expected 6 rules, got %d", len(rulesArray))
	}
}

func TestGeneratePostUpClientToClient(t *testing.T) {
	rules := GeneratePostUp("awg0", "eth0", "10.8.0.0/24", true)

	// Verify all basic rules are present
	if !strings.Contains(rules, "iptables -A INPUT -i awg0 -j ACCEPT") {
		t.Errorf("Missing INPUT rule")
	}

	// Verify client-to-client rule is present when true
	if !strings.Contains(rules, "iptables -A FORWARD -i awg0 -o awg0 -j ACCEPT") {
		t.Errorf("Missing client-to-client rule when clientToClient is true")
	}

	// Verify we have 7 rules when clientToClient is true
	rulesArray := strings.Split(rules, "; ")
	if len(rulesArray) != 7 {
		t.Errorf("Expected 7 rules with clientToClient, got %d", len(rulesArray))
	}
}

func TestGeneratePostDown(t *testing.T) {
	rules := GeneratePostDown("awg0", "eth0", "10.8.0.0/24", false)

	// Verify all rules use -D instead of -A
	if strings.Contains(rules, " -A ") {
		t.Errorf("PostDown should use -D instead of -A, found: %s", rules)
	}

	// Verify all expected rules are present with -D
	if !strings.Contains(rules, "iptables -D INPUT -i awg0 -j ACCEPT") {
		t.Errorf("Missing INPUT delete rule")
	}
	if !strings.Contains(rules, "iptables -D OUTPUT -o awg0 -j ACCEPT") {
		t.Errorf("Missing OUTPUT delete rule")
	}
	// Verify permissive FORWARD rule is NOT present
	if strings.Contains(rules, "iptables -D FORWARD -i awg0 -j ACCEPT") {
		t.Errorf("Permissive FORWARD delete rule should not be present")
	}
	if !strings.Contains(rules, "iptables -D FORWARD -i awg0 -o eth0 -s 10.8.0.0/24 -j ACCEPT") {
		t.Errorf("Missing FORWARD tunnel to main interface delete rule")
	}
	if !strings.Contains(rules, "iptables -D FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT") {
		t.Errorf("Missing ESTABLISHED,RELATED delete rule")
	}
	if !strings.Contains(rules, "iptables -t nat -D POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE") {
		t.Errorf("Missing MASQUERADE delete rule")
	}

	// Verify rules are joined with "; "
	rulesArray := strings.Split(rules, "; ")
	if len(rulesArray) != 6 {
		t.Errorf("Expected 6 rules, got %d", len(rulesArray))
	}
}

func TestGeneratePostDownClientToClient(t *testing.T) {
	rules := GeneratePostDown("awg0", "eth0", "10.8.0.0/24", true)

	// Verify client-to-client delete rule is present when true
	if !strings.Contains(rules, "iptables -D FORWARD -i awg0 -o awg0 -j ACCEPT") {
		t.Errorf("Missing client-to-client delete rule when clientToClient is true")
	}

	// Verify we have 7 rules when clientToClient is true
	rulesArray := strings.Split(rules, "; ")
	if len(rulesArray) != 7 {
		t.Errorf("Expected 7 rules with clientToClient, got %d", len(rulesArray))
	}
}
