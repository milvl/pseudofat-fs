// utils is a package that contains utility functions for the ZOS project.
package utils

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"kiv-zos-semestral-work/pseudo_fat"
	"math"
	"strings"
)

// GetClusterChain traverses the FAT to collect all clusters in the directory's chain.
func GetClusterChain(startCluster uint32, fat []int32) ([]uint32, error) {
	if int32(startCluster) == consts.FatFree {
		return nil, custom_errors.ErrInvalStartCluster
	}

	var chain []uint32
	cycleMap := make(map[uint32]bool)
	current := startCluster

	for {
		// validate cluster index
		if current >= uint32(len(fat)) {
			return nil, fmt.Errorf("cluster index %d out of bounds", current)
		}

		chain = append(chain, current)
		cycleMap[current] = true

		next := fat[current]
		_, exists := cycleMap[uint32(next)]
		if exists {
			return nil, fmt.Errorf("CYCLE DETECTED AT CLUSTER %d", current)
		}

		if next == consts.FatFileEnd {
			break
		} else if next == consts.FatBadCluster {
			return nil, custom_errors.ErrBadCluster
		} else if next < 0 {
			// negative values (other than consts.FatFileEnd) can be considered invalid or used for other purposes
			return nil, fmt.Errorf("invalid FAT entry at cluster %d: %d", current, next)
		}

		current = uint32(next)
	}

	return chain, nil
}

// ReadDirectoryEntryFromCluster deserializes DirectoryEntry structs from a specific cluster.
func ReadDirectoryEntryFromCluster(clusterData []byte) (*pseudo_fat.DirectoryEntry, error) {
	// sanity check
	if clusterData == nil {
		return nil, custom_errors.ErrNilPointer
	}

	entry := pseudo_fat.DirectoryEntry{}
	err := BytesToStruct(clusterData, &entry)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize directory entry: %w", err)
	}

	return &entry, nil
}

// findFreeCluster finds the first free cluster in the FAT.
func findFreeCluster(fat []int32) (uint32, error) {
	for i, entry := range fat {
		if entry == consts.FatFree {
			return uint32(i), nil
		}
	}

	return 0, custom_errors.ErrNoFreeCluster
}

// findFreeClustersForFile tries to find enough free clusters for the file.
func findFreeClustersForFile(clustersNeeded int, fat []int32) ([]uint32, error) {
	freeClusters := make([]uint32, 0, clustersNeeded)
	for i, entry := range fat {
		if entry == consts.FatFree {
			freeClusters = append(freeClusters, uint32(i))
		}

		if len(freeClusters) == clustersNeeded {
			break
		}
	}

	if len(freeClusters) < clustersNeeded {
		return nil, custom_errors.ErrNoFreeCluster
	}

	return freeClusters, nil
}

// GetRootDirEntry retrieves the root directory entry.
func GetRootDirEntry(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte) (*pseudo_fat.DirectoryEntry, error) {
	// sanity checks
	if pFs == nil || fats == nil || data == nil {
		return nil, custom_errors.ErrNilPointer
	}

	// get the root directory cluster
	rootCluster := uint32(0)

	// read the cluster and deserialize the directory entry
	byteOffset := int(rootCluster)
	clusterData := data[byteOffset : byteOffset+int(pFs.ClusterSize)]

	pDirEntry, err := ReadDirectoryEntryFromCluster(clusterData)
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory entry: %w", err)
	}

	return pDirEntry, nil
}

// GetDirEntries returns a slice of pointers to DirectoryEntry structs that belong to the specified directory.
//
// NOTE: It returns the directory entries of that are from the parent's cluster chain.
// NOTE: It ommits the self reference entry.
func GetDirEntries(pFs *pseudo_fat.FileSystem, pDir *pseudo_fat.DirectoryEntry, fats [][]int32, data []byte) ([](*pseudo_fat.DirectoryEntry), error) {
	// sanity checks
	if pFs == nil || pDir == nil || fats == nil || data == nil {
		return nil, custom_errors.ErrNilPointer
	}
	if pDir.IsFile {
		return nil, custom_errors.ErrIsFile
	}

	fat := fats[0]

	logging.Debug(fmt.Sprintf("Getting directory entries for directory: \"%s\"", GetNormalizedStrFromMem(pDir.Name[:])))

	// get the cluster chain for the directory
	clusterChain, err := GetClusterChain(pDir.StartCluster, fat)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster chain: %w", err)
	}

	var entries [](*pseudo_fat.DirectoryEntry)
	var byteOffset int
	for _, cluster := range clusterChain {
		byteOffset = int(cluster) * int(pFs.ClusterSize)
		clusterData := data[byteOffset : byteOffset+int(pFs.ClusterSize)]
		if IsClusterEmpty(clusterData) {
			logging.Warn(fmt.Sprintf("Cluster %d to read directory entry is empty, skipping...", cluster))
			continue
		}

		pDirEntry, err := ReadDirectoryEntryFromCluster(clusterData)
		if err != nil {
			logging.Error(fmt.Sprintf("Failed to read directory entry from cluster %d: %s", cluster, err))
			continue
		}

		logging.Debug(fmt.Sprintf("Directory entry: \"%s\"", pDirEntry.ToString()))

		if pDirEntry.StartCluster == pDir.StartCluster {
			logging.Debug(fmt.Sprintf("Skipping parent directory entry: \"%s\"", pDirEntry.Name))
			continue
		}
		entries = append(entries, pDirEntry)
	}

	return entries, nil
}

