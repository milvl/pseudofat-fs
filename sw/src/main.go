// main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"kiv-zos-semestral-work/arg_parser"
	"kiv-zos-semestral-work/cmd"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// acceptCmds reads from stdin and parses the input
// into a command. It sends the command to the cmdOut
// channel.
//
// Its goroutine is closed by program termination, because
// the scanner.Scan() is blocking so it was too complicated
// to try to synchronize its termination with the main
// goroutine.
func acceptCmds(scanner *bufio.Scanner, cmdOut chan *cmd.Command, endFlagChan chan struct{}) {
	defer logging.Debug("acceptCmds goroutine finished")

	for {
		if scanner.Scan() {
			input := scanner.Text()
			logging.Debug(fmt.Sprintf("Read from stdin: \"%s\"", input))
			pCommand, err := cmd.ParseCommand(input)
			if err != nil {
				logging.Error(fmt.Sprintf("Error parsing command: %s", err))
			} else if err := cmd.ValidateCommand(pCommand); err != nil {
				switch err {
				case custom_errors.ErrUnknownCmd:
					logging.Info(fmt.Sprintf("Unknown command: \"%s\"", pCommand.Name))
					fmt.Println(consts.UnknownCmdMsg)
					fmt.Println(consts.HintMsg)
				default:
					logging.Error(fmt.Sprintf("Not specified err: %s", err))
				}
			} else {
				logging.Debug(fmt.Sprintf("Parsed command: %s", pCommand))
				cmdOut <- pCommand
			}

		} else if err := scanner.Err(); err != nil {
			// there was an error reading from stdin
			logging.Error(fmt.Sprintf("Error reading from stdin: %s", err))
			endFlagChan <- struct{}{}
			return

		} else {
			// the input is closed (EOF), break the loop
			logging.Debug("EOF reached, sending end flag...")
			endFlagChan <- struct{}{}
			return
		}
	}
}

// interpretCmds reads the commands from the cmdIn channel
// and interprets them. It sends the result to the stdout.
func interpretCmds(cmdIn chan *cmd.Command, fsPath string, wg *sync.WaitGroup) {
	defer wg.Done()

	endLoop := false
	for {
		if endLoop {
			break
		}

		pCommand, ok := <-cmdIn
		if !ok {
			logging.Debug("cmdIn channel closed, exiting interpretCmds...")
			endLoop = true
			break
		}

		logging.Debug(fmt.Sprintf("Interpreting command: %s", pCommand))
		// code
	}
}

func main() {
	fsPath, err := arg_parser.GetFilenameFromArgs(os.Args)
	if err != nil {
		switch err {
		case custom_errors.ErrInvalArgsCount:
			logging.Info("User provided invalid number of arguments")
			fmt.Println("Invalid number of arguments")
			os.Exit(consts.ExitFailure)
		case custom_errors.ErrHelpWanted:
			logging.Info("Help requested")
			fmt.Print(consts.HelpMsg)
			os.Exit(consts.ExitSuccess)
		default:
			logging.Error(fmt.Sprintf("Not specified err: %s", err))
			os.Exit(consts.ExitFailure)
		}
	}

	logging.Debug(fmt.Sprintf("Filesystem path: %s", fsPath))

	// to handle signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// scanner is used to read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(nil, int(consts.MaxInputBufferSize))

	// cmdBufferChan is a channel to send the command to the interpreter
	cmdBufferChan := make(chan *cmd.Command)

	scannerEndChan := make(chan struct{})

	// to ensure that the goroutine is finished in case of signal
	var wg sync.WaitGroup
	wg.Add(1)

	go acceptCmds(scanner, cmdBufferChan, scannerEndChan)
	go interpretCmds(cmdBufferChan, fsPath, &wg)

	// wait for signal or EOF
	select {
	// SIGINT received
	case <-ctx.Done():
		print("\n")
		logging.Debug("Received SIGINT, closing the cmdBufferChan...")
		close(cmdBufferChan)

		// join the interpret goroutine
		wg.Wait()

	// EOF received - here goroutine is already finished
	case <-scannerEndChan:
		logging.Debug("Scanner ended, sending end flag to cmdBufferChan...")
		close(cmdBufferChan)

		// join the interpret goroutine
		wg.Wait()
	}
}
