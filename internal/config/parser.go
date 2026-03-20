package config

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

func ParseServerConfig(r io.Reader) (ServerConfig, error) {
	var cfg ServerConfig
	var currentSection string
	var currentPeer PeerConfig

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#_") {
			// Skip empty lines and regular comments
			continue
		}

		// Section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentSection == "[Peer]" && currentPeer.PublicKey != "" {
				cfg.Peers = append(cfg.Peers, currentPeer)
				currentPeer = PeerConfig{}
			}
			currentSection = line
			continue
		}

		// Key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle commented fields (#_Name, #_PrivateKey, etc.)
		if strings.HasPrefix(key, "#_") {
			fieldName := strings.TrimPrefix(key, "#_")
			value = strings.Trim(value, `"'`)

			if currentSection == "[Peer]" {
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
			} else if currentSection == "[Interface]" {
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
		if currentSection == "[Interface]" {
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
				// TODO: Parse as HeaderRange (min,max) format
				if parts := strings.Split(value, ","); len(parts) == 2 {
					if min, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32); err1 == nil {
						if max, err2 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32); err2 == nil {
							cfg.Obfuscation.H1 = HeaderRange{Min: uint32(min), Max: uint32(max)}
						}
					}
				}
			case "H2":
				// TODO: Parse as HeaderRange (min,max) format
				if parts := strings.Split(value, ","); len(parts) == 2 {
					if min, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32); err1 == nil {
						if max, err2 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32); err2 == nil {
							cfg.Obfuscation.H2 = HeaderRange{Min: uint32(min), Max: uint32(max)}
						}
					}
				}
			case "H3":
				// TODO: Parse as HeaderRange (min,max) format
				if parts := strings.Split(value, ","); len(parts) == 2 {
					if min, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32); err1 == nil {
						if max, err2 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32); err2 == nil {
							cfg.Obfuscation.H3 = HeaderRange{Min: uint32(min), Max: uint32(max)}
						}
					}
				}
			case "H4":
				// TODO: Parse as HeaderRange (min,max) format
				if parts := strings.Split(value, ","); len(parts) == 2 {
					if min, err1 := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32); err1 == nil {
						if max, err2 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32); err2 == nil {
							cfg.Obfuscation.H4 = HeaderRange{Min: uint32(min), Max: uint32(max)}
						}
					}
				}
				// I1-I5 are client-only fields, should be in ParseClientConfig
				// case "I1":
				// 	cfg.Obfuscation.I1 = value
				// case "I2":
				// 	cfg.Obfuscation.I2 = value
				// case "I3":
				// 	cfg.Obfuscation.I3 = value
				// case "I4":
				// 	cfg.Obfuscation.I4 = value
				// case "I5":
				// 	cfg.Obfuscation.I5 = value
			}
		} else if currentSection == "[Peer]" {
			switch key {
			case "PublicKey":
				currentPeer.PublicKey = value
			case "PresharedKey":
				currentPeer.PresharedKey = value
			case "AllowedIPs":
				currentPeer.AllowedIPs = value
			}
		}
	}

	// Don't forget the last peer
	if currentSection == "[Peer]" && currentPeer.PublicKey != "" {
		cfg.Peers = append(cfg.Peers, currentPeer)
	}

	if err := scanner.Err(); err != nil {
		return ServerConfig{}, err
	}

	return cfg, nil
}
