package utils

import (
	"Builder/spinner"
	"os"
	"os/exec"
	"strings"
)

// CopyDir copies the contents of the hidden dir into the workspace dir.
func CopyDir() {
	hiddenDir := os.Getenv("BUILDER_HIDDEN_DIR")
	workspaceDir := os.Getenv("BUILDER_WORKSPACE_DIR")
	cmd := exec.Command("cp", "-a", hiddenDir+"/.", workspaceDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		spinner.LogMessage("copying source into workspace failed: "+err.Error()+": "+strings.TrimSpace(string(out)), "fatal")
	}
}
