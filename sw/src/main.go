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
	"kiv-zos-semestral-work/pseudo_fat"
	"kiv-zos-semestral-work/utils"
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
func interpretCmds(cmdIn chan *cmd.Command, endFlagChan chan struct{}, fsPath string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		pCommand, ok := <-cmdIn
		if !ok {
			logging.Debug("cmdIn channel closed, exiting interpretCmds...")
			break
		}

		logging.Debug(fmt.Sprintf("Interpreting command: %s", pCommand))
		cmd.ExecuteCommand(pCommand, endFlagChan)
	}
}

// handleArgsParserErrAndQuit handles the errors returned by the argument parser.
func handleArgsParserErrAndQuit(err error) {
	switch err {
	case custom_errors.ErrInvalArgsCount:
		logging.Info("User provided invalid number of arguments")
		fmt.Printf("%s\n\n%s\n", consts.InvalProgArgsCount, consts.LaunchHintMsg)
		os.Exit(consts.ExitFailure)

	case custom_errors.ErrHelpWanted:
		logging.Info("Help requested")
		fmt.Print(consts.HelpMsg)
		os.Exit(consts.ExitSuccess)

	case custom_errors.ErrInvalidPathCharacter:
		logging.Info("User provided invalid path for the file system file")
		fmt.Printf("%s\n\n%s\n", consts.InvalFSPathChars, consts.LaunchHintMsg)
		os.Exit(consts.ExitFailure)

	default:
		logging.Error(fmt.Sprintf("Not specified err: %s", err))
		os.Exit(consts.ExitFailure)
	}
}

// handleFilepathErr handles the errors returned by the filesystem path validation.
func handleFilepathErr(err error, fsPath string) {
	switch err {
	case custom_errors.ErrIsDir:
		logging.Info(fmt.Sprintf("Filesystem file \"%s\" is a directory", fsPath))
		fmt.Printf("%s\n\n%s\n", consts.FSPathIsDir, consts.LaunchHintMsg)
	default:
		logging.Error(fmt.Sprintf("Not specified err: %s", err))
	}
	os.Exit(consts.ExitFailure)
}

// handleProgramTermination handles the program termination
func handleProgramTermination(
	ctx context.Context,
	cmdBufferChan chan *cmd.Command,
	pWg *sync.WaitGroup,
	scannerEndChan chan struct{},
	interpreterEndChan chan struct{}) {

	select {
	// SIGINT received
	case <-ctx.Done():
		print("\n")
		logging.Debug("Received SIGINT, closing the cmdBufferChan...")
		close(cmdBufferChan)

		pWg.Wait()

	// EOF or scanner error received
	case <-scannerEndChan:
		logging.Debug("Scanner ended, sending end flag to cmdBufferChan...")
		close(cmdBufferChan)

		pWg.Wait()

	// 'exit' command received
	case <-interpreterEndChan:
		logging.Debug("cmdBufferChan closed (exit command received), exiting...")
		close(cmdBufferChan)

		pWg.Wait()
	}
}

func main() {
	// INITIALIZATION OF CONTROL VARIABLES //
	// to handle signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// scanner is used to read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(nil, int(consts.MaxInputBufferSize))

	// cmdBufferChan is a channel to send the command to the interpreter
	cmdBufferChan := make(chan *cmd.Command)

	// interpretEndChan is a channel to signal the end of the interpreter
	interpreterEndChan := make(chan struct{})
	// scannerEndChan is a channel to signal the end of the scanner
	scannerEndChan := make(chan struct{})

	// to ensure that the goroutine is finished in case of signal
	var wg sync.WaitGroup
	wg.Add(1)

	// INITIALIZATION OF THE FILE SYSTEM //
	fsPath, err := arg_parser.GetFilenameFromArgs(os.Args)
	if err != nil {
		handleArgsParserErrAndQuit(err)
	}
	logging.Debug(fmt.Sprintf("Filesystem path: %s", fsPath))

	fileExists, err := utils.FilepathValid(fsPath)
	if err != nil {
		handleFilepathErr(err, fsPath)
	}

	var file *os.File
	if !fileExists {
		// 	logging.Info(fmt.Sprintf("Filesystem file \"%s\" does not exist, creating it...", fsPath))
		// 	file, err = os.Create(fsPath)
		// 	if err != nil {
		// 		logging.Error(fmt.Sprintf("Error creating filesystem file: %s", err))
		// 		os.Exit(consts.ExitFailure)
		// 	}
		// } else {
		// 	file, err = os.Open(fsPath)
		// 	if err != nil {
		// 		logging.Error(fmt.Sprintf("Error opening filesystem file: %s", err))
		// 		os.Exit(consts.ExitFailure)
		// 	}
	}
	logging.Warn(fmt.Sprintf("DEBUG - Creating filesystem file \"%s\"...", fsPath))
	file, err = os.Create(fsPath)
	if err != nil {
		logging.Error(fmt.Sprintf("Error creating filesystem file: %s", err))
		os.Exit(consts.ExitFailure)
	}
	defer fmt.Println("Closing file...")
	defer file.Close()

	// TODO: load the filesystem from the file
	// create a dummy filesystem for now
	pTmpFs := pseudo_fat.GetUninitializedFileSystem()
	pTmpFs.DiskSize = 4008032
	pTmpFs.ClusterSize = 4000
	pTmpFs.FatCount = 1000
	pTmpFs.Fat01StartAddr = 32
	pTmpFs.Fat02StartAddr = 4032
	pTmpFs.DataStartAddr = 8032
	copy(pTmpFs.Signature[:], consts.AuthorID)
	logging.Debug(fmt.Sprintf("Dummy filesystem: %s", pTmpFs.ToString()))
	// convert the filesystem to bytes
	fsBytes, err := utils.StructToBytes(pTmpFs)
	if err != nil {
		logging.Error(fmt.Sprintf("Error converting filesystem to bytes: %s", err))
		os.Exit(consts.ExitFailure)
	}

	// write the filesystem to the file
	// expand the fsBytes to the size of the filesystem
	fsBytes = append(fsBytes, make([]byte, pTmpFs.DiskSize-uint32(len(fsBytes)))...)

	_, err = file.Write(fsBytes)
	if err != nil {
		logging.Error(fmt.Sprintf("Error writing filesystem to the file: %s", err))
		os.Exit(consts.ExitFailure)
	}

	_, _, _ = pseudo_fat.GetFileSystem(file)

	// USER INTERACTION HANDLING //
	go acceptCmds(scanner, cmdBufferChan, scannerEndChan)
	go interpretCmds(cmdBufferChan, interpreterEndChan, fsPath, &wg)

	// PROGRAM TERMINATION HANDLING //
	handleProgramTermination(ctx, cmdBufferChan, &wg, scannerEndChan, interpreterEndChan)
}
