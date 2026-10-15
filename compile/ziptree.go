package compile

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// zipEntry is one file to archive: its name inside the zip and where to read it.
type zipEntry struct {
	name string
	path string
	size int64
}

// largeZipFile is the size above which a file is streamed through the zip
// writer instead of being compressed in parallel in memory.
const largeZipFile = 64 << 20

// zipTree writes every file under root into zipPath, in the same order and with
// the same names as the original recursive walk (sorted per directory,
// depth-first), skipping .git. Symlinks to files store the target's content;
// symlinks to directories store the target's files under the link's name, once
// (loops are skipped). Entries carry no timestamps, so the zip is
// reproducible. Files are deflated in parallel at the default level (so
// sizes match the previous serial writer) and written in order.
func zipTree(zipPath, root string) error {
	var entries []zipEntry
	visited := map[string]bool{}
	var walk func(dir, prefix string) error
	walk = func(dir, prefix string) error {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return err
		}
		if visited[real] {
			return nil // symlink loop
		}
		visited[real] = true
		defer delete(visited, real)

		list, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range list {
			path := filepath.Join(dir, e.Name())
			info, err := os.Stat(path) // follows symlinks
			if err != nil {
				continue // dangling symlink
			}
			if info.IsDir() {
				if e.Name() == ".git" {
					// Git metadata isn't part of the artifact, and its index and
					// logs change on every clone, which would make zips
					// non-reproducible.
					continue
				}
				if err := walk(path, prefix+e.Name()+"/"); err != nil {
					return err
				}
			} else if info.Mode().IsRegular() {
				entries = append(entries, zipEntry{prefix + e.Name(), path, info.Size()})
			}
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return err
	}

	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()
	w := zip.NewWriter(out)
	w.RegisterCompressor(zip.Deflate, func(dst io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(dst, flate.DefaultCompression)
	})

	type result struct {
		hdr  zip.FileHeader
		data []byte
		err  error
	}
	workers := runtime.NumCPU()
	results := make([]chan result, len(entries))
	for i := range results {
		results[i] = make(chan result, 1)
	}
	jobs := make(chan int)
	slots := make(chan struct{}, workers*4) // bounds compressed data held in memory

	for n := 0; n < workers; n++ {
		go func() {
			for i := range jobs {
				e := entries[i]
				if e.size > largeZipFile {
					results[i] <- result{} // written by streaming below
					continue
				}
				raw, err := os.ReadFile(e.path)
				if err != nil {
					results[i] <- result{err: err}
					continue
				}
				var buf bytes.Buffer
				fw, _ := flate.NewWriter(&buf, flate.DefaultCompression)
				fw.Write(raw)
				fw.Close()
				results[i] <- result{hdr: zip.FileHeader{
					Name: e.name, Method: zip.Deflate, CRC32: crc32.ChecksumIEEE(raw),
					CompressedSize64: uint64(buf.Len()), UncompressedSize64: uint64(len(raw)),
				}, data: buf.Bytes()}
			}
		}()
	}
	go func() {
		for i := range entries {
			slots <- struct{}{}
			jobs <- i
		}
		close(jobs)
	}()

	var firstErr error
	for i, e := range entries {
		r := <-results[i]
		<-slots
		if firstErr != nil {
			continue // keep draining so workers finish
		}
		switch {
		case r.err != nil:
			firstErr = r.err
		case e.size > largeZipFile:
			firstErr = streamZipFile(w, e)
		default:
			hdr := r.hdr
			var fw io.Writer
			if fw, firstErr = w.CreateRaw(&hdr); firstErr == nil {
				_, firstErr = fw.Write(r.data)
			}
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return w.Close()
}

func streamZipFile(w *zip.Writer, e zipEntry) error {
	f, err := os.Open(e.path)
	if err != nil {
		return err
	}
	defer f.Close()
	fw, err := w.CreateHeader(&zip.FileHeader{Name: e.name, Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, f)
	return err
}

// moveFile renames src to dst, copying only when they're on different
// filesystems.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}
