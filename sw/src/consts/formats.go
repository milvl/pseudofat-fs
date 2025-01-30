// consts contains all constants used in the application
package consts

// AllowedPathCharacters is a string containing all allowed characters in a path
const AllowedPathCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_./"

// UnitB is the unit for kilobytes
const UnitKb = "KB"

// UnitMb is the unit for megabytes
const UnitMb = "MB"

// UnitGb is the unit for gigabytes
const UnitGb = "GB"

// UnitB is the unit for bytes
const UnitB = "B"

// AllowedFilesizeUnits is a slice containing all allowed units for a file size.
//
// The B unit needs to be the last one because of the way the units are parsed
// (substring parsing).
var AllowedFilesizeUnits = []string{UnitKb, UnitMb, UnitGb, UnitB}

// AuthorID is the ID of the author of the file system
const AuthorID = "A21B0318P"

// RootDirName is the name of the root directory
const PathDelimiter = "/"

// CurrDirSymbol is the symbol for the current directory
const CurrDirSymbol = "."

// ParentDirSymbol is the symbol for the parent directory
const ParentDirSymbol = ".."

// NewFilePermissions is the default permissions for a new file
const NewFilePermissions = 0644
