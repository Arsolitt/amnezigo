//nolint:cyclop // config parsing is inherently complex with many fields
package amnezigo

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	maxSplitParts    = 2
	sectionInterface = "[Interface]"
	sectionPeer      = "[Peer]"
)

// ParseOptions controls optional behavior of ParseServerConfigWithOptions.
type ParseOptions struct {
	// Strict, when true, causes the parser to collect (rather than silently
	// ignore) unknown INI keys, raw <c> tag literals, and other non-fatal
	// anomalies into the returned []ParseWarning. The structural validation
	// (H-range integrity) remains in error form.
	Strict bool
}

// ParseWarning is a non-fatal observation made during strict parsing.
type ParseWarning struct {
	Message string
	Key     string
	Code    string
	Line    int
}

// ParseServerConfig delegates to ParseServerConfigWithOptions with default
// options. Back-compat shim for callers that don't need warnings.
func ParseServerConfig(r io.Reader) (ServerConfig, error) {
	cfg, _, err := ParseServerConfigWithOptions(r, ParseOptions{})
	return cfg, err
}

// knownInterfaceKeys is the set of keys the parser recognises in [Interface].
var knownInterfaceKeys = map[string]bool{
	"PrivateKey": true, "PublicKey": true, "Address": true,
	"ListenPort": true, "MTU": true, "DNS": true,
	"PersistentKeepalive": true, "PostUp": true, "PostDown": true,
	"Jc": true, "Jmin": true, "Jmax": true,
	"S1": true, "S2": true, "S3": true, "S4": true,
	"H1": true, "H2": true, "H3": true, "H4": true,
}

// knownPeerKeys is the set of keys the parser recognises in [Peer].
var knownPeerKeys = map[string]bool{
	"PublicKey": true, "PresharedKey": true, "AllowedIPs": true,
}

