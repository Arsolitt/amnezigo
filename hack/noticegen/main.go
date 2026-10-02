// Package main regenerates NOTICE and licenses/ from the module graph of the
// released amnezigo binaries.
//
// Run it from the repository root:
//
//	go run ./hack/noticegen
//
// The module cache must be warm (go mod download), because the sources of every
// dependency are read to collect the license files they ship.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// mainModule is the module the released binary belongs to; its packages are
	// not third-party components.
	mainModule = "github.com/Arsolitt/amnezigo"

	// noticePath is the generated attribution file.
	noticePath = "NOTICE"
	// licensesDir is the generated directory holding the upstream license texts.
	licensesDir = "licenses"

	// licenseFileMode is the mode of the copied license texts. They are meant to
	// be read by everyone, including processes inside the published image.
	licenseFileMode = 0o644
	// licenseDirMode is the mode of the license directories.
	licenseDirMode = 0o755

	// spdxUnknown is reported for components whose license cannot be identified.
	spdxUnknown = "unknown"
	// missingLicense is reported for components that ship no attribution file.
	missingLicense = "missing (no license file in the module cache)"
)

// packages are the entry points GoReleaser builds and ships.
var packages = []string{"./cmd/amnezigo"}

// target is one (GOOS, GOARCH) pair of the release matrix.
type target struct {
	goos   string
	goarch string
}

// targets mirrors the release matrix in .goreleaser.yaml (linux/amd64,
// darwin/amd64). Keep the two in sync: a target added there must be added here,
// otherwise components that are only linked for it are silently missing from
// NOTICE.
var targets = []target{
	{"linux", "amd64"},
	{"darwin", "amd64"},
}

// licensePrefixes are the lowercased file name prefixes that identify the
// attribution and license files an upstream module ships. A file matches when
// its name equals a prefix or continues with a separator, see
// hasLicensePrefix.
var licensePrefixes = []string{"license", "licence", "notice", "copying", "authors", "patents"}

// licensePattern pairs the SPDX identifier of a license with lowercased phrases
// that must all occur in the license text.
type licensePattern struct {
	spdx    string
	phrases []string
}

// licensePatterns is ordered most specific first. Every matching pattern is
// reported as an SPDX expression in that order, except for patterns whose
// phrases are a subset of an already matched pattern (BSD-2-Clause behind
// BSD-3-Clause, for example). The GPL family is matched on the title dates to
// avoid confusing the variants with each other.
var licensePatterns = []licensePattern{
	{spdx: "AGPL-3.0-only", phrases: []string{"gnu affero general public license", "version 3, 19 november 2007"}},
	{spdx: "LGPL-3.0-only", phrases: []string{"gnu lesser general public license", "version 3, 29 june 2007"}},
	{spdx: "GPL-3.0-only", phrases: []string{"gnu general public license", "version 3, 29 june 2007"}},
	{spdx: "MPL-2.0", phrases: []string{"mozilla public license version 2.0"}},
	{spdx: "Apache-2.0", phrases: []string{"apache license", "version 2.0, january 2004"}},
	{spdx: "BSD-3-Clause", phrases: []string{"redistribution and use in source and binary forms", "neither the name"}},
	{spdx: "BSD-2-Clause", phrases: []string{"redistribution and use in source and binary forms"}},
	{spdx: "ISC", phrases: []string{"permission to use, copy, modify, and/or distribute this software"}},
	{spdx: "MIT", phrases: []string{"permission is hereby granted, free of charge, to any person obtaining a copy"}},
}

// module is one third-party component of the dependency graph.
type module struct {
	path    string
	version string
	dir     string
}

// moduleFiles is one module together with the attribution files it ships.
type moduleFiles struct {
	names   []string // selected file names, sorted
	content map[string][]byte
}

// noticeHeader is reproduced verbatim at the top of NOTICE.
const noticeHeader = `amnezigo
========

amnezigo is a declarative AmneziaWG v3.1 configuration generator, released as
the amnezigo CLI and as a Go library. This file lists the third-party
components linked into the released binaries and the license files they ship.
The full text of every third-party license is bundled in licenses/ next to this
file.

amnezigo itself is licensed under the GNU General Public License version 3
(SPDX: GPL-3.0-only, LICENSE).
`

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "noticegen:", err)
		os.Exit(1)
	}
}

