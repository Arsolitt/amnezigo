package amnezigo

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// WireGuard message size constants from amneziawg-go device/noise-protocol.go.
// These are the on-the-wire sizes BEFORE AWG S-padding is applied.
const (
	WGInitiationSize  = 148
	WGResponseSize    = 92
	WGCookieReplySize = 64
	WGTransportSize   = 32
)

// wgMessageTypeMin and wgMessageTypeMax bound the standard WireGuard
// message type-ids (1 = Initiation, 2 = Response, 3 = Cookie Reply,
// 4 = Transport). H1-H4 ranges must never include any value in
// [wgMessageTypeMin..wgMessageTypeMax] — otherwise vanilla WireGuard
// traffic would be accepted by the AWG-aware peer, defeating
// obfuscation. Source: amneziawg-go device/noise-protocol.go.
//
// If a future AWG version introduces a new message type-id, expand
// wgMessageTypeMax — that single edit point updates both the generator
// retry loop and the loader rejection path.
const (
	wgMessageTypeMin = uint32(1)
	wgMessageTypeMax = uint32(4)
)

// PacketSizeCollisionError describes a single size-classification collision in
// a config. It is returned by ValidatePacketSizes when any of the AWG 2.0
// invariants is violated.
type PacketSizeCollisionError struct {
	// Kind is one of: "s-pair", "i-packet", "junk-range".
	Kind string
	// Pair names the colliding entities, e.g. "S1+148 vs S2+92" or
	// "I3 vs S4+32" or "[Jmin..Jmax] contains 148".
	Pair string
	// Size is the colliding numeric size in bytes (or the boundary value for
	// junk ranges).
	Size int
}

func (e *PacketSizeCollisionError) Error() string {
	return fmt.Sprintf("packet size collision (%s): %s at %d bytes", e.Kind, e.Pair, e.Size)
}

// ErrEmptyJunkRange is returned when jmin > jmax. ValidatePacketSizes treats
// this as a structural error in the input, not a collision.
var ErrEmptyJunkRange = errors.New("junk range is empty (jmin > jmax)")

// PaddedSizes returns the four AWG-padded packet sizes.
// Order: init, response, cookie, transport.
func PaddedSizes(s1, s2, s3, s4 int) [4]int {
	return [4]int{
		s1 + WGInitiationSize,
		s2 + WGResponseSize,
		s3 + WGCookieReplySize,
		s4 + WGTransportSize,
	}
}

// ValidatePacketSizes enforces the AWG 2.0 size-classification invariant:
//  1. The four S-padded handshake sizes are pairwise distinct.
//  2. No I-packet length equals any of the four padded sizes.
//  3. The junk range [jmin..jmax] does not include any of the four padded
//     sizes and does not include any of the four raw WireGuard message sizes.
//
// It returns nil if all invariants hold, ErrEmptyJunkRange if jmin > jmax, or
// a *PacketSizeCollisionError describing the first violation found. Order of
// checks: S-pairs → I-packets → junk range.
//
// Designed to be reused by the future `amnezigo validate` CLI command.
func ValidatePacketSizes(s1, s2, s3, s4 int, iPacketSizes []int, jmin, jmax int) error {
	if jmin > jmax {
		return ErrEmptyJunkRange
	}
	padded := PaddedSizes(s1, s2, s3, s4)
	pairLabels := [4]string{"S1+148", "S2+92", "S3+64", "S4+32"}

	// 1. Six pairwise S-padding checks.
	for i := range 4 {
		for j := i + 1; j < 4; j++ {
			if padded[i] == padded[j] {
				return &PacketSizeCollisionError{
					Kind: "s-pair",
					Pair: fmt.Sprintf("%s vs %s", pairLabels[i], pairLabels[j]),
					Size: padded[i],
				}
			}
		}
	}

	// 2. I-packet vs padded-size checks.
	for idx, sz := range iPacketSizes {
		for i, p := range padded {
			if sz == p {
				return &PacketSizeCollisionError{
					Kind: "i-packet",
					Pair: fmt.Sprintf("I%d vs %s", idx+1, pairLabels[i]),
					Size: sz,
				}
			}
		}
	}

	// 3. Junk range vs padded sizes and raw WG constants. Eight forbidden
	// integers; any inside [jmin..jmax] (inclusive) is a collision.
	forbidden := [...]int{
		padded[0], padded[1], padded[2], padded[3],
		WGInitiationSize, WGResponseSize, WGCookieReplySize, WGTransportSize,
	}
	for _, f := range forbidden {
		if f >= jmin && f <= jmax {
			return &PacketSizeCollisionError{
				Kind: "junk-range",
				Pair: fmt.Sprintf("[Jmin..Jmax] contains %d", f),
				Size: f,
			}
		}
	}

	return nil
}

