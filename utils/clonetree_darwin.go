package utils

import "golang.org/x/sys/unix"

// cloneEntry clones a file, symlink or whole directory tree with clonefile(2)
// (APFS copy-on-write). It fails on filesystems without clone support.
func cloneEntry(src, dst string) error {
	return unix.Clonefile(src, dst, unix.CLONE_NOFOLLOW)
}
