// Package provenance computes content-addressed build keys, caches build
// outputs by key, and writes SLSA provenance for every artifact.
package provenance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

// cacheSchema is folded into every key; bump it when the key inputs change.
const cacheSchema = "builder-cache/v1"

// lockfiles are recorded as resolved dependencies in provenance.
var lockfiles = map[string]bool{
	"go.sum": true, "package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"cargo.lock": true, "gemfile.lock": true, "requirements.txt": true, "poetry.lock": true,
	"pom.xml": true, "packages.lock.json": true, "npm-shrinkwrap.json": true, "uv.lock": true, "pipfile.lock": true,
	"bun.lock": true, "bun.lockb": true,
}

// Lockfile is a dependency manifest found in the source tree.
type Lockfile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Source describes the exact source tree being built.
type Source struct {
	Digest    string     `json:"digest"`
	Commit    string     `json:"commit,omitempty"`
	Dirty     bool       `json:"dirty"`
	Lockfiles []Lockfile `json:"lockfiles,omitempty"`
}

// Config is the subset of builder.yaml that changes what gets built.
type Config struct {
	ProjectType     string `json:"projectType"`
	BuildTool       string `json:"buildTool,omitempty"`
	BuildFile       string `json:"buildFile,omitempty"`
	BuildCommand    string `json:"buildCommand,omitempty"`
	PreBuildCommand string `json:"preBuildCommand,omitempty"`
	ConfigCommand   string `json:"configCommand,omitempty"`
}

// Inputs is everything that determines the build output. Its hash is the build key.
type Inputs struct {
	Schema         string            `json:"schema"`
	Source         string            `json:"source"`
	Config         Config            `json:"config"`
	Platform       string            `json:"platform"`
	Toolchain      string            `json:"toolchain"`
	BuildEnv       map[string]string `json:"buildEnv,omitempty"`
	BuilderVersion string            `json:"builderVersion"`
}

// Key returns the content address of a build.
func (in Inputs) Key() string {
	b, _ := json.Marshal(in)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ConfigFromEnv reads the build config builder.yaml (or a compiler) has set.
func ConfigFromEnv() Config {
	return Config{
		ProjectType:     strings.ToLower(os.Getenv("BUILDER_PROJECT_TYPE")),
		BuildTool:       os.Getenv("BUILDER_BUILD_TOOL"),
		BuildFile:       os.Getenv("BUILDER_BUILD_FILE"),
		BuildCommand:    os.Getenv("BUILDER_BUILD_COMMAND"),
		PreBuildCommand: os.Getenv("BUILDER_PREBUILD_COMMAND"),
		ConfigCommand:   os.Getenv("BUILDER_CONFIG_COMMAND"),
	}
}

// NewInputs assembles the key inputs for a source tree and config.
func NewInputs(src Source, cfg Config) Inputs {
	return Inputs{
		Schema:         cacheSchema,
		Source:         src.Digest,
		Config:         cfg,
		Platform:       Platform(),
		Toolchain:      withBuildTool(Toolchain(cfg.ProjectType), cfg.BuildCommand),
		BuildEnv:       buildEnv(),
		BuilderVersion: Version(),
	}
}

// Platform is the OS/arch artifacts are built for. Builds are native today.
func Platform() string {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if v := os.Getenv("GOOS"); v != "" {
		goos = v
	}
	if v := os.Getenv("GOARCH"); v != "" {
		goarch = v
	}
	return goos + "/" + goarch
}

// buildEnv captures environment variables that change compiler output.
func buildEnv() map[string]string {
	env := map[string]string{}
	for _, k := range []string{"GOOS", "GOARCH", "GOFLAGS", "CGO_ENABLED", "NODE_ENV", "RUSTFLAGS", "CFLAGS", "LDFLAGS", "SOURCE_DATE_EPOCH", "BUNDLE_FROZEN"} {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}
	if len(env) == 0 {
		return nil
	}
	return env
}

// Version identifies this Builder binary: the git revision it was built from.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "unknown"
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}

// Toolchain returns the version strings of the compilers a project type uses.
func Toolchain(projectType string) string {
	var cmds [][]string
	switch projectType {
	case "go":
		cmds = [][]string{{"go", "version"}}
	case "rust":
		cmds = [][]string{{"rustc", "--version"}, {"cargo", "--version"}}
	case "node", "npm":
		cmds = [][]string{{"node", "--version"}, {"npm", "--version"}}
	case "java":
		cmds = [][]string{{"mvn", "-v"}, {"java", "-version"}}
	case "ruby":
		cmds = [][]string{{"ruby", "--version"}, {"bundle", "--version"}}
	case "python":
		cmds = [][]string{{"python3", "--version"}, {"pip3", "--version"}}
	case "c", "c++":
		cmds = [][]string{{"cc", "--version"}, {"make", "--version"}}
	case "c#":
		cmds = [][]string{{"dotnet", "--version"}}
	default:
		return ""
	}
	var out []string
	for _, c := range cmds {
		out = append(out, firstLine(c...))
	}
	return strings.Join(out, "; ")
}

// withBuildTool adds the version of a custom build command's executable
// (e.g. uv, bun, yarn) when the language defaults don't already cover it.
func withBuildTool(toolchain, buildCmd string) string {
	fields := strings.Fields(buildCmd)
	if len(fields) == 0 {
		return toolchain
	}
	tool := filepath.Base(fields[0])
	for _, part := range strings.Split(toolchain, "; ") {
		if strings.HasPrefix(strings.ToLower(part), tool+" ") {
			return toolchain
		}
	}
	v := firstLine(fields[0], "--version")
	if toolchain == "" {
		return v
	}
	return toolchain + "; " + v
}

func firstLine(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return args[0] + ": unavailable"
	}
	return strings.TrimSpace(strings.SplitN(string(b), "\n", 2)[0])
}

// HashSource content-addresses a source tree. It skips .git and builder.yaml
// (the yaml's build-relevant fields are part of Config instead), so a fresh
// clone and a later `builder` run in the same repo hash identically.
func HashSource(dir string) (Source, error) {
	h := sha256.New()
	var locks []Lockfile
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if name == "builder.yaml" {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		var sum string
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			sum = "symlink:" + target
		case info.Mode().IsRegular():
			s, err := hashFile(path)
			if err != nil {
				return err
			}
			sum = s
			if lockfiles[strings.ToLower(name)] {
				locks = append(locks, Lockfile{Path: rel, SHA256: s})
			}
		default:
			return nil
		}
		execBit := "0"
		if info.Mode()&0o111 != 0 {
			execBit = "1"
		}
		io.WriteString(h, rel+"\x00"+execBit+"\x00"+sum+"\n")
		return nil
	})
	if err != nil {
		return Source{}, err
	}
	sort.Slice(locks, func(i, j int) bool { return locks[i].Path < locks[j].Path })
	commit, dirty := gitState(dir)
	return Source{
		Digest:    hex.EncodeToString(h.Sum(nil)),
		Commit:    commit,
		Dirty:     dirty,
		Lockfiles: locks,
	}, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// gitState returns the full commit SHA and whether tracked files differ from it.
func gitState(dir string) (string, bool) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", false
	}
	status, err := exec.Command("git", "-C", dir, "status", "--porcelain", "--untracked-files=no").Output()
	return strings.TrimSpace(string(out)), err == nil && len(strings.TrimSpace(string(status))) > 0
}