// Severity classifies the impact of a validation finding.
type Severity string

const (
	// SeverityError indicates a violation that prevents the config from working.
	SeverityError Severity = "error"
	// SeverityWarning indicates a non-fatal risk or deprecation signal.
	SeverityWarning Severity = "warning"
	// SeverityInfo is reserved for noteworthy but harmless observations.
	SeverityInfo Severity = "info"
)

// Location pinpoints where a finding originates within a config file.
type Location struct {
	File string `json:"file,omitempty"`
	Key  string `json:"key,omitempty"`
	Line int    `json:"line,omitempty"`
}

// Finding is a single validation observation with severity, stable code,
// location, and human-readable message. P1.4 (`analyze` command) reuses
// these types — do not break wire-compatibility without coordinating.
type Finding struct {
	Message  string   `json:"message"`
	Detail   string   `json:"detail,omitempty"`
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Location Location `json:"location,omitzero"`
}

// OneLine returns the canonical single-line representation of a finding,
// suitable for CLI text output and log lines. Format:
//
//	[<SEVERITY> <CODE>] <file>:<line> (key=<key>): <message>
//
// Line and key segments are omitted when empty.
func (f Finding) OneLine() string {
	var locParts []string
	if f.Location.File != "" {
		locParts = append(locParts, f.Location.File)
	}
	if f.Location.Line > 0 && len(locParts) > 0 {
		locParts[len(locParts)-1] += fmt.Sprintf(":%d", f.Location.Line)
	}
	loc := strings.Join(locParts, "")
	if f.Location.Key != "" {
		loc += fmt.Sprintf(" (key=%s)", f.Location.Key)
	}
	if loc != "" {
		loc = " " + loc
	}
	return fmt.Sprintf("[%s %s]%s: %s",
		strings.ToUpper(string(f.Severity)), f.Code, loc, f.Message)
}

// ValidateHeaderRange returns a non-nil error if the range includes any of
// the standard WireGuard message type-ids (1..4) or is structurally invalid
// (Max < Min). H1-H4 ranges that include WG type-ids would accept vanilla
// WireGuard packets, breaking the obfuscation guarantee that AWG and
// vanilla-WG networks are inert to each other.
//
// The check is inclusive on both ends because parser/writer use inclusive
// "Min-Max" notation.
func ValidateHeaderRange(r HeaderRange) error {
	if r.Max < r.Min {
		return fmt.Errorf("invalid header range: Max (%d) < Min (%d)", r.Max, r.Min)
	}
	if r.Min <= wgMessageTypeMax && r.Max >= wgMessageTypeMin {
		return fmt.Errorf("header range [%d-%d] contains forbidden WG type-id(s) in [%d..%d]",
			r.Min, r.Max, wgMessageTypeMin, wgMessageTypeMax)
	}
	return nil
}

