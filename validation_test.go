package amnezigo

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestValidatePacketSizes_AllDistinct asserts the happy path: distinct S-padded
// sizes, I-packet sizes that do not collide with any padded size, and a junk
// range that excludes both padded and raw WG message sizes.
func TestValidatePacketSizes_AllDistinct(t *testing.T) {
	err := ValidatePacketSizes(10, 20, 30, 40,
		[]int{200, 250, 300, 350, 400}, 500, 900)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestValidatePacketSizes_S1S2Collision verifies the Init/Response collision
// detection: S1+148 == S2+92 with s1=0, s2=56.
func TestValidatePacketSizes_S1S2Collision(t *testing.T) {
	err := ValidatePacketSizes(0, 56, 30, 40,
		[]int{200, 250, 300, 350, 400}, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) {
		t.Fatalf("expected *PacketSizeCollisionError, got %T", err)
	}
	if collErr.Kind != "s-pair" {
		t.Errorf("expected Kind=s-pair, got %q", collErr.Kind)
	}
	if collErr.Pair != "S1+148 vs S2+92" {
		t.Errorf("expected Pair=%q, got %q", "S1+148 vs S2+92", collErr.Pair)
	}
	if collErr.Size != 148 {
		t.Errorf("expected Size=148, got %d", collErr.Size)
	}
}

// TestValidatePacketSizes_S1S3Collision verifies Init vs Cookie collision:
// S1+148 == S3+64 with s1=0, s3=84.
func TestValidatePacketSizes_S1S3Collision(t *testing.T) {
	err := ValidatePacketSizes(0, 20, 84, 40,
		nil, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) {
		t.Fatalf("expected *PacketSizeCollisionError, got %T", err)
	}
	if collErr.Kind != "s-pair" {
		t.Errorf("expected Kind=s-pair, got %q", collErr.Kind)
	}
}

// TestValidatePacketSizes_S1S4Collision verifies Init vs Transport collision.
// s1=0, s4=116 yields 148 == 148. Note: s4 is outside the legal generator range
// [0,32], but ValidatePacketSizes is generic for the future `validate` CLI.
func TestValidatePacketSizes_S1S4Collision(t *testing.T) {
	err := ValidatePacketSizes(0, 20, 30, 116,
		nil, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "s-pair" {
		t.Errorf("expected s-pair collision, got %v", err)
	}
}

// TestValidatePacketSizes_S2S3Collision verifies Response vs Cookie collision.
func TestValidatePacketSizes_S2S3Collision(t *testing.T) {
	err := ValidatePacketSizes(10, 0, 28, 40,
		nil, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "s-pair" {
		t.Errorf("expected s-pair collision, got %v", err)
	}
}

// TestValidatePacketSizes_S2S4Collision verifies Response vs Transport collision.
func TestValidatePacketSizes_S2S4Collision(t *testing.T) {
	err := ValidatePacketSizes(10, 0, 30, 60,
		nil, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "s-pair" {
		t.Errorf("expected s-pair collision, got %v", err)
	}
}

// TestValidatePacketSizes_S3S4Collision verifies Cookie vs Transport collision.
func TestValidatePacketSizes_S3S4Collision(t *testing.T) {
	err := ValidatePacketSizes(10, 20, 0, 32,
		nil, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "s-pair" {
		t.Errorf("expected s-pair collision, got %v", err)
	}
}

// TestValidatePacketSizes_IPacketEqualsPadded_Init flags an I-packet whose
// length equals S1+148.
func TestValidatePacketSizes_IPacketEqualsPadded_Init(t *testing.T) {
	s1 := 4
	err := ValidatePacketSizes(s1, 20, 30, 10,
		[]int{200, s1 + 148, 300, 350, 400}, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) {
		t.Fatalf("expected *PacketSizeCollisionError, got %T", err)
	}
	if collErr.Kind != "i-packet" {
		t.Errorf("expected Kind=i-packet, got %q", collErr.Kind)
	}
}

// TestValidatePacketSizes_IPacketEqualsPadded_Response flags an I-packet
// matching S2+92.
func TestValidatePacketSizes_IPacketEqualsPadded_Response(t *testing.T) {
	s2 := 8
	err := ValidatePacketSizes(4, s2, 30, 10,
		[]int{200, 250, s2 + 92, 350, 400}, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "i-packet" {
		t.Errorf("expected i-packet collision, got %v", err)
	}
}

// TestValidatePacketSizes_IPacketEqualsPadded_Cookie flags S3+64.
func TestValidatePacketSizes_IPacketEqualsPadded_Cookie(t *testing.T) {
	s3 := 12
	err := ValidatePacketSizes(4, 8, s3, 10,
		[]int{200, 250, 300, s3 + 64, 400}, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "i-packet" {
		t.Errorf("expected i-packet collision, got %v", err)
	}
}

// TestValidatePacketSizes_IPacketEqualsPadded_Transport flags S4+32.
func TestValidatePacketSizes_IPacketEqualsPadded_Transport(t *testing.T) {
	s4 := 16
	err := ValidatePacketSizes(4, 8, 12, s4,
		[]int{200, 250, 300, 350, s4 + 32}, 500, 900)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "i-packet" {
		t.Errorf("expected i-packet collision, got %v", err)
	}
}

// TestValidatePacketSizes_JunkRangeIncludesPadded flags a junk range that
// straddles a padded size.
func TestValidatePacketSizes_JunkRangeIncludesPadded(t *testing.T) {
	s1 := 10
	padded := s1 + 148 // 158
	err := ValidatePacketSizes(s1, 20, 30, 5,
		nil, padded-5, padded+5)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) || collErr.Kind != "junk-range" {
		t.Errorf("expected junk-range collision, got %v", err)
	}
}

// TestValidatePacketSizes_JunkRangeIncludesRawWGSize flags a junk range that
// covers a raw WireGuard message size (148, 92, 64, or 32).
func TestValidatePacketSizes_JunkRangeIncludesRawWGSize(t *testing.T) {
	// Pick s-prefixes such that none of the padded sizes fall in [140, 160],
	// so the failure is unambiguously due to raw 148.
	err := ValidatePacketSizes(50, 50, 50, 0,
		nil, 140, 160)
	if err == nil {
		t.Fatal("expected collision error, got nil")
	}
	var collErr *PacketSizeCollisionError
	if !errors.As(err, &collErr) {
		t.Fatalf("expected *PacketSizeCollisionError, got %T", err)
	}
	if collErr.Kind != "junk-range" {
		t.Errorf("expected Kind=junk-range, got %q", collErr.Kind)
	}
}

// TestValidatePacketSizes_JunkRangeBoundaryExact verifies the boundary is
// inclusive: jmin equals a forbidden size.
func TestValidatePacketSizes_JunkRangeBoundaryExact(t *testing.T) {
	s1 := 4
	padded := s1 + 148
	err := ValidatePacketSizes(s1, 20, 30, 5,
		nil, padded, padded+10)
	if err == nil {
		t.Fatal("expected collision error at boundary, got nil")
	}
}

// TestValidatePacketSizes_JunkRangeBoundaryAdjacent verifies that a junk range
// starting just past a forbidden size does NOT collide.
func TestValidatePacketSizes_JunkRangeBoundaryAdjacent(t *testing.T) {
	s1 := 4
	padded := s1 + 148
	// jmin = padded+1 keeps the forbidden size out of [jmin, jmax]
	// (assuming no other forbidden size lands in the chosen range).
	err := ValidatePacketSizes(s1, 50, 50, 0,
		nil, padded+1, padded+10)
	if err != nil {
		t.Errorf("expected nil for adjacent boundary, got %v", err)
	}
}

// TestValidatePacketSizes_NilIPacketSlice verifies nil I-packet slices are
// treated as no-I-packets-to-check.
func TestValidatePacketSizes_NilIPacketSlice(t *testing.T) {
	err := ValidatePacketSizes(10, 20, 30, 40,
		nil, 500, 900)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestValidatePacketSizes_EmptyIPacketSlice verifies empty I-packet slices.
func TestValidatePacketSizes_EmptyIPacketSlice(t *testing.T) {
	err := ValidatePacketSizes(10, 20, 30, 40,
		[]int{}, 500, 900)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestValidatePacketSizes_DuplicateIPacketSizes_OK documents that duplicate
// I-packet sizes are allowed. The AWG receiver classifies handshake by length
// against handshake/transport sizes; multiple I-packets sharing a length is
// not a classification error — both are still I-packets.
func TestValidatePacketSizes_DuplicateIPacketSizes_OK(t *testing.T) {
	err := ValidatePacketSizes(10, 20, 30, 40,
		[]int{200, 200, 250, 300, 350}, 500, 900)
	if err != nil {
		t.Errorf("duplicate I-packet sizes must be allowed, got %v", err)
	}
}

// TestValidatePacketSizes_EmptyJunkRange returns the structural sentinel when
// jmin > jmax.
func TestValidatePacketSizes_EmptyJunkRange(t *testing.T) {
	err := ValidatePacketSizes(10, 20, 30, 40, nil, 900, 500)
	if !errors.Is(err, ErrEmptyJunkRange) {
		t.Errorf("expected ErrEmptyJunkRange, got %v", err)
	}
}

// TestValidatePacketSizes_DTagDoesNotMaskCollisions guards against a future
// regression where <d>'s zero-byte semantic is mistakenly used to "fix" a
// collision by zeroing the affected interval. ValidatePacketSizes operates on
// pre-computed byte sizes (post-calculateCPSLength); a <d>-only interval has
// size 0, which trivially does not collide with any S-padded handshake size
// (>= wgTransportSize=32). This test pins that <d>'s zero-ness does not
// silence real collisions in OTHER intervals of the same config.
func TestValidatePacketSizes_DTagDoesNotMaskCollisions(t *testing.T) {
	// Construct a config where I3 collides with S1+148 — but I2 is "<d>"
	// (size 0). The validator must still flag the I3 collision, not skip it.
	s1, s2, s3, s4 := 32, 64, 128, 200
	padded := s1 + WGInitiationSize // 180
	// I1=10, I2=0 (the <d>-only interval), I3=collision, I4=20, I5=30.
	iPacketSizes := []int{10, 0, padded, 20, 30}
	err := ValidatePacketSizes(s1, s2, s3, s4, iPacketSizes, 500, 900) // safe junk range
	if err == nil {
		t.Fatal("expected I3 collision, got nil")
	}
	var collisionErr *PacketSizeCollisionError
	if !errors.As(err, &collisionErr) {
		t.Fatalf("expected *PacketSizeCollisionError, got %T: %v", err, err)
	}
	if collisionErr.Kind != "i-packet" {
		t.Errorf("got Kind=%q, want %q", collisionErr.Kind, "i-packet")
	}
	if collisionErr.Size != padded {
		t.Errorf("got Size=%d, want %d", collisionErr.Size, padded)
	}
}

// TestSeverityValues pins the string representation of Severity constants.
func TestSeverityValues(t *testing.T) {
	cases := map[Severity]string{
		SeverityError:   "error",
		SeverityWarning: "warning",
		SeverityInfo:    "info",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("Severity %v = %q, want %q", got, string(got), want)
		}
	}
}

// TestFindingFormatsLine verifies the OneLine() text representation includes
// code and message.
func TestFindingFormatsLine(t *testing.T) {
	f := Finding{
		Severity: SeverityError,
		Code:     "PSC001",
		Location: Location{File: "/tmp/x.conf", Line: 0, Key: ""},
		Message:  "S1+148 vs S2+92",
	}
	line := f.OneLine()
	if !strings.Contains(line, "PSC001") || !strings.Contains(line, "S1+148") {
		t.Errorf("OneLine() = %q, missing code or message", line)
	}
}

// TestFinding_JSONShape pins the wire format. P1.4 (`analyze` command)
// shares these types — breaking the JSON keys is a cross-plan contract change.
func TestFinding_JSONShape(t *testing.T) {
	// Empty Location and Detail must not appear in the JSON output.
	f := Finding{
		Severity: SeverityError,
		Code:     "PSC001",
		Message:  "size collision",
	}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"severity":"error"`) {
		t.Errorf("expected lowercase severity key in %q", got)
	}
	if !strings.Contains(got, `"code":"PSC001"`) {
		t.Errorf("expected lowercase code key in %q", got)
	}
	if !strings.Contains(got, `"message":"size collision"`) {
		t.Errorf("expected lowercase message key in %q", got)
	}
	if strings.Contains(got, `"location"`) {
		t.Errorf("empty Location must be omitted via omitempty, got %q", got)
	}
	if strings.Contains(got, `"detail"`) {
		t.Errorf("empty Detail must be omitted via omitempty, got %q", got)
	}

	// Populated Location must serialize sub-fields with lowercase keys.
	f.Location = Location{File: "/tmp/x.conf", Line: 42, Key: "S1"}
	b, err = json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal with location: %v", err)
	}
	got = string(b)
	for _, want := range []string{`"file":"/tmp/x.conf"`, `"line":42`, `"key":"S1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %s in %q", want, got)
		}
	}
}

// freshServerConfig produces a known-good ServerConfig via the generator.
func freshServerConfig(t *testing.T) ServerConfig {
	t.Helper()
	obf := GenerateServerConfig(1280, 32, 5)
	return ServerConfig{
		Interface: InterfaceConfig{
			PrivateKey: "aaa", PublicKey: "bbb",
			Address: "10.0.0.1/24", ListenPort: 51820, MTU: 1280,
		},
		Obfuscation: obf,
	}
}

func containsCode(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

// valValidHeaderProtectionKey returns a valid 44-char base64
// header-protection key (32 zero bytes) as the engine decodes it.
func valValidHeaderProtectionKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, keyLength))
}

// valFindFinding returns the first finding with the given code, or nil when
// none matches.
func valFindFinding(findings []Finding, code string) *Finding {
	for i := range findings {
		if findings[i].Code == code {
			return &findings[i]
		}
	}
	return nil
}

func TestValidateServerConfig_CleanGeneratedConfig(t *testing.T) {
	cfg := freshServerConfig(t)
	findings := ValidateServerConfig(&cfg)
	var errs int
	for _, f := range findings {
		if f.Severity == SeverityError {
			errs++
		}
	}
	if errs != 0 {
		t.Errorf("freshly generated config produced %d errors: %+v", errs, findings)
	}
}

func TestValidateServerConfig_DetectsSPrefixCollision(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Obfuscation.S1 = 0
	cfg.Obfuscation.S2 = 56 // 0+148 == 56+92
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "PSC001") {
		t.Errorf("S-collision not detected: %+v", findings)
	}
}

func TestValidateServerConfig_DetectsHeaderTypeIDOverlap(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Obfuscation.H1 = HeaderRange{Min: 1, Max: 100}
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "HDR001") {
		t.Errorf("H1 overlap not detected: %+v", findings)
	}
}

