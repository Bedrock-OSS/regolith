package regolith

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/otiai10/copy"
)

// CopyDirOrFallback attempts to copy a directory using the upstream copy library.
// On Windows, NTFS junctions (directory reparse points) inside the directory can
// cause issues for libraries that rely on generic lstat checks. To improve reliability,
// we pre-scan for junctions and use SyncDirectories when detected, and otherwise
// fallback only on specific Windows error messages.
func CopyDirOrFallback(src, dst string) error {
	if runtime.GOOS == "windows" {
		hasJunc, _ := hasWindowsJunction(src)
		if hasJunc {
			return SyncDirectories(src, dst, false)
		}
	}

	err := copy.Copy(src, dst, copy.Options{PreserveTimes: false, Sync: false})
	if err == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		// Fallback for common junction-related errors observed on Windows.
		msg := err.Error()
		if strings.Contains(msg, "Incorrect function") ||
			strings.Contains(msg, "reparse") ||
			strings.Contains(msg, "The system cannot find the path specified") {
			return SyncDirectories(src, dst, false)
		}
	}
	return err
}

// hasWindowsJunction walks the directory and returns true if any entry is a Windows
// NTFS junction (directory reparse point). It stops early on first detection.
func hasWindowsJunction(root string) (bool, error) {
	var junctionFound bool
	var sentinel = errors.New("junction-found")
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if junctionFound {
			return sentinel
		}
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return statErr
		}
		if wad, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
			const FILE_ATTRIBUTE_DIRECTORY = 0x10
			const FILE_ATTRIBUTE_REPARSE_POINT = 0x0400
			if wad.FileAttributes&FILE_ATTRIBUTE_REPARSE_POINT != 0 && wad.FileAttributes&FILE_ATTRIBUTE_DIRECTORY != 0 {
				junctionFound = true
				return sentinel // abort walk early
			}
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, sentinel) {
		return junctionFound, walkErr
	}
	return junctionFound, nil
}
