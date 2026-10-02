package amnezigo

import (
	"strings"
	"testing"
	"time"
)

func TestParseServerConfig(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	// Verify Interface section
	if cfg.Interface.PrivateKey != "abc123" {
		t.Errorf("Expected PrivateKey 'abc123', got '%s'", cfg.Interface.PrivateKey)
	}
	if cfg.Interface.PublicKey != "xyz789" {
		t.Errorf("Expected PublicKey 'xyz789', got '%s'", cfg.Interface.PublicKey)
	}
	if cfg.Interface.Address != "10.8.0.1/24" {
		t.Errorf("Expected Address '10.8.0.1/24', got '%s'", cfg.Interface.Address)
	}
	if cfg.Interface.ListenPort != 55424 {
		t.Errorf("Expected ListenPort 55424, got %d", cfg.Interface.ListenPort)
	}

	// Verify Peer section
	if len(cfg.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].Name != "client1" {
		t.Errorf("Expected Peer Name 'client1', got '%s'", cfg.Peers[0].Name)
	}
	if cfg.Peers[0].PublicKey != "peerpub1" {
		t.Errorf("Expected Peer PublicKey 'peerpub1', got '%s'", cfg.Peers[0].PublicKey)
	}
	if cfg.Peers[0].AllowedIPs != "10.8.0.2/32" {
		t.Errorf("Expected Peer AllowedIPs '10.8.0.2/32', got '%s'", cfg.Peers[0].AllowedIPs)
	}
}

func TestParseObfuscationParams(t *testing.T) {
	input := `
[Interface]
Jc = 3
Jmin = 64
Jmax = 512
S1 = 1
S2 = 2
S3 = 3
S4 = 4
H1 = 1020325451-1020325451
H2 = 2020325452-2020325452
H3 = 3020325453-3020325453
H4 = 4020325454-4020325454
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	// Verify obfuscation params
	if cfg.Obfuscation.Jc != 3 {
		t.Errorf("Expected Jc 3, got %d", cfg.Obfuscation.Jc)
	}
	if cfg.Obfuscation.Jmin != 64 {
		t.Errorf("Expected Jmin 64, got %d", cfg.Obfuscation.Jmin)
	}
	if cfg.Obfuscation.Jmax != 512 {
		t.Errorf("Expected Jmax 512, got %d", cfg.Obfuscation.Jmax)
	}
	if cfg.Obfuscation.S1 != 1 {
		t.Errorf("Expected S1 1, got %d", cfg.Obfuscation.S1)
	}
	if cfg.Obfuscation.H1.Min != 1020325451 || cfg.Obfuscation.H1.Max != 1020325451 {
		t.Errorf("Expected H1 {1020325451,1020325451}, got {%d,%d}", cfg.Obfuscation.H1.Min, cfg.Obfuscation.H1.Max)
	}
	// I1-I5 are client-only fields, not in ServerObfuscationConfig
}

func TestParseMultiplePeers(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
#_Name = client1
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
#_GenKeyTime = 2024-01-15T10:30:00Z

[Peer]
#_Name = client2
PublicKey = peerpub2
AllowedIPs = 10.8.0.3/32
#_GenKeyTime = 2024-01-16T11:45:00Z
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	// Verify two peers
	if len(cfg.Peers) != 2 {
		t.Fatalf("Expected 2 peers, got %d", len(cfg.Peers))
	}

	// First peer
	if cfg.Peers[0].Name != "client1" {
		t.Errorf("Expected Peer[0] Name 'client1', got '%s'", cfg.Peers[0].Name)
	}
	expectedTime1, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	if !cfg.Peers[0].CreatedAt.Equal(expectedTime1) {
		t.Errorf("Expected Peer[0] CreatedAt %v, got %v", expectedTime1, cfg.Peers[0].CreatedAt)
	}

	// Second peer
	if cfg.Peers[1].Name != "client2" {
		t.Errorf("Expected Peer[1] Name 'client2', got '%s'", cfg.Peers[1].Name)
	}
	expectedTime2, _ := time.Parse(time.RFC3339, "2024-01-16T11:45:00Z")
	if !cfg.Peers[1].CreatedAt.Equal(expectedTime2) {
		t.Errorf("Expected Peer[1] CreatedAt %v, got %v", expectedTime2, cfg.Peers[1].CreatedAt)
	}
}

func TestParsePostScripts(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT
MTU = 1420

[Peer]
PublicKey = peerpub1
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	if cfg.Interface.PostUp != "iptables -A FORWARD -i wg0 -j ACCEPT" {
		t.Errorf("Expected PostUp 'iptables -A FORWARD -i wg0 -j ACCEPT', got '%s'", cfg.Interface.PostUp)
	}
	if cfg.Interface.PostDown != "iptables -D FORWARD -i wg0 -j ACCEPT" {
		t.Errorf("Expected PostDown 'iptables -D FORWARD -i wg0 -j ACCEPT', got '%s'", cfg.Interface.PostDown)
	}
	if cfg.Interface.MTU != 1420 {
		t.Errorf("Expected MTU 1420, got %d", cfg.Interface.MTU)
	}
}

func TestParsePeerPresharedKey(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
PublicKey = xyz789
Address = 10.8.0.1/24
ListenPort = 55424

[Peer]
PublicKey = peerpub1
PresharedKey = psk123
AllowedIPs = 10.8.0.2/32
`

	cfg, err := ParseServerConfig(strings.NewReader(input))

	if err != nil {
		t.Fatalf("ParseServerConfig failed: %v", err)
	}

	if len(cfg.Peers) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(cfg.Peers))
	}
	if cfg.Peers[0].PublicKey != "peerpub1" {
		t.Errorf("Expected Peer PublicKey 'peerpub1', got '%s'", cfg.Peers[0].PublicKey)
	}
	if cfg.Peers[0].PresharedKey != "psk123" {
		t.Errorf("Expected Peer PresharedKey 'psk123', got '%s'", cfg.Peers[0].PresharedKey)
	}
	if cfg.Peers[0].AllowedIPs != "10.8.0.2/32" {
		t.Errorf("Expected Peer AllowedIPs '10.8.0.2/32', got '%s'", cfg.Peers[0].AllowedIPs)
	}
}

