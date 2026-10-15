package compile

import (
	"Builder/spinner"
	"os/exec"
	"strings"
)

// runChecked runs a helper command (cp, rm, ...) and reports failure at the
// given log level ("fatal" stops the build).
func runChecked(level string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		spinner.LogMessage(cmd.String()+" failed: "+err.Error()+": "+strings.TrimSpace(string(out)), level)
	}
}
