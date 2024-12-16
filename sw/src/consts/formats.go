// consts contains all constants used in the application
package consts

// AllowedPathCharacters is a string containing all allowed characters in a path
const AllowedPathCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_./"

// AllowedFilesizeUnits is a slice containing all allowed units for a file size.
//
// The B unit needs to be the last one because of the way the units are parsed
// (substring parsing).
var AllowedFilesizeUnits = []string{"KB", "MB", "GB", "B"}

// AuthorID is the ID of the author of the file system
const AuthorID = "A21B0318P"
