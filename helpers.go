package amnezigo

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net"
	"slices"
	"strconv"
)

const (
	minPort   = 10000
	portRange = 55536
)

// IsValidIPAddr checks if the given string is a valid IP address in CIDR notation.
func IsValidIPAddr(ipaddr string) bool {
	ip, _, err := net.ParseCIDR(ipaddr)
	return err == nil && ip != nil
}

// ExtractSubnet extracts the subnet from an IP address in CIDR notation.
func ExtractSubnet(ipaddr string) string {
	_, ipnet, err := net.ParseCIDR(ipaddr)
	if err != nil {
		return ipaddr
	}
	ones, _ := ipnet.Mask.Size()
	return ipnet.IP.String() + "/" + strconv.Itoa(ones)
}

// GenerateRandomPort generates a random port number in the range [10000, 65535].
func GenerateRandomPort() (int, error) {
	maxPort := big.NewInt(portRange)
	n, err := rand.Int(rand.Reader, maxPort)
	if err != nil {
		return 0, err
	}
	return int(n.Int64()) + minPort, nil
}

// DetectMainInterface returns the name of the first non-loopback network
// interface that is up and has at least one address.
func DetectMainInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				return iface.Name
			}
		}
	}

	return ""
}

// FindNextAvailableIP finds the next available IP address in the subnet
// defined by serverAddress, skipping any IPs in the existingIPs list.
func FindNextAvailableIP(serverAddress string, existingIPs []string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(serverAddress)
	if err != nil {
		return "", err
	}

	existing := make(map[string]bool)
	for _, ipStr := range existingIPs {
		existing[ipStr] = true
	}

	for i := 2; i <= 254; i++ {
		ipBytes := ip.To4()
		if ipBytes == nil {
			return "", errors.New("not an IPv4 address")
		}

		ipBytes[3] = byte(i)
		candidateIP := ipBytes.String()

		if existing[candidateIP] {
			continue
		}

		if !ipnet.Contains(net.ParseIP(candidateIP)) {
			continue
		}

		return candidateIP, nil
	}

	return "", nil
}

// StringContains checks if a string slice contains a given string.
func StringContains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
