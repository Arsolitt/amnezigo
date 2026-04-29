package amnezigo

import (
	"errors"
	"fmt"
)

// WireGuard message size constants from amneziawg-go device/noise-protocol.go.
// These are the on-the-wire sizes BEFORE AWG S-padding is applied.
const (
	wgInitiationSize  = 148
	wgResponseSize    = 92
	wgCookieReplySize = 64
	wgTransportSize   = 32
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
