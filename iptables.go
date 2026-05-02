package amnezigo

import "strings"

// GeneratePostUp generates iptables rules to set up the VPN interface for routing.
// Rules are joined with "; " for execution in PostUp hook.
func GeneratePostUp(tunName, mainIface, subnet string, clientToClient bool) string {
	var rules []string

	// Accept traffic on tunnel interface
	rules = append(rules, "iptables -A INPUT -i "+tunName+" -j ACCEPT")
	rules = append(rules, "iptables -A OUTPUT -o "+tunName+" -j ACCEPT")

	// Forward traffic from tunnel to main interface (for internet access)
	rules = append(rules, "iptables -A FORWARD -i "+tunName+" -o "+mainIface+" -s "+subnet+" -j ACCEPT")

	// Allow established/related connections (for return traffic)
	rules = append(rules, "iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT")

	// Allow forwarding from main interface to tunnel for return traffic
	rules = append(
		rules,
		"iptables -A FORWARD -i "+mainIface+" -o "+tunName+" -d "+subnet+" -m state --state ESTABLISHED,RELATED -j ACCEPT",
	)

	// Allow client-to-client traffic if enabled
	if clientToClient {
		rules = append(rules, "iptables -A FORWARD -i "+tunName+" -o "+tunName+" -j ACCEPT")
	}

	// NAT/masquerade for internet access
	rules = append(rules, "iptables -t nat -A POSTROUTING -s "+subnet+" -o "+mainIface+" -j MASQUERADE")

	return strings.Join(rules, "; ")
}

// GeneratePostDown generates iptables rules to tear down the VPN interface routing.
// Rules are the same as GeneratePostUp but use -D (delete) instead of -A (append).
func GeneratePostDown(tunName, mainIface, subnet string, clientToClient bool) string {
	var rules []string

	// Accept traffic on tunnel interface
	rules = append(rules, "iptables -D INPUT -i "+tunName+" -j ACCEPT")
	rules = append(rules, "iptables -D OUTPUT -o "+tunName+" -j ACCEPT")

	// Forward traffic from tunnel to main interface
	rules = append(rules, "iptables -D FORWARD -i "+tunName+" -o "+mainIface+" -s "+subnet+" -j ACCEPT")

	// Allow established/related connections (for return traffic)
	rules = append(rules, "iptables -D FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT")

	// Allow forwarding from main interface to tunnel for return traffic
	rules = append(
		rules,
		"iptables -D FORWARD -i "+mainIface+" -o "+tunName+" -d "+subnet+" -m state --state ESTABLISHED,RELATED -j ACCEPT",
	)

	// Allow client-to-client traffic if enabled
	if clientToClient {
		rules = append(rules, "iptables -D FORWARD -i "+tunName+" -o "+tunName+" -j ACCEPT")
	}

	// NAT/masquerade for internet access
	rules = append(rules, "iptables -t nat -D POSTROUTING -s "+subnet+" -o "+mainIface+" -j MASQUERADE")

	return strings.Join(rules, "; ")
}

// GeneratePostUp6 generates ip6tables rules to set up IPv6 routing for the VPN interface.
// Rules are joined with "; " for execution in PostUp hook.
func GeneratePostUp6(tunName, mainIface, subnet string, clientToClient bool) string {
	var rules []string

	rules = append(rules, "ip6tables -A INPUT -i "+tunName+" -j ACCEPT")
	rules = append(rules, "ip6tables -A OUTPUT -o "+tunName+" -j ACCEPT")

	rules = append(rules, "ip6tables -A FORWARD -i "+tunName+" -o "+mainIface+" -s "+subnet+" -j ACCEPT")

	rules = append(rules, "ip6tables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT")

	rules = append(
		rules,
		"ip6tables -A FORWARD -i "+mainIface+" -o "+tunName+" -d "+subnet+" -m state --state ESTABLISHED,RELATED -j ACCEPT",
	)

	if clientToClient {
		rules = append(rules, "ip6tables -A FORWARD -i "+tunName+" -o "+tunName+" -j ACCEPT")
	}

	rules = append(rules, "ip6tables -t nat -A POSTROUTING -s "+subnet+" -o "+mainIface+" -j MASQUERADE")

	return strings.Join(rules, "; ")
}

// GeneratePostDown6 generates ip6tables rules to tear down IPv6 routing for the VPN interface.
// Rules use -D (delete) instead of -A (append).
func GeneratePostDown6(tunName, mainIface, subnet string, clientToClient bool) string {
	var rules []string

	rules = append(rules, "ip6tables -D INPUT -i "+tunName+" -j ACCEPT")
	rules = append(rules, "ip6tables -D OUTPUT -o "+tunName+" -j ACCEPT")

	rules = append(rules, "ip6tables -D FORWARD -i "+tunName+" -o "+mainIface+" -s "+subnet+" -j ACCEPT")

	rules = append(rules, "ip6tables -D FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT")

	rules = append(
		rules,
		"ip6tables -D FORWARD -i "+mainIface+" -o "+tunName+" -d "+subnet+" -m state --state ESTABLISHED,RELATED -j ACCEPT",
	)

	if clientToClient {
		rules = append(rules, "ip6tables -D FORWARD -i "+tunName+" -o "+tunName+" -j ACCEPT")
	}

	rules = append(rules, "ip6tables -t nat -D POSTROUTING -s "+subnet+" -o "+mainIface+" -j MASQUERADE")

	return strings.Join(rules, "; ")
}