// ValidateServerConfig runs every validation rule against the parsed config
// and returns all findings. The slice is empty when the config is clean.
// Severity ranking, ordering, and code allocation are documented in
// docs/plans/p1.3-validate-command.md § 4.8.
func ValidateServerConfig(cfg *ServerConfig) []Finding {
	var findings []Finding

	findings = append(findings, validateRequiredFields(cfg)...)
	findings = append(findings, validateSPrefixes(cfg)...)
	findings = append(findings, validateJunkRange(cfg)...)
	findings = append(findings, validateHeaderRanges(cfg)...)
	findings = append(findings, validateHeaderProtection(cfg)...)
	findings = append(findings, validateRandomTrailers(cfg)...)
	findings = append(findings, validateTransportRanges(cfg)...)

	return findings
}

func validateRequiredFields(cfg *ServerConfig) []Finding {
	var out []Finding
	add := func(key string) {
		out = append(out, Finding{
			Severity: SeverityError,
			Code:     "FLD001",
			Location: Location{Key: key},
			Message:  fmt.Sprintf("required field %q is missing", key),
			Detail:   "server configs require PrivateKey, Address, and ListenPort to function.",
		})
	}
	if cfg.Interface.PrivateKey == "" {
		add("PrivateKey")
	}
	if cfg.Interface.Address == "" {
		add("Address")
	}
	if cfg.Interface.ListenPort == 0 {
		add("ListenPort")
	}
	return out
}

func validateSPrefixes(cfg *ServerConfig) []Finding {
	o := cfg.Obfuscation
	err := ValidatePacketSizes(o.S1, o.S2, o.S3, o.S4, nil, o.Jmin, o.Jmax)
	return findingsFromValidationError(err)
}

func validateJunkRange(cfg *ServerConfig) []Finding {
	o := cfg.Obfuscation
	if o.Jmin > o.Jmax {
		return []Finding{{
			Severity: SeverityError,
			Code:     "JNK001",
			Message:  fmt.Sprintf("junk range Jmin (%d) > Jmax (%d)", o.Jmin, o.Jmax),
		}}
	}
	return nil
}

func validateHeaderRanges(cfg *ServerConfig) []Finding {
	var out []Finding
	ranges := [4]HeaderRange{
		cfg.Obfuscation.H1, cfg.Obfuscation.H2,
		cfg.Obfuscation.H3, cfg.Obfuscation.H4,
	}
	for i, r := range ranges {
		err := ValidateHeaderRange(r)
		if err == nil {
			continue
		}
		// With a header-protection key the 4-byte type is encrypted and the
		// engine uses H1..H4 = 1..4, so WG type-ids are legitimate; only the
		// structural Max < Min check still applies.
		if cfg.Obfuscation.HeaderProtectionKey != "" && r.Max >= r.Min {
			continue
		}
		code := "HDR001"
		if r.Max < r.Min {
			code = "HDR002"
		}
		out = append(out, Finding{
			Severity: SeverityError,
			Code:     code,
			Location: Location{Key: fmt.Sprintf("H%d", i+1)},
			Message:  err.Error(),
			Detail:   "H1-H4 ranges must avoid WG message type-ids 1..4.",
		})
	}
	return out
}

// validateHeaderProtection checks the AWG 3.x header-protection key and the
// S-prefix floor the ChaCha20 nonce imposes on it.
func validateHeaderProtection(cfg *ServerConfig) []Finding {
	o := cfg.Obfuscation
	if o.HeaderProtectionKey == "" {
		return nil
	}
	var out []Finding
	if o.S1 < headerProtectionNonceSize || o.S2 < headerProtectionNonceSize ||
		o.S3 < headerProtectionNonceSize || o.S4 < headerProtectionNonceSize {
		out = append(out, Finding{
			Severity: SeverityError,
			Code:     "HPK001",
			Location: Location{Key: keyHeaderProtection},
			Message: fmt.Sprintf(
				"header protection requires S1-S4 >= %d (got S1=%d, S2=%d, S3=%d, S4=%d)",
				headerProtectionNonceSize, o.S1, o.S2, o.S3, o.S4),
			Detail: "Header protection encrypts the 4-byte message header at offset S{n}; " +
				"the ChaCha20 nonce is the first 12 bytes of the packet.",
		})
	}
	if !validHeaderProtectionKey(o.HeaderProtectionKey) {
		out = append(out, Finding{
			Severity: SeverityError,
			Code:     "HPK003",
			Location: Location{Key: keyHeaderProtection},
			Message:  "HeaderProtectionKey is not 44-char base64 of 32 bytes",
			Detail: "The engine decodes the key into a 32-byte ChaCha20 key; " +
				"a malformed value cannot protect the header.",
		})
	}
	return out
}

