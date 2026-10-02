// Package buildinfo carries the version and commit stamp injected into the
// binary at build time via -ldflags -X. The CLI reports them through
// `amnezigo version`; library users can read the same values.
package buildinfo

// Version is the release version of the build: a git tag such as v0.4.0, a
// snapshot version, or "dev" for a plain `go build`. Release tooling overrides
// it with:
//
//	-ldflags "-X github.com/Arsolitt/amnezigo/internal/buildinfo.Version=v0.4.0"
var Version = "dev"

// Commit is the short hash of the commit the binary was built from, or "none"
// for a plain `go build`. Release tooling overrides it with:
//
//	-ldflags "-X github.com/Arsolitt/amnezigo/internal/buildinfo.Commit=abc1234"
var Commit = "none"
