// pseudo_fat package contains the implementation of a pseudo FAT file system.
package pseudo_fat

import (
	"fmt"
	"io"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"kiv-zos-semestral-work/utils"
	"os"
	"unsafe"
)

// validateFileSystem checks if the file system is valid by performing a series of logical checks.
func validateFileSystem(pFs *FileSystem) error {
	// sanity check
	if pFs == nil {
		return custom_errors.ErrNilPointer
	}

	// check basic things
	if string(pFs.Signature[:]) != consts.AuthorID {
		logging.Info(fmt.Sprintf("Invalid signature: %s", string(pFs.Signature[:])))
		return custom_errors.ErrInvalidFileSys
	}

	// no size
	if pFs.ClusterSize <= 0 || pFs.DiskSize <= 0 || pFs.FatCount <= 0 {
		logging.Info(fmt.Sprintf("Size too small (clusterSize: %d, diskSize: %d, fatCount: %d)", pFs.ClusterSize, pFs.DiskSize, pFs.FatCount))
		return custom_errors.ErrInvalidFileSys
	}

	// beyond limits
	if pFs.ClusterSize > consts.ClusterSize || pFs.DiskSize > consts.MaxFilesystemSize || pFs.FatCount > consts.MaxClusterCount {
		logging.Info(fmt.Sprintf("Size beyond limits (clusterSize: %d, diskSize: %d, fatCount: %d)", pFs.ClusterSize, pFs.DiskSize, pFs.FatCount))
		return custom_errors.ErrInvalidFileSys
	}

	// check if disk size can accommodate the FAT tables and data
	fatSize := pFs.FatCount * uint32(unsafe.Sizeof(int32(0)))

	minRequiredSize := 2*fatSize + uint32(pFs.ClusterSize)
	if pFs.DiskSize < minRequiredSize {
		logging.Info(fmt.Sprintf("Disk size too small (required: %d, available: %d)", minRequiredSize, pFs.DiskSize))
		return custom_errors.ErrInvalidFileSys
	}

	// check if the FAT tables overlap
	if pFs.Fat01StartAddr+fatSize > pFs.Fat02StartAddr {
		logging.Info(fmt.Sprintf("FAT tables overlap (fat01StartAddr: %d, fat02StartAddr: %d)", pFs.Fat01StartAddr, pFs.Fat02StartAddr))
		return custom_errors.ErrInvalidFileSys
	} else if pFs.DataStartAddr < (pFs.Fat02StartAddr + fatSize) {
		logging.Info(fmt.Sprintf("Data region overlaps FAT tables (dataStartAddr: %d, fat02StartAddr: %d, fatSize: %d)", pFs.DataStartAddr, pFs.Fat02StartAddr, fatSize))
		return custom_errors.ErrInvalidFileSys
	} else if pFs.Fat01StartAddr != uint32(unsafe.Sizeof(FileSystem{})) {
		logging.Debug(fmt.Sprintf("size of FileSystem: %d", unsafe.Sizeof(FileSystem{})))
		logging.Info(fmt.Sprintf("FAT01 overlaps the file system structure (fat01StartAddr: %d)", pFs.Fat01StartAddr))
		return custom_errors.ErrInvalidFileSys
	}

	allocatableSpace := pFs.DiskSize - pFs.DataStartAddr
	clusterCount := allocatableSpace / uint32(pFs.ClusterSize)
	if clusterCount != pFs.FatCount {
		logging.Info(fmt.Sprintf("Cluster count does not match FAT count (clusterCount: %d, fatCount: %d)", clusterCount, pFs.FatCount))
		return custom_errors.ErrInvalidFileSys
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
	err = validateFileSystem(&pFs)
	if err != nil {
		return nil, nil, err
	}

	// exit the system now (DEBUG)
	logging.Critical("all good, now please be so kind and exit the freaking unfinished system")
	return &pFs, fsBytes, err
}
