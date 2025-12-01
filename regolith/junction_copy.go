package regolith

import (
	"runtime"
	"strings"

	"github.com/otiai10/copy"
)

// CopyDirOrFallback attempts to copy a directory using the upstream copy library.
// On Windows, NTFS junctions inside the directory may cause an "Incorrect function" error
// when misclassified by lstat. If that specific error occurs, fall back to SyncDirectories
// which performs a manual recursive copy and treats ModeIrregular reparse points as directories.
func CopyDirOrFallback(src, dst string) error {
	err := copy.Copy(src, dst, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		if runtime.GOOS == "windows" && strings.Contains(err.Error(), "Incorrect function") {
			// Fallback path for Windows junctions
			return SyncDirectories(src, dst, false)
		}
		return err
	}
	return nil
}
