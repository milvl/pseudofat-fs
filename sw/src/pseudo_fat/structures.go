// pseudo_fat package contains the implementation of a pseudo FAT file system.
package pseudo_fat

import "kiv-zos-semestral-work/consts"

// FileSystem is a struct representing the pseudo FAT file system
type FileSystem struct {
	// signature is the ID of the author of the file system
	signature [consts.StudentNumLen]byte
	// diskSize is the size of the disk in bytes
	diskSize uint32
	// clusterSize is the size of a cluster in bytes
	clusterSize uint32
	// fatCount is the number of FATs in the file system
	fatCount uint32
	// fat01StartAddr is the start address of the first FAT
	fat01StartAddr uint32
	// fat02StartAddr is the start address of the second FAT
	fat02StartAddr uint32
	// dataStartAddr is the start address of the data region
	dataStartAddr uint32
}

// GetUninitializedFileSystem returns an uninitialized file system
func GetUninitializedFileSystem() *FileSystem {
	return &FileSystem{}
}

// DirectoryEntry is a struct representing an item in a directory
type DirectoryEntry struct {
	// name is the name of the file or directory
	name string
	// isFile is a flag indicating if the item is a file
	isFile bool
	// size is the size of the file in bytes
	size uint32
	// startCluster is the start cluster of the file
	startCluster uint32
}
