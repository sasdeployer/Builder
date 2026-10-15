package provenance

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	cp "github.com/otiai10/copy"
)

// Files Builder writes next to artifacts; they are never cached as artifacts.
var builderFiles = map[string]bool{
	"metadata.json": true, "metadata.yaml": true, "provenance.json": true,
}

// Entry is the manifest stored with each cached build.
type Entry struct {
	Key          string            `json:"key"`
	Inputs       Inputs            `json:"inputs"`
	Artifacts    map[string]string `json:"artifacts"` // name -> sha256
	InvocationID string            `json:"invocationId"`
	CreatedAt    time.Time         `json:"createdAt"`
}

// Enabled reports whether the cache may be used for this run.
// BUILDER_CACHE=off disables it. Compressed artifacts embed the build
// timestamp in their names, and -D needs a populated workspace, so both bypass it.
func Enabled() (bool, string) {
	if os.Getenv("BUILDER_CACHE") == "off" {
		return false, "BUILDER_CACHE=off"
	}
	if os.Getenv("ARTIFACT_ZIP_ENABLED") == "true" {
		return false, "compressed artifacts are not cached"
	}
	for _, a := range os.Args[1:] {
		if a == "-D" || a == "--docker" {
			return false, "docker builds are not cached"
		}
	}
	return true, ""
}

// Dir is the cache root: $BUILDER_CACHE_DIR, else Builder's data dir + /cache.
func Dir() string {
	if d := os.Getenv("BUILDER_CACHE_DIR"); d != "" {
		return d
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("APPDATA")
		}
		return filepath.Join(base, "Builder", "cache")
	}
	u, err := user.Current()
	if err != nil {
		return filepath.Join(os.TempDir(), "builder-cache")
	}
	return filepath.Join(u.HomeDir, ".builder", "cache")
}

// Lookup returns the cached entry for key after re-verifying every artifact's checksum.
func Lookup(key string) (*Entry, bool) {
	dir := filepath.Join(Dir(), key)
	b, err := os.ReadFile(filepath.Join(dir, "entry.json"))
	if err != nil {
		return nil, false
	}
	var e Entry
	if json.Unmarshal(b, &e) != nil || e.Key != key || len(e.Artifacts) == 0 {
		return nil, false
	}
	for name, want := range e.Artifacts {
		got, err := hashPath(filepath.Join(dir, "artifacts", name))
		if err != nil || got != want {
			return nil, false
		}
	}
	return &e, true
}

// Restore copies a cached build's artifacts into artifactDir and returns their names.
func Restore(e *Entry, artifactDir string) ([]string, error) {
	src := filepath.Join(Dir(), e.Key, "artifacts")
	names := sortedNames(e.Artifacts)
	for _, name := range names {
		if err := cp.Copy(filepath.Join(src, name), filepath.Join(artifactDir, name)); err != nil {
			return nil, err
		}
	}
	return names, nil
}

// Store saves the artifacts in artifactDir under key. It writes to a temp dir
// and renames, so concurrent builds never see a half-written entry.
func Store(in Inputs, invocationID, artifactDir string) error {
	key := in.Key()
	arts, err := Artifacts(artifactDir)
	if err != nil {
		return err
	}
	if len(arts) == 0 {
		return fmt.Errorf("no artifacts in %s", artifactDir)
	}
	root := Dir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	final := filepath.Join(root, key)
	if _, err := os.Stat(final); err == nil {
		if _, ok := Lookup(key); ok {
			return nil
		}
		// Corrupt or tampered entry: replace it.
		if err := os.RemoveAll(final); err != nil {
			return err
		}
	}
	tmp, err := os.MkdirTemp(root, ".tmp-"+key[:12]+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	for name := range arts {
		if err := cp.Copy(filepath.Join(artifactDir, name), filepath.Join(tmp, "artifacts", name)); err != nil {
			return err
		}
	}
	b, _ := json.MarshalIndent(Entry{
		Key: key, Inputs: in, Artifacts: arts, InvocationID: invocationID, CreatedAt: time.Now().UTC(),
	}, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, "entry.json"), b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, final); err != nil && !os.IsExist(err) {
		if _, statErr := os.Stat(final); statErr == nil {
			return nil // another build stored the same key first
		}
		return err
	}
	return nil
}

// Artifacts returns name -> sha256 for every artifact in dir (top level only).
// A directory artifact is hashed over its contents.
func Artifacts(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, e := range entries {
		if builderFiles[e.Name()] {
			continue
		}
		sum, err := hashPath(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out[e.Name()] = sum
	}
	return out, nil
}

func hashPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		src, err := HashSource(path)
		return src.Digest, err
	}
	return hashFile(path)
}

func sortedNames(m map[string]string) []string {
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