// GetAbsolutePathFromPwd retrieves the absolute path of the specified directory.
func GetAbsolutePathFromPwd(pFs *pseudo_fat.FileSystem, pDir *pseudo_fat.DirectoryEntry, fats [][]int32, data []byte) (string, error) {
	// sanity checks
	if pFs == nil || pDir == nil || fats == nil || data == nil {
		return "", custom_errors.ErrNilPointer
	}

	currDirName := GetNormalizedStrFromMem(pDir.Name[:])
	res := currDirName

	// is root
	if currDirName == consts.PathDelimiter {
		return consts.PathDelimiter, nil
	}

	// traverse the directory tree through parent clusters
	pCurrDir := pDir
	for {
		parentClusterDataIndex := int(pCurrDir.ParentCluster) * int(pFs.ClusterSize)
		parentClusterData := data[parentClusterDataIndex : parentClusterDataIndex+int(pFs.ClusterSize)]
		pParentDir, err := ReadDirectoryEntryFromCluster(parentClusterData)
		if err != nil {
			return "", fmt.Errorf("failed to read parent directory entry: %w", err)
		}

		// prepend the directory name to the result
		parentDirName := GetNormalizedStrFromMem(pParentDir.Name[:])
		if pParentDir.StartCluster == pParentDir.ParentCluster {
			if parentDirName != consts.PathDelimiter {
				return "", fmt.Errorf("highest parent directory is not root")
			}

			// parent will be root
			res = parentDirName + res
			break
		}

		res = parentDirName + consts.PathDelimiter + res
		pCurrDir = pParentDir
	}

	return res, nil
}

// GetBranchDirEntriesFromRoot returns a slice of pointers to DirectoryEntry structs that represent the
// directory entries on the specified path.
//
// Returns ErrEntryNotFound if some entry on the path does not exist.
func GetBranchDirEntriesFromRoot(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte, absPath string) ([](*pseudo_fat.DirectoryEntry), error) {
	// sanity checks
	if pFs == nil || fats == nil || data == nil || absPath == "" {
		return nil, custom_errors.ErrNilPointer
	}

	// get the root directory entry
	pRootDirEntry, err := GetRootDirEntry(pFs, fats, data)
	if err != nil {
		return nil, fmt.Errorf("failed to get root directory entry: %w", err)
	}

	resEntries := make([](*pseudo_fat.DirectoryEntry), 0)

	// edge case: root directory
	if absPath == consts.PathDelimiter || absPath == consts.PathDelimiter+consts.CurrDirSymbol {
		resEntries = append(resEntries, pRootDirEntry)
		return resEntries, nil
	}

	// normalize the path into individual directory names
	nodes, err := GetNormalizedPathNodes(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get normalized path: %w", err)
	}

	// traverse each directory in the path
	pCurrDirEntry := pRootDirEntry
	resEntries = append(resEntries, pCurrDirEntry)
	for i, dirName := range nodes {
		entries, err := GetDirEntries(pFs, pCurrDirEntry, fats, data)
		if err != nil {
			return nil, fmt.Errorf("failed to get directory entries: %w", err)
		}

		nodeFound := false
		for _, pEntry := range entries {
			if GetNormalizedStrFromMem(pEntry.Name[:]) == dirName {
				// skip potential file entries with same name as the last directory in the path (should still work correctly)
				if i < len(nodes)-1 && pEntry.IsFile {
					continue
				}
				logging.Debug(fmt.Sprintf("Found node: \"%s\" on path: \"%s\"", dirName, absPath))
				pCurrDirEntry = pEntry
				nodeFound = true
				resEntries = append(resEntries, pEntry)
				break
			}
		}

		if !nodeFound {
			return nil, custom_errors.ErrEntryNotFound
		}
	}

	return resEntries, nil
}