func run() error {
	mods, err := collectModules()
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(mods))
	for path := range mods {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	// Drop stale copies first so that components removed from the module graph
	// cannot linger in licenses/.
	if err := os.RemoveAll(licensesDir); err != nil {
		return fmt.Errorf("remove %s: %w", licensesDir, err)
	}

	files, written, err := copyLicenseFiles(paths, mods)
	if err != nil {
		return err
	}

	notice, notices, missing := renderNotice(paths, mods, files)
	if err := os.WriteFile(noticePath, notice, licenseFileMode); err != nil {
		return fmt.Errorf("write %s: %w", noticePath, err)
	}

	for _, path := range paths {
		if len(files[path].names) == 0 {
			fmt.Fprintf(os.Stderr, "noticegen: warning: %s %s ships no license file\n",
				path, mods[path].version)
		}
	}

	fmt.Fprintf(
		os.Stdout,
		"noticegen: %d components (%d missing licenses), %d with NOTICE files, %d license files written\n",
		len(paths),
		missing,
		notices,
		written,
	)
	return nil
}

// collectModules resolves the dependency graph of every released target and
// unions the modules found into one map keyed by module path.
func collectModules() (map[string]module, error) {
	const format = "{{if .Module}}{{.Module.Path}}\t{{.Module.Version}}\t{{.Module.Dir}}{{end}}"

	mods := make(map[string]module)
	for _, t := range targets {
		args := append([]string{"list", "-deps", "-f", format}, packages...)
		cmd := exec.CommandContext(context.Background(), "go", args...)
		cmd.Env = append(os.Environ(), "GOOS="+t.goos, "GOARCH="+t.goarch)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("go list -deps for %s/%s: %w: %s",
				t.goos, t.goarch, err, strings.TrimSpace(stderr.String()))
		}

		for line := range strings.SplitSeq(stdout.String(), "\n") {
			if strings.TrimSpace(line) == "" {
				continue // stdlib packages have no module.
			}
			path, rest, ok := strings.Cut(line, "\t")
			if !ok {
				return nil, fmt.Errorf("unexpected go list output line %q", line)
			}
			version, dir, ok := strings.Cut(rest, "\t")
			if !ok {
				return nil, fmt.Errorf("unexpected go list output line %q", line)
			}
			if path == mainModule || strings.HasPrefix(path, mainModule+"/") {
				continue
			}
			mods[path] = module{path: path, version: version, dir: dir}
		}
	}
	return mods, nil
}

// copyLicenseFiles copies the attribution files of every component into
// licenses/<module-path>/ and reports how many files were written.
func copyLicenseFiles(paths []string, mods map[string]module) (map[string]*moduleFiles, int, error) {
	if err := os.MkdirAll(licensesDir, licenseDirMode); err != nil {
		return nil, 0, fmt.Errorf("create %s: %w", licensesDir, err)
	}
	// os.Root confines every write to licenses/, whatever the module path is.
	root, err := os.OpenRoot(licensesDir)
	if err != nil {
		return nil, 0, fmt.Errorf("open %s: %w", licensesDir, err)
	}
	defer root.Close()

	files := make(map[string]*moduleFiles, len(paths))
	written := 0
	for _, path := range paths {
		mf, count, err := copyModuleFiles(root, path, mods[path])
		if err != nil {
			return nil, 0, err
		}
		files[path] = mf
		written += count
	}
	return files, written, nil
}

// copyModuleFiles collects the attribution files one module ships in the module
// cache and copies them under licenses/<module-path>/.
func copyModuleFiles(root *os.Root, path string, m module) (*moduleFiles, int, error) {
	mf := &moduleFiles{content: map[string][]byte{}}
	if m.dir == "" {
		return mf, 0, nil
	}

	names, err := selectLicenseFiles(m.dir)
	if err != nil {
		return nil, 0, err
	}
	mf.names = names

	written := 0
	for _, name := range names {
		src := filepath.Join(m.dir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, 0, fmt.Errorf("read %s: %w", src, err)
		}
		mf.content[name] = data

		dst := filepath.Join(filepath.FromSlash(path), name)
		if err := root.MkdirAll(filepath.Dir(dst), licenseDirMode); err != nil {
			return nil, 0, fmt.Errorf("create %s: %w", filepath.Join(licensesDir, filepath.Dir(dst)), err)
		}
		if err := root.WriteFile(dst, data, licenseFileMode); err != nil {
			return nil, 0, fmt.Errorf("write %s: %w", filepath.Join(licensesDir, dst), err)
		}
		written++
	}
	return mf, written, nil
}

