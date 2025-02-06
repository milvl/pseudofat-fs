// command_parser.go contains the command parser for the command line interface.
package cmd

import (
	"bytes"
	"fmt"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"kiv-zos-semestral-work/pseudo_fat"
	"kiv-zos-semestral-work/utils"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"unsafe"
)

// P_CurrDir is a global variable that holds the current directory
var P_CurrDir *pseudo_fat.DirectoryEntry = nil

// sortDirectoryEntries sorts the entries placing directories first and then sorting by name.
// It sorts the slice in place.
func sortDirectoryEntries(entries []*pseudo_fat.DirectoryEntry) {
	sort.Slice(entries, func(i, j int) bool {
		// directories first
		if entries[i].IsFile != entries[j].IsFile {
			return !entries[i].IsFile
		}

		// then lexicographical comparison
		return bytes.Compare(entries[i].Name[:], entries[j].Name[:]) < 0
	})
}

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
	logging.Debug(fmt.Sprintf("FAT tables count: %d", consts.FATableCount))
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

// makePathNormAbs tries to make the given path normalized and absolute.
func makePathNormAbs(path string, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (string, error) {
	var res string

	// path is absolute
	if strings.HasPrefix(path, consts.PathDelimiter) {
		absPathNodes, err := utils.GetNormalizedPathNodes(path)
		if err != nil {
			return "", err
		}

		res = consts.PathDelimiter + strings.Join(absPathNodes, consts.PathDelimiter)

		// path is relative
	} else {
		// construct the absolute path
		absCurrPath, err := utils.GetAbsolutePathFromPwd(pFs, P_CurrDir, fatsRef, dataRef)
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
func changeDirCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || fatsRef[0] == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	var absPath string
	var err error

	// check if the path is absolute
	absPath, err = makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}
	logging.Debug(fmt.Sprintf("Absolute path from pwd constructed: %s", absPath))

	// get the directory entry
	pDirEntries, err := utils.GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, absPath)
	if err != nil {
		return false, err
	}

	// check if the last entry is a directory
	if pDirEntries[len(pDirEntries)-1].IsFile {
		return false, custom_errors.ErrIsFile
	}

	// set the current directory
	P_CurrDir = pDirEntries[len(pDirEntries)-1]
	logging.Debug(fmt.Sprintf("Current directory set to: %s", P_CurrDir.ToString()))

	return false, nil
}

// mkdirCommand creates a new directory
func mkdirCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	absNormPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	err = utils.Mkdir(pFs, fatsRef, dataRef, absNormPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

// rmdirCommand tries to remove the directory.
func rmdirCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	// check if the directory name is valid
	dirName := utils.GetPathBasename(pCommand.Args[0])
	if dirName == consts.CurrDirSymbol || dirName == consts.ParentDirSymbol || dirName == "" {
		return false, custom_errors.ErrInvalidDirEntryName
	}

	absPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	err = utils.Rmdir(pFs, fatsRef, dataRef, P_CurrDir, absPath)
	if err != nil {
		switch err {
		case custom_errors.ErrDirNotFound:
			fmt.Println(consts.DirNotFound)
			return false, err
		case custom_errors.ErrDirNotEmpty:
			fmt.Println(consts.NotEmpty)
			return false, err
		default:
			return false, fmt.Errorf("error removing directory: %s", err)
		}
	}

	return true, nil
}

// removeCommand removes a file from the filesystem.
func removeCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}

	// check if the file name is valid
	fileName := utils.GetPathBasename(pCommand.Args[0])
	if fileName == consts.CurrDirSymbol || fileName == consts.ParentDirSymbol || fileName == "" {
		return false, custom_errors.ErrInvalidDirEntryName
	}

	normAbsPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	err = utils.RemoveFile(pFs, fatsRef, dataRef, normAbsPath)
	if err != nil {
		switch err {
		case custom_errors.ErrEntryNotFound:
			fmt.Println(consts.FileNotFound)
		default:
			return false, fmt.Errorf("error removing file: %s", err)
		}

		return false, nil
	}

	return true, nil
}