// validHeaderProtectionKey reports whether key is base64 of exactly keyLength
// (32) bytes, the ChaCha20 header-key size the engine decodes.
func validHeaderProtectionKey(key string) bool {
	raw, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(raw) == keyLength
}

// validateRandomTrailers warns when random trailers are enabled while the
// S-prefixes differ; the AWG 3.1 reference recommends equal S values there.
func validateRandomTrailers(cfg *ServerConfig) []Finding {
	o := cfg.Obfuscation
	if !o.RandomTrailers {
		return nil
	}
	if o.S1 == o.S2 && o.S2 == o.S3 && o.S3 == o.S4 {
		return nil
	}
	return []Finding{{
		Severity: SeverityWarning,
		Code:     "TRL001",
		Location: Location{Key: keyRandomTrailers},
		Message: "RandomTrailers is enabled while S1..S4 differ; " +
			"the AWG 3.1 reference recommends equal S values to avoid packet-type misclassification",
		Detail: "The receiver classifies packets by size and only accepts size > expected when trailers are enabled, " +
			"so unequal S values can misclassify padded handshake packets.",
	}}
}

// validateTransportRanges checks the six AWG 3.x uint16 ranges for structural
// errors: Max < Min or exactly one zero bound (mixed "0-N"/"N-0").
func validateTransportRanges(cfg *ServerConfig) []Finding {
	o := cfg.Obfuscation
	ranges := []struct {
		key string
		r   U16Range
	}{
		{keyContentPadding, o.ContentPadding},
		{keyRekeyAfterTime, o.RekeyAfterTime},
		{keyRekeyTimeout, o.RekeyTimeout},
		{keyRejectAfterTime, o.RejectAfterTime},
		{keyKeepaliveTimeout, o.KeepaliveTimeout},
		{keyMaxHandshakeAttempts, o.MaxHandshakeAttempts},
	}
	var out []Finding
	for _, entry := range ranges {
		if entry.r.Max < entry.r.Min {
			out = append(out, Finding{
				Severity: SeverityError,
				Code:     "TRM001",
				Location: Location{Key: entry.key},
				Message: fmt.Sprintf(
					"invalid %s range: max (%d) is below min (%d)",
					entry.key, entry.r.Max, entry.r.Min),
			})
			continue
		}
		if (entry.r.Min == 0) != (entry.r.Max == 0) {
			out = append(out, Finding{
				Severity: SeverityError,
				Code:     "TRM001",
				Location: Location{Key: entry.key},
				Message: fmt.Sprintf(
					"invalid %s range: bounds must both be zero or both non-zero (got %d-%d)",
					entry.key, entry.r.Min, entry.r.Max),
			})
		}
	}
	return out
}

func findingsFromValidationError(err error) []Finding {
	if err == nil {
		return nil
	}
	var psc *PacketSizeCollisionError
	if errors.As(err, &psc) {
		code := map[string]string{
			"s-pair":     "PSC001",
			"i-packet":   "PSC003",
			"junk-range": "PSC002",
		}[psc.Kind]
		return []Finding{{
			Severity: SeverityError,
			Code:     code,
			Message:  err.Error(),
		}}
	}
	if errors.Is(err, ErrEmptyJunkRange) {
		return []Finding{{
			Severity: SeverityError,
			Code:     "JNK001",
			Message:  err.Error(),
		}}
	}
	return []Finding{{
		Severity: SeverityError,
		Code:     "PSC000",
		Message:  err.Error(),
	}}
}