func TestValidateServerConfig_DetectsHeaderStructuralInvalid(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Obfuscation.H2 = HeaderRange{Min: 100, Max: 50}
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "HDR002") {
		t.Errorf("H2 structural error not detected: %+v", findings)
	}
}

func TestValidateServerConfig_DetectsMissingPrivateKey(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Interface.PrivateKey = ""
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "FLD001") {
		t.Errorf("missing PrivateKey not detected: %+v", findings)
	}
}

func TestValidateServerConfig_DetectsMissingAddress(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Interface.Address = ""
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "FLD001") {
		t.Errorf("missing Address not detected: %+v", findings)
	}
}

func TestValidateServerConfig_DetectsMissingListenPort(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Interface.ListenPort = 0
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "FLD001") {
		t.Errorf("missing ListenPort not detected: %+v", findings)
	}
}

func TestValidateServerConfig_DetectsJunkRangeStructural(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Obfuscation.Jmin = 200
	cfg.Obfuscation.Jmax = 100 // jmin > jmax
	findings := ValidateServerConfig(&cfg)
	if !containsCode(findings, "JNK001") {
		t.Errorf("junk range structural error not detected: %+v", findings)
	}
}

// TestValidateServerConfig_HPK001 verifies the header-protection S-prefix
// floor: a key with any S below the 12-byte ChaCha20 nonce is an error, while
// S values >= 12 pass.
func TestValidateServerConfig_HPK001(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Obfuscation.HeaderProtectionKey = valValidHeaderProtectionKey()
	cfg.Obfuscation.S3 = 8
	findings := ValidateServerConfig(&cfg)
	f := valFindFinding(findings, "HPK001")
	if f == nil {
		t.Fatalf("HPK001 not found: %+v", findings)
	}
	if f.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q", f.Severity, SeverityError)
	}
	if !strings.Contains(f.Message, "header protection requires S1-S4 >= 12") {
		t.Errorf("message %q does not mention the S floor", f.Message)
	}
	if !strings.Contains(f.Message, "S3=8") {
		t.Errorf("message %q does not identify S3", f.Message)
	}

	ok := freshServerConfig(t)
	ok.Obfuscation.HeaderProtectionKey = valValidHeaderProtectionKey()
	ok.Obfuscation.S1, ok.Obfuscation.S2 = 12, 13
	ok.Obfuscation.S3, ok.Obfuscation.S4 = 12, 14
	okFindings := ValidateServerConfig(&ok)
	if containsCode(okFindings, "HPK001") {
		t.Errorf("HPK001 fired for S1-S4 all >= 12: %+v", okFindings)
	}
}