// listCommand lists the directory entries for a specified path.
func listCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) ([]*pseudo_fat.DirectoryEntry, error) {
	// sanity check
	if pCommand == nil || pFs == nil {
		return nil, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return nil, custom_errors.ErrFSUninitialized
	}

	var desiredPath string
	if len(pCommand.Args) == 0 {
		desiredPath = consts.CurrDirSymbol
	} else {
		desiredPath = pCommand.Args[0]
	}

	normAbsPath, err := makePathNormAbs(desiredPath, pFs, fatsRef, dataRef)
	if err != nil {
		return nil, err
	}

	// get the directory entries
	branchDirEntries, err := utils.GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, normAbsPath)
	if err != nil {
		return nil, err
	}

	// get the target directory
	targetDir := branchDirEntries[len(branchDirEntries)-1]

	// get the directory entries
	dirEntries, err := utils.GetDirEntries(pFs, targetDir, fatsRef, dataRef)
	if err != nil {
		return nil, err
	}

	return dirEntries, nil
}

// copyInsideFS copies a file to the filesystem.
func copyInsideFS(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return false, custom_errors.ErrFSUninitialized
	}
	if len(pCommand.Args) != 2 {
		return false, custom_errors.ErrInvalArgsCount
	}

	// check if the input file exists
	_, err := os.Stat(pCommand.Args[0])
	if os.IsNotExist(err) {
		return false, custom_errors.ErrInFileNotFound
	} else if err != nil {
		return false, err
	}

	// load the file
	fileData, err := os.ReadFile(pCommand.Args[0])
	if err != nil {
		return false, err
	}

	// check if the file name is valid
	baseName := utils.GetPathBasename(pCommand.Args[1])
	if baseName == consts.CurrDirSymbol || baseName == consts.ParentDirSymbol || baseName == "" {
		return false, custom_errors.ErrInvalidDirEntryName
	}

	absPath, err := makePathNormAbs(pCommand.Args[1], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	err = utils.CopyInsideFS(pFs, fatsRef, dataRef, absPath, fileData)
	if err != nil {
		return false, err
	}

	return true, nil
}

// moveCommand moves a file to a new location/renames it.
func moveCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if len(pCommand.Args) != 2 {
		return false, custom_errors.ErrInvalArgsCount
	}

	// check if the file name is valid
	srcBasename := utils.GetPathBasename(pCommand.Args[0])
	if srcBasename == consts.CurrDirSymbol || srcBasename == consts.ParentDirSymbol || srcBasename == "" {
		return false, custom_errors.ErrInvalidDirEntryName
	}

	// get the source path
	srcPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	unprocessedDestPath := pCommand.Args[1]
	if strings.HasSuffix(unprocessedDestPath, consts.PathDelimiter) {
		unprocessedDestPath = unprocessedDestPath + srcBasename
	}

	// get the destination path
	destPath, err := makePathNormAbs(unprocessedDestPath, pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	if srcPath == destPath {
		return false, nil
	}

	err = utils.MoveFile(pFs, fatsRef, dataRef, srcPath, destPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

// copyCommand copies a file to a new location.
func copyCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pCommand == nil || pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if len(pCommand.Args) != 2 {
		return false, custom_errors.ErrInvalArgsCount
	}

	// check if the file name is valid
	srcBasename := utils.GetPathBasename(pCommand.Args[0])
	if srcBasename == consts.CurrDirSymbol || srcBasename == consts.ParentDirSymbol || srcBasename == "" {
		return false, custom_errors.ErrInvalidDirEntryName
	}

	// get the source path
	srcPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	// get the destination path
	unprocessedDestPath := pCommand.Args[1]
	if strings.HasSuffix(unprocessedDestPath, consts.PathDelimiter) {
		unprocessedDestPath = unprocessedDestPath + srcBasename
	}

	destPath, err := makePathNormAbs(unprocessedDestPath, pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	err = utils.CopyFile(pFs, fatsRef, dataRef, srcPath, destPath)
	if err != nil {
		return false, err
	}

	return true, nil
}

// concatCommand handles the concatenation command.
func concatCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) ([]byte, error) {
	// sanity check
	if pFs == nil || fatsRef == nil || dataRef == nil || pCommand == nil {
		return nil, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return nil, custom_errors.ErrFSUninitialized
	}
	if len(pCommand.Args) != 1 {
		return nil, custom_errors.ErrInvalArgsCount
	}

	// get the path
	normAbsPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return nil, err
	}

	// get the file data
	fileData, err := utils.GetFileBytes(pFs, fatsRef, dataRef, normAbsPath)
	if err != nil {
		return nil, err
	}

	return fileData, nil
}

