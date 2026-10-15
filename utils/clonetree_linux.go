package utils

import "os/exec"

// cloneEntry copies with reflinks where the filesystem supports them (Btrfs,
// XFS, overlay on those); cp falls back to a regular copy otherwise.
func cloneEntry(src, dst string) error {
	return exec.Command("cp", "-a", "--reflink=auto", src, dst).Run()
}
