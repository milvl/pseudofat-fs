// util package provides utility functions for the project
package utils

import (
	"kiv-zos-semestral-work/custom_errors"
	"os"
)

// FilepathValid checks if a file exists and is not a directory
func FilepathValid(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else if info.IsDir() {
		return false, custom_errors.ErrIsDir
	}

	return true, nil
}