// infoCommand handles the info command.
func infoCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) ([]uint32, error) {
	// sanity check
	if pFs == nil || fatsRef == nil || dataRef == nil || pCommand == nil {
		return nil, custom_errors.ErrNilPointer
	}
	if P_CurrDir == nil {
		return nil, custom_errors.ErrFSUninitialized
	}
	if len(pCommand.Args) != 1 {
		return nil, custom_errors.ErrInvalArgsCount
	}

	// get the path
	normAbsPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return nil, err
	}

	branchDirEntries, err := utils.GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, normAbsPath)
	if err != nil {
		return nil, err
	}

	// get the target entry
	targetEntry := branchDirEntries[len(branchDirEntries)-1]
	if !targetEntry.IsFile {
		return nil, custom_errors.ErrIsDir
	}

	referencedFat := fatsRef[0]

	// get the cluster chain
	clusters, err := utils.GetClusterChain(targetEntry.StartCluster, referencedFat)
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

// checkCommand checks the filesystem.
func checkCommand(pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) {
	// sanity check
	if pFs == nil || fatsRef == nil || dataRef == nil {
		fmt.Println("FILESYSTEM ERROR: FILESYSTEM IS NOT INITIALIZED")
		return
	}

	// check for any broken FAT entries
	noErrs := true
	for i := 0; i < len(fatsRef); i++ {
		for j := 0; j < len(fatsRef[i]); j++ {
			if fatsRef[i][j] == consts.FatBadCluster {
				if noErrs {
					fmt.Println("FILESYSTEM CORRUPTED:")
					noErrs = false
				}

				fmt.Printf("BAD CLUSTER AT: FAT%d[%d]", i, j)
			}
		}
	}

	// walk the filesystem and check for any inconsistencies
	queue := make([]*pseudo_fat.DirectoryEntry, 0)
	pRootDir, err := utils.GetRootDirEntry(pFs, fatsRef, dataRef)
	if err != nil {
		fmt.Println("FILESYSTEM CORRUPTED WITH ERROR: ", err)
	}

	queue = append(queue, pRootDir)
	for len(queue) > 0 {
		pCurrEntry := queue[0]
		var pCurrEntryFromData *pseudo_fat.DirectoryEntry
		queue = queue[1:]

		if pCurrEntry.IsFile {
			// check the cluster chain
			pCurrEntryFromData, err = utils.ReadDirectoryEntryFromCluster(dataRef[pCurrEntry.StartCluster*uint32(pFs.ClusterSize) : (pCurrEntry.StartCluster+1)*uint32(pFs.ClusterSize)])
			if err != nil {
				fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED WITH ERROR: %s\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]), err)
				noErrs = false
			}

			if utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]) != utils.GetNormalizedStrFromMem(pCurrEntryFromData.Name[:]) ||
				pCurrEntry.IsFile != pCurrEntryFromData.IsFile ||
				pCurrEntry.StartCluster != pCurrEntryFromData.StartCluster ||
				pCurrEntry.ParentCluster != pCurrEntryFromData.ParentCluster ||
				pCurrEntry.Size != pCurrEntryFromData.Size {
				fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED: METADATA MISSMATCH COMPARED TO PARENT REFERENCE\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]))
				noErrs = false
			}

			// check the fat chains
			for i := 0; i < len(fatsRef); i++ {
				clusterChain, err := utils.GetClusterChain(pCurrEntry.StartCluster, fatsRef[i])
				if err != nil {
					fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED WHILE READING FAT%d: %s\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]), i, err)
					noErrs = false
				}

				if len(clusterChain)-1 != int(math.Ceil(float64(pCurrEntry.Size)/float64(pFs.ClusterSize))) { // -1 because the last cluster is not counted
					fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED WHILE READING FAT%d: DATA SIZE MISMATCH\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]), i)
					noErrs = false
				}
			}

		} else {
			// attempt to read the self reference entry if the entry is not root
			if utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]) != consts.PathDelimiter {
				pCurrEntryFromData, err = utils.ReadDirectoryEntryFromCluster(dataRef[pCurrEntry.StartCluster*uint32(pFs.ClusterSize) : (pCurrEntry.StartCluster+1)*uint32(pFs.ClusterSize)])
				if err != nil {
					fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED WITH ERROR: %s\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]), err)
					noErrs = false
				}

				if utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]) != utils.GetNormalizedStrFromMem(pCurrEntryFromData.Name[:]) ||
					pCurrEntry.IsFile != pCurrEntryFromData.IsFile ||
					pCurrEntry.StartCluster != pCurrEntryFromData.StartCluster ||
					pCurrEntry.ParentCluster != pCurrEntryFromData.ParentCluster ||
					pCurrEntry.Size != pCurrEntryFromData.Size {
					fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]))
					noErrs = false
				}
			}

			// add the children to the queue
			children, err := utils.GetDirEntries(pFs, pCurrEntry, fatsRef, dataRef)
			if err != nil {
				fmt.Printf("FILESYSTEM ENTRY \"%s\" CORRUPTED WITH ERROR: %s\n", utils.GetNormalizedStrFromMem(pCurrEntry.Name[:]), err)
				noErrs = false
			}

			if len(children) > 0 {
				queue = append(queue, children...)
			}
		}
	}

	if noErrs {
		fmt.Println(consts.CmdSuccessMsg)
	}
}

