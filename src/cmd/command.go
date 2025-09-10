// command.go contains the command struct and the command interface
package cmd

import "strings"

// Command is a struct representing a command
type Command struct {
	// Name is the name of the command
	Name string
	// Args are the arguments of the command
	Args []string
}

// ToString returns the command as a string
func (c *Command) ToString() string {
	return c.Name + " " + strings.Join(c.Args, " ")
}