// TestValidateServerConfig_HPK003 verifies the key shape check: any non-empty
// value that is not 44-char base64 of 32 bytes is an error.
func TestValidateServerConfig_HPK003(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"malformed_base64", "not-valid-base64!!", true},
		{"wrong_length", base64.StdEncoding.EncodeToString(make([]byte, keyLength-1)), true},
		{"valid_key", valValidHeaderProtectionKey(), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := freshServerConfig(t)
			cfg.Obfuscation.HeaderProtectionKey = tc.key
			findings := ValidateServerConfig(&cfg)
			f := valFindFinding(findings, "HPK003")
			if !tc.want {
				if f != nil {
					t.Errorf("unexpected HPK003: %+v", f)
				}
				return
			}
			if f == nil {
				t.Fatalf("HPK003 not found: %+v", findings)
			}
			if f.Severity != SeverityError {
				t.Errorf("Severity = %q, want %q", f.Severity, SeverityError)
			}
			wantMsg := "HeaderProtectionKey is not 44-char base64 of 32 bytes"
			if !strings.Contains(f.Message, wantMsg) {
				t.Errorf("message %q does not contain %q", f.Message, wantMsg)
			}
		})
	}
}

// TestValidateServerConfig_TRL001 verifies the random-trailers warning fires
// only when trailers are on and S1..S4 differ.
func TestValidateServerConfig_TRL001(t *testing.T) {
	tests := []struct {
		name           string
		randomTrailers bool
		s1, s2, s3, s4 int
		want           bool
	}{
		{"trailers_on_unequal_s", true, 30, 35, 20, 12, true},
		{"trailers_on_equal_s", true, 30, 30, 30, 30, false},
		{"trailers_off_unequal_s", false, 30, 35, 20, 12, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := freshServerConfig(t)
			cfg.Obfuscation.RandomTrailers = tc.randomTrailers
			cfg.Obfuscation.S1, cfg.Obfuscation.S2 = tc.s1, tc.s2
			cfg.Obfuscation.S3, cfg.Obfuscation.S4 = tc.s3, tc.s4
			findings := ValidateServerConfig(&cfg)
			f := valFindFinding(findings, "TRL001")
			if !tc.want {
				if f != nil {
					t.Errorf("unexpected TRL001: %+v", f)
				}
				return
			}
			if f == nil {
				t.Fatalf("TRL001 not found: %+v", findings)
			}
			if f.Severity != SeverityWarning {
				t.Errorf("Severity = %q, want %q", f.Severity, SeverityWarning)
			}
			wantMsg := "RandomTrailers is enabled while S1..S4 differ"
			if !strings.Contains(f.Message, wantMsg) {
				t.Errorf("message %q does not contain %q", f.Message, wantMsg)
			}
		})
	}
}

