//go:build e2e

package e2e

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	// serverTunnelIP/clientTunnelIP are the tunnel addresses of the server and
	// client peers in every scenario manifest.
	serverTunnelIP = "10.77.0.1"
	clientTunnelIP = "10.77.0.2"
	// serverAddr/clientAddr are the same addresses in CIDR form.
	serverAddr = serverTunnelIP + "/24"
	clientAddr = clientTunnelIP + "/32"
	// tunnelSubnet is routed over the tunnel on the client, whose /32 address
	// installs no connected route by itself.
	tunnelSubnet = "10.77.0.0/24"
	// vpnPort is the server listen port and the port the client sends to.
	vpnPort = 51820
	// vpnMTU is the tunnel MTU from the scenario manifests.
	vpnMTU = 1280
	// wgInitiationSize is the fixed WireGuard handshake-initiation size.
	wgInitiationSize = 148
	// udpIPOverhead is the IPv4 + UDP header size added to a UDP payload.
	udpIPOverhead = 28
)

// e2eManifest renders the scenario manifest around an obfuscation object.
func e2eManifest(obfuscation string) string {
	return fmt.Sprintf(`{
  "version": 1,
  "network": {"mtu": %d},
  "obfuscation": %s,
  "peers": {
    "server": {"address": %q, "endpoint": "203.0.113.1:%d", "listen_port": %d},
    "client": {"address": %q}
  }
}`, vpnMTU, obfuscation, serverAddr, vpnPort, vpnPort, clientAddr)
}

// startTunnel deploys conf on a fresh node pair and waits until the handshake
// completes on both sides.
func startTunnel(t *testing.T, network string, serverConf, clientConf []byte) (server, client node) {
	t.Helper()
	server = startNode(t, network, "server")
	client = startNode(t, network, "client")
	applyConf(t, server, serverConf, client.ip, serverAddr, vpnMTU)
	applyConf(t, client, clientConf, server.ip, clientAddr, vpnMTU, tunnelSubnet)

	ok := waitFor(func() bool {
		return latestHandshake(t, server) > 0 && latestHandshake(t, client) > 0
	}, 20*time.Second)
	if !ok {
		logs, _ := execIn(t, server, "cat "+daemonLogPath+" 2>&1")
		t.Fatalf("tunnel did not come up: server handshake %d, client handshake %d\nserver daemon log:\n%s",
			latestHandshake(t, server), latestHandshake(t, client), logs)
	}
	return server, client
}

