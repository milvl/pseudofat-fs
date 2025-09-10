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
//
// NOTE: It does not include the root directory in the result.
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

// GetPathBasename returns the last segment of the path
func GetPathBasename(path string) string {
	if path == "" {
		return ""
	}

	segments := strings.Split(path, consts.PathDelimiter)
	return segments[len(segments)-1]
}

// GetPathAndBasename returns the path and the last segment of the path
func GetPathAndBasename(path string) (string, string) {
	if path == "" {
		return "", ""
	}

	segments := strings.Split(path, consts.PathDelimiter)
	return strings.Join(segments[:len(segments)-1], consts.PathDelimiter), segments[len(segments)-1]
}

// GetPathSegments returns a slice of path segments
func GetPathSegments(path string) []string {
	segments := make([]string, 0)
	segments = append(segments, consts.PathDelimiter)
	split := strings.Split(path, consts.PathDelimiter)[1:]
	for _, segment := range split {
		if segment == "" {
			continue
		}
		segments = append(segments, segment)
	}

	return segments
}