// TestValidateServerConfig_TRM001 verifies structural checks on the six AWG 3.x
// ranges: Max < Min and mixed zero bounds are errors, all-zero means disabled.
func TestValidateServerConfig_TRM001(t *testing.T) {
	tests := []struct {
		name    string
		set     func(o *ServerObfuscationConfig)
		wantKey string
		wantMsg string
	}{
		{
			name:    "max_below_min",
			set:     func(o *ServerObfuscationConfig) { o.RekeyTimeout = U16Range{Min: 10, Max: 5} },
			wantKey: "RekeyTimeout",
			wantMsg: "max (5) is below min (10)",
		},
		{
			name:    "mixed_zero_bounds",
			set:     func(o *ServerObfuscationConfig) { o.ContentPadding = U16Range{Min: 0, Max: 5} },
			wantKey: "ContentPaddingAddition",
			wantMsg: "bounds must both be zero or both non-zero (got 0-5)",
		},
		{
			name: "all_zero_bounds",
			set: func(o *ServerObfuscationConfig) {
				o.ContentPadding = U16Range{}
				o.RekeyAfterTime = U16Range{}
				o.RekeyTimeout = U16Range{}
				o.RejectAfterTime = U16Range{}
				o.KeepaliveTimeout = U16Range{}
				o.MaxHandshakeAttempts = U16Range{}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := freshServerConfig(t)
			tc.set(&cfg.Obfuscation)
			findings := ValidateServerConfig(&cfg)
			if tc.wantMsg == "" {
				if containsCode(findings, "TRM001") {
					t.Errorf("unexpected TRM001: %+v", findings)
				}
				return
			}
			f := valFindFinding(findings, "TRM001")
			if f == nil {
				t.Fatalf("TRM001 not found: %+v", findings)
			}
			if f.Severity != SeverityError {
				t.Errorf("Severity = %q, want %q", f.Severity, SeverityError)
			}
			if f.Location.Key != tc.wantKey {
				t.Errorf("Location.Key = %q, want %q", f.Location.Key, tc.wantKey)
			}
			if !strings.Contains(f.Message, tc.wantMsg) {
				t.Errorf("message %q does not contain %q", f.Message, tc.wantMsg)
			}
		})
	}
}

