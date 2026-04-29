package amnezigo

import (
	"errors"
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
