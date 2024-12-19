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
	"strings"
)

// P_CurrDir is a global variable that holds the current directory
var P_CurrDir *pseudo_fat.DirectoryEntry = nil

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
	rootDir := utils.NewDirectoryEntry(false, 0, 0, 0, consts.PathDelimiter)

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

	// assign current directory
	P_CurrDir = &rootDir

	fmt.Printf("Filesystem formatted to %d bytes. Allocatable data space: %d bytes\n", size, allocatableSize)
	return true, nil
}

// makeThePathAbsolute tries to make the given path absolute.
func makeThePathAbsolute(path string, pFs *pseudo_fat.FileSystem, pFatsRef *[][]int32, pDataRef *[]byte) (string, error) {
	var res string

	if strings.HasPrefix(path, consts.PathDelimiter) {
		res = path
	} else {
		// construct the absolute path
		absCurrPath, err := utils.GetAbsolutePathFromPwd(pFs, P_CurrDir, *pFatsRef, *pDataRef)
		if err != nil {
			return "", err
		}

		absPathNodes, err := utils.GetNormalizedPathNodes(absCurrPath + consts.PathDelimiter + path)
		if err != nil {
			return "", err
		}

		res = consts.PathDelimiter + strings.Join(absPathNodes, consts.PathDelimiter)
	}

	return res, nil
}

// changeDirCommand changes the current directory.
//
// Always returns false as the filesystem is not changed.
// Returns ErrDirNotFound if the directory is not found.
// Otherwise an error is returned.
func changeDirCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, pFatsRef *[][]int32, pDataRef *[]byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || pFatsRef == nil || pDataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	var absPath string
	var err error

	// check if the path is absolute
	absPath, err = makeThePathAbsolute(pCommand.Args[0], pFs, pFatsRef, pDataRef)
	if err != nil {
		return false, err
	}
	logging.Debug(fmt.Sprintf("Absolute path from pwd constructed: %s", absPath))

	// get the directory entry
	pDirEntries, err := utils.GetDirEntriesFromRoot(pFs, *pFatsRef, *pDataRef, absPath)
	if err != nil {
		return false, err
	}

	// set the current directory
	P_CurrDir = &pDirEntries[len(pDirEntries)-1]
	logging.Debug(fmt.Sprintf("Current directory set to: %s", P_CurrDir.ToString()))

	return false, nil
}

// mkdirCommand creates a new directory
func mkdirCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, pFatsRef *[][]int32, pDataRef *[]byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || pFatsRef == nil || pDataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	absPath, err := makeThePathAbsolute(pCommand.Args[0], pFs, pFatsRef, pDataRef)
	if err != nil {
		return false, err
	}

	err = utils.Mkdir(pFs, *pFatsRef, *pDataRef, absPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

// rmdirCommand tries to remove the directory.
func rmdirCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, pFatsRef *[][]int32, pDataRef *[]byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || pFatsRef == nil || pDataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	absPath, err := makeThePathAbsolute(pCommand.Args[0], pFs, pFatsRef, pDataRef)
	if err != nil {
		return false, err
	}

	err = utils.Rmdir(pFs, *pFatsRef, *pDataRef, absPath)
	if err != nil {
		switch err {
		case custom_errors.ErrDirNotFound:
			fmt.Println(consts.FileNotFound)
		case custom_errors.ErrDirNotEmpty:
			fmt.Println(consts.NotEmpty)
		default:
			return false, fmt.Errorf("error removing directory: %s", err)
		}

		return false, nil
	}

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
	case consts.DebugCommand:
		dirEntries, err := utils.GetDirEntries(pFs, P_CurrDir, *pFatsRef, *pDataRef)
		if err != nil {
			logging.Error(fmt.Sprintf("Error getting the directory entries: %s", err))
			os.Exit(consts.ExitFailure)
		}

		logging.Debug(fmt.Sprintf("Directory entries count: %d", len(dirEntries)))
		for i, entry := range dirEntries {
			logging.Debug(fmt.Sprintf("Entry %d: %s", i, entry.ToString()))
		}

		logging.Debug("debug pwd: ")
		if P_CurrDir == nil {
			logging.Debug(fmt.Sprintf("Current directory: %s", consts.FSUninitializedMsg))
		} else {
			pwd, err := utils.GetAbsolutePathFromPwd(pFs, P_CurrDir, *pFatsRef, *pDataRef)
			if err != nil {
				return err
			}
			logging.Debug(fmt.Sprintf("Current directory: %s", pwd))
		}

		logging.Debug(fmt.Sprintf("FATS: \n%s", utils.PFormatFats(*pFatsRef)))

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

	case consts.CurrDirCommand:
		if P_CurrDir == nil {
			fmt.Println(consts.FSUninitializedMsg)
		} else {
			pwd, err := utils.GetAbsolutePathFromPwd(pFs, P_CurrDir, *pFatsRef, *pDataRef)
			if err != nil {
				return err
			}
			fmt.Println(pwd)
		}

	case consts.ChangeDirCommand:
		fsChanged, err = changeDirCommand(pCommand, pFs, pFatsRef, pDataRef)
		if err != nil {
			return err
		}

	case consts.MakeDirCommand:
		fsChanged, err = mkdirCommand(pCommand, pFs, pFatsRef, pDataRef)
		if err != nil {
			return err
		}

	case consts.RemoveDirCommand:
		fsChanged, err = rmdirCommand(pCommand, pFs, pFatsRef, pDataRef)
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
