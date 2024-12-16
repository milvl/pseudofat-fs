// pseudo_fat package contains the implementation of a pseudo FAT file system.
package pseudo_fat

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
)

// FileSystem is a struct representing the pseudo FAT file system. It is 31 bytes long.
//
// WARNING: The variables are ordered in a way that they are aligned in memory with the
// smallest possible padding. This is important for the byte handling in the loader.go.
type FileSystem struct {
	// DiskSize is the size of the disk in bytes
	DiskSize uint32
	// FatCount is the number of records in the FAT
	FatCount uint32
	// Fat01StartAddr is the start address of the first FAT
	Fat01StartAddr uint32
	// Fat02StartAddr is the start address of the second FAT
	Fat02StartAddr uint32
	// DataStartAddr is the start address of the data region
	DataStartAddr uint32
	// ClusterSize is the size of a cluster in bytes
	ClusterSize uint16
	// Signature is the ID of the author of the file system
	Signature [consts.StudentNumLen]byte
}

// ToString returns a string representation of the file system
func (fs *FileSystem) ToString() string {
	signature := string(fs.Signature[:])
	return "FileSystem{" +
		"Signature: " + signature +
		", DiskSize: " + fmt.Sprint(fs.DiskSize) +
		", ClusterSize: " + fmt.Sprint(fs.ClusterSize) +
		", FatCount: " + fmt.Sprint(fs.FatCount) +
		", Fat01StartAddr: " + fmt.Sprint(fs.Fat01StartAddr) +
		", Fat02StartAddr: " + fmt.Sprint(fs.Fat02StartAddr) +
		", DataStartAddr: " + fmt.Sprint(fs.DataStartAddr) +
		"}"
}

// GetUninitializedFileSystem returns an uninitialized file system
func GetUninitializedFileSystem() *FileSystem {
	return &FileSystem{}
}

// DirectoryEntry is a struct representing an item in a directory. It is 20 bytes long.
type DirectoryEntry struct {
	// name is the name of the file or directory
	name [consts.MaxFileNameLength]byte
	// isFile is a flag indicating if the item is a file
	isFile bool
	// size is the size of the file in bytes
	size uint32
	// startCluster is the start cluster of the file
	startCluster uint32
}