// bugCommand handles the bug command.
func bugCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) (bool, error) {
	// sanity check
	if pFs == nil || fatsRef == nil || dataRef == nil {
		return false, custom_errors.ErrNilPointer
	}

	// check if the command is valid
	if len(pCommand.Args) != 1 {
		return false, custom_errors.ErrInvalArgsCount
	}

	// get the path
	normAbsPath, err := makePathNormAbs(pCommand.Args[0], pFs, fatsRef, dataRef)
	if err != nil {
		return false, err
	}

	// get the file data
	branchDirEntries, err := utils.GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, normAbsPath)
	if err != nil {
		return false, err
	}

	pEntry := branchDirEntries[len(branchDirEntries)-1]
	if !pEntry.IsFile {
		return false, custom_errors.ErrIsDir
	}

	// randomly corrupt the file
	corruptFat := 0      // 50% chance to corrupt the fat table entry
	corruptDirEntry := 1 // 50% chance to corrupt the directory entry
	randCorrupt := rand.Intn(corruptDirEntry + 1)

	switch randCorrupt {
	case corruptFat:
		clusterChain, err := utils.GetClusterChain(pEntry.StartCluster, fatsRef[0])
		if err != nil {
			return false, err
		}

		randCluster := clusterChain[rand.Intn(len(clusterChain))]
		randFAT := rand.Intn(len(fatsRef))

		// randomly corrupt the cluster chain
		corruptionFileFree := 0   // 33% chance to corrupt with a file free
		corruptionBadCluster := 1 // 33% chance to corrupt with a bad cluster
		corruptionCycle := 2      // 33% chance to corrupt with a cycle
		randCorruptVal := rand.Intn(corruptionCycle + 1)

		switch randCorruptVal {
		case corruptionFileFree:
			fatsRef[randFAT][randCluster] = consts.FatFree
			logging.Debug(fmt.Sprintf("Corrupted FAT%d[%d] with free cluster flag", randFAT, randCluster))
		case corruptionBadCluster:
			fatsRef[randFAT][randCluster] = consts.FatBadCluster
			logging.Debug(fmt.Sprintf("Corrupted FAT%d[%d] with bad cluster", randFAT, randCluster))
		case corruptionCycle:
			lastInChain := clusterChain[len(clusterChain)-1]
			fatsRef[randFAT][lastInChain] = int32(randCluster)
			logging.Debug(fmt.Sprintf("Corrupted FAT%d[%d] with cycle to %d", randFAT, lastInChain, randCluster))
		}

	case corruptDirEntry:
		logging.Debug(fmt.Sprintf("Corrupting directory entry %s", utils.GetNormalizedStrFromMem(pEntry.Name[:])))
		// randomize the bytes
		// debug print the bytes from dataRef before the corruption
		fmt.Println(dataRef[pEntry.StartCluster*uint32(pFs.ClusterSize) : (pEntry.StartCluster+1)*uint32(pFs.ClusterSize)])
		for b := 0; b < int(unsafe.Sizeof(*pEntry)); b++ {
			dataRef[pEntry.StartCluster*uint32(pFs.ClusterSize)+uint32(b)] = byte(rand.Intn(consts.ByteSizeInt))
		}
		logging.Debug(fmt.Sprintf("Corrupted directory entry %s", utils.GetNormalizedStrFromMem(pEntry.Name[:])))
		// print the bytes from dataRef after the corruption
		fmt.Println(dataRef[pEntry.StartCluster*uint32(pFs.ClusterSize) : (pEntry.StartCluster+1)*uint32(pFs.ClusterSize)])
	}

	return true, nil
}

