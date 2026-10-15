// Package output gives machine callers (agents, CI, other services) one JSON
// result per run, built from the record Builder already writes: metadata.json,
// provenance.json and logs.json.
package output

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var (
	jsonMode bool
	stdout   = os.Stdout
	started  = time.Now()
)

// init enables JSON mode for `--json` or BUILDER_OUTPUT=json, before any other
// package prints. In JSON mode the real stdout is reserved for the single
// result; everything else Builder prints goes to stderr.
func init() {
	for _, a := range os.Args[1:] {
		if a == "--json" {
			jsonMode = true
		}
	}
	if os.Getenv("BUILDER_OUTPUT") == "json" {
		jsonMode = true
	}
	if jsonMode {
		stdout = os.Stdout
		os.Stdout = os.Stderr
	}
}

// JSON reports whether a machine-readable result was requested.
func JSON() bool { return jsonMode }

// Interactive reports whether Builder may prompt: not in JSON mode, and only
// when stdin is a terminal.
func Interactive() bool {
	if jsonMode {
		return false
	}
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// TTY reports whether stderr/stdout is a terminal (for the spinner).
func TTY() bool {
	if jsonMode {
		return false
	}
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

type artifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type failure struct {
	Message string   `json:"message"`
	At      string   `json:"at,omitempty"`
	LogTail []string `json:"logTail,omitempty"`
}

type result struct {
	Status         string          `json:"status"`
	Command        string          `json:"command"`
	DurationMs     int64           `json:"durationMs"`
	Project        string          `json:"project,omitempty"`
	ProjectType    string          `json:"projectType,omitempty"`
	BuildID        string          `json:"buildId,omitempty"`
	ArtifactDir    string          `json:"artifactDir,omitempty"`
	Artifacts      []artifact      `json:"artifacts,omitempty"`
	MetadataPath   string          `json:"metadataPath,omitempty"`
	ProvenancePath string          `json:"provenancePath,omitempty"`
	LogsPath       string          `json:"logsPath,omitempty"`
	Cache          json.RawMessage `json:"cache,omitempty"`
	Determinism    json.RawMessage `json:"determinism,omitempty"`
	Docker         json.RawMessage `json:"docker,omitempty"`
	Error          *failure        `json:"error,omitempty"`
}

// Success writes the result for a finished build (JSON mode only).
func Success() {
	if !jsonMode {
		return
	}
	r := base("success")
	dir := abs(os.Getenv("BUILDER_ARTIFACT_DIR"))
	if dir != "" {
		r.ArtifactDir = dir
		if p := filepath.Join(dir, "metadata.json"); exists(p) {
			r.MetadataPath = p
			var m map[string]interface{}
			if b, err := os.ReadFile(p); err == nil && json.Unmarshal(b, &m) == nil {
				// Same ID Builder stores in builds.json (utils.GetBuildID).
				sum := sha256.Sum256(b)
				r.BuildID = hex.EncodeToString(sum[:])[:9]
				if v, ok := m["BuildID"].(string); ok {
					r.BuildID = v
				}
				if v, ok := m["DockerRepositoryTag"]; ok {
					d, _ := json.Marshal(map[string]interface{}{
						"repositoryTag": v, "repository": m["DockerRepository"], "tags": m["DockerTags"],
					})
					r.Docker = d
				}
			}
		}
		if p := filepath.Join(dir, "provenance.json"); exists(p) {
			r.ProvenancePath = p
			r.Cache, r.Determinism = provenanceDecisions(p)
		}
		r.Artifacts = artifacts(dir)
	}
	emit(r)
}

// Failure writes the result for a failed run (JSON mode only).
func Failure(msg, at string) {
	if !jsonMode {
		return
	}
	r := base("failed")
	r.Error = &failure{Message: msg, At: at, LogTail: logTail(r.LogsPath, 20)}
	emit(r)
}

func base(status string) result {
	cmd := "builder"
	if len(os.Args) > 1 && (os.Args[1] == "init" || os.Args[1] == "config" || os.Args[1] == "docker") {
		cmd = os.Args[1]
	}
	r := result{
		Status: status, Command: cmd, DurationMs: time.Since(started).Milliseconds(),
		Project: os.Getenv("BUILDER_DIR_NAME"), ProjectType: os.Getenv("BUILDER_PROJECT_TYPE"),
	}
	if logs := os.Getenv("BUILDER_LOGS_DIR"); logs != "" {
		if p := abs(filepath.Join(logs, "logs.json")); exists(p) {
			r.LogsPath = p
		}
	}
	return r
}

func emit(r result) {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.Encode(r)
}

func provenanceDecisions(path string) (json.RawMessage, json.RawMessage) {
	var st struct {
		Predicate struct {
			RunDetails struct {
				Byproducts []struct {
					Name        string          `json:"name"`
					Annotations json.RawMessage `json:"annotations"`
				} `json:"byproducts"`
			} `json:"runDetails"`
		} `json:"predicate"`
	}
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &st) != nil {
		return nil, nil
	}
	var cache, det json.RawMessage
	for _, bp := range st.Predicate.RunDetails.Byproducts {
		switch bp.Name {
		case "cache":
			cache = bp.Annotations
		case "determinism":
			det = bp.Annotations
		}
	}
	return cache, det
}

func artifacts(dir string) []artifact {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []artifact
	for _, e := range entries {
		switch e.Name() {
		case "metadata.json", "metadata.yaml", "provenance.json":
			continue
		}
		p := filepath.Join(dir, e.Name())
		info, err := os.Stat(p)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		h := sha256.New()
		io.Copy(h, f)
		f.Close()
		out = append(out, artifact{e.Name(), p, hex.EncodeToString(h.Sum(nil)), info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// logTail returns the last n messages from a zap JSON log file.
func logTail(path string, n int) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var l struct {
			Msg string `json:"msg"`
		}
		if json.Unmarshal(sc.Bytes(), &l) == nil && l.Msg != "" {
			lines = append(lines, l.Msg)
			if len(lines) > n {
				lines = lines[1:]
			}
		}
	}
	return lines
}

func abs(p string) string {
	if p == "" {
		return ""
	}
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
