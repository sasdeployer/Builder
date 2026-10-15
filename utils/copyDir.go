package utils

import (
	"Builder/spinner"
	"os"
)

// CopyDir copies the contents of the hidden dir into the workspace dir.
func CopyDir() {
	hiddenDir := os.Getenv("BUILDER_HIDDEN_DIR")
	workspaceDir := os.Getenv("BUILDER_WORKSPACE_DIR")
	if err := CloneTree(hiddenDir, workspaceDir, nil); err != nil {
		spinner.LogMessage("copying source into workspace failed: "+err.Error(), "fatal")
	}
}
