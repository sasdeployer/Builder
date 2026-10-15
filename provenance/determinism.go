package provenance

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PinBuildEnv sets environment variables that make build output depend only
// on the source, for every build command Builder runs (defaults and custom
// buildcmd alike). Values the user already set are kept.
func PinBuildEnv(src Source, sourceDir string) {
	// Tools that embed timestamps (gcc __DATE__, Python, many packagers)
	// use this instead of the wall clock.
	if os.Getenv("SOURCE_DATE_EPOCH") == "" {
		if out, err := exec.Command("git", "-C", sourceDir, "log", "-1", "--format=%ct").Output(); err == nil {
			if ts := strings.TrimSpace(string(out)); ts != "" {
				os.Setenv("SOURCE_DATE_EPOCH", ts)
			}
		}
	}

	// Go: no absolute build paths and no VCS stamp (an untracked builder.yaml
	// marks the tree dirty) in binaries.
	goflags := os.Getenv("GOFLAGS")
	for _, f := range []string{"-trimpath", "-buildvcs=false"} {
		if !strings.Contains(goflags, f) {
			goflags = strings.TrimSpace(goflags + " " + f)
		}
	}
	os.Setenv("GOFLAGS", goflags)

	// Ruby: bundler refuses to change Gemfile.lock.
	if hasLockfile(src, "gemfile.lock") && os.Getenv("BUNDLE_FROZEN") == "" {
		os.Setenv("BUNDLE_FROZEN", "true")
	}
}

// LockStatus reports whether the build's dependencies are pinned, judged from
// the build command that will actually run. Only locked builds are cached:
// an unlocked install can resolve different versions on the next run, so a
// cached artifact would no longer match what a fresh build produces.
func LockStatus(cfg Config, src Source, sourceDir string) (bool, string) {
	cmd := strings.ToLower(cfg.BuildCommand)
	has := func(s ...string) bool {
		for _, x := range s {
			if strings.Contains(cmd, x) {
				return true
			}
		}
		return false
	}
	switch cfg.ProjectType {
	case "go":
		return true, "Go modules verify go.sum"
	case "ruby":
		if hasLockfile(src, "gemfile.lock") {
			return true, "Gemfile.lock enforced with BUNDLE_FROZEN"
		}
		return false, "no Gemfile.lock"
	case "rust":
		if has("--locked", "--frozen") {
			return true, "cargo runs with --locked"
		}
		return false, "cargo runs without --locked"
	case "node", "npm":
		if strings.HasPrefix(cmd, "npm ci") || has("--frozen-lockfile", "--immutable") {
			return true, "install enforces the lockfile"
		}
		return false, "dependency install doesn't enforce a lockfile (use npm ci, or --frozen-lockfile with yarn/pnpm/bun)"
	case "python":
		switch {
		case has("uv sync") && has("--frozen", "--locked"):
			return true, "uv sync with --frozen/--locked"
		case has("poetry install") && hasLockfile(src, "poetry.lock"):
			return true, "poetry install from poetry.lock"
		case has("--require-hashes"):
			return true, "pip with --require-hashes"
		case has("pip") && has("requirements.txt"):
			if pinned, why := requirementsPinned(filepath.Join(sourceDir, "requirements.txt")); pinned {
				return true, "every requirement pinned with =="
			} else {
				return false, why
			}
		}
		return false, "python install doesn't enforce a lockfile"
	case "":
		return false, "project type not known yet"
	}
	return false, "Builder doesn't verify dependency locking for " + cfg.ProjectType
}

func hasLockfile(src Source, name string) bool {
	for _, l := range src.Lockfiles {
		if strings.EqualFold(filepath.Base(l.Path), name) {
			return true
		}
	}
	return false
}

// requirementsPinned reports whether every requirement is an exact == pin.
// Includes (-r, -c, -e) and URL/VCS requirements can't be verified.
func requirementsPinned(path string) (bool, string) {
	f, err := os.Open(path)
	if err != nil {
		return false, "requirements.txt not readable"
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			for _, p := range []string{"-r", "--requirement", "-c", "--constraint", "-e", "--editable"} {
				if strings.HasPrefix(line, p) {
					return false, "requirements.txt uses " + p + ", which Builder can't verify"
				}
			}
			continue // index options such as --index-url
		}
		if !strings.Contains(line, "==") || strings.Contains(line, "*") {
			return false, "requirement not pinned with ==: " + line
		}
	}
	return true, ""
}
