//go:build e2e

// Package e2e contains container-based end-to-end tests that exercise
// generated AmneziaWG configs against the real amneziawg-go engine and
// amneziawg-tools inside Docker.
//
// The package is guarded by the `e2e` build tag and never imports the root
// module: it builds the amnezigo CLI once (TestMain) and drives it as a
// black box, so the tests keep working while the library evolves.
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// awgGoTag is the pinned amneziawg-go release built into the test image.
	awgGoTag = "v3.1.20260828"
	// awgToolsTag is the pinned amneziawg-tools release built into the test image.
	awgToolsTag = "v3.1.20260812"
	// imageTag is the local Docker tag of the e2e test image.
	imageTag = "amnezigo-e2e-awg:31"
	// dockerTimeout bounds a single docker CLI invocation (pull excluded).
	dockerTimeout = 120 * time.Second
	// cliBuildTimeout bounds the one-time amnezigo CLI build.
	cliBuildTimeout = 5 * time.Minute
	// imageBuildTimeout bounds the one-time test image build.
	imageBuildTimeout = 30 * time.Minute
	// rootModuleDir is the module root relative to the e2e package directory.
	rootModuleDir = ".."
)

var (
	// dockerOK reports whether a Docker daemon is reachable.
	dockerOK bool
	// tunOK reports whether containers can open /dev/net/tun.
	tunOK bool
	// buildErr records a failed CLI or image build. It fails tests instead of
	// skipping them, because a broken build is our bug, not the environment's.
	buildErr error
	// cliPath is the absolute path of the amnezigo binary built for the tests.
	cliPath string
)

// TestMain probes the environment, builds the CLI and the test image once,
// then runs the suite. Tests skip themselves through requireHarness when the
// environment cannot support them.
func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

// runMain performs one-time harness setup and returns the process exit code.
// Every probe is attempted even when an earlier build failed, so that
// requireHarness can still distinguish environment skips from real failures.
func runMain(m *testing.M) int {
	tmpDir, err := os.MkdirTemp("", "amnezigo-e2e-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: create temp dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	if out, err := dockerRun("info"); err != nil {
		fmt.Printf("e2e: docker unavailable, container tests will be skipped: %v\n%s\n", err, out)
		return m.Run()
	}
	dockerOK = true

	root, err := filepath.Abs(rootModuleDir)
	if err != nil {
		buildErr = fmt.Errorf("resolve repo root: %w", err)
	} else if _, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr != nil {
		buildErr = fmt.Errorf("repo root %s has no go.mod: %w", root, statErr)
	}

	if buildErr == nil {
		cliPath = filepath.Join(tmpDir, "amnezigo")
		if out, err := runCommand("go", []string{"build", "-o", cliPath, "./cmd/amnezigo"}, root, nil, cliBuildTimeout); err != nil {
			buildErr = fmt.Errorf("build amnezigo CLI: %v\n%s", err, out)
		}
	}

	if buildErr == nil {
		buildArgs := []string{
			"build", "-f", "e2e/awg/Dockerfile",
			"--build-arg", "AWG_GO_TAG=" + awgGoTag,
			"--build-arg", "AWG_TOOLS_TAG=" + awgToolsTag,
			"-t", imageTag,
			"e2e/awg",
		}
		if out, err := runCommand("docker", buildArgs, root, nil, imageBuildTimeout); err != nil {
			buildErr = fmt.Errorf("build e2e image: %v\n%s", err, out)
		}
	}

	if out, err := dockerRun(
		"run", "--rm", "--cap-add=NET_ADMIN", "--device", "/dev/net/tun",
		"alpine:3.19", "sh", "-c", "ls /dev/net/tun",
	); err != nil {
		fmt.Printf("e2e: /dev/net/tun unavailable in containers, tunnel tests will be skipped: %v\n%s\n", err, out)
	} else {
		tunOK = true
	}

	if buildErr != nil {
		fmt.Printf("e2e: harness setup failed: %v\n", buildErr)
	}
	return m.Run()
}

// requireHarness skips the test when Docker or /dev/net/tun is unavailable and
// fails it when the harness could not be built.
func requireHarness(t *testing.T) {
	t.Helper()
	if !dockerOK {
		t.Skip("e2e: docker unavailable")
	}
	if !tunOK {
		t.Skip("e2e: /dev/net/tun unavailable in containers")
	}
	if buildErr != nil {
		t.Fatalf("e2e harness setup failed: %v", buildErr)
	}
}

// TestE2E_DockerfileTagDefaults pins the image build args to the Go constants
// so the pinned upstream revisions can never drift between the two.
func TestE2E_DockerfileTagDefaults(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("awg", "Dockerfile"))
	if err != nil {
		t.Fatalf("read e2e/awg/Dockerfile: %v", err)
	}
	for _, want := range []string{"ARG AWG_GO_TAG=" + awgGoTag, "ARG AWG_TOOLS_TAG=" + awgToolsTag} {
		if !strings.Contains(string(data), want) {
			t.Errorf("e2e/awg/Dockerfile does not contain %q", want)
		}
	}
}
