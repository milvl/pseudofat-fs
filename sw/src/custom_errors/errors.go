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

// ErrParsingUnits is an error for parsing units
var ErrParsingUnits = errors.New("error parsing units")

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

// ErrInvalidFileSys is an error for invalid file system
var ErrInvalidFileSys = errors.New("invalid file system")

// ErrCreatingFile is an error for creating file
var ErrCreatingFile = errors.New("error creating file")

// ErrOpeningFile is an error for opening file
var ErrOpeningFile = errors.New("error opening file")

// ErrFSUninitialized is an error for file system is uninitialized
var ErrFSUninitialized = errors.New("file system is uninitialized")

// ErrDiskTooSmall is an error for disk too small for the filesystem
var ErrDiskTooSmall = errors.New("chosen disk size is too small for the filesystem")

// ErrInvalidFatCount is an error for invalid FAT count
var ErrInvalidFatCount = errors.New("invalid FAT count")

// ErrReadingFat is an error for reading FAT
var ErrReadingFat = errors.New("error reading FAT")

// ErrConvertingFat is an error for converting FAT
var ErrConvertingFat = errors.New("error converting FAT")

// ErrDataTooSmall is an error for data region is too small
var ErrDataTooSmall = errors.New("data region is too small")

// ErrInvalStartCluster is an error for invalid start cluster
var ErrInvalStartCluster = errors.New("invalid start cluster")

// ErrNoFreeCluster is an error for no free cluster
var ErrNoFreeCluster = errors.New("no free cluster")

// ErrDirNotFound is an error for directory not found
var ErrDirNotFound = errors.New("directory not found")

// ErrFileNotFound is an error for file not found
var ErrInvalidPath = errors.New("invalid path")

// ErrDirNotEmpty is an error for directory not empty
var ErrDirNotEmpty = errors.New("directory not empty")