// entryExists checks if the entry exists on the specified path.
func entryExists(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte, normAbsPath string) (bool, error) {
	// sanity checks
	if pFs == nil || fats == nil || data == nil || normAbsPath == "" {
		return false, custom_errors.ErrNilPointer
	}

	// edge case: root directory
	if normAbsPath == consts.PathDelimiter || normAbsPath == consts.PathDelimiter+consts.CurrDirSymbol {
		return true, nil
	}

	// try to get the directory entry
	_, err := GetBranchDirEntriesFromRoot(pFs, fats, data, normAbsPath)
	if err != nil {
		if err == custom_errors.ErrEntryNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// addToFat adds a new cluster to the FAT chain.
func addToFat(fats [][]int32, clusterIndex uint32, newClusterIndex uint32) {
	for i := 0; i < len(fats); i++ {
		logging.Debug(fmt.Sprintf("Chain for FAT%d: %d -> from %d to %d", i, clusterIndex, fats[i][clusterIndex], newClusterIndex))
		fats[i][clusterIndex] = int32(newClusterIndex)
		logging.Debug(fmt.Sprintf("Chain for FAT%d: %d -> from %d to %d", i, newClusterIndex, fats[i][newClusterIndex], consts.FatFileEnd))
		fats[i][newClusterIndex] = consts.FatFileEnd
	}
}

// markEndOfChain marks the end of the chain in the FAT.
func markEndOfChain(fats [][]int32, clusterIndex uint32) {
	for i := 0; i < len(fats); i++ {
		logging.Debug(fmt.Sprintf("Chain for FAT%d: %d -> from %d to %d", i, clusterIndex, fats[i][clusterIndex], consts.FatFileEnd))
		fats[i][clusterIndex] = consts.FatFileEnd
	}
}

// markFreeCluster marks a cluster as free in the FAT.
func markFreeCluster(fats [][]int32, clusterIndex uint32) {
	for i := 0; i < len(fats); i++ {
		logging.Debug(fmt.Sprintf("Chain for FAT%d: %d -> from %d to %d", i, clusterIndex, fats[i][clusterIndex], consts.FatFree))
		fats[i][clusterIndex] = consts.FatFree
	}
}

// inheritValOfCluster inherits the value of the target cluster to the specified cluster in the FAT.
func inheritValOfCluster(fats [][]int32, receiverClusterIndex uint32, targetClusterIndex uint32) {
	for i := 0; i < len(fats); i++ {
		logging.Debug(fmt.Sprintf("Chain for FAT%d: %d -> from %d to %d", i, receiverClusterIndex, fats[i][receiverClusterIndex], fats[i][targetClusterIndex]))
		fats[i][receiverClusterIndex] = fats[i][targetClusterIndex]
	}
}

// Mkdir creates a new directory in the specified parent directory.
// Expects the absPathToDir to be a valid normalized absolute path.
//
// It returns ErrNoFreeCluster if there are no free clusters in the FAT.
// It returns ErrNilPointer if any of the pointers are nil.
func Mkdir(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte, absNormPathToDir string) error {
	// sanity checks
	if pFs == nil || fats == nil || data == nil {
		return custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Creating directory: \"%s\"", absNormPathToDir))

	pathSegments := GetPathSegments(absNormPathToDir)
	targetDirName := pathSegments[len(pathSegments)-1]

	// if the taget is root
	if targetDirName == consts.PathDelimiter {
		return custom_errors.ErrInvalidPath
	}

	ancestorBranchPath := pathSegments[0] + strings.Join(pathSegments[1:len(pathSegments)-1], consts.PathDelimiter)

	// sanity check for invalid ancestor path
	ancestorEntriesRef, err := GetBranchDirEntriesFromRoot(pFs, fats, data, ancestorBranchPath)
	if err != nil {
		if err == custom_errors.ErrEntryNotFound {
			return custom_errors.ErrPathNotFound
		}
		return err
	}

	// traverse the directory branch from the root directory - everything should be a directory
	for _, pEntry := range ancestorEntriesRef {
		if pEntry.IsFile {
			logging.Warn(fmt.Sprintf("Target entry \"%s\" is a file, not a directory", GetNormalizedStrFromMem(pEntry.Name[:])))
			return custom_errors.ErrInvalidPath
		}
	}

	referencedFat := fats[0]

	pLastDir := ancestorEntriesRef[len(ancestorEntriesRef)-1]

	// check if the target directory entry already exists
	exists, err := entryExists(pFs, fats, data, absNormPathToDir)
	if err != nil {
		return err
	}
	if exists {
		return custom_errors.ErrEntryExists
	}

	// get the cluster chain for the parent directory
	clusterChain, err := GetClusterChain(pLastDir.StartCluster, referencedFat)
	if err != nil {
		return fmt.Errorf("failed to get cluster chain: %w", err)
	}
	clusterEndIndex := clusterChain[len(clusterChain)-1]

	// find a free cluster for the parent directory new entry
	freeClusterIndexParent, err := findFreeCluster(referencedFat)
	if err != nil {
		return err
	}

	// update the parent directory entry chain in the FAT
	addToFat(fats, clusterEndIndex, freeClusterIndexParent)

	// find a free cluster for the new directory entries (including reference to itself)
	freeClusterIndex, err := findFreeCluster(referencedFat)
	if err != nil {
		return err
	}

	// write the new directory entry
	pNewDirEntry := NewDirectoryEntry(false, 0, freeClusterIndex, pLastDir.StartCluster, targetDirName)
	markEndOfChain(fats, freeClusterIndex)

	// serialize the directory entry
	newDirEntryBytes, err := StructToBytes(pNewDirEntry)
	if err != nil {
		return fmt.Errorf("failed to serialize directory entry: %w", err)
	}

	// write the new directory entry to the parent directory cluster
	byteOffset := int(freeClusterIndexParent) * int(pFs.ClusterSize)
	copy(data[byteOffset:], newDirEntryBytes)
	// write the new directory entry to the its own cluster
	byteOffset = int(freeClusterIndex) * int(pFs.ClusterSize)
	copy(data[byteOffset:], newDirEntryBytes)

	return nil
}

// removeParentTargetEntry removes the target entry from the parent directory entry chain.
func removeParentTargetEntry(
	pFs *pseudo_fat.FileSystem,
	fats [][]int32,
	data []byte,
	pParentDirEntry *pseudo_fat.DirectoryEntry,
	pTargetDirEntry *pseudo_fat.DirectoryEntry) error {

	// sanity checks
	if pFs == nil || fats == nil || data == nil || pParentDirEntry == nil || pTargetDirEntry == nil {
		return custom_errors.ErrNilPointer
	}

	fat := fats[0]

	// get the cluster chain for the parent directory
	parentClusterChain, err := GetClusterChain(pParentDirEntry.StartCluster, fat)
	if err != nil {
		return fmt.Errorf("failed to get cluster chain: %w", err)
	}

	var prevClusterIndex int = -1
	var nextClusterIndex int = -1
	for i := 0; i < len(parentClusterChain); i++ {
		currentClusterIndex := int(parentClusterChain[i])

		// load the cluster data
		byteOffset := currentClusterIndex * int(pFs.ClusterSize)
		clusterData := data[byteOffset : byteOffset+int(pFs.ClusterSize)]
		pEntry, err := ReadDirectoryEntryFromCluster(clusterData)
		if err != nil {
			return fmt.Errorf("failed to read directory entry: %w", err)
		}

		if i < len(parentClusterChain)-1 {
			nextClusterIndex = int(parentClusterChain[i+1])
		} else {
			nextClusterIndex = -1
		}

		// found the target entry (with skipped self reference)
		if pEntry.StartCluster != pTargetDirEntry.ParentCluster && GetNormalizedStrFromMem(pEntry.Name[:]) == GetNormalizedStrFromMem(pTargetDirEntry.Name[:]) {
			// target is the only entry in the parent directory but ending index does not exist
			if prevClusterIndex == -1 && nextClusterIndex == -1 {
				return fmt.Errorf("the directory should not exist as parent directory is empty - logic error")

				// target is the only entry in the parent directory
			} else if prevClusterIndex == -1 {
				return fmt.Errorf("the parent directory does not contain self reference - logic error")

				// target is in the middle of the parent directory
			} else if nextClusterIndex != -1 {
				inheritValOfCluster(fats, uint32(prevClusterIndex), uint32(currentClusterIndex))
				markFreeCluster(fats, uint32(currentClusterIndex))

				// target is the last entry in the parent directory
			} else {
				markFreeCluster(fats, uint32(currentClusterIndex))
				markEndOfChain(fats, uint32(prevClusterIndex))
			}

			// free the target parent directory entry
			bytesOffset := currentClusterIndex * int(pFs.ClusterSize)
			copy(data[bytesOffset:], make([]byte, int(pFs.ClusterSize)))

			break
		}

		prevClusterIndex = currentClusterIndex
	}

	return nil
}

// Rmdir removes an existing directory from the specified parent directory.
// Expects the absNormPathToDir to be a valid normalized absolute path.
//
// It returns ErrDirNotFound if the target directory does not exist.
// It returns ErrDirectoryNotEmpty if the directory is not empty.
// It returns ErrInvalidPath if the path is invalid or points to a file.
// It returns ErrNilPointer if any of the pointers are nil.
func Rmdir(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte, p_pwd *pseudo_fat.DirectoryEntry, absNormPathToDir string) error {
	// sanity checks
	if pFs == nil || fats == nil || data == nil || absNormPathToDir == "" {
		return custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Removing directory: \"%s\"", absNormPathToDir))

	pathSegments := GetPathSegments(absNormPathToDir)

	// if the taget is root
	if len(pathSegments) == 0 {
		return custom_errors.ErrInvalidPath
	}

	// get the directory entry of the target directory
	pDirEntries, err := GetBranchDirEntriesFromRoot(pFs, fats, data, absNormPathToDir)
	if err != nil {
		return err
	}
	// all entries should be directories
	for _, pDirEntry := range pDirEntries {
		if pDirEntry.IsFile {
			logging.Warn(fmt.Sprintf("Target entry \"%s\" is a file, not a directory", GetNormalizedStrFromMem(pDirEntry.Name[:])))
			return custom_errors.ErrInvalidPath
		}
	}

	pTargetDirEntry := pDirEntries[len(pDirEntries)-1]
	// check if deletion is valid
	targetEntries, err := GetDirEntries(pFs, pTargetDirEntry, fats, data)
	if err != nil {
		return fmt.Errorf("failed to get directory entries: %w", err)
	}
	// check if the directory is empty
	if len(targetEntries) > 0 {
		return custom_errors.ErrDirNotEmpty
	}

	// if pwd is the target directory
	if pTargetDirEntry.StartCluster == p_pwd.StartCluster {
		return custom_errors.ErrDirInUse
	}

	// get the parent directory entry
	pParentDirEntry := pDirEntries[len(pDirEntries)-2]
	err = removeParentTargetEntry(pFs, fats, data, pParentDirEntry, pTargetDirEntry)
	if err != nil {
		return fmt.Errorf("failed to remove target entry from the parent directory: %w", err)
	}

	// remove the target directory entry
	markFreeCluster(fats, pTargetDirEntry.StartCluster)
	bytesOffset := int(pTargetDirEntry.StartCluster) * int(pFs.ClusterSize)
	copy(data[bytesOffset:], make([]byte, int(pFs.ClusterSize)))

	return nil
}

