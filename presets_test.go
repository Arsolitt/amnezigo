package amnezigo

import (
	"testing"
)

// TestPresetsValidate verifies that every built-in preset produces obfuscation
// parameters that pass ValidatePacketSizes with zero findings.
func TestPresetsValidate(t *testing.T) {
	presets := ListPresets()
	if len(presets) == 0 {
		t.Fatal("ListPresets returned empty list")
	}

	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			padded := paddedSizes(p.S1, p.S2, p.S3, p.S4)
			// Use empty I-packet sizes — presets define server-side params only;
			// I-packets are generated per-peer at export time.
			err := ValidatePacketSizes(p.S1, p.S2, p.S3, p.S4, nil, p.Jmin, p.Jmax)
			if err != nil {
				t.Errorf("preset %q fails ValidatePacketSizes: %v (padded sizes: %v)",
					p.Name, err, padded)
			}
		})
	}
}

// TestPresetsHeaderRangesValid verifies that every preset's H1-H4 header ranges
// are non-overlapping and do not contain WG message type-ids.
func TestPresetsHeaderRangesValid(t *testing.T) {
	presets := ListPresets()
	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			ranges := [4]HeaderRange{p.H1, p.H2, p.H3, p.H4}
			for i, r := range ranges {
				if err := validateHeaderRange(r); err != nil {
					t.Errorf("preset %q: H%d range [%d-%d] invalid: %v",
						p.Name, i+1, r.Min, r.Max, err)
				}
			}
			// Check pairwise non-overlap (sorted by Min).
			for i := range 3 {
				if ranges[i].Max >= ranges[i+1].Min {
					t.Errorf("preset %q: H%d [%d-%d] overlaps H%d [%d-%d]",
						p.Name, i+1, ranges[i].Min, ranges[i].Max,
						i+2, ranges[i+1].Min, ranges[i+1].Max)
				}
			}
		})
	}
}

// TestGetPreset_Known verifies that GetPreset returns the correct preset for
// each known name.
func TestGetPreset_Known(t *testing.T) {
	expectedNames := []string{
		"lan-conservative",
		"home-balanced",
		"mobile-aggressive",
		"test-minimal",
	}
	for _, name := range expectedNames {
		t.Run(name, func(t *testing.T) {
			p, err := GetPreset(name)
			if err != nil {
				t.Fatalf("GetPreset(%q) returned error: %v", name, err)
			}
			if p.Name != name {
				t.Errorf("GetPreset(%q) returned preset with Name=%q", name, p.Name)
			}
			if p.Description == "" {
				t.Errorf("preset %q has empty Description", name)
			}
		})
	}
}

// TestGetPreset_Unknown verifies that GetPreset returns an error for an
// unregistered preset name.
func TestGetPreset_Unknown(t *testing.T) {
	_, err := GetPreset("nonexistent-preset")
	if err == nil {
		t.Fatal("expected error for unknown preset, got nil")
	}
}

// TestListPresets_ReturnsAll verifies that ListPresets returns at least the
// four required presets and that every entry has a non-empty Name.
func TestListPresets_ReturnsAll(t *testing.T) {
	presets := ListPresets()
	if len(presets) < 4 {
		t.Errorf("expected at least 4 presets, got %d", len(presets))
	}
	seen := make(map[string]bool)
	for _, p := range presets {
		if p.Name == "" {
			t.Error("found preset with empty Name")
		}
		if seen[p.Name] {
			t.Errorf("duplicate preset name: %q", p.Name)
		}
		seen[p.Name] = true
	}

	required := []string{"lan-conservative", "home-balanced", "mobile-aggressive", "test-minimal"}
	for _, name := range required {
		if !seen[name] {
			t.Errorf("required preset %q not found in ListPresets()", name)
		}
	}
}

// TestPresetRoundTrip verifies each preset can be used to create a
// ServerConfig and the resulting obfuscation params pass validation.
func TestPresetRoundTrip(t *testing.T) {
	presets := ListPresets()
	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			cfg := p.ToServerObfuscation()

			err := ValidatePacketSizes(cfg.S1, cfg.S2, cfg.S3, cfg.S4, nil, cfg.Jmin, cfg.Jmax)
			if err != nil {
				t.Errorf("preset %q round-trip failed ValidatePacketSizes: %v", p.Name, err)
			}

			ranges := [4]HeaderRange{cfg.H1, cfg.H2, cfg.H3, cfg.H4}
			for i, r := range ranges {
				if err := validateHeaderRange(r); err != nil {
					t.Errorf("preset %q round-trip: H%d invalid: %v", p.Name, i+1, err)
				}
			}
		})
	}
}

// TestPresetMTU verifies each preset has a valid MTU value (positive, <= 1500).
func TestPresetMTU(t *testing.T) {
	presets := ListPresets()
	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			if p.MTU <= 0 || p.MTU > 1500 {
				t.Errorf("preset %q has invalid MTU: %d", p.Name, p.MTU)
			}
		})
	}
}

// TestPresetDefaultProtocol verifies each preset has a valid default protocol.
func TestPresetDefaultProtocol(t *testing.T) {
	validProtocols := map[string]bool{
		"random": true,
		"quic":   true,
		"dns":    true,
		"dtls":   true,
		"stun":   true,
	}
	presets := ListPresets()
	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			if !validProtocols[p.DefaultProtocol] {
				t.Errorf("preset %q has invalid DefaultProtocol: %q", p.Name, p.DefaultProtocol)
			}
		})
	}
}

// TestPresetJunkRange verifies each preset has Jmin < Jmax.
func TestPresetJunkRange(t *testing.T) {
	presets := ListPresets()
	for _, p := range presets {
		t.Run(p.Name, func(t *testing.T) {
			if p.Jmin >= p.Jmax {
				t.Errorf("preset %q has invalid junk range: Jmin=%d >= Jmax=%d",
					p.Name, p.Jmin, p.Jmax)
			}
		})
	}
}
