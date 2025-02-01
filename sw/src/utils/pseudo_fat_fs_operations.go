// utils is a package that contains utility functions for the ZOS project.
package utils

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"kiv-zos-semestral-work/pseudo_fat"
	"strings"
)

// getClusterChain traverses the FAT to collect all clusters in the directory's chain.
func getClusterChain(startCluster uint32, fat []int32) ([]uint32, error) {
	if int32(startCluster) == consts.FatFree {
		return nil, custom_errors.ErrInvalStartCluster
	}

	var chain []uint32
	current := startCluster

	for {
		// validate cluster index
		if current >= uint32(len(fat)) {
			return nil, fmt.Errorf("cluster index %d out of bounds", current)
		}

		chain = append(chain, current)

		next := fat[current]

		if next == consts.FatFileEnd {
			break
		}

		if next < 0 {
			// negative values (other than consts.FatFileEnd) can be considered invalid or used for other purposes
			return nil, fmt.Errorf("invalid FAT entry at cluster %d: %d", current, next)
		}

		current = uint32(next)
	}

	return chain, nil
}

// readDirectoryEntryFromCluster deserializes DirectoryEntry structs from a specific cluster.
func readDirectoryEntryFromCluster(clusterData []byte) (*pseudo_fat.DirectoryEntry, error) {
	// sanity check
	if clusterData == nil {
		return nil, custom_errors.ErrNilPointer
	}

	entry := pseudo_fat.DirectoryEntry{}
	err := BytesToStruct(clusterData, &entry)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize directory entry: %w", err)
	}

	if entry.IsFile {
		return nil, fmt.Errorf("entry is a file, not a directory")
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

	pDirEntry, err := readDirectoryEntryFromCluster(clusterData)
	if err != nil {
		return nil, fmt.Errorf("failed to read root directory entry: %w", err)
	}

	return pDirEntry, nil
}

