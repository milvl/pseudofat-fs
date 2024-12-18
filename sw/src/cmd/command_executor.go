// command_parser.go contains the command parser for the command line interface.
package cmd

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"kiv-zos-semestral-work/pseudo_fat"
	"kiv-zos-semestral-work/utils"
	"os"
)

// helpCommand prints the help message
func helpCommand() error {
	fmt.Print(consts.HelpMsg)
	return nil
}

// formatCommand formats the filesystem
func formatCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, pFatsRef *[][]int32, pDataRef *[]byte) (bool, error) {
	// parse the size
	size, err := utils.ParseFSSize(pCommand.Args[0])
	if err != nil {
		return false, err
	}

	clusterCount, fatSize, allocatableSize := utils.CalculateFSSizes(size)
	logging.Debug(fmt.Sprintf("To format filesystem to %d bytes", size))
	logging.Debug(fmt.Sprintf("Cluster count: %d", clusterCount))
	logging.Debug(fmt.Sprintf("FAT space: %d", fatSize))
	logging.Debug(fmt.Sprintf("Allocatable space: %d", allocatableSize))

	// allocate the pFatsRef
	*pFatsRef = make([][]int32, consts.FATableCount)
	for i := range *pFatsRef {
		(*pFatsRef)[i] = make([]int32, clusterCount)
	}
	// initialize the FATs
	for i := range *pFatsRef {
		for j := range (*pFatsRef)[i] {
			(*pFatsRef)[i][j] = consts.FatFree
		}
	}

	// initialize the filesystem
	pFs.DiskSize = size
	pFs.FatCount = clusterCount
	pFs.Fat01StartAddr = uint32(pseudo_fat.GetSizeOfFileSystem())
	pFs.Fat02StartAddr = pFs.Fat01StartAddr + fatSize
	pFs.DataStartAddr = pFs.Fat02StartAddr + (pFs.Fat02StartAddr - pFs.Fat01StartAddr)
	pFs.ClusterSize = consts.ClusterSize
	copy(pFs.Signature[:], consts.AuthorID)

	logging.Debug(fmt.Sprintf("Filesystem initialized: %s", pFs.ToString()))

	// prepare root directory
	rootDir := pseudo_fat.DirectoryEntry{
		IsFile:        false,
		Size:          0,
		StartCluster:  0,
		ParentCluster: 0,
	}
	copy(rootDir.Name[:], []byte(consts.PathDelimiter))

	// assign the root directory to the first cluster
	for i := 0; i < len((*pFatsRef)); i++ {
		(*pFatsRef)[i][rootDir.StartCluster] = consts.FatFileEnd
	}

	*pDataRef = make([]byte, allocatableSize)

	// write the root directory to the data region
	rootDirBytes, err := utils.StructToBytes(rootDir)
	if err != nil {
		return false, err
	}
	copy((*pDataRef)[0:], rootDirBytes)

	fmt.Printf("Filesystem formatted to %d bytes. Allocatable data space: %d bytes\n", size, allocatableSize)
	return true, nil
}

// ExecuteCommand executes the given command
func ExecuteCommand(
	pCommand *Command,
	endFlag chan struct{},
	pFile *os.File,
	pFs *pseudo_fat.FileSystem,
	pFatsRef *[][]int32,
	pDataRef *[]byte) error {

	// sanity check
	if pCommand == nil || endFlag == nil || pFs == nil {
		return custom_errors.ErrNilPointer
	}
	// check if the filesystem is initialized
	if pFatsRef == nil || pDataRef == nil {
		if pCommand.Name != consts.ExitCommand && pCommand.Name != consts.HelpCommand && pCommand.Name != consts.FormatCommand {
			return custom_errors.ErrFSUninitialized
		}
	}

	fsChanged := false
	var err error

	// execute the command
	switch pCommand.Name {
	case consts.HelpCommand:
		return helpCommand()

	case consts.ExitCommand:
		close(endFlag)
		return nil

	case consts.FormatCommand:
		fsChanged, err = formatCommand(pCommand, pFs, pFatsRef, pDataRef)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown command to execute (logic error): %s", pCommand.Name)
	}

	// if the filesystem was changed, write it to the file
	if fsChanged {
		err := utils.WriteFileSystem(pFile, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return err
		}
	}

	return nil
}
