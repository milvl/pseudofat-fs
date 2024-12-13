// command_parser.go contains the command parser for the command line interface.
package cmd

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
)

// helpCommand prints the help message
func helpCommand() error {
	fmt.Print(consts.HelpMsg)
	return nil
}

// ExecuteCommand executes the given command
func ExecuteCommand(command *Command, endFlag chan struct{}) error {
	switch command.Name {
	case consts.HelpCommand:
		return helpCommand()
	case consts.ExitCommand:
		close(endFlag)
		return nil

	default:
		return fmt.Errorf("unknown command to execute (logic error): %s", command.Name)
	}

}
