package regolith

import (
	"errors"
	"io/ioutil"

	"github.com/denisbrodbeck/machineid"
)

// Returns true if safe mode is unlocked
func IsUnlocked() bool {
	// TODO - maybe consider caching this result to avoid reading the file every time
	id, err := GetMachineId()
	if err != nil {
		Logger.Info("Failed to get machine ID.", err)
		return false
	}

	lockedId, err := ioutil.ReadFile(".regolith/cache/lockfile.txt")
	if err != nil {
		return false
	}

	unlocked := id == string(lockedId)

	if !unlocked {
		Logger.Info("Safe mode is locked. Unlock it by running `regolith unlock`.")
	}

	return unlocked
}

// Returns machine ID
func GetMachineId() (string, error) {
	id, err := machineid.ProtectedID("regolith")
	if err != nil {
		return "", wrapError(err, "Failed to create unique machine ID.")
	}
	return id, nil
}

// Unlocks safe mode, by signing the machine ID into lockfile.txt
func Unlock() error {

	if !IsProjectInitialized() {
		return errors.New("this does not appear to be a Regolith project")
	}

	id, err := GetMachineId()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(".regolith/cache/lockfile.txt", []byte(id), 0666)
	if err != nil {
		return wrapError(err, "Failed to write lock file.")
	}

	return nil
}
