// pseudo_fat package contains the implementation of a pseudo FAT file system.
package pseudo_fat

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
	"unsafe"
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

// GetSizeOfFileSystem returns the size of the FileSystem struct in bytes
func GetSizeOfFileSystem() uintptr {
	fs := GetUninitializedFileSystem()
	size := uintptr(0)
	size += unsafe.Sizeof(fs.DiskSize)
	size += unsafe.Sizeof(fs.FatCount)
	size += unsafe.Sizeof(fs.Fat01StartAddr)
	size += unsafe.Sizeof(fs.Fat02StartAddr)
	size += unsafe.Sizeof(fs.DataStartAddr)
	size += unsafe.Sizeof(fs.ClusterSize)
	size += unsafe.Sizeof(fs.Signature)

	return size
}

// DirectoryEntry is a struct representing an item in a directory. It is 20 bytes long.
type DirectoryEntry struct {
	// Name is the name of the file or directory
	Name [consts.MaxFileNameLength]byte
	// IsFile is a flag indicating if the item is a file
	IsFile bool
	// Size is the size of the file in bytes
	Size uint32
	// StartCluster is the start cluster of the file
	StartCluster uint32
	// ParentCluster is the start cluster of the parent directory
	ParentCluster uint32
}

// ToString returns a string representation of the directory entry
func (d *DirectoryEntry) ToString() string {
	return "DirectoryEntry{" +
		"Name: " + string(d.Name[:]) +
		", IsFile: " + fmt.Sprint(d.IsFile) +
		", Size: " + fmt.Sprint(d.Size) +
		", StartCluster: " + fmt.Sprint(d.StartCluster) +
		", ParentCluster: " + fmt.Sprint(d.ParentCluster) +
		"}"
}