// interpretScriptCommand interprets the script command.
func interpretScriptCommand(pCommand *Command, pFs *pseudo_fat.FileSystem, pFatsRef *[][]int32, pDataRef *[]byte, endFlag chan struct{}) (bool, error) {
	// sanity checks
	if pCommand == nil || pFs == nil || pFatsRef == nil || pDataRef == nil {
		return false, custom_errors.ErrNilPointer
	}
	if len(pCommand.Args) != 1 {
		return false, custom_errors.ErrInvalArgsCount
	}

	fsChanged := false
	var err error = nil
	commands := make([]*Command, 0)
	commands = append(commands, pCommand)
	for {
		pCurrCommand := commands[0]
		commands = commands[1:]

		if pCurrCommand.Name == consts.InterpretScriptCommand {
			pScriptFile, err := os.ReadFile(pCurrCommand.Args[0])
			if err != nil {
				return fsChanged, err
			}

			scriptLines := strings.Split(string(pScriptFile), consts.ScriptDelimiter)
			trailingCommands := commands
			commands = make([]*Command, 0)
			for _, line := range scriptLines {
				// parse command
				if line != "" && !strings.HasPrefix(line, consts.CommentSymbol) {
					pCommand, err = ParseCommand(line)
					if err != nil {
						return fsChanged, err
					}

					// append the command to the commands
					commands = append(commands, pCommand)
				}
			}

			commands = append(commands, trailingCommands...)

		} else {
			if P_CurrDir == nil {
				fsChanged, err = handleUninitializedFSCmd(pFs, pFatsRef, pDataRef, pCurrCommand, endFlag)
			} else {
				fsChanged, err = handleInitializedFSCmd(pFs, pFatsRef, pDataRef, pCurrCommand, endFlag)
			}
		}

		if len(commands) < 1 {
			break
		}
	}

	return fsChanged, err
}