// CopyInsideFS copies a file to a new location in the filesystem.
func CopyInsideFS(pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte, absNormDestPath string, fileDataRef []byte) error {
	// sanity checks
	if pFs == nil || fatsRef == nil || dataRef == nil || absNormDestPath == "" || fileDataRef == nil {
		return custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Copying file \"%s\" to \"%s\"", absNormDestPath, absNormDestPath))

	// check if the destination path already exists
	exists, err := entryExists(pFs, fatsRef, dataRef, absNormDestPath)
	if err != nil {
		return err
	}
	if exists {
		return custom_errors.ErrEntryExists
	}

	pathSegments := GetPathSegments(absNormDestPath)
	fileName := pathSegments[len(pathSegments)-1]

	// if the taget is root
	if len(pathSegments) == 0 {
		return custom_errors.ErrInvalidPath
	}

	// get the branch for the parent directory
	ancestorBranchPath := strings.Join(pathSegments[:len(pathSegments)-1], consts.PathDelimiter)
	ancestorEntriesRef, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, ancestorBranchPath)
	if err != nil {
		if err == custom_errors.ErrEntryNotFound {
			return custom_errors.ErrPathNotFound
		}
		return err
	}

	// all entries should be directories
	for _, pEntry := range ancestorEntriesRef {
		if pEntry.IsFile {
			logging.Warn(fmt.Sprintf("Target entry \"%s\" is a file, not a directory", GetNormalizedStrFromMem(pEntry.Name[:])))
			return custom_errors.ErrInvalidPath
		}
	}

	referencedFat := fatsRef[0]

	// get the parent directory entry
	pParentDirEntry := ancestorEntriesRef[len(ancestorEntriesRef)-1]

	// figure out if the file will fit into the filesystem
	clustersNeededSelfRef := 1
	clustersNeededParentRef := 1
	clustersNeededData := int(math.Ceil(float64(len(fileDataRef)) / float64(pFs.ClusterSize)))
	clustersNeeded := clustersNeededSelfRef + clustersNeededParentRef + clustersNeededData
	clustersReady, err := findFreeClustersForFile(clustersNeeded, referencedFat)
	if err != nil {
		return err
	}

	// get the cluster chain for the parent directory
	clusterChain, err := GetClusterChain(pParentDirEntry.StartCluster, referencedFat)
	if err != nil {
		return fmt.Errorf("failed to get cluster chain: %w", err)
	}
	clusterEndIndex := clusterChain[len(clusterChain)-1]

	// find a free cluster for the parent directory new entry
	freeClusterIndexParent := clustersReady[0]
	freeClusterIndex := clustersReady[1]
	freeClusterIndicesData := clustersReady[2:]

	// prepare the new directory entry
	pNewDirEntry := NewDirectoryEntry(true, uint32(len(fileDataRef)), freeClusterIndex, pParentDirEntry.StartCluster, fileName)
	newDirEntryBytes, err := StructToBytes(pNewDirEntry)
	if err != nil {
		return fmt.Errorf("failed to serialize directory entry: %w", err)
	}

	// write the new directory entry to the parent directory cluster
	addToFat(fatsRef, clusterEndIndex, freeClusterIndexParent)
	byteOffset := int(freeClusterIndexParent) * int(pFs.ClusterSize)
	copy(dataRef[byteOffset:], newDirEntryBytes)

	// write the new directory entry to the its own cluster
	markEndOfChain(fatsRef, freeClusterIndex)
	byteOffset = int(freeClusterIndex) * int(pFs.ClusterSize)
	copy(dataRef[byteOffset:], newDirEntryBytes)

	// write the file data to the filesystem
	prevIndex := freeClusterIndex
	for i, clusterIndex := range freeClusterIndicesData {
		addToFat(fatsRef, prevIndex, clusterIndex)
		byteOffset = int(clusterIndex) * int(pFs.ClusterSize)
		copy(dataRef[byteOffset:], fileDataRef[i*int(pFs.ClusterSize):])
		prevIndex = clusterIndex
	}

	return nil
}

