// consts contains all constants used in the application
package consts

// Command names
const (
	// ONE WORD COMMANDS //

	// CurrentDirCommand represents the format of the current directory command
	CurrDirCommand = "pwd"
	// HelpCommand represents the format of the help command
	HelpCommand = "help"
	// ExitCommand represents the format of the exit command
	ExitCommand = "exit"
	// DebugCommand represents the format of the debug command. This command is used for debugging purposes.
	// Remove this code for production
	DebugCommand = "debug"

	// TWO WORD COMMANDS //

	// RemoveCommand represents the format of the remove command
	RemoveCommand = "rm"
	// MakeDirCommand represents the format of the make directory command
	MakeDirCommand = "mkdir"
	// RemoveDirCommand represents the format of the remove directory command
	RemoveDirCommand = "rmdir"
	// ConcatCommand represents the format of the concatenate command
	ConcatCommand = "cat"
	// ChangeDirCommand represents the format of the change directory command
	ChangeDirCommand = "cd"
	// InfoCommand represents the format of the info command
	InfoCommand = "info"
	// LoadInterpretScriptCommand represents the format of the load interpret script command
	InterpretScriptCommand = "load"
	// DefragCommand represents the format of the defrag command
	DefragCommand = "defrag"
	// FormatCommand represents the format of the format command
	FormatCommand = "format"

	// THREE WORD COMMANDS //

	// CopyCommand represents the format of the copy command
	CopyCommand = "cp"
	// MoveCommand represents the format of the move command
	MoveCommand = "mv"
	// CopyInsideFSCommand represents the format of the copy inside filesystem command
	CopyInsideFSCommand = "incp"
	// CopyOutsideFSCommand represents the format of the copy outside filesystem command
	CopyOutsideFSCommand = "outcp"

	// VARIABLE WORD COUNT COMMANDS //

	// ListCommand represents the format of the list command
	ListCommand = "ls"
)
