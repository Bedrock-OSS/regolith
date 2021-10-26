package regolith

import (
	"fmt"
	"io/ioutil"

	"github.com/denisbrodbeck/machineid"
)

// Returns true if safe mode is unlocked
func IsUnlocked() bool {
	id, err := GetMachineId()
	if err != nil {
		fmt.Println("Failed to get machine ID.", err)
		return false
	}

	lockedId, err := ioutil.ReadFile(".regolith/cache/lockfile.txt")
	if err != nil {
		return false
	}

	unlocked := id == string(lockedId)

	if !unlocked {
		fmt.Println("Safe mode is locked. Unlock it by running `regolith unlock`.")
	}

	return unlocked
}

// Returns machine ID
func GetMachineId() (string, error) {
	id, err := machineid.ProtectedID("regolith")
	if err != nil {
		return "", wrapError("Failed to create unique machine ID.", err)
	}
	return id, nil
}

// Unlocks safe mode, by signing the machine ID into lockfile.txt
func Unlock() error {

	if !IsProjectConfigured() {
		return fmt.Errorf("This does not appear to be a Regolith project.")
	}

	id, err := GetMachineId()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(".regolith/cache/lockfile.txt", []byte(id), 0666)
	if err != nil {
		return wrapError("Failed to write lock file.", err)
	}

	return nil
}
