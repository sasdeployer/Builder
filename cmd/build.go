package cmd

import (
	"Builder/artifact"
	"Builder/derive"
	"Builder/directory"
	"Builder/provenance"
	"Builder/spinner"
	"Builder/utils"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	cp "github.com/otiai10/copy"
)

// runBuild derives the project type and compiles it, unless an identical build
// (same source, config, toolchain and platform) is already cached. Either way
// it writes provenance.json next to the artifacts.
func runBuild() {
	started := time.Now()
	invocationID := provenance.NewInvocationID()

	cacheOn, why := provenance.Enabled()
	src, err := provenance.HashSource(os.Getenv("BUILDER_HIDDEN_DIR"))
	if err != nil {
		spinner.LogMessage("Could not hash source, skipping cache: "+err.Error(), "warn")
		cacheOn = false
	}
	if !cacheOn && why != "" {
		spinner.LogMessage("Build cache skipped: "+why, "info")
	}

	// The key needs the project type, which is only known up front when
	// builder.yaml supplies it (`builder` and `builder config`).
	if cacheOn && os.Getenv("BUILDER_PROJECT_TYPE") != "" {
		in := provenance.NewInputs(src, provenance.ConfigFromEnv())
		if entry, ok := provenance.Lookup(in.Key()); ok {
			restoreCachedBuild(entry, src, in, invocationID, started)
			return
		}
	}

	derive.ProjectType()

	// Compilers fill in defaults (project type, build command), so compute the key after.
	in := provenance.NewInputs(src, provenance.ConfigFromEnv())
	artifactDir := os.Getenv("BUILDER_ARTIFACT_DIR")
	writeProvenance(artifactDir, provenance.Run{
		InvocationID: invocationID, Started: started, Finished: time.Now(),
		Source: src, Inputs: in,
	})
	if cacheOn {
		if err := provenance.Store(in, invocationID, artifactDir); err != nil {
			spinner.LogMessage("Could not cache build: "+err.Error(), "warn")
		} else {
			spinner.LogMessage("Build cached under key "+in.Key()[:12], "info")
		}
	}
}

// restoreCachedBuild lays out the same parent/artifact dirs a compiler would,
// then fills them from the cache instead of compiling.
func restoreCachedBuild(entry *provenance.Entry, src provenance.Source, in provenance.Inputs, invocationID string, started time.Time) {
	now := time.Now().Format(time.RFC850)
	os.Setenv("BUILD_START_TIME", now)
	os.Setenv("BUILD_END_TIME", now)
	directory.UpdateParentDirName("")
	artifact.ArtifactDir()
	artifactDir := os.Getenv("BUILDER_ARTIFACT_DIR")

	names, err := provenance.Restore(entry, artifactDir)
	if err != nil {
		spinner.LogMessage("Could not restore cached build: "+err.Error(), "fatal")
	}
	os.Setenv("BUILDER_ARTIFACT_NAMES", strings.Join(names, ","))

	if outputPath := os.Getenv("BUILDER_OUTPUT_PATH"); outputPath != "" {
		for _, name := range names {
			if err := cp.Copy(filepath.Join(artifactDir, name), filepath.Join(outputPath, name)); err != nil {
				spinner.LogMessage(err.Error(), "warn")
			}
		}
	}

	msg := "cache hit: key " + entry.Key + ", artifacts from build " + entry.InvocationID
	writeCacheHitLog(msg)
	spinner.LogMessage(msg, "info")

	utils.Metadata(artifactDir)
	writeProvenance(artifactDir, provenance.Run{
		InvocationID: invocationID, Started: started, Finished: time.Now(),
		Source: src, Inputs: in, CacheHit: true, CachedFrom: entry.InvocationID,
	})
}

// writeCacheHitLog keeps logs/logs.json present (push reads it) with one line
// in the same shape as the compilers' zap output.
func writeCacheHitLog(msg string) {
	line, _ := json.Marshal(map[string]string{
		"level": "info", "timestamp": time.Now().Format("2006-01-02T15:04:05.000-0700"),
		"caller": "cmd/build.go", "msg": msg,
	})
	path := filepath.Join(os.Getenv("BUILDER_LOGS_DIR"), "logs.json")
	if err := os.WriteFile(path, append(line, '\n'), 0o644); err != nil {
		spinner.LogMessage("Could not write logs.json: "+err.Error(), "warn")
	}
}

func writeProvenance(artifactDir string, run provenance.Run) {
	run.RepoURL = utils.GetRepoURL()
	run.Branch = os.Getenv("REPO_BRANCH_NAME")
	run.LogsPath = filepath.Join(os.Getenv("BUILDER_LOGS_DIR"), "logs.json")
	if err := provenance.Write(artifactDir, run); err != nil {
		spinner.LogMessage("Could not write provenance.json: "+err.Error(), "warn")
		return
	}
	spinner.LogMessage("Provenance written.", "info")
}
