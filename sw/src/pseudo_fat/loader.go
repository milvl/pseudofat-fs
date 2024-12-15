// pseudo_fat package contains the implementation of a pseudo FAT file system.
package pseudo_fat

import (
	"fmt"
	"io"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/utils"
	"os"
	"unsafe"
)

// validateFileSystem checks if the file system is valid by performing a series of logical checks.
func validateFileSystem(pFs *FileSystem) error {
	// TODO: Check this

	// sanity check
	if pFs == nil {
		return custom_errors.ErrNilPointer
	}

	// check basic things
	if string(pFs.signature[:]) != consts.AuthorID {
		return custom_errors.ErrInvalidFileSys
	}

	if pFs.clusterSize == 0 || pFs.diskSize == 0 || pFs.fatCount == 0 {
		return custom_errors.ErrInvalidFileSys
	}

	// check if disk size can accommodate the FAT tables and data
	fatSize := pFs.fatCount * 4 // Assuming 4 bytes per FAT entry
	dataRegionStart := pFs.fat02StartAddr

	if pFs.fat01StartAddr+fatSize > pFs.fat02StartAddr {
		return fmt.Errorf("FAT1 overlaps with FAT2")
	}

	if dataRegionStart < (pFs.fat02StartAddr + fatSize) {
		return fmt.Errorf("Data region overlaps with FAT2")
	}

	// Check if the disk size is sufficient for clusters and metadata
	totalClusters := (pFs.diskSize - pFs.dataStartAddr) / pFs.clusterSize
	if totalClusters == 0 {
		return fmt.Errorf("Insufficient space for any clusters")
	}

	if totalClusters < pFs.fatCount {
		return fmt.Errorf("Number of FAT entries exceeds total clusters")
	}

	// Ensure cluster size is a reasonable value (e.g., not too small or large)
	if pFs.clusterSize < 512 || pFs.clusterSize > 1024*1024 {
		return fmt.Errorf("Cluster size %d is out of acceptable range (512B - 1MB)", pFs.clusterSize)
	}

	// Check if the start addresses align with cluster boundaries
	if pFs.fat01StartAddr%pFs.clusterSize != 0 ||
		pFs.fat02StartAddr%pFs.clusterSize != 0 ||
		pFs.dataStartAddr%pFs.clusterSize != 0 {
		return fmt.Errorf("Start addresses must align with cluster boundaries")
	}

	return nil
}

func GetFileSystem(file *os.File) (*FileSystem, []byte, error) {
	// sanity check
	if file == nil {
		return nil, nil, custom_errors.ErrNilPointer
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	if fileInfo.Size() == int64(0) {
		// if the file is empty, return an uninitialized file system
		return GetUninitializedFileSystem(), nil, nil

	} else if fileInfo.Size() < int64(unsafe.Sizeof(FileSystem{})) {
		// if the file is smaller than it does not contain a file system or is corrupted
		// inform the user and return an uninitialized file system
		fmt.Println(consts.FileNotFilesys)
		return GetUninitializedFileSystem(), nil, nil
	}

	// try to read the file system
	pFs := FileSystem{}
	fsBytes := make([]byte, unsafe.Sizeof(FileSystem{}))
	_, err = file.ReadAt(fsBytes, io.SeekStart)
	if err != nil {
		return nil, nil, err
	}

	// try to convert the bytes to a file system
	err = utils.BytesToStruct(fsBytes, &pFs)
	if err != nil {
		return nil, nil, err
	}

	// check if the file system is valid
	err = validateFileSystem(pFs)
	return nil, nil, nil //TODO: Implement this

}
