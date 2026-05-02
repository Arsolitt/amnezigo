package amnezigo

// GenerateOptions configures the generate pipeline.
type GenerateOptions struct {
	ProjectDir string
	OutputDir  string
	JpathDirs  []string
	PeerFilter []string
	DryRun     bool
	FullReset  bool
}

// GenerateResult holds the output of a generate run.
type GenerateResult struct {
	ServerPeer  string
	Files       []FileOutput
	ClientPeers []string
	Findings    []Finding
}

// FileOutput represents a single file to be written.
type FileOutput struct {
	RelPath string
	Content []byte
}

// resolveObfuscation merges explicit manifest values with randomly generated ones.
// If ObfuscationManifest has no values (all nil), generates everything randomly.
// If some values are set, preserves them and generates the rest.
func resolveObfuscation(obf ObfuscationManifest) (ServerObfuscationConfig, error) {
	result := ServerObfuscationConfig{
		S1: resolveInt(obf.S1),
		S2: resolveInt(obf.S2),
		S3: resolveInt(obf.S3),
		S4: resolveInt(obf.S4),
	}

	result = fillMissingSPrefixes(result)

	result = fillMissingHeaders(obf, result)

	junkResult, err := fillMissingJunk(obf, result)
	if err != nil {
		return ServerObfuscationConfig{}, err
	}
	result = junkResult

	return result, nil
}

// fillMissingSPrefixes generates S-prefixes for any zero values.
// Retries until all zero fields get non-zero values, since GenerateSPrefixes
// can produce 0 (rand.Int [0,64)).
func fillMissingSPrefixes(cfg ServerObfuscationConfig) ServerObfuscationConfig {
	if cfg.S1 != 0 && cfg.S2 != 0 && cfg.S3 != 0 && cfg.S4 != 0 {
		return cfg
	}
	for range 100 {
		p := GenerateSPrefixes()
		cfg.S1 = pickNonZero(cfg.S1, p.S1)
		cfg.S2 = pickNonZero(cfg.S2, p.S2)
		cfg.S3 = pickNonZero(cfg.S3, p.S3)
		cfg.S4 = pickNonZero(cfg.S4, p.S4)
		if cfg.S1 != 0 && cfg.S2 != 0 && cfg.S3 != 0 && cfg.S4 != 0 {
			return cfg
		}
	}
	return cfg
}

// fillMissingHeaders generates header ranges for any nil values.
func fillMissingHeaders(obf ObfuscationManifest, cfg ServerObfuscationConfig) ServerObfuscationConfig {
	if obf.H1 == nil || obf.H2 == nil || obf.H3 == nil || obf.H4 == nil {
		headers := GenerateHeaderRanges()
		cfg.H1 = resolveHeader(obf.H1, headers[0])
		cfg.H2 = resolveHeader(obf.H2, headers[1])
		cfg.H3 = resolveHeader(obf.H3, headers[2])
		cfg.H4 = resolveHeader(obf.H4, headers[3])
		return cfg
	}
	cfg.H1 = *obf.H1
	cfg.H2 = *obf.H2
	cfg.H3 = *obf.H3
	cfg.H4 = *obf.H4
	return cfg
}

// fillMissingJunk generates junk parameters for any nil values.
// Retries until Jc is non-zero, since GenerateJunkParamsWithForbidden
// can produce Jc=0 (rand.Int [0,11)).
func fillMissingJunk(obf ObfuscationManifest, cfg ServerObfuscationConfig) (ServerObfuscationConfig, error) {
	if obf.Jc != nil && obf.Jmin != nil && obf.Jmax != nil {
		cfg.Jc = *obf.Jc
		cfg.Jmin = *obf.Jmin
		cfg.Jmax = *obf.Jmax
		return cfg, nil
	}
	forbidden := [4]int{
		cfg.S1 + 148, // wgInitiationSize
		cfg.S2 + 92,  // wgResponseSize
		cfg.S3 + 64,  // wgCookieReplySize
		cfg.S4 + 32,  // wgTransportSize
	}
	jc := resolveInt(obf.Jc)
	jmin := resolveInt(obf.Jmin)
	jmax := resolveInt(obf.Jmax)
	for range 100 {
		junk, err := GenerateJunkParamsWithForbidden(forbidden)
		if err != nil {
			return ServerObfuscationConfig{}, err
		}
		cfg.Jc = pickNonZero(jc, junk.Jc)
		cfg.Jmin = pickNonZero(jmin, junk.Jmin)
		cfg.Jmax = pickNonZero(jmax, junk.Jmax)
		if cfg.Jc != 0 {
			return cfg, nil
		}
	}
	return cfg, nil
}

// resolveInt returns the dereferenced value or 0 if nil.
func resolveInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// resolveHeader returns the explicit header or the generated fallback.
func resolveHeader(explicit *HeaderRange, fallback HeaderRange) HeaderRange {
	if explicit == nil {
		return fallback
	}
	return *explicit
}

// pickNonZero returns current if non-zero, otherwise generated.
func pickNonZero(current, generated int) int {
	if current == 0 && generated != 0 {
		return generated
	}
	return current
}
