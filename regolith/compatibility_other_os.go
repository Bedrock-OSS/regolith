//go:build !windows
// +build !windows

package regolith

// venvScriptsPath is a folder name between "venv" and "python" that leads to
// the python executable.
const venvScriptsPath = "bin"

// exeSuffix is a suffix for executable files.
const exeSuffix = ""

// copyFileSecurityInfo placeholder for a function which is necessary only
// on Windows.
func copyFileSecurityInfo(source string, target string) error {
	return nil
}

type DirWatcher struct;

func NewDirWatcher(path string) (*DirWatcher, error) {
	return nil, fmt.Errorf("Not implemented for this system.")
}

func (d *DirWatcher) WaitForChange() error {
	return fmt.Errorf("Not implemented for this system.")
}

func (d *DirWatcher) WaitForChangeGroup(groupTimeout uint32) error {
	return fmt.Errorf("Not implemented for this system.")
}

func (d *DirWatcher) Close() error {
	return fmt.Errorf("Not implemented for this system.")
}
