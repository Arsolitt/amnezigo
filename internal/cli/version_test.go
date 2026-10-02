package cli

import (
	"bytes"
	"testing"

	"github.com/Arsolitt/amnezigo/internal/buildinfo"
)

func TestVersionCommand_Output(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		want    string
	}{
		{
			name:    "default build",
			version: "dev",
			commit:  "none",
			want:    "amnezigo dev (none)\n",
		},
		{
			name:    "release build",
			version: "v0.4.0",
			commit:  "abc1234",
			want:    "amnezigo v0.4.0 (abc1234)\n",
		},
		{
			name:    "snapshot build",
			version: "v0.4.1-snapshot.deadbee",
			commit:  "deadbee",
			want:    "amnezigo v0.4.1-snapshot.deadbee (deadbee)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBuildinfo(t, tt.version, tt.commit)

			var stdout, stderr bytes.Buffer
			cmd := NewVersionCommand()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("version failed: %v\nstderr=%s", err, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Errorf("output = %q, want %q", got, tt.want)
			}
		})
	}
}

// setBuildinfo overrides the build stamps for one test and restores the
// previous values when the test finishes.
func setBuildinfo(t *testing.T, version, commit string) {
	t.Helper()

	origVersion, origCommit := buildinfo.Version, buildinfo.Commit
	t.Cleanup(func() {
		buildinfo.Version = origVersion
		buildinfo.Commit = origCommit
	})

	buildinfo.Version = version
	buildinfo.Commit = commit
}