// TestE2E_31_FullDefaults_TunnelUp generates a default AWG 3.1 config, applies
// it to two real nodes and verifies both the negotiated parameters and traffic.
func TestE2E_31_FullDefaults_TunnelUp(t *testing.T) {
	requireHarness(t)
	serverConf, clientConf := generate(t, e2eManifest(`{"awg_version": "3.1"}`))

	// Standard AWG 3.1 defaults, applied identically to both peers.
	defaults := map[string]string{
		"ContentPaddingAddition": "2-10",
		"RekeyAfterTime":         "120-180",
		"RekeyTimeout":           "5-8",
		"RejectAfterTime":        "180-240",
		"KeepaliveTimeout":       "8-12",
		"MaxHandshakeAttempts":   "16-20",
		"RandomTrailers":         "on",
		"DisableCookies":         "on",
	}
	for key, want := range defaults {
		for name, conf := range map[string][]byte{"server": serverConf, "client": clientConf} {
			if got := mustConfValue(t, conf, key); got != want {
				t.Errorf("generated %s config %s = %q, want %q", name, key, got, want)
			}
		}
	}
	// With header protection the reference uses the plain WireGuard type ids.
	headerDefaults := map[string]string{"H1": "1-1", "H2": "2-2", "H3": "3-3", "H4": "4-4"}
	for key, want := range headerDefaults {
		if got := mustConfValue(t, clientConf, key); got != want {
			t.Errorf("generated client config %s = %q, want %q", key, got, want)
		}
	}
	hpKey := mustConfValue(t, clientConf, "HeaderProtectionKey")
	if got := mustConfValue(t, serverConf, "HeaderProtectionKey"); got != hpKey {
		t.Fatalf("server header protection key %q differs from client key %q", got, hpKey)
	}
	rawKey, err := base64.StdEncoding.DecodeString(hpKey)
	if err != nil || len(hpKey) != 44 || len(rawKey) != 32 {
		t.Fatalf("HeaderProtectionKey %q is not 44-char base64 of 32 bytes (decoded %d bytes, err %v)",
			hpKey, len(rawKey), err)
	}
	for _, key := range []string{"S1", "S2", "S3", "S4"} {
		s, err := strconv.Atoi(mustConfValue(t, clientConf, key))
		if err != nil || s < 12 {
			t.Fatalf("generated client config %s = %q, want an integer >= 12", key, mustConfValue(t, clientConf, key))
		}
	}

	network := newNet(t)
	server, client := startTunnel(t, network, serverConf, clientConf)

	expect := map[string]string{
		"header_protection_key":    hpKey,
		"content_padding_addition": "2-10",
		"rekey_after_time":         "120-180",
		"rekey_timeout":            "5-8",
		"reject_after_time":        "180-240",
		"keepalive_timeout":        "8-12",
		"max_handshake_attempts":   "16-20",
		"random_trailers":          "on",
		"disable_cookies":          "on",
	}
	for i := 1; i <= 4; i++ {
		expect[fmt.Sprintf("s%d", i)] = mustConfValue(t, clientConf, fmt.Sprintf("S%d", i))
		expect[fmt.Sprintf("h%d", i)] = dumpRange(mustConfValue(t, clientConf, fmt.Sprintf("H%d", i)))
	}
	for name, n := range map[string]node{"server": server, "client": client} {
		dump := awgDump(t, n)
		for field, want := range expect {
			if got := dump[field]; got != want {
				t.Errorf("%s node dump %s = %q, want %q", name, field, got, want)
			}
		}
	}

	mustPing(t, client, serverTunnelIP)
	mustPing(t, server, clientTunnelIP)
	mustPing(t, client, serverTunnelIP, "-c", "1", "-W", "5", "-M", "do", "-s", "1200")
}

// TestE2E_HeaderProtection_Mismatch_NoHandshake proves the header protection
// key is actually used on the wire: a client with a different key can never
// complete a handshake, while the untouched config can.
func TestE2E_HeaderProtection_Mismatch_NoHandshake(t *testing.T) {
	requireHarness(t)
	serverConf, clientConf := generate(t, e2eManifest(`{"awg_version": "3.1"}`))
	brokenClient := replaceConfValue(t, clientConf, "HeaderProtectionKey", randomHPKey(t))

	network := newNet(t)
	server := startNode(t, network, "server")
	client := startNode(t, network, "client")
	applyConf(t, server, serverConf, client.ip, serverAddr, vpnMTU)
	applyConf(t, client, brokenClient, server.ip, clientAddr, vpnMTU, tunnelSubnet)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_ = ping(t, client, serverTunnelIP, "-c", "1", "-W", "1")
	}
	if hs := latestHandshake(t, client); hs != 0 {
		t.Fatalf("mismatched header protection key still completed a handshake on the client (timestamp %d)", hs)
	}
	if hs := latestHandshake(t, server); hs != 0 {
		t.Fatalf("mismatched header protection key still completed a handshake on the server (timestamp %d)", hs)
	}

	// Control: the untouched config on a fresh pair of nodes does handshake.
	controlServer, controlClient := startTunnel(t, newNet(t), serverConf, clientConf)
	mustPing(t, controlClient, serverTunnelIP)
	mustPing(t, controlServer, clientTunnelIP)
}

