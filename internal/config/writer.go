package config

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"time"
)

// randInRange returns a random number in the range [min, max]
func randInRange(min, max uint32) uint32 {
	if min >= max {
		return min
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return min + uint32(n.Uint64())
}

func WriteServerConfig(w io.Writer, cfg ServerConfig) error {
	fmt.Fprintln(w, "[Interface]")
	fmt.Fprintf(w, "PrivateKey = %s\n", cfg.Interface.PrivateKey)
	fmt.Fprintf(w, "Address = %s\n", cfg.Interface.Address)
	fmt.Fprintf(w, "ListenPort = %d\n", cfg.Interface.ListenPort)
	fmt.Fprintf(w, "MTU = %d\n", cfg.Interface.MTU)

	if cfg.Interface.PostUp != "" {
		fmt.Fprintf(w, "PostUp = %s\n", cfg.Interface.PostUp)
	}
	if cfg.Interface.PostDown != "" {
		fmt.Fprintf(w, "PostDown = %s\n", cfg.Interface.PostDown)
	}

	fmt.Fprintf(w, "Jc = %d\n", cfg.Obfuscation.Jc)
	fmt.Fprintf(w, "Jmin = %d\n", cfg.Obfuscation.Jmin)
	fmt.Fprintf(w, "Jmax = %d\n", cfg.Obfuscation.Jmax)
	fmt.Fprintf(w, "S1 = %d\n", cfg.Obfuscation.S1)
	fmt.Fprintf(w, "S2 = %d\n", cfg.Obfuscation.S2)
	fmt.Fprintf(w, "S3 = %d\n", cfg.Obfuscation.S3)
	fmt.Fprintf(w, "S4 = %d\n", cfg.Obfuscation.S4)
	fmt.Fprintf(w, "H1 = %d,%d\n", cfg.Obfuscation.H1.Min, cfg.Obfuscation.H1.Max)
	fmt.Fprintf(w, "H2 = %d,%d\n", cfg.Obfuscation.H2.Min, cfg.Obfuscation.H2.Max)
	fmt.Fprintf(w, "H3 = %d,%d\n", cfg.Obfuscation.H3.Min, cfg.Obfuscation.H3.Max)
	fmt.Fprintf(w, "H4 = %d,%d\n", cfg.Obfuscation.H4.Min, cfg.Obfuscation.H4.Max)
	// I1-I5 are client-only fields, not in ServerObfuscationConfig

	// Write metadata comments
	if cfg.Interface.EndpointV4 != "" {
		fmt.Fprintf(w, "#_EndpointV4 = %s\n", cfg.Interface.EndpointV4)
	}
	if cfg.Interface.EndpointV6 != "" {
		fmt.Fprintf(w, "#_EndpointV6 = %s\n", cfg.Interface.EndpointV6)
	}
	fmt.Fprintf(w, "#_ClientToClient = %v\n", cfg.Interface.ClientToClient)
	if cfg.Interface.TunName != "" {
		fmt.Fprintf(w, "#_TunName = %s\n", cfg.Interface.TunName)
	}
	if cfg.Interface.MainIface != "" {
		fmt.Fprintf(w, "#_MainIface = %s\n", cfg.Interface.MainIface)
	}

	for _, peer := range cfg.Peers {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "[Peer]")
		if peer.Name != "" {
			fmt.Fprintf(w, "#_Name = %s\n", peer.Name)
		}
		if peer.PrivateKey != "" {
			fmt.Fprintf(w, "#_PrivateKey = %s\n", peer.PrivateKey)
		}
		fmt.Fprintf(w, "PublicKey = %s\n", peer.PublicKey)
		if peer.PresharedKey != "" {
			fmt.Fprintf(w, "PresharedKey = %s\n", peer.PresharedKey)
		}
		fmt.Fprintf(w, "AllowedIPs = %s\n", peer.AllowedIPs)
		if !peer.CreatedAt.IsZero() {
			fmt.Fprintf(w, "#_GenKeyTime = %s\n", peer.CreatedAt.Format(time.RFC3339))
		}
	}

	return nil
}

func WriteClientConfig(w io.Writer, cfg ClientConfig) error {
	fmt.Fprintln(w, "[Interface]")
	fmt.Fprintf(w, "PrivateKey = %s\n", cfg.Interface.PrivateKey)
	fmt.Fprintf(w, "Address = %s\n", cfg.Interface.Address)
	fmt.Fprintf(w, "DNS = %s\n", cfg.Interface.DNS)
	fmt.Fprintf(w, "MTU = %d\n", cfg.Interface.MTU)

	fmt.Fprintf(w, "Jc = %d\n", cfg.Interface.Obfuscation.Jc)
	fmt.Fprintf(w, "Jmin = %d\n", cfg.Interface.Obfuscation.Jmin)
	fmt.Fprintf(w, "Jmax = %d\n", cfg.Interface.Obfuscation.Jmax)
	fmt.Fprintf(w, "S1 = %d\n", cfg.Interface.Obfuscation.S1)
	fmt.Fprintf(w, "S2 = %d\n", cfg.Interface.Obfuscation.S2)
	fmt.Fprintf(w, "S3 = %d\n", cfg.Interface.Obfuscation.S3)
	fmt.Fprintf(w, "S4 = %d\n", cfg.Interface.Obfuscation.S4)
	// Write H1-H4 as single values picked from range
	fmt.Fprintf(w, "H1 = %d\n", randInRange(cfg.Interface.Obfuscation.H1.Min, cfg.Interface.Obfuscation.H1.Max))
	fmt.Fprintf(w, "H2 = %d\n", randInRange(cfg.Interface.Obfuscation.H2.Min, cfg.Interface.Obfuscation.H2.Max))
	fmt.Fprintf(w, "H3 = %d\n", randInRange(cfg.Interface.Obfuscation.H3.Min, cfg.Interface.Obfuscation.H3.Max))
	fmt.Fprintf(w, "H4 = %d\n", randInRange(cfg.Interface.Obfuscation.H4.Min, cfg.Interface.Obfuscation.H4.Max))
	// Write I1-I5 in a loop
	iValues := []struct {
		name  string
		value string
	}{
		{"I1", cfg.Interface.Obfuscation.I1},
		{"I2", cfg.Interface.Obfuscation.I2},
		{"I3", cfg.Interface.Obfuscation.I3},
		{"I4", cfg.Interface.Obfuscation.I4},
		{"I5", cfg.Interface.Obfuscation.I5},
	}
	for _, iv := range iValues {
		if iv.value != "" {
			fmt.Fprintf(w, "%s = %s\n", iv.name, iv.value)
		}
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "[Peer]")
	fmt.Fprintf(w, "PublicKey = %s\n", cfg.Peer.PublicKey)
	fmt.Fprintf(w, "PresharedKey = %s\n", cfg.Peer.PresharedKey)
	fmt.Fprintf(w, "Endpoint = %s\n", cfg.Peer.Endpoint)
	fmt.Fprintf(w, "AllowedIPs = %s\n", cfg.Peer.AllowedIPs)
	fmt.Fprintf(w, "PersistentKeepalive = %d\n", cfg.Peer.PersistentKeepalive)

	return nil
}
