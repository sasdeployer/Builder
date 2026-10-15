package utils

import (
	"Builder/spinner"
	"os"
	"path/filepath"
)

// CloneRepoFiles copies the repo (or the current dir) into the hidden source
// snapshot, leaving out Builder's builds dir at any depth.
func CloneRepoFiles(from string, to string) {
	//Get absolute paths in case relative paths are given
	fromPath, _ := filepath.Abs(from)
	toPath, _ := filepath.Abs(to)

	buildsDir := "builder"                     // Builds are stored in default named 'builder' folder
	if os.Getenv("BUILDER_BUILDS_DIR") != "" { // A different folder to store builds is provided
		buildsDir = os.Getenv("BUILDER_BUILDS_DIR")
	}
	skip := func(name string) bool { return name == buildsDir }
	if err := CloneTree(fromPath, toPath, skip); err != nil {
		spinner.LogMessage(err.Error(), "fatal")
	}
}