// TestE2E_HeaderProtection_EncryptsWireHeader inspects the on-wire handshake
// header: with header protection it must be ciphertext, without it a plaintext
// value from the H1 range.
func TestE2E_HeaderProtection_EncryptsWireHeader(t *testing.T) {
	requireHarness(t)
	cases := []struct {
		name            string
		obfuscation     string
		headerProtected bool
	}{
		{
			name:            "hp_on",
			obfuscation:     `{"awg_version": "3.1", "random_trailers": false}`,
			headerProtected: true,
		},
		{
			name:            "hp_off",
			obfuscation:     `{"awg_version": "3.1", "header_protection": false, "random_trailers": false}`,
			headerProtected: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			serverConf, clientConf := generate(t, e2eManifest(tc.obfuscation))
			s1, err := strconv.Atoi(mustConfValue(t, clientConf, "S1"))
			if err != nil {
				t.Fatalf("parse S1: %v", err)
			}
			h1Min, h1Max := parseHeaderRange(t, mustConfValue(t, clientConf, "H1"))

			network := newNet(t)
			server := startNode(t, network, "server")
			applyConf(t, server, serverConf, "", serverAddr, vpnMTU)

			payload := captureInitPacket(t, server, vpnPort, wgInitiationSize+s1+udpIPOverhead, func() {
				client := startNode(t, network, "client")
				applyConf(t, client, clientConf, server.ip, clientAddr, vpnMTU, tunnelSubnet)
				_ = ping(t, client, serverTunnelIP)
			})
			if want := s1 + wgInitiationSize; len(payload) != want {
				t.Fatalf("handshake initiation payload is %d bytes, want %d (S1=%d)", len(payload), want, s1)
			}
			wireType := binary.LittleEndian.Uint32(payload[s1 : s1+4])
			if tc.headerProtected {
				if wireType >= 1 && wireType <= 4 {
					t.Fatalf("header-protected wire type at offset S1=%d is %d, want outside 1..4", s1, wireType)
				}
				return
			}
			if wireType < h1Min || wireType > h1Max {
				t.Fatalf("unprotected wire type at offset S1=%d is %d, want within H1 range %d-%d",
					s1, wireType, h1Min, h1Max)
			}
		})
	}
}

// TestE2E_20Mode_No31Keys_TunnelUp verifies that 2.0 output carries none of
// the 3.x keys and still tunnels through the 3.1 engine.
func TestE2E_20Mode_No31Keys_TunnelUp(t *testing.T) {
	requireHarness(t)
	serverConf, clientConf := generate(t, e2eManifest(`{"awg_version": "2.0"}`))

	keys := []string{
		"HeaderProtectionKey", "ContentPaddingAddition", "RekeyAfterTime", "RekeyTimeout",
		"RejectAfterTime", "KeepaliveTimeout", "MaxHandshakeAttempts", "RandomTrailers", "DisableCookies",
	}
	for name, conf := range map[string][]byte{"server": serverConf, "client": clientConf} {
		for _, key := range keys {
			if strings.Contains(string(conf), key) {
				t.Errorf("generated %s 2.0 config unexpectedly contains %s:\n%s", name, key, conf)
			}
		}
	}

	network := newNet(t)
	server, client := startTunnel(t, network, serverConf, clientConf)
	for name, n := range map[string]node{"server": server, "client": client} {
		dump := awgDump(t, n)
		if got := dump["header_protection_key"]; got != "(none)" {
			t.Errorf("%s node dump header_protection_key = %q, want %q", name, got, "(none)")
		}
		if got := dump["random_trailers"]; got != "off" {
			t.Errorf("%s node dump random_trailers = %q, want %q", name, got, "off")
		}
		if got := dump["disable_cookies"]; got != "off" {
			t.Errorf("%s node dump disable_cookies = %q, want %q", name, got, "off")
		}
	}
	mustPing(t, client, serverTunnelIP)
	mustPing(t, server, clientTunnelIP)
}

