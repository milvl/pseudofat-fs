// util package provides utility functions for the project
package utils

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"os"
	"strings"
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

// GetNormalizedPathNodes processes the absolute path and returns a slice of normalized path nodes.
// Handles "." and ".." segments.
func GetNormalizedPathNodes(absPath string) ([]string, error) {
	if !strings.HasPrefix(absPath, consts.PathDelimiter) {
		return nil, fmt.Errorf("path must be absolute")
	}

	// trim leading and trailing delimiters and split the path
	trimmedPath := strings.Trim(absPath, consts.PathDelimiter)
	segments := strings.Split(trimmedPath, consts.PathDelimiter)

	var stack []string
	for _, segment := range segments {
		switch segment {
		case "", consts.CurrDirSymbol:
			// skip empty or current directory segments
			continue

		case consts.ParentDirSymbol:
			if len(stack) > 0 {
				// pop the last valid directory if possible
				stack = stack[:len(stack)-1]
			}
			// if stack is empty, we're at root; do not pop further

		default:
			// push the valid directory segment onto the stack
			stack = append(stack, segment)
		}
	}

	return stack, nil
}
