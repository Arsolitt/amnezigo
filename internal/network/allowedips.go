package network

// CalculateAllowedIPs returns comma-separated CIDRs that are the complement of
// private IPv4 ranges plus the AWG subnet. This routes public internet through
// the VPN while excluding private IP ranges.
func CalculateAllowedIPs(awgSubnet string) string {
	// Public IPv4 ranges (complement of private ranges)
	publicRanges := []string{
		"1.0.0.0/8",
		"2.0.0.0/7",
		"4.0.0.0/6",
		"8.0.0.0/7",
		"11.0.0.0/8",
		"12.0.0.0/6",
		"16.0.0.0/4",
		"32.0.0.0/3",
		"64.0.0.0/2",
		"96.0.0.0/3",
		"104.0.0.0/5",
		"112.0.0.0/4",
		"128.0.0.0/3",
		"160.0.0.0/5",
		"168.0.0.0/6",
		"172.32.0.0/11",
		"172.64.0.0/10",
		"173.0.0.0/8",
		"174.0.0.0/7",
		"176.0.0.0/4",
		"192.0.1.0/24",
		"192.0.3.0/24",
		"192.0.4.0/22",
		"192.0.8.0/21",
		"192.0.16.0/20",
		"192.0.32.0/19",
		"192.0.64.0/18",
		"192.0.128.0/17",
		"192.1.0.0/16",
		"192.2.0.0/15",
		"192.4.0.0/14",
		"192.8.0.0/13",
		"192.16.0.0/12",
		"192.32.0.0/11",
		"192.64.0.0/10",
		"192.128.0.0/9",
		"193.0.0.0/8",
		"194.0.0.0/7",
		"196.0.0.0/6",
		"200.0.0.0/5",
		"208.0.0.0/4",
	}

	result := awgSubnet
	for _, r := range publicRanges {
		result += ", " + r
	}

	return result
}
