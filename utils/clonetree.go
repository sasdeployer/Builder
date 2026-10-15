package utils

import (
	"io/fs"
	"os"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

// CloneTree copies the contents of src into dst, like copying "src/." with
// otiai10/copy, but using copy-on-write clones where the filesystem supports
// them (APFS clonefile, Linux reflinks), so a snapshot of a large tree costs
// almost nothing until a file is modified. skip, if set, excludes entries by
// name at any depth, exactly as before; directories that contain a skipped
// entry are walked into instead of cloned whole. Entries that already exist in
// dst, or that can't be cloned, fall back to a regular copy.
func CloneTree(src, dst string, skip func(name string) bool) error {
	// Directories (relative to src) that contain a skipped entry somewhere below.
	hasSkipped := map[string]bool{}
	if skip != nil {
		err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != src && skip(d.Name()) {
				for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
					rel, _ := filepath.Rel(src, dir)
					hasSkipped[rel] = true
					if dir == src || dir == filepath.Dir(dir) {
						break
					}
				}
				if d.IsDir() {
					return filepath.SkipDir
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return cloneDir(src, dst, ".", skip, hasSkipped)
}

func cloneDir(srcRoot, dst, rel string, skip func(string) bool, hasSkipped map[string]bool) error {
	srcDir := filepath.Join(srcRoot, rel)
	dstDir := filepath.Join(dst, rel)
	info, err := os.Stat(srcDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, info.Mode().Perm()|0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if skip != nil && skip(e.Name()) {
			continue
		}
		childRel := filepath.Join(rel, e.Name())
		if e.IsDir() && hasSkipped[childRel] {
			if err := cloneDir(srcRoot, dst, childRel, skip, hasSkipped); err != nil {
				return err
			}
			continue
		}
		s, d := filepath.Join(srcRoot, childRel), filepath.Join(dst, childRel)
		if _, err := os.Lstat(d); err != nil && cloneEntry(s, d) == nil {
			continue
		}
		if err := cp.Copy(s, d); err != nil {
			return err
		}
	}
	return nil
}
