package provenance

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

const (
	statementType = "https://in-toto.io/Statement/v1"
	predicateType = "https://slsa.dev/provenance/v1"
	buildType     = "https://github.com/sasdeployer/Builder/buildtypes/local/v1"
	builderID     = "https://github.com/sasdeployer/Builder"
)

// Run describes one Builder invocation.
type Run struct {
	InvocationID string
	Started      time.Time
	Finished     time.Time
	Source       Source
	Inputs       Inputs
	RepoURL      string
	Branch       string
	LogsPath     string
	CacheHit     bool
	CachedFrom   string // invocation that originally produced a cache hit's artifacts
	Locked       bool   // dependencies verifiably pinned (see LockStatus)
	LockReason   string
}

// NewInvocationID returns a random ID for a build run.
func NewInvocationID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Requester is who asked for the build: a person or an agent session.
type Requester struct {
	Kind     string `json:"kind"` // "human" or "agent"
	ID       string `json:"id,omitempty"`
	Agent    string `json:"agent,omitempty"`
	Session  string `json:"session,omitempty"`
	Attended string `json:"attended,omitempty"`
	Task     string `json:"task,omitempty"`
}

// DetectRequester prefers an explicit BUILDER_REQUESTED_BY, then known agent
// environments, then the local OS user.
func DetectRequester() Requester {
	r := Requester{Task: os.Getenv("BUILDER_TASK")}
	switch {
	case os.Getenv("BUILDER_REQUESTED_BY") != "":
		r.ID = os.Getenv("BUILDER_REQUESTED_BY")
		r.Kind = os.Getenv("BUILDER_REQUESTER_KIND")
		if r.Kind == "" {
			r.Kind = "human"
		}
	case os.Getenv("CLAUDECODE") == "1":
		r.Kind = "agent"
		r.Agent = os.Getenv("AI_AGENT")
		if r.Agent == "" {
			r.Agent = "claude-code"
		}
		r.Session = os.Getenv("CLAUDE_CODE_SESSION_ID")
		r.Attended = os.Getenv("CLAUDE_CODE_SESSION_ATTENDED")
	case os.Getenv("AI_AGENT") != "":
		r.Kind = "agent"
		r.Agent = os.Getenv("AI_AGENT")
	default:
		r.Kind = "human"
		if u, err := user.Current(); err == nil {
			r.ID = u.Username
		}
	}
	return r
}

type resourceDescriptor struct {
	Name        string            `json:"name,omitempty"`
	URI         string            `json:"uri,omitempty"`
	Digest      map[string]string `json:"digest,omitempty"`
	Annotations interface{}       `json:"annotations,omitempty"`
}

// Write emits provenance.json (an unsigned in-toto Statement carrying a SLSA
// v1 provenance predicate) for every artifact in artifactDir.
func Write(artifactDir string, run Run) error {
	arts, err := Artifacts(artifactDir)
	if err != nil {
		return err
	}
	var subjects []resourceDescriptor
	for _, name := range sortedNames(arts) {
		subjects = append(subjects, resourceDescriptor{Name: name, Digest: map[string]string{"sha256": arts[name]}})
	}

	srcURI := run.RepoURL
	if srcURI != "" && !strings.HasPrefix(srcURI, "git+") {
		srcURI = "git+" + srcURI
	}
	if run.Branch != "" && srcURI != "" {
		srcURI += "@refs/heads/" + run.Branch
	}
	srcDigest := map[string]string{"sha256": run.Source.Digest}
	if run.Source.Commit != "" {
		srcDigest["gitCommit"] = run.Source.Commit
	}
	deps := []resourceDescriptor{{
		Name: "source", URI: srcURI, Digest: srcDigest,
		Annotations: map[string]bool{"dirty": run.Source.Dirty},
	}}
	for _, l := range run.Source.Lockfiles {
		deps = append(deps, resourceDescriptor{Name: l.Path, Digest: map[string]string{"sha256": l.SHA256}})
	}

	byproducts := []resourceDescriptor{
		{Name: "requester", Annotations: DetectRequester()},
		{Name: "cache", Annotations: map[string]interface{}{
			"key": run.Inputs.Key(), "hit": run.CacheHit, "producedBy": run.CachedFrom,
		}},
		{Name: "determinism", Annotations: map[string]interface{}{
			"dependenciesLocked": run.Locked, "reason": run.LockReason,
			"sourceDateEpoch": os.Getenv("SOURCE_DATE_EPOCH"), "goflags": os.Getenv("GOFLAGS"),
		}},
	}
	if run.LogsPath != "" {
		if abs, err := filepath.Abs(run.LogsPath); err == nil {
			byproducts = append(byproducts, resourceDescriptor{Name: "logs", URI: "file://" + filepath.ToSlash(abs)})
		}
	}

	stmt := map[string]interface{}{
		"_type":         statementType,
		"subject":       subjects,
		"predicateType": predicateType,
		"predicate": map[string]interface{}{
			"buildDefinition": map[string]interface{}{
				"buildType":          buildType,
				"externalParameters": map[string]interface{}{"repository": run.RepoURL, "branch": run.Branch, "config": run.Inputs.Config},
				"internalParameters": map[string]interface{}{
					"platform": run.Inputs.Platform, "toolchain": run.Inputs.Toolchain, "buildEnv": run.Inputs.BuildEnv,
				},
				"resolvedDependencies": deps,
			},
			"runDetails": map[string]interface{}{
				"builder": map[string]interface{}{"id": builderID, "version": map[string]string{"builder": run.Inputs.BuilderVersion}},
				"metadata": map[string]string{
					"invocationId": run.InvocationID,
					"startedOn":    run.Started.UTC().Format(time.RFC3339Nano),
					"finishedOn":   run.Finished.UTC().Format(time.RFC3339Nano),
				},
				"byproducts": byproducts,
			},
		},
	}
	b, err := json.MarshalIndent(stmt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(artifactDir, "provenance.json"), b, 0o644)
}