// TestValidateServerConfig_HeaderProtectionSuppressesHDR001 verifies that a
// header-protection key makes H1..H4 = 1..4 legal while structural HDR002
// (Max < Min) still fires.
func TestValidateServerConfig_HeaderProtectionSuppressesHDR001(t *testing.T) {
	cfg := freshServerConfig(t)
	cfg.Obfuscation.HeaderProtectionKey = valValidHeaderProtectionKey()
	cfg.Obfuscation.H1 = HeaderRange{Min: 1, Max: 1}
	cfg.Obfuscation.H2 = HeaderRange{Min: 2, Max: 2}
	cfg.Obfuscation.H3 = HeaderRange{Min: 3, Max: 3}
	cfg.Obfuscation.H4 = HeaderRange{Min: 4, Max: 4}
	findings := ValidateServerConfig(&cfg)
	assertFindingAbsent(t, findings, "HDR001")
	assertFindingAbsent(t, findings, "HDR002")

	bad := freshServerConfig(t)
	bad.Obfuscation.HeaderProtectionKey = valValidHeaderProtectionKey()
	bad.Obfuscation.H2 = HeaderRange{Min: 5, Max: 1}
	findings = ValidateServerConfig(&bad)
	assertFindingPresent(t, findings, "HDR002")
}

