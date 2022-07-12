package regolith

import (
	"io/ioutil"
	"path/filepath"

	"github.com/denisbrodbeck/machineid"
)

// Returns true if safe mode is unlocked
func IsUnlocked(dotRegolithPath string) bool {
	// TODO - maybe consider caching this result to avoid reading the file every time
	id, err := GetMachineId()
	if err != nil {
		Logger.Info("Failed to get machine ID.", err)
		return false
	}

	lockedId, err := ioutil.ReadFile(filepath.Join(dotRegolithPath, "cache/lockfile.txt"))
	if err != nil {
		return false
	}

	unlocked := id == string(lockedId)

	if !unlocked {
		Logger.Info(
			"Safe mode is locked. Unlock it by running \"regolith unlock\".")
	}

	return unlocked
}

// Returns machine ID
func GetMachineId() (string, error) {
	id, err := machineid.ProtectedID("regolith")
	if err != nil {
		return "", WrapError(err, "Failed to create unique machine ID.")
	}
	return id, nil
}
