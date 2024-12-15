// errors.go contains the error definitions for the project.
package custom_errors

import (
	"errors"
)

// ErrInvalArgsCount is an error for invalid number of arguments
var ErrInvalArgsCount = errors.New("invalid number of arguments)")

// ErrEmptyPath is an error for empty path
var ErrEmptyPath = errors.New("empty path")

// ErrPathTooLong is an error for path too long
var ErrPathTooLong = errors.New("path too long")

// ErrInvalidPathCharacter is an error for invalid path character
var ErrInvalidPathCharacter = errors.New("invalid path character")

// ErrUnknownPathsCount is an error for unknown count of paths
var ErrUnknownPathsCount = errors.New("unknown count of paths for command (logic error)")

// ErrInvalFormatUnits is an error for invalid units for the format command
var ErrInvalFormatUnits = errors.New("invalid format units")

// ErrInvalidFilesizeFormat is an error for invalid file size format
var ErrInvalidFilesizeFormat = errors.New("invalid file size format")

// ErrNilCmd is an error for nil command
var ErrNilCmd = errors.New("nil command")

// ErrEmptyCmdName is an error for empty command name
var ErrEmptyCmdName = errors.New("empty command name")

// ErrUnknownCmd is an error for unknown command
var ErrUnknownCmd = errors.New("unknown command")

// ErrHelpWanted is an error for requesting help
var ErrHelpWanted = errors.New("user requested help message")

// ErrIsDir is an error for chosen file is a directory
var ErrIsDir = errors.New("chosen file is a directory")

// ErrNilPointer is an error for nil pointer
var ErrNilPointer = errors.New("nil pointer provided")

// ErrStructToBytes is an error for converting struct to byte
var ErrStructToBytes = errors.New("error converting struct to bytes")

// ErrBytesToStruct is an error for converting bytes to struct
var ErrBytesToStruct = errors.New("error converting bytes to struct")

// ErrNotPtr is an error for not a pointer
var ErrNotPtr = errors.New("not a pointer")

// ErrEmptyData is an error for empty data
var ErrEmptySlice = errors.New("empty slice provided")

// ErrInvalidFileSys is an error for invalid file system
var ErrInvalidFileSys = errors.New("invalid file system")
