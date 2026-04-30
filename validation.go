package amnezigo

import (
	"errors"
	"fmt"
	"strings"
)

// WireGuard message size constants from amneziawg-go device/noise-protocol.go.
// These are the on-the-wire sizes BEFORE AWG S-padding is applied.
const (
	wgInitiationSize  = 148
	wgResponseSize    = 92
	wgCookieReplySize = 64
	wgTransportSize   = 32
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

// paddedSizes returns the four AWG-padded packet sizes.
// Order: init, response, cookie, transport.
func paddedSizes(s1, s2, s3, s4 int) [4]int {
	return [4]int{
		s1 + wgInitiationSize,
		s2 + wgResponseSize,
		s3 + wgCookieReplySize,
		s4 + wgTransportSize,
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
	padded := paddedSizes(s1, s2, s3, s4)
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
		wgInitiationSize, wgResponseSize, wgCookieReplySize, wgTransportSize,
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
