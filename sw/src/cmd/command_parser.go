// command_parser.go contains the command parser for the command line interface.
package cmd

import (
	"errors"
	"strings"
)

func ParseCommand(input string) (*Command, error) {
	// split the input into words
	words := strings.Fields(input)
	if len(words) == 0 {
		return nil, errors.New("empty input")
	}

	cmdName := words[0]
	cmdArgs := words[1:]

	return &Command{Name: cmdName, Args: cmdArgs}, nil
}