// ParseServerConfigWithOptions is ParseServerConfig with optional behavior.
// In non-strict mode (default), the returned []ParseWarning is always nil.
//
//nolint:funlen,gocognit,gocyclo // parser with many config fields
func ParseServerConfigWithOptions(r io.Reader, opts ParseOptions) (ServerConfig, []ParseWarning, error) {
	var cfg ServerConfig
	var currentSection string
	var currentPeer PeerConfig
	var warnings []ParseWarning

	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#_") {
			// Check raw <c> even in comment lines in strict mode.
			if opts.Strict && strings.Contains(rawLine, "<c>") {
				warnings = append(warnings, ParseWarning{
					Code:    "CPS001",
					Line:    lineNo,
					Message: "raw <c> tag detected; rejected by amneziawg-go and AmneziaVPN clients",
				})
			}
			continue
		}

		// Check for raw <c> tag in any line (strict mode only).
		if opts.Strict && strings.Contains(rawLine, "<c>") {
			warnings = append(warnings, ParseWarning{
				Code:    "CPS001",
				Line:    lineNo,
				Message: "raw <c> tag detected; rejected by amneziawg-go and AmneziaVPN clients",
			})
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentSection == sectionPeer && currentPeer.PublicKey != "" {
				cfg.Peers = append(cfg.Peers, currentPeer)
				currentPeer = PeerConfig{}
			}
			currentSection = line
			continue
		}

		// Key-value pairs
		parts := strings.SplitN(line, "=", maxSplitParts)
		if len(parts) != maxSplitParts {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle commented fields (#_Name, #_PrivateKey, etc.)
		if after, ok := strings.CutPrefix(key, "#_"); ok {
			fieldName := after
			value = strings.Trim(value, `"'`)

			switch currentSection {
			case sectionPeer:
				switch fieldName {
				case "Name":
					currentPeer.Name = value
				case "PrivateKey":
					currentPeer.PrivateKey = value
				case "GenKeyTime":
					if t, err := time.Parse(time.RFC3339, value); err == nil {
						currentPeer.CreatedAt = t
					}
				}
			case sectionInterface:
				switch fieldName {
				case "EndpointV4":
					cfg.Interface.EndpointV4 = value
				case "EndpointV6":
					cfg.Interface.EndpointV6 = value
				case "ClientToClient":
					cfg.Interface.ClientToClient = value == "true"
				case "TunName":
					cfg.Interface.TunName = value
				case "MainIface":
					cfg.Interface.MainIface = value
				}
			}
			continue
		}

		// Regular fields
		matched := false
		switch currentSection {
		case sectionInterface:
			matched = knownInterfaceKeys[key]
			switch key {
			case "PrivateKey":
				cfg.Interface.PrivateKey = value
			case "PublicKey":
				cfg.Interface.PublicKey = value
			case "Address":
				cfg.Interface.Address = value
			case "ListenPort":
				if port, err := strconv.Atoi(value); err == nil {
					cfg.Interface.ListenPort = port
				}
			case "MTU":
				if mtu, err := strconv.Atoi(value); err == nil {
					cfg.Interface.MTU = mtu
				}
			case "DNS":
				cfg.Interface.DNS = value
			case "PersistentKeepalive":
				if ka, err := strconv.Atoi(value); err == nil {
					cfg.Interface.PersistentKeepalive = ka
				}
			case "PostUp":
				cfg.Interface.PostUp = value
			case "PostDown":
				cfg.Interface.PostDown = value
			case "Jc":
				if jc, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.Jc = jc
				}
			case "Jmin":
				if jmin, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.Jmin = jmin
				}
			case "Jmax":
				if jmax, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.Jmax = jmax
				}
			case "S1":
				if s, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.S1 = s
				}
			case "S2":
				if s, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.S2 = s
				}
			case "S3":
				if s, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.S3 = s
				}
			case "S4":
				if s, err := strconv.Atoi(value); err == nil {
					cfg.Obfuscation.S4 = s
				}
			case "H1":
				cfg.Obfuscation.H1 = parseHeaderRange(value)
			case "H2":
				cfg.Obfuscation.H2 = parseHeaderRange(value)
			case "H3":
				cfg.Obfuscation.H3 = parseHeaderRange(value)
			case "H4":
				cfg.Obfuscation.H4 = parseHeaderRange(value)
				// I1-I5 are client-only fields, should be in ParseClientConfig
			}
		case sectionPeer:
			matched = knownPeerKeys[key]
			switch key {
			case "PublicKey":
				currentPeer.PublicKey = value
			case "PresharedKey":
				currentPeer.PresharedKey = value
			case "AllowedIPs":
				currentPeer.AllowedIPs = value
			}
		}

		if opts.Strict && !matched {
			warnings = append(warnings, ParseWarning{
				Code:    "KEY001",
				Line:    lineNo,
				Key:     key,
				Message: fmt.Sprintf("unknown INI key %q in %s section", key, currentSection),
			})
		}
	}

	// Don't forget the last peer
	if currentSection == sectionPeer && currentPeer.PublicKey != "" {
		cfg.Peers = append(cfg.Peers, currentPeer)
	}

	if err := scanner.Err(); err != nil {
		return ServerConfig{}, warnings, err
	}

	// Validate H1-H4 ranges do not overlap WG message type-ids (1..4).
	// Such overlaps would let vanilla WireGuard packets be accepted by the
	// AWG-aware peer, defeating the obfuscation guarantee.
	for k, r := range []HeaderRange{
		cfg.Obfuscation.H1,
		cfg.Obfuscation.H2,
		cfg.Obfuscation.H3,
		cfg.Obfuscation.H4,
	} {
		if err := ValidateHeaderRange(r); err != nil {
			return ServerConfig{}, warnings, fmt.Errorf("invalid H%d: %w", k+1, err)
		}
	}

	return cfg, warnings, nil
}

func parseHeaderRange(value string) HeaderRange {
	parts := strings.Split(value, "-")
	if len(parts) != maxSplitParts {
		return HeaderRange{}
	}
	minVal, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
	maxVal, err2 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
	if err1 != nil || err2 != nil {
		return HeaderRange{}
	}
	return HeaderRange{Min: uint32(minVal), Max: uint32(maxVal)}
}