// TestParseServerConfig_RejectsWGTypeIDInH1 verifies that a config whose H1
// range includes any of the standard WG message type-ids (1..4) is rejected.
// The error message must mention the offending field (H1).
func TestParseServerConfig_RejectsWGTypeIDInH1(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.0.0.1/24
ListenPort = 51820
H1 = 0-10
H2 = 1000000-2000000
H3 = 3000000-4000000
H4 = 5000000-6000000
`
	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for H1 containing WG type-id, got nil")
	}
	if !strings.Contains(err.Error(), "H1") {
		t.Errorf("error message should mention H1, got: %v", err)
	}
}

// TestParseServerConfig_RejectsWGTypeIDInH2 mirrors the H1 case for H2.
func TestParseServerConfig_RejectsWGTypeIDInH2(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.0.0.1/24
ListenPort = 51820
H1 = 1000000-2000000
H2 = 0-10
H3 = 3000000-4000000
H4 = 5000000-6000000
`
	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for H2 containing WG type-id, got nil")
	}
	if !strings.Contains(err.Error(), "H2") {
		t.Errorf("error message should mention H2, got: %v", err)
	}
}

// TestParseServerConfig_RejectsWGTypeIDInH3 mirrors the H1 case for H3.
func TestParseServerConfig_RejectsWGTypeIDInH3(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.0.0.1/24
ListenPort = 51820
H1 = 1000000-2000000
H2 = 3000000-4000000
H3 = 0-10
H4 = 5000000-6000000
`
	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for H3 containing WG type-id, got nil")
	}
	if !strings.Contains(err.Error(), "H3") {
		t.Errorf("error message should mention H3, got: %v", err)
	}
}

// TestParseServerConfig_RejectsWGTypeIDInH4 mirrors the H1 case for H4.
func TestParseServerConfig_RejectsWGTypeIDInH4(t *testing.T) {
	input := `
[Interface]
PrivateKey = abc123
Address = 10.0.0.1/24
ListenPort = 51820
H1 = 1000000-2000000
H2 = 3000000-4000000
H3 = 5000000-6000000
H4 = 0-10
`
	_, err := ParseServerConfig(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for H4 containing WG type-id, got nil")
	}
	if !strings.Contains(err.Error(), "H4") {
		t.Errorf("error message should mention H4, got: %v", err)
	}
}

func TestParseServerConfigWithOptions_StrictReportsUnknownKey(t *testing.T) {
	input := `[Interface]
PrivateKey = aaa
Address = 10.0.0.1/24
ListenPort = 51820
WeirdKey = nope
`
	cfg, warnings, err := ParseServerConfigWithOptions(
		strings.NewReader(input), ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict parse failed: %v", err)
	}
	if cfg.Interface.PrivateKey != "aaa" {
		t.Errorf("PrivateKey lost during strict parse")
	}
	var found bool
	for _, w := range warnings {
		if w.Key == "WeirdKey" {
			found = true
		}
	}
	if !found {
		t.Errorf("WeirdKey not reported as warning, got %+v", warnings)
	}
}

func TestParseServerConfigWithOptions_StrictReportsRawCounterTag(t *testing.T) {
	input := `[Interface]
PrivateKey = aaa
PostUp = ip link set up dev awg0 # CPS template <c> stays
`
	_, warnings, err := ParseServerConfigWithOptions(
		strings.NewReader(input), ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict parse failed: %v", err)
	}
	var found bool
	for _, w := range warnings {
		if w.Code == "CPS001" {
			found = true
		}
	}
	if !found {
		t.Errorf("raw <c> not reported, got %+v", warnings)
	}
}

func TestParseServerConfig_BackCompatLegacy(t *testing.T) {
	// Existing callers must keep their arity.
	_, err := ParseServerConfig(strings.NewReader("[Interface]\nPrivateKey = aaa\n"))
	if err != nil {
		t.Fatalf("legacy ParseServerConfig broke: %v", err)
	}
}

// TestParseServerConfigWithOptions_DefaultsAreSilent pins the contract for
// non-strict mode: zero warnings on every warnable condition.
func TestParseServerConfigWithOptions_DefaultsAreSilent(t *testing.T) {
	input := `[Interface]
PrivateKey = aaa
Address = 10.0.0.1/24
ListenPort = 51820
WeirdKey = nope
PostUp = echo '<c>' # legacy migration artifact
`
	_, warnings, err := ParseServerConfigWithOptions(
		strings.NewReader(input), ParseOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("default mode emitted %d warnings; expected zero: %+v", len(warnings), warnings)
	}
}

// TestParseServerConfigWithOptions_StrictUnknownKeyInPeer verifies unknown
// keys in the [Peer] section are also reported.
func TestParseServerConfigWithOptions_StrictUnknownKeyInPeer(t *testing.T) {
	input := `[Interface]
PrivateKey = aaa

[Peer]
PublicKey = bbb
AllowedIPs = 10.0.0.2/32
Bogus = something
`
	_, warnings, err := ParseServerConfigWithOptions(
		strings.NewReader(input), ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict parse failed: %v", err)
	}
	var found bool
	for _, w := range warnings {
		if w.Key == "Bogus" && w.Code == "KEY001" {
			found = true
		}
	}
	if !found {
		t.Errorf("Bogus key in [Peer] not reported, got %+v", warnings)
	}
}

func TestParseServerConfig_AWG31_RoundTrip(t *testing.T) {
	cases := []struct {
		name           string
		randomTrailers bool
		disableCookies bool
	}{
		{"bools_on", true, true},
		{"bools_off", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := ServerObfuscationConfig{
				Jc:                   3,
				Jmin:                 64,
				Jmax:                 512,
				S1:                   12,
				S2:                   13,
				S3:                   14,
				S4:                   15,
				H1:                   HeaderRange{Min: 1, Max: 1},
				H2:                   HeaderRange{Min: 2, Max: 2},
				H3:                   HeaderRange{Min: 3, Max: 3},
				H4:                   HeaderRange{Min: 4, Max: 4},
				Version:              AWG31,
				HeaderProtectionKey:  ioTestHPKey,
				ContentPadding:       U16Range{Min: 2, Max: 10},
				RekeyAfterTime:       U16Range{Min: 120, Max: 180},
				RekeyTimeout:         U16Range{Min: 5, Max: 8},
				RejectAfterTime:      U16Range{Min: 180, Max: 240},
				KeepaliveTimeout:     U16Range{Min: 8, Max: 12},
				MaxHandshakeAttempts: U16Range{Min: 16, Max: 20},
				RandomTrailers:       tc.randomTrailers,
				DisableCookies:       tc.disableCookies,
			}
			cfg := ServerConfig{
				Interface: InterfaceConfig{
					PrivateKey: "server_priv_key",
					Address:    "10.0.0.1/24",
					ListenPort: 51820,
					MTU:        1420,
				},
				Obfuscation: want,
			}

			var buf strings.Builder
			if err := WriteServerConfig(&buf, cfg); err != nil {
				t.Fatalf("WriteServerConfig failed: %v", err)
			}
			parsed, err := ParseServerConfig(strings.NewReader(buf.String()))
			if err != nil {
				t.Fatalf("ParseServerConfig failed: %v\nconfig:\n%s", err, buf.String())
			}

			got := parsed.Obfuscation
			if got.HeaderProtectionKey != want.HeaderProtectionKey {
				t.Errorf("HeaderProtectionKey = %q, want %q",
					got.HeaderProtectionKey, want.HeaderProtectionKey)
			}
			ranges := []struct {
				name string
				got  U16Range
				want U16Range
			}{
				{"ContentPadding", got.ContentPadding, want.ContentPadding},
				{"RekeyAfterTime", got.RekeyAfterTime, want.RekeyAfterTime},
				{"RekeyTimeout", got.RekeyTimeout, want.RekeyTimeout},
				{"RejectAfterTime", got.RejectAfterTime, want.RejectAfterTime},
				{"KeepaliveTimeout", got.KeepaliveTimeout, want.KeepaliveTimeout},
				{"MaxHandshakeAttempts", got.MaxHandshakeAttempts, want.MaxHandshakeAttempts},
			}
			for _, r := range ranges {
				if r.got != r.want {
					t.Errorf("%s = %+v, want %+v", r.name, r.got, r.want)
				}
			}
			if got.RandomTrailers != tc.randomTrailers {
				t.Errorf("RandomTrailers = %v, want %v", got.RandomTrailers, tc.randomTrailers)
			}
			if got.DisableCookies != tc.disableCookies {
				t.Errorf("DisableCookies = %v, want %v", got.DisableCookies, tc.disableCookies)
			}
		})
	}
}

func TestParseServerConfig_RejectsMalformedTransportKeys(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantErr []string
	}{
		{
			"header_protection_key_not_base64",
			"HeaderProtectionKey = not-base64!!",
			[]string{"invalid HeaderProtectionKey", "must be 44-char base64 of 32 bytes"},
		},
		{
			"header_protection_key_wrong_length",
			"HeaderProtectionKey = " + ioTestHPKeyShort,
			[]string{"invalid HeaderProtectionKey", "must be 44-char base64 of 32 bytes"},
		},
		{
			"content_padding_addition_junk",
			"ContentPaddingAddition = abc",
			[]string{`invalid ContentPaddingAddition "abc": expected "N" or "N-M"`},
		},
		{
			"rekey_timeout_max_below_min",
			"RekeyTimeout = 10-5",
			[]string{`invalid RekeyTimeout "10-5": max (5) is below min (10)`},
		},
		{
			"random_trailers_junk",
			"RandomTrailers = maybe",
			[]string{`invalid RandomTrailers "maybe": expected on/off/0/1`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := "[Interface]\nPrivateKey = aaa\n" + tc.line + "\n"
			_, err := ParseServerConfig(strings.NewReader(input))
			if err == nil {
				t.Fatalf("ParseServerConfig(%q) = nil error, want error containing %v", tc.line, tc.wantErr)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
			}
		})
	}
}

func TestParseServerConfig_TransportKeySpellings(t *testing.T) {
	t.Run("booleans", func(t *testing.T) {
		cases := []struct {
			key   string
			value string
			want  bool
		}{
			{"RandomTrailers", "on", true},
			{"RandomTrailers", "ON", true},
			{"RandomTrailers", "1", true},
			{"RandomTrailers", "off", false},
			{"RandomTrailers", "OFF", false},
			{"RandomTrailers", "0", false},
			{"DisableCookies", "On", true},
			{"DisableCookies", "0", false},
		}

		for _, tc := range cases {
			t.Run(tc.key+"_"+tc.value, func(t *testing.T) {
				input := "[Interface]\nPrivateKey = aaa\n" + tc.key + " = " + tc.value + "\n"
				cfg, err := ParseServerConfig(strings.NewReader(input))
				if err != nil {
					t.Fatalf("ParseServerConfig failed: %v", err)
				}
				var got bool
				switch tc.key {
				case "RandomTrailers":
					got = cfg.Obfuscation.RandomTrailers
				case "DisableCookies":
					got = cfg.Obfuscation.DisableCookies
				default:
					t.Fatalf("unhandled key %q in test table", tc.key)
				}
				if got != tc.want {
					t.Errorf("%s = %v for value %q, want %v", tc.key, got, tc.value, tc.want)
				}
			})
		}
	})

	t.Run("ranges", func(t *testing.T) {
		cases := []struct {
			value string
			want  U16Range
		}{
			{"120", U16Range{Min: 120, Max: 120}},
			{"5-8", U16Range{Min: 5, Max: 8}},
		}

		for _, tc := range cases {
			t.Run(tc.value, func(t *testing.T) {
				input := "[Interface]\nPrivateKey = aaa\nMaxHandshakeAttempts = " + tc.value + "\n"
				cfg, err := ParseServerConfig(strings.NewReader(input))
				if err != nil {
					t.Fatalf("ParseServerConfig failed: %v", err)
				}
				if got := cfg.Obfuscation.MaxHandshakeAttempts; got != tc.want {
					t.Errorf("MaxHandshakeAttempts = %+v, want %+v", got, tc.want)
				}
			})
		}
	})
}

func TestParseServerConfig_AcceptsWGTypeIDsUnderHeaderProtection(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			"key_before_headers",
			"[Interface]\nPrivateKey = aaa\n" +
				"HeaderProtectionKey = " + ioTestHPKey + "\n" +
				"H1 = 1-1\nH2 = 2-2\nH3 = 3-3\nH4 = 4-4\n",
		},
		{
			"key_after_headers",
			"[Interface]\nPrivateKey = aaa\n" +
				"H1 = 1-1\nH2 = 2-2\nH3 = 3-3\nH4 = 4-4\n" +
				"HeaderProtectionKey = " + ioTestHPKey + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseServerConfig(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("ParseServerConfig failed: %v", err)
			}
			if cfg.Obfuscation.HeaderProtectionKey != ioTestHPKey {
				t.Errorf("HeaderProtectionKey = %q, want %q",
					cfg.Obfuscation.HeaderProtectionKey, ioTestHPKey)
			}
			for i, got := range []HeaderRange{
				cfg.Obfuscation.H1,
				cfg.Obfuscation.H2,
				cfg.Obfuscation.H3,
				cfg.Obfuscation.H4,
			} {
				want := HeaderRange{Min: uint32(i + 1), Max: uint32(i + 1)}
				if got != want {
					t.Errorf("H%d = {%d,%d}, want {%d,%d}",
						i+1, got.Min, got.Max, want.Min, want.Max)
				}
			}
		})
	}
}