// GetFileBytes retrieves the content of the specified file.
func GetFileBytes(pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte, absNormSrcPath string) ([]byte, error) {
	// sanity checks
	if pFs == nil || fatsRef == nil || dataRef == nil || absNormSrcPath == "" {
		return nil, custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Returning content of file \"%s\"", absNormSrcPath))

	// get the branch for the source file (returns error if the file does not exist)
	fileEntriesRef, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, absNormSrcPath)
	if err != nil {
		return nil, err
	}

	// get the last entry in the branch
	pEntry := fileEntriesRef[len(fileEntriesRef)-1]
	if !pEntry.IsFile {
		return nil, custom_errors.ErrIsDir
	}

	referecedFat := fatsRef[0]

	// get the cluster chain for the file
	clusterChain, err := GetClusterChain(pEntry.StartCluster, referecedFat)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster chain: %w", err)
	}

	onlyDataChain := clusterChain[1:] // skip the first cluster (directory entry)

	// read the file data
	fileData := make([]byte, 0, int(pEntry.Size))
	remainingSize := int(pEntry.Size)
	for _, clusterIndex := range onlyDataChain {
		byteOffset := int(clusterIndex) * int(pFs.ClusterSize)
		var endOffset int
		if remainingSize > int(pFs.ClusterSize) {
			endOffset = byteOffset + int(pFs.ClusterSize)
			remainingSize -= int(pFs.ClusterSize)
		} else {
			endOffset = byteOffset + remainingSize
		}
		clusterData := dataRef[byteOffset:endOffset]
		fileData = append(fileData, clusterData...)
	}

	return fileData, nil
}