func TestValidateServerConfig_RoundTripGenerated_AllProtocols(t *testing.T) {
	// Property: every config GenerateServerConfig produces validates clean.
	for i := range 50 {
		cfg := freshServerConfig(t)
		findings := ValidateServerConfig(&cfg)
		for _, f := range findings {
			if f.Severity == SeverityError {
				t.Fatalf("iteration %d: %+v on freshly generated config", i, f)
			}
		}
	}
}

// TestValidateServerConfig_PSC003UnreachableWithNilIPackets pins the current
// behavior that PSC003 never fires when iPacketSizes=nil.
func TestValidateServerConfig_PSC003UnreachableWithNilIPackets(t *testing.T) {
	cfg := freshServerConfig(t)
	findings := ValidateServerConfig(&cfg)
	for _, f := range findings {
		if f.Code == "PSC003" {
			t.Fatalf("PSC003 fired despite nil iPackets")
		}
	}
}

// TestValidateHeaderRange_Exported verifies the promoted public API works.
func TestValidateHeaderRange_Exported(t *testing.T) {
	// Range that overlaps WG type-id 4 must be rejected.
	err := ValidateHeaderRange(HeaderRange{Min: 1, Max: 100})
	if err == nil {
		t.Fatal("ValidateHeaderRange should reject [1..100]")
	}
	// Legal range above wgMessageTypeMax must pass.
	if err := ValidateHeaderRange(HeaderRange{Min: 5, Max: 1000}); err != nil {
		t.Fatalf("ValidateHeaderRange([5..1000]) = %v, want nil", err)
	}
}

