// command_validator.go contains the implementation of the command validator
package cmd

import (
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"strconv"
	"strings"
)

// floatBitSize is the size of the float to be parsed
var floatBitSize = 64

// validateOneWordCommand validates the current directory command
func validateOneWordCommand(cmd *Command) error {
	// check if the number of arguments is correct
	if len(cmd.Args) != 0 {
		return custom_errors.ErrInvalArgsCount
	}

	return nil
}

// validatePathFormat validates the path
func validatePathFormat(path string) error {
	// check if the path is empty
	if path == "" {
		return custom_errors.ErrEmptyPath
	}

	// check if the path is too long
	if len(path) > consts.MaxFileNameLength {
		return custom_errors.ErrPathTooLong
	}

	// check if the path format is valid
	for _, c := range path {
		if !strings.Contains(consts.AllowedPathCharacters, string(c)) {
			return custom_errors.ErrInvalidPathCharacter
		}
	}

	return nil
}

// validateListCommand validates the list command
func validateListCommand(cmd *Command) error {
	if len(cmd.Args) > 1 {
		return custom_errors.ErrInvalArgsCount

	} else if len(cmd.Args) == 1 {
		path := cmd.Args[0]
		// check if the path format is valid
		err := validatePathFormat(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// getPathsCountForCommand returns the number of paths for the command
func getPathsCountForCommand(cmdName string) (int, error) {
	switch cmdName {
	case
		consts.RemoveCommand,
		consts.MakeDirCommand,
		consts.RemoveDirCommand,
		consts.ConcatCommand,
		consts.ChangeDirCommand,
		consts.InfoCommand,
		consts.InterpretScriptCommand,
		consts.DefragCommand:
		return 1, nil

	case
		consts.CopyCommand,
		consts.MoveCommand,
		consts.CopyInsideFSCommand,
		consts.CopyOutsideFSCommand:
		return 2, nil

	default:
		return 0, custom_errors.ErrUnknownPathsCount
	}
}

// validateArgPathsCommand validates the command with arguments that are only paths
func validateArgPathsCommand(cmd *Command) error {
	// check if the number of arguments is correct
	expectedPathsCount, err := getPathsCountForCommand(cmd.Name)
	if err != nil {
		return err
	} else if len(cmd.Args) != expectedPathsCount {
		return custom_errors.ErrInvalArgsCount
	}

	for _, path := range cmd.Args {
		// check if the path format is valid
		err := validatePathFormat(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateFormatCommand validates the format command
func validateFormatCommand(cmd *Command) error {
	// check if the number of arguments is correct
	if len(cmd.Args) != 1 {
		return custom_errors.ErrInvalArgsCount
	}

	// check if the filesize is valid (600MB, 1.2GB, 1.44MB, 0.5KB, ...)
	filesize := cmd.Args[0]
	detected_unit := ""
	for _, unit := range consts.AllowedFilesizeUnits {
		if strings.HasSuffix(filesize, unit) {
			detected_unit = unit
			break
		}
	}

	if detected_unit == "" {
		return custom_errors.ErrInvalFormatUnits
	}

	// check if the format is a valid number
	unit_index := strings.Index(filesize, detected_unit)
	onlyNumber := filesize[:unit_index]
	// try to convert the number to float
	_, err := strconv.ParseFloat(onlyNumber, floatBitSize)
	if err != nil {
		return custom_errors.ErrInvalidFilesizeFormat
	}

	return nil
}

// ValidateCommand validates the command
func ValidateCommand(cmd *Command) error {
	// sanity check
	if cmd == nil {
		return custom_errors.ErrNilCmd
	}

	// check if the command name is empty
	if cmd.Name == "" {
		return custom_errors.ErrEmptyCmdName
	}

	switch cmd.Name {
	// one word commands
	case
		consts.CurrDirCommand,
		consts.HelpCommand,
		consts.ExitCommand:
		return validateOneWordCommand(cmd)

	// two or three word commands with only paths as arguments
	case
		consts.RemoveCommand,
		consts.MakeDirCommand,
		consts.RemoveDirCommand,
		consts.ConcatCommand,
		consts.ChangeDirCommand,
		consts.InfoCommand,
		consts.InterpretScriptCommand,
		consts.DefragCommand,
		consts.CopyCommand,
		consts.MoveCommand,
		consts.CopyInsideFSCommand,
		consts.CopyOutsideFSCommand:
		return validateArgPathsCommand(cmd)

	// edge cases
	case consts.FormatCommand:
		return validateFormatCommand(cmd)
	case consts.ListCommand:
		return validateListCommand(cmd)

	default:
		return custom_errors.ErrUnknownCmd
	}
}
