// command.go contains the command struct and the command interface
package cmd

// Command is a struct representing a command
type Command struct {
	// Name is the name of the command
	Name string
	// Args are the arguments of the command
	Args []string
}