// GetDirEntries returns a slice of pointers to DirectoryEntry structs that belong to the specified directory.
func GetDirEntries(pFs *pseudo_fat.FileSystem, pDir *pseudo_fat.DirectoryEntry, fats [][]int32, data []byte) ([](*pseudo_fat.DirectoryEntry), error) {
	// sanity checks
	if pFs == nil || pDir == nil || fats == nil || data == nil {
		return nil, custom_errors.ErrNilPointer
	}
	if pDir.IsFile {
		return nil, custom_errors.ErrIsFile
	}

	fat := fats[0]

	// get the cluster chain for the directory
	clusterChain, err := getClusterChain(pDir.StartCluster, fat)
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

		pDirEntry, err := readDirectoryEntryFromCluster(clusterData)
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
// TODO: check if this really works
func GetAbsolutePathFromPwd(pFs *pseudo_fat.FileSystem, pDir *pseudo_fat.DirectoryEntry, fats [][]int32, data []byte) (string, error) {
	// sanity checks
	if pFs == nil || pDir == nil || fats == nil || data == nil {
		return "", custom_errors.ErrNilPointer
	}

	currDirName := getNormalizedStrFromMem(pDir.Name[:])
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
		pParentDir, err := readDirectoryEntryFromCluster(parentClusterData)
		if err != nil {
			return "", fmt.Errorf("failed to read parent directory entry: %w", err)
		}

		// prepend the directory name to the result
		parentDirName := getNormalizedStrFromMem(pParentDir.Name[:])
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
	for _, dirName := range nodes {
		entries, err := GetDirEntries(pFs, pCurrDirEntry, fats, data)
		if err != nil {
			return nil, fmt.Errorf("failed to get directory entries: %w", err)
		}

		nodeFound := false
		for _, pEntry := range entries {
			if !pEntry.IsFile && getNormalizedStrFromMem(pEntry.Name[:]) == dirName {
				logging.Debug(fmt.Sprintf("Found node: \"%s\" on path: \"%s\"", dirName, absPath))
				pCurrDirEntry = pEntry
				nodeFound = true
				resEntries = append(resEntries, pEntry)
				break
			}
		}

		if !nodeFound {
			return nil, custom_errors.ErrDirNotFound
		}
	}

	return resEntries, nil
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

// connectClusters connects two clusters in the FAT chain.
func connectClusters(fats [][]int32, clusterIndex uint32, nextClusterIndex uint32) {
	for i := 0; i < len(fats); i++ {
		logging.Debug(fmt.Sprintf("Chain for FAT%d: %d -> from %d to %d", i, clusterIndex, fats[i][clusterIndex], nextClusterIndex))
		fats[i][clusterIndex] = int32(nextClusterIndex)
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

	pathSegments := GetPathSegments(absNormPathToDir)
	targetDirName := pathSegments[len(pathSegments)-1]
	ancestorBranchPath := strings.Join(pathSegments[:len(pathSegments)-1], consts.PathDelimiter)

	// sanity check for invalid ancestor path
	ancestorEntriesRef, err := GetBranchDirEntriesFromRoot(pFs, fats, data, ancestorBranchPath)
	if err != nil {
		return err
	}

	// traverse the directory branch from the root directory - everything should be a directory
	for _, pEntry := range ancestorEntriesRef {
		if pEntry.IsFile {
			logging.Warn(fmt.Sprintf("Target entry \"%s\" is a file, not a directory", getNormalizedStrFromMem(pEntry.Name[:])))
			return custom_errors.ErrInvalidPath
		}
	}

	referencedFat := fats[0]

	pLastDir := ancestorEntriesRef[len(ancestorEntriesRef)-1]

	// get the cluster chain for the parent directory
	clusterChain, err := getClusterChain(pLastDir.StartCluster, referencedFat)
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
func removeParentTargetEntry(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte, pParentDirEntry *pseudo_fat.DirectoryEntry, pTargetDirEntry *pseudo_fat.DirectoryEntry) error {
	// sanity checks
	if pFs == nil || fats == nil || data == nil || pParentDirEntry == nil || pTargetDirEntry == nil {
		return custom_errors.ErrNilPointer
	}

	fat := fats[0]

	// get the cluster chain for the parent directory
	clusterChain, err := getClusterChain(pParentDirEntry.StartCluster, fat)
	if err != nil {
		return fmt.Errorf("failed to get cluster chain: %w", err)
	}

	// find the index of entry in the parent directory entry chain and the index of its ancestor
	var targetEntryIndex int = -1
	var targetEntryAncestorIndex int = -1
	for _, entryIndex := range clusterChain {
		// load the cluster data
		byteOffset := int(entryIndex) * int(pFs.ClusterSize)
		clusterData := data[byteOffset : byteOffset+int(pFs.ClusterSize)]
		pEntry, err := readDirectoryEntryFromCluster(clusterData)
		if err != nil {
			return fmt.Errorf("failed to read directory entry: %w", err)
		}

		if getNormalizedStrFromMem(pEntry.Name[:]) == getNormalizedStrFromMem(pTargetDirEntry.Name[:]) {
			targetEntryIndex = int(entryIndex)
			break
		}

		targetEntryAncestorIndex = int(entryIndex)
	}

	if targetEntryIndex == -1 {
		return fmt.Errorf("target directory entry not found in the parent directory - logic error")
	}

	// target is last entry in the parent directory
	if fat[targetEntryIndex] == consts.FatFileEnd {
		// mark the parent directory entry as empty
		markFreeCluster(fats, uint32(targetEntryIndex))
		markEndOfChain(fats, uint32(targetEntryAncestorIndex))
	} else {
		// rewiring the FAT chain
		connectClusters(fats, uint32(targetEntryAncestorIndex), uint32(fat[targetEntryIndex]))
		markFreeCluster(fats, uint32(targetEntryIndex))
	}

	// free the target parent directory entry
	bytesOffset := targetEntryIndex * int(pFs.ClusterSize)
	copy(data[bytesOffset:], make([]byte, int(pFs.ClusterSize)))

	return nil
}

// Rmdir removes an existing directory from the specified parent directory.
// Expects the absNormPathToDir to be a valid normalized absolute path.
//
// It returns ErrDirNotFound if the target directory does not exist.
// It returns ErrDirectoryNotEmpty if the directory is not empty.
// It returns ErrInvalidPath if the path is invalid or points to a file.
// It returns ErrNilPointer if any of the pointers are nil.
func Rmdir(pFs *pseudo_fat.FileSystem, fats [][]int32, data []byte, absNormPathToDir string) error {
	// sanity checks
	if pFs == nil || fats == nil || data == nil || absNormPathToDir == "" {
		return custom_errors.ErrNilPointer
	}

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
			logging.Warn(fmt.Sprintf("Target entry \"%s\" is a file, not a directory", getNormalizedStrFromMem(pDirEntry.Name[:])))
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