// handleUninitializedFSCmd handles the command when the filesystem is not initialized.
func handleUninitializedFSCmd(pFs *pseudo_fat.FileSystem,
	pFatsRef *[][]int32,
	pDataRef *[]byte,
	pCommand *Command,
	endFlag chan struct{}) (bool, error) {

	fsChanged := false
	var err error = nil

	switch pCommand.Name {
	case consts.FormatCommand:
		fsChanged, err = formatCommand(pCommand, pFs, pFatsRef, pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.HelpCommand:
		err = helpCommand()
		if err != nil {
			return fsChanged, err
		}

	case consts.ExitCommand:
		close(endFlag)
		return fsChanged, err

	case consts.DebugCommand,
		consts.CurrDirCommand,
		consts.ChangeDirCommand,
		consts.ListCommand,
		consts.MakeDirCommand,
		consts.RemoveDirCommand,
		consts.RemoveCommand,
		consts.ConcatCommand,
		consts.InfoCommand,
		consts.InterpretScriptCommand,
		consts.CheckCommand,
		consts.BugCommand,
		consts.CopyCommand,
		consts.MoveCommand,
		consts.CopyInsideFSCommand,
		consts.CopyOutsideFSCommand:
		fmt.Println(consts.FSUninitializedMsg)

	default:
		return fsChanged, fmt.Errorf("unknown command to execute (logic error): %s", pCommand.Name)
	}

	return fsChanged, err
}

// handleInitializedFSCmd handles the command when the filesystem is initialized.
func handleInitializedFSCmd(pFs *pseudo_fat.FileSystem,
	pFatsRef *[][]int32,
	pDataRef *[]byte,
	pCommand *Command,
	endFlag chan struct{}) (bool, error) {

	fsChanged := false
	var err error = nil

	switch pCommand.Name {
	case consts.HelpCommand:
		err = helpCommand()
		return fsChanged, err

	case consts.ExitCommand:
		close(endFlag)
		return fsChanged, err

	case consts.FormatCommand:
		fsChanged, err = formatCommand(pCommand, pFs, pFatsRef, pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.CurrDirCommand:
		if P_CurrDir == nil {
			fmt.Println(consts.FSUninitializedMsg)
		} else {
			pwd, err := utils.GetAbsolutePathFromPwd(pFs, P_CurrDir, *pFatsRef, *pDataRef)
			if err != nil {
				return fsChanged, err
			}
			fmt.Println(pwd)
		}
		return fsChanged, err

	case consts.ChangeDirCommand:
		fsChanged, err = changeDirCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.MakeDirCommand:
		fsChanged, err = mkdirCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.RemoveDirCommand:
		fsChanged, err = rmdirCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.RemoveCommand:
		fsChanged, err = removeCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.ListCommand:
		var entries []*pseudo_fat.DirectoryEntry
		entries, err = listCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		} else {
			sortDirectoryEntries(entries)
			for _, entry := range entries {
				fmt.Println(entry.ToStringLS())
			}
		}
		return fsChanged, err

	case consts.CopyInsideFSCommand:
		fsChanged, err = copyInsideFS(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.CopyOutsideFSCommand:
		normAbsSrcPath, err := makePathNormAbs(pCommand.Args[0], pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

		var dataRef []byte
		dataRef, err = utils.GetFileBytes(pFs, *pFatsRef, *pDataRef, normAbsSrcPath)
		if err != nil {
			return fsChanged, err
		}

		err = os.WriteFile(pCommand.Args[1], dataRef, consts.NewFilePermissions)
		if err != nil {
			return fsChanged, err
		}

	case consts.MoveCommand:
		fsChanged, err = moveCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.CopyCommand:
		fsChanged, err = copyCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.InterpretScriptCommand:
		fsChanged, err = interpretScriptCommand(pCommand, pFs, pFatsRef, pDataRef, endFlag)
		return fsChanged, err

	case consts.ConcatCommand:
		var dataRef []byte
		dataRef, err = concatCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

		fmt.Println(string(dataRef))
		return fsChanged, err

	case consts.InfoCommand:
		var clusters []uint32
		clusters, err = infoCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

		filename := utils.GetPathBasename(pCommand.Args[0])

		res := fmt.Sprintf("%s ", filename)
		for i, cluster := range clusters {
			if i == 0 {
				res += fmt.Sprintf("%d", cluster)
			} else {
				res += fmt.Sprintf(",%d", cluster)
			}
		}

		fmt.Println(res)

	case consts.CheckCommand:
		checkCommand(pFs, *pFatsRef, *pDataRef)
		return fsChanged, err

	case consts.BugCommand:
		fsChanged, err = bugCommand(pCommand, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

	case consts.DebugCommand:
		logging.Debug(fmt.Sprintf("FATS: \n%s", utils.PFormatFats(*pFatsRef)))
		pEntries := make([]*pseudo_fat.DirectoryEntry, 0)
		queue := make([]*pseudo_fat.DirectoryEntry, 0)
		pRootDir, err := utils.GetRootDirEntry(pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return fsChanged, err
		}

		queue = append(queue, pRootDir)
		for {
			if len(queue) == 0 {
				break
			}

			pCurrEntry := queue[0]
			queue = queue[1:]

			pEntries = append(pEntries, pCurrEntry)

			if !pCurrEntry.IsFile {
				children, err := utils.GetDirEntries(pFs, pCurrEntry, *pFatsRef, *pDataRef)
				if err != nil {
					return fsChanged, err
				}

				queue = append(queue, children...)
			}
		}

		logging.Debug("DEBUG STOP")
	}

	fmt.Printf("%s\n", consts.CmdSuccessMsg)

	return fsChanged, err
}

// ExecuteCommand executes the given command.
// It returns an error if any.
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

	var fsChanged bool
	var err error

	if P_CurrDir == nil {
		fsChanged, err = handleUninitializedFSCmd(pFs, pFatsRef, pDataRef, pCommand, endFlag)
	} else {
		fsChanged, err = handleInitializedFSCmd(pFs, pFatsRef, pDataRef, pCommand, endFlag)
	}

	if err != nil {
		return fmt.Errorf("error executing command: %s", err)
	}

	// if the filesystem was changed, write it to the file
	if fsChanged {
		fmt.Println("Filesystem changed, writing to the file...")
		err := utils.WriteFileSystem(pFile, pFs, *pFatsRef, *pDataRef)
		if err != nil {
			return err
		}
		fmt.Println("Updated filesystem written to the file.")
	}

	return nil
}
