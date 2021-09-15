package src

import (
	"errors"
	"fmt"
)

func StringArrayContains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func wrapError(text string, err error) error {
	return errors.New(fmt.Sprintf("%s\nCaused by: %s", text, err.Error()))
}