// selectLicenseFiles returns the sorted names of the attribution files a module
// ships in its root directory.
func selectLicenseFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".go") {
			continue // Go sources are never attribution files.
		}
		for _, prefix := range licensePrefixes {
			if hasLicensePrefix(lower, prefix) {
				names = append(names, name)
				break
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

// hasLicensePrefix reports whether a lowercased file name is an attribution
// file: it must either equal a prefix or continue with a separator, so that
// sources like noticedep.go are not mistaken for NOTICE files.
func hasLicensePrefix(name, prefix string) bool {
	rest, ok := strings.CutPrefix(name, prefix)
	if !ok {
		return false
	}
	return rest == "" || rest[0] == '.' || rest[0] == '-' || rest[0] == '_'
}

// renderNotice composes NOTICE and reports how many components ship a NOTICE
// file and how many have no license file at all. Everything is sorted so that
// repeated runs are byte-identical.
func renderNotice(paths []string, mods map[string]module, files map[string]*moduleFiles) ([]byte, int, int) {
	var b strings.Builder
	b.WriteString(noticeHeader)

	writeHeading(&b, "Third-party components")
	b.WriteString("\n")
	missing := 0
	for _, path := range paths {
		m := mods[path]
		mf := files[path]
		if len(mf.names) == 0 {
			missing++
		}
		fmt.Fprintf(&b, "%s %s — %s — %s\n", m.path, m.version, detectSPDX(mf), licenseSummary(path, mf))
	}

	notices := writeNotices(&b, paths, mods, files)
	return []byte(strings.TrimRight(b.String(), "\n") + "\n"), notices, missing
}

// writeNotices reproduces the NOTICE files shipped by third-party components
// and reports how many components ship one.
func writeNotices(b *strings.Builder, paths []string, mods map[string]module, files map[string]*moduleFiles) int {
	notices := 0
	for _, path := range paths {
		for _, name := range files[path].names {
			if isNoticeFile(name) {
				notices++
			}
		}
	}
	if notices == 0 {
		return 0
	}

	writeHeading(b, "Third-party attribution notices")
	for _, path := range paths {
		m := mods[path]
		mf := files[path]
		for _, name := range mf.names {
			if !isNoticeFile(name) {
				continue
			}
			fmt.Fprintf(b, "\n## %s %s\n\n", m.path, m.version)
			content := mf.content[name]
			b.Write(content)
			if len(content) == 0 || content[len(content)-1] != '\n' {
				b.WriteString("\n")
			}
		}
	}
	return notices
}

// detectSPDX identifies the SPDX license expression of a component from the
// attribution files it ships. It returns spdxUnknown when no pattern matches.
func detectSPDX(mf *moduleFiles) string {
	var text strings.Builder
	for _, name := range mf.names {
		text.Write(mf.content[name])
		text.WriteString("\n")
	}
	normalized := normalizeLicenseText(text.String())

	var matched []licensePattern
	for _, pattern := range licensePatterns {
		if matchesAll(normalized, pattern.phrases) && !subsumed(pattern, matched) {
			matched = append(matched, pattern)
		}
	}
	if len(matched) == 0 {
		return spdxUnknown
	}
	ids := make([]string, 0, len(matched))
	for _, pattern := range matched {
		ids = append(ids, pattern.spdx)
	}
	return strings.Join(ids, " AND ")
}

// subsumed reports whether the phrases of a matching pattern add nothing
// because a previously matched license already requires all of them. This keeps
// BSD-3-Clause from being followed by its BSD-2-Clause subset.
func subsumed(pattern licensePattern, matched []licensePattern) bool {
	for _, other := range matched {
		if isPhraseSubset(pattern.phrases, other.phrases) {
			return true
		}
	}
	return false
}

// isPhraseSubset reports whether every phrase of subset occurs in superset.
func isPhraseSubset(subset, superset []string) bool {
	for _, phrase := range subset {
		if !slices.Contains(superset, phrase) {
			return false
		}
	}
	return true
}

// normalizeLicenseText lowercases text and collapses every whitespace run into
// a single space so that phrases can be matched regardless of line breaks.
func normalizeLicenseText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

// matchesAll reports whether every phrase occurs in the normalized text.
func matchesAll(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if !strings.Contains(text, phrase) {
			return false
		}
	}
	return true
}

// licenseSummary returns the bundled file list shown for one component, or an
// explicit missing marker when the component ships no license file.
func licenseSummary(path string, mf *moduleFiles) string {
	if len(mf.names) == 0 {
		return missingLicense
	}
	bundled := make([]string, 0, len(mf.names))
	for _, name := range mf.names {
		bundled = append(bundled, licensesDir+"/"+path+"/"+name)
	}
	return strings.Join(bundled, ", ")
}

// isNoticeFile reports whether an attribution file is an upstream NOTICE that
// must be reproduced in NOTICE itself.
func isNoticeFile(name string) bool {
	return hasLicensePrefix(strings.ToLower(name), "notice")
}

// writeHeading writes a section title underlined with the same convention as
// the file header.
func writeHeading(b *strings.Builder, title string) {
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", len(title)))
	b.WriteString("\n")
}
