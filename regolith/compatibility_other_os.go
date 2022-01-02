//go:build !windows
// +build !windows

package regolith

// copyFileSecurityInfo placeholder for a function which is necessary only
// on Windows.
func copyFileSecurityInfo(source string, target string) error {
	return nil
}