// TestValidateHeaderRange asserts that validateHeaderRange rejects ranges
// containing any standard WireGuard message type-id (1..4) and accepts ranges
// strictly above 4. Boundary cases at 0, 1, 2, 3, 4, 5 are all covered.
func TestValidateHeaderRange(t *testing.T) {
	tests := []struct {
		name    string
		r       HeaderRange
		wantErr bool
	}{
		// Bad: contains forbidden WG type-ids.
		{"contains_all_wg_typeids", HeaderRange{Min: 0, Max: 5}, true},
		{"starts_at_zero_includes_typeids", HeaderRange{Min: 0, Max: 4}, true},
		{"starts_at_one_single", HeaderRange{Min: 1, Max: 1}, true},
		{"starts_at_two_single", HeaderRange{Min: 2, Max: 2}, true},
		{"starts_at_three_single", HeaderRange{Min: 3, Max: 3}, true},
		{"starts_at_four_single", HeaderRange{Min: 4, Max: 4}, true},
		{"crosses_upper_bound_of_typeids", HeaderRange{Min: 4, Max: 10}, true},
		{"spans_typeids_only", HeaderRange{Min: 1, Max: 4}, true},
		// Good: starts strictly above 4.
		{"just_above_typeids", HeaderRange{Min: 5, Max: 100}, false},
		{"large_range", HeaderRange{Min: 100, Max: 1000000}, false},
		{"max_uint32_window", HeaderRange{Min: 1000000, Max: 2147483647}, false},
		// Good: pure-zero range does not contain any forbidden id (Min<=4 holds
		// but Max>=1 fails). Out-of-scope per P0.4 plan §7.6 — kept passing for
		// fixture compatibility; full zero-range hardening lives in P1.3.
		{"zero_range_passes", HeaderRange{Min: 0, Max: 0}, false},
		// Bad: structurally invalid (Max < Min).
		{"max_less_than_min", HeaderRange{Min: 100, Max: 50}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateHeaderRange(tc.r)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateHeaderRange(%+v) error = %v, wantErr %v", tc.r, err, tc.wantErr)
			}
		})
	}
}
