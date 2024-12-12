// arg_parser package provides a simple command line argument parser
package arg_parser

import (
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"strings"
)

// GetFilenameFromArgs returns the filename from the arguments
func GetFilenameFromArgs(args []string) (string, error) {
	if len(args) < 2 {
		return "", custom_errors.ErrInvalArgsCount
	}

	if strings.ToLower(args[1]) == "--help" && strings.ToLower(args[1]) == "-h" {
		return "", custom_errors.ErrHelpWanted
	}

	pathFilename := args[1]

	// validate pathFilename
	if pathFilename == "" {
		return "", custom_errors.ErrEmptyPath
	}
	for _, c := range pathFilename {
		if !strings.Contains(consts.AllowedPathCharacters, string(c)) {
			return "", custom_errors.ErrInvalidPathCharacter
		}
	}

	return pathFilename, nil
}