// RemoveFile removes an existing file from the specified parent directory.
// Expects the absNormPathToFile to be a valid normalized absolute path.
func RemoveFile(pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte, absNormPathToFile string) error {
	// sanity checks
	if pFs == nil || fatsRef == nil || dataRef == nil || absNormPathToFile == "" {
		return custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Removing file: \"%s\"", absNormPathToFile))

	// get the directory entry of the target file
	pDirEntries, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, absNormPathToFile)
	if err != nil {
		return err
	}

	// get the last entry in the branch
	pTargetFileEntry := pDirEntries[len(pDirEntries)-1]
	if !pTargetFileEntry.IsFile {
		return custom_errors.ErrIsDir
	}

	// get the parent directory entry
	pParentDirEntry := pDirEntries[len(pDirEntries)-2]

	// remove the target file entry
	err = removeParentTargetEntry(pFs, fatsRef, dataRef, pParentDirEntry, pTargetFileEntry)
	if err != nil {
		return fmt.Errorf("failed to remove target entry from the parent directory: %w", err)
	}

	// get the cluster chain for the file (for freeing the clusters)
	clusterChain, err := GetClusterChain(pTargetFileEntry.StartCluster, fatsRef[0])
	if err != nil {
		return fmt.Errorf("failed to get cluster chain: %w", err)
	}

	// free the file clusters
	for _, clusterIndex := range clusterChain {
		markFreeCluster(fatsRef, clusterIndex)
		bytesOffset := int(clusterIndex) * int(pFs.ClusterSize)
		copy(dataRef[bytesOffset:], make([]byte, int(pFs.ClusterSize)))
	}

	return nil
}

