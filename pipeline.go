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
