// consts package contains all the constants used in the project
package consts

const HelpMsg = `Usage: myfilesystem <filesystem_path>
A simplified filesystem program based on pseudoFAT. The <filesystem_path> must be a valid path to a pseudoFAT filesystem file.

Commands:
  help           - Display this help message.
  exit           - Exit the program.
  cp s1 s2       - Copy file "s1" to destination "s2".
  mv s1 s2       - Move file "s1" to "s2" or rename "s1" to "s2".
  rm s1          - Delete file "s1".
  mkdir a1       - Create directory "a1".
  rmdir a1       - Remove empty directory "a1".
  ls [a1]        - List contents of directory "a1" (or current directory if not specified).
  cat s1         - Display contents of file "s1".
  cd a1          - Change current directory to "a1".
  pwd            - Print the current working directory.
  info s1        - Display cluster information of file "s1".
  incp s1 s2     - Import file "s1" from disk to location "s2" in the filesystem.
  outcp s1 s2    - Export file "s1" from filesystem to "s2" on the disk.
  load s1        - Load and execute commands from file "s1" sequentially (one command per line).
  format <size>  - Format the filesystem to the specified size, overwriting existing data.
  check          - Check the filesystem for errors.
  bug s1         - Simulate a bug in the filesystem for file "s1".

Example:
  To create a filesystem, format it, and perform operations:
    $ myfilesystem myfs.pseudo
    format 600MB
    mkdir documents
    incp report.txt documents/report.txt
    ls documents
    info documents/report.txt
`

// UnknownCmdMsg is the message displayed when an unknown command is entered
const UnknownCmdMsg = "UNKNOWN COMMAND"

// HintMsg is the message displayed to prompt the user for input
const HintMsg = "Type 'help' for usage information."

// LaunchHintMsg is the message displayed to prompt the user for input
const LaunchHintMsg = "Launch only with '-h' or '--help' for usage information."

// InvalCmdArgsMsg is the message displayed when the command arguments are invalid
const InvalFSPathChars = "The path to the filesystem contains invalid characters."

// InvalProgArgsCount is the message displayed when the program arguments are invalid
const InvalProgArgsCount = "Invalid number of program arguments"

// FSPathTooLong is the message displayed when the filesystem path is too long
const FSPathIsDir = "The chosen file is a directory."

// FileNotFilesys is the message displayed when the file is not a pseudoFAT filesystem file
const FileNotFilesys = "Warning: The file is not a pseudoFAT filesystem file. It may be corrupted. It can only be formatted which will ERASE ALL DATA. Proceed with caution."

// FSUninitializedMsg is the message displayed when the filesystem is uninitialized
const FSUninitializedMsg = "File system is uninitialized. It cannot be used until it is formatted."

// FileNotFound is the message displayed when the file is not found
const FileNotFound = "FILE NOT FOUND"

// DirNotFound is the message displayed when the directory is not found
const DirNotFound = "DIRECTORY NOT FOUND"

// InFileNotFound is the message displayed when the input file is not found
const InFileNotFound = "INPUT FILE NOT FOUND"

// NotEmpty is the message displayed when the directory is not empty
const NotEmpty = "NOT EMPTY"

// InvalidPath is the message displayed when the path is invalid
const InvalidPath = "INVALID PATH CHOICE FOR SELECTED OPERATION"

// CmdSuccessMsg is the message displayed when the command is successful
const CmdSuccessMsg = "OK"
