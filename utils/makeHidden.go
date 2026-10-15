package utils

import (
	"Builder/spinner"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// MakeHidden hides the source snapshot dir (Windows) and makes it read-only.
func MakeHidden() {
	hiddenDir := os.Getenv("BUILDER_HIDDEN_DIR")
	if os.Getenv("HIDDEN_DIR_ENABLED") != "true" {
		return
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("attrib", "+h", "+r", hiddenDir, "/s", "/d")
	} else {
		// a-w keeps directories traversable (0444 made them unreadable).
		cmd = exec.Command("chmod", "-R", "a-w", hiddenDir)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		spinner.LogMessage("could not make hidden dir read-only: "+err.Error()+": "+strings.TrimSpace(string(out)), "warn")
	}
}