// MoveFile moves an existing file to a new location in the filesystem
// or renames the file if the target path is in the same directory.
//
// Expects the absNormSrcPath and absNormDestPath to be valid normalized absolute paths.
func MoveFile(pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte, absNormSrcPath string, absNormDestPath string) error {
	// sanity checks
	if pFs == nil || fatsRef == nil || dataRef == nil || absNormSrcPath == "" || absNormDestPath == "" {
		return custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Moving file \"%s\" to \"%s\"", absNormSrcPath, absNormDestPath))

	// get the branch for the source file (returns error if the file does not exist)
	fileEntriesRef, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, absNormSrcPath)
	if err != nil {
		return err
	}

	// get the last entry in the branch and its parent
	pSrcEntry := fileEntriesRef[len(fileEntriesRef)-1]
	if !pSrcEntry.IsFile {
		return custom_errors.ErrIsDir
	}
	pSrcParentEntry := fileEntriesRef[len(fileEntriesRef)-2]

	// check if the destination path already exists
	exists, err := entryExists(pFs, fatsRef, dataRef, absNormDestPath)
	if err != nil {
		return err
	}
	if exists {
		return custom_errors.ErrEntryExists
	}

	// get the closest common ancestor of the source and destination paths
	srcSegments := GetPathSegments(absNormSrcPath)
	destSegments := GetPathSegments(absNormDestPath)
	originalSrcName := srcSegments[len(srcSegments)-1]
	destName := destSegments[len(destSegments)-1]
	ancestorSrc := strings.Join(srcSegments[:len(srcSegments)-1], consts.PathDelimiter)
	ancestorDest := strings.Join(destSegments[:len(destSegments)-1], consts.PathDelimiter)

	// prepare the new directory entry
	pNewDirEntry := NewDirectoryEntry(true, pSrcEntry.Size, pSrcEntry.StartCluster, pSrcParentEntry.StartCluster, destName)
	newDirEntryBytes, err := StructToBytes(pNewDirEntry)
	if err != nil {
		return fmt.Errorf("failed to serialize directory entry: %w", err)
	}

	referecedFat := fatsRef[0]

	// if the source and destination are the same, only the name is changed
	if ancestorSrc == ancestorDest {
		// find the cluster index of the source parents entry
		srcParentClusterChain, err := GetClusterChain(pSrcParentEntry.StartCluster, referecedFat)
		if err != nil {
			return fmt.Errorf("failed to get cluster chain: %w", err)
		}
		for _, clusterIndex := range srcParentClusterChain {
			byteOffset := int(clusterIndex) * int(pFs.ClusterSize)
			clusterData := dataRef[byteOffset : byteOffset+int(pFs.ClusterSize)]
			pEntry, err := ReadDirectoryEntryFromCluster(clusterData)
			if err != nil {
				return fmt.Errorf("failed to read directory entry: %w", err)
			}

			if GetNormalizedStrFromMem(pEntry.Name[:]) == originalSrcName {
				copy(dataRef[byteOffset:], newDirEntryBytes)
				break
			}
		}

		// write the new directory entry to the its own cluster
		byteOffset := int(pNewDirEntry.StartCluster) * int(pFs.ClusterSize)
		copy(dataRef[byteOffset:], newDirEntryBytes)

		return nil

		// if the source and destination are different, the file is moved
	} else {
		// get the branch for the destination directory
		pDestEntries, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, ancestorDest)
		if err != nil {
			if err == custom_errors.ErrEntryNotFound {
				return custom_errors.ErrPathNotFound
			}
			return err
		}
		pNewParentEntry := pDestEntries[len(pDestEntries)-1]
		if pNewParentEntry.IsFile {
			return custom_errors.ErrIsFile
		}

		// remove the source file entry from the parent directory entry chain
		err = removeParentTargetEntry(pFs, fatsRef, dataRef, pSrcParentEntry, pSrcEntry)
		if err != nil {
			return fmt.Errorf("failed to remove target entry from the parent directory: %w", err)
		}

		// find a free cluster for the new parent directory entry
		freeClusterIndexParent, err := findFreeCluster(referecedFat)
		if err != nil {
			return err
		}

		// find last cluster in the new parent directory chain
		clusterChain, err := GetClusterChain(pNewParentEntry.StartCluster, referecedFat)
		if err != nil {
			return fmt.Errorf("failed to get cluster chain: %w", err)
		}
		pNewParentLastCluster := clusterChain[len(clusterChain)-1]

		// update the parent directory entry chain in the FAT
		addToFat(fatsRef, pNewParentLastCluster, freeClusterIndexParent)

		// write the new directory entry to the parent directory cluster
		byteOffset := int(freeClusterIndexParent) * int(pFs.ClusterSize)
		copy(dataRef[byteOffset:], newDirEntryBytes)

		// write the new directory entry to the its own cluster
		byteOffset = int(pNewDirEntry.StartCluster) * int(pFs.ClusterSize)
		copy(dataRef[byteOffset:], newDirEntryBytes)
	}

	return nil
}