// TestE2E_RekeyRange_Applied verifies an explicit rekey-after-time range is
// handed to the engine and causes a real rehandshake.
func TestE2E_RekeyRange_Applied(t *testing.T) {
	requireHarness(t)
	const obfuscation = `{"awg_version": "3.1", "rekey_after_time": {"min": 5, "max": 6}}`
	serverConf, clientConf := generate(t, e2eManifest(obfuscation))

	network := newNet(t)
	server, client := startTunnel(t, network, serverConf, clientConf)
	for name, n := range map[string]node{"server": server, "client": client} {
		if got := awgDump(t, n)["rekey_after_time"]; got != "5-6" {
			t.Fatalf("%s node dump rekey_after_time = %q, want %q", name, got, "5-6")
		}
	}

	first := latestHandshake(t, client)
	time.Sleep(7 * time.Second)
	mustPing(t, client, serverTunnelIP, "-c", "1", "-W", "5")
	rekeyed := waitFor(func() bool {
		return latestHandshake(t, client) > first
	}, 15*time.Second)
	if !rekeyed {
		t.Fatalf("client did not rekey within 15s: first handshake %d, latest %d",
			first, latestHandshake(t, client))
	}
}

// TestE2E_OverlappingHeaders_RejectedByEngine pins the engine's own rejection
// of overlapping H ranges that the validator's comments rely on.
func TestE2E_OverlappingHeaders_RejectedByEngine(t *testing.T) {
	requireHarness(t)
	const obfuscation = `{"awg_version": "3.1",
		"h1": {"min": 10, "max": 20},
		"h2": {"min": 30, "max": 40},
		"h3": {"min": 50, "max": 60},
		"h4": {"min": 70, "max": 80}}`
	serverConf, _ := generate(t, e2eManifest(obfuscation))

	network := newNet(t)
	server := startNode(t, network, "server")
	overlapping := replaceConfValue(t, serverConf, "H2", "10-40")
	writeFileInNode(t, server, confPathInNode, []byte(prepareConf(t, overlapping, "")))
	startAWG(t, server)
	waitUAPISocket(t, server)

	if err := setconf(t, server, confPathInNode); err == nil {
		t.Fatal("awg setconf accepted overlapping H ranges, want a failure")
	}
	if log := awgLog(t, server); !strings.Contains(log, "headers must not overlap") {
		t.Errorf("amneziawg-go daemon log does not mention the overlap rejection:\n%s", log)
	}

	// Control: the untouched, non-overlapping config is accepted.
	writeFileInNode(t, server, confPathInNode, []byte(prepareConf(t, serverConf, "")))
	if err := setconf(t, server, confPathInNode); err != nil {
		t.Fatalf("awg setconf rejected the non-overlapping config: %v", err)
	}
}

// randomHPKey returns a fresh 44-char base64 key (32 random bytes).
func randomHPKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate header protection key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

// replaceConfValue returns conf with the first line whose key matches replaced
// by "<key> = <value>". Scenario helper for the unique [Interface] keys.
func replaceConfValue(t *testing.T, conf []byte, key, value string) []byte {
	t.Helper()
	lines := strings.Split(string(conf), "\n")
	for i, raw := range lines {
		field, _, ok := strings.Cut(strings.TrimSpace(raw), "=")
		if !ok || strings.TrimSpace(field) != key {
			continue
		}
		lines[i] = key + " = " + value
		return []byte(strings.Join(lines, "\n"))
	}
	t.Fatalf("generated config has no %s line:\n%s", key, conf)
	return conf
}

// dumpRange converts a config range ("N-M") to the form `awg show dump`
// prints, which collapses equal bounds to a single number.
func dumpRange(value string) string {
	lo, hi, ok := strings.Cut(value, "-")
	if !ok || lo != hi {
		return value
	}
	return lo
}

// parseHeaderRange parses an "N-M" header range from a generated config.
func parseHeaderRange(t *testing.T, value string) (uint32, uint32) {
	t.Helper()
	lo, hi, ok := strings.Cut(value, "-")
	if !ok {
		t.Fatalf("header range %q is not in N-M form", value)
	}
	minVal, err := strconv.ParseUint(strings.TrimSpace(lo), 10, 32)
	if err != nil {
		t.Fatalf("parse header range %q: %v", value, err)
	}
	maxVal, err := strconv.ParseUint(strings.TrimSpace(hi), 10, 32)
	if err != nil {
		t.Fatalf("parse header range %q: %v", value, err)
	}
	return uint32(minVal), uint32(maxVal)
}
