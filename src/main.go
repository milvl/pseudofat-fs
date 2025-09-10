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
// into a command. It sends the command to the cmdIn
// channel.
//
// Its goroutine is closed by program termination, because
// the scanner.Scan() is blocking so it was too complicated
// to try to synchronize its termination with the main
// goroutine.
func acceptCmds(scanner *bufio.Scanner, cmdIn chan *cmd.Command, endFlagChan chan struct{}) {
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
				cmdIn <- pCommand
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
func interpretCmds(cmdOut chan *cmd.Command,
	endFlagChan chan struct{},
	fsPath string,
	wg *sync.WaitGroup,
	pFile *os.File,
	pFs *pseudo_fat.FileSystem,
	pFatsRef *[][]int32,
	pDataRef *[]byte) {

	defer wg.Done()

	for {
		pCommand, ok := <-cmdOut
		if !ok {
			logging.Debug("cmdIn channel closed, exiting interpretCmds...")
			break
		}

		logging.Debug(fmt.Sprintf("Interpreting command: %s", pCommand))
		err := cmd.ExecuteCommand(pCommand, endFlagChan, pFile, pFs, pFatsRef, pDataRef)
		if err != nil {
			switch err {
			case custom_errors.ErrNilPointer:
				logging.Error("Nil pointer provided to ExecuteCommand")
			case custom_errors.ErrFSUninitialized:
				logging.Info("File system is uninitialized (command requires initialized file system)")
				fmt.Println(consts.FSUninitializedMsg)
				fmt.Println(consts.HintMsg)
			default:
				logging.Error(fmt.Sprintf("Not specified err: %s", err))
			}
		}
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

// handleFileErr handles the errors returned by the filesystem path validation.
func handleFileErr(err error, fsPath string) {
	switch err {
	case custom_errors.ErrIsDir:
		logging.Info(fmt.Sprintf("Filesystem file \"%s\" is a directory", fsPath))
		fmt.Printf("%s\n\n%s\n", consts.FSPathIsDir, consts.LaunchHintMsg)
	case custom_errors.ErrCreatingFile:
		logging.Error(fmt.Sprintf("Error creating filesystem file \"%s\"", fsPath))
	case custom_errors.ErrOpeningFile:
		logging.Error(fmt.Sprintf("Error opening filesystem file \"%s\"", fsPath))
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

	logging.Info("Exiting...")
}

// getFileFromPath returns the file from the path
func getFileFromPath(fsPath string) (*os.File, error) {
	fileExists, err := utils.FilepathValid(fsPath)
	if err != nil {
		return nil, err
	}

	var pFile *os.File
	if !fileExists {
		logging.Info(fmt.Sprintf("Filesystem file \"%s\" does not exist, creating it...", fsPath))
		pFile, err = os.Create(fsPath)
		if err != nil {
			return nil, custom_errors.ErrCreatingFile
		}
	} else {
		pFile, err = os.OpenFile(fsPath, os.O_CREATE|os.O_RDWR, consts.NewFilePermissions)
		if err != nil {
			return nil, custom_errors.ErrOpeningFile
		}
	}

	return pFile, nil
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
	// get the filesystem path from the arguments
	fsPath, err := arg_parser.GetFilenameFromArgs(os.Args)
	if err != nil {
		handleArgsParserErrAndQuit(err)
	}
	logging.Debug(fmt.Sprintf("Filesystem path: %s", fsPath))

	// get the pFile from the path
	pFile, err := getFileFromPath(fsPath)
	if err != nil {
		handleFileErr(err, fsPath)
	}
	defer fmt.Println("Closing file...")
	defer pFile.Close()

	// load the filesystem from the file
	pFs, pFats, pData, err := utils.GetFileSystem(pFile)
	if err != nil {
		logging.Error(fmt.Sprintf("Error getting the filesystem: %s", err))
		os.Exit(consts.ExitFailure)
	}

	// set the current directory to the root directory
	if (*pFats != nil) && (*pData != nil) {
		cmd.P_CurrDir, err = utils.GetRootDirEntry(pFs, *pFats, *pData)
		if err != nil {
			logging.Error(fmt.Sprintf("Error getting the root directory entry: %s", err))
			os.Exit(consts.ExitFailure)
		}
	}

	// USER INTERACTION HANDLING //
	go acceptCmds(scanner, cmdBufferChan, scannerEndChan)
	go interpretCmds(cmdBufferChan, interpreterEndChan, fsPath, &wg, pFile, pFs, pFats, pData)

	// PROGRAM TERMINATION HANDLING //
	handleProgramTermination(ctx, cmdBufferChan, &wg, scannerEndChan, interpreterEndChan)
}