// CopyFile copies a file to a new location in the filesystem.
//
// Expects the absNormSrcPath and absNormDestPath to be valid normalized absolute paths.
func CopyFile(pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte, absNormSrcPath string, absNormDestPath string) error {
	// sanity checks
	if pFs == nil || fatsRef == nil || dataRef == nil || absNormSrcPath == "" || absNormDestPath == "" {
		return custom_errors.ErrNilPointer
	}

	logging.Debug(fmt.Sprintf("Copying file \"%s\" to \"%s\"", absNormSrcPath, absNormDestPath))

	// get the branch for the source file (returns error if the file does not exist)
	fileEntriesRef, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, absNormSrcPath)
	if err != nil {
		return err
	}

	// get the last entry in the branch
	pSrcEntry := fileEntriesRef[len(fileEntriesRef)-1]
	if !pSrcEntry.IsFile {
		return custom_errors.ErrIsDir
	}

	referecedFat := fatsRef[0]

	// check if the destination path already exists
	exists, err := entryExists(pFs, fatsRef, dataRef, absNormDestPath)
	if err != nil {
		return err
	}
	if exists {
		return custom_errors.ErrEntryExists
	}

	// get the closest common ancestor of the source and destination paths
	destSegments := GetPathSegments(absNormDestPath)
	destName := destSegments[len(destSegments)-1]
	ancestorDest := strings.Join(destSegments[:len(destSegments)-1], consts.PathDelimiter)

	// prepare the space for the new file
	clustersNeededData := int(math.Ceil(float64(pSrcEntry.Size) / float64(pFs.ClusterSize)))
	clustersNeededSelfRef := 1
	clustersNeededParentRef := 1
	clustersNeeded := clustersNeededData + clustersNeededSelfRef + clustersNeededParentRef

	// find clusters for the new file
	clustersReady, err := findFreeClustersForFile(clustersNeeded, referecedFat)
	if err != nil {
		return err
	}

	freeClusterIndexParRef := clustersReady[0]
	freeClusterIndexSelfRef := clustersReady[1]
	freeClusterIndicesData := clustersReady[2:]

	// get the branch for the destination directory
	pDestEntries, err := GetBranchDirEntriesFromRoot(pFs, fatsRef, dataRef, ancestorDest)
	if err != nil {
		if err == custom_errors.ErrEntryNotFound {
			return custom_errors.ErrPathNotFound
		}
		return err
	}
	pNewParentEntry := pDestEntries[len(pDestEntries)-1]
	if pNewParentEntry.IsFile {
		return custom_errors.ErrIsFile
	}

	// prepare the new directory entry
	pNewDirEntry := NewDirectoryEntry(true, pSrcEntry.Size, freeClusterIndexSelfRef, pNewParentEntry.StartCluster, destName)
	newDirEntryBytes, err := StructToBytes(pNewDirEntry)
	if err != nil {
		return fmt.Errorf("failed to serialize directory entry: %w", err)
	}

	// find the last cluster in the new parent directory chain
	clusterChain, err := GetClusterChain(pNewParentEntry.StartCluster, referecedFat)
	if err != nil {
		return fmt.Errorf("failed to get cluster chain: %w", err)
	}
	lastParentClusterIndex := clusterChain[len(clusterChain)-1]

	// write the new directory entry to the parent directory cluster
	addToFat(fatsRef, lastParentClusterIndex, freeClusterIndexParRef)
	byteOffset := int(freeClusterIndexParRef) * int(pFs.ClusterSize)
	copy(dataRef[byteOffset:], newDirEntryBytes)

	// write the new directory entry to the its own cluster
	markEndOfChain(fatsRef, freeClusterIndexSelfRef)
	byteOffset = int(freeClusterIndexSelfRef) * int(pFs.ClusterSize)
	copy(dataRef[byteOffset:], newDirEntryBytes)

	// get the file data
	fileData, err := GetFileBytes(pFs, fatsRef, dataRef, absNormSrcPath)
	if err != nil {
		return fmt.Errorf("failed to get file data: %w", err)
	}

	// write the file data to the filesystem
	prevIndex := freeClusterIndexSelfRef
	for i, clusterIndex := range freeClusterIndicesData {
		addToFat(fatsRef, prevIndex, clusterIndex)
		byteOffset = int(clusterIndex) * int(pFs.ClusterSize)
		copy(dataRef[byteOffset:], fileData[i*int(pFs.ClusterSize):])
		prevIndex = clusterIndex
	}

	return nil
}
