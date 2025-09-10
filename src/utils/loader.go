// utils package contains utility functions for the file system.
package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"kiv-zos-semestral-work/consts"
	"kiv-zos-semestral-work/custom_errors"
	"kiv-zos-semestral-work/logging"
	"kiv-zos-semestral-work/pseudo_fat"
	"os"
	"unsafe"
)

// validateFileSystem checks if the file system is valid by performing a series of logical checks.
func validateFileSystem(pFs *pseudo_fat.FileSystem) error {
	// sanity check
	if pFs == nil {
		return custom_errors.ErrNilPointer
	}

	logging.Info(fmt.Sprintf("Validating file system: %s", pFs.ToString()))

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
	logging.Debug(fmt.Sprintf("FAT size calculated: %d", fatSize))

	minRequiredSize := uint32(consts.FATableCount)*fatSize + uint32(pFs.ClusterSize)
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
	} else if pFs.Fat01StartAddr != uint32(pseudo_fat.GetSizeOfFileSystem()) {
		logging.Debug(fmt.Sprintf("size of FileSystem: %d", uint32(pseudo_fat.GetSizeOfFileSystem())))
		logging.Info(fmt.Sprintf("FAT01 overlaps the file system structure (fat01StartAddr: %d)", pFs.Fat01StartAddr))
		return custom_errors.ErrInvalidFileSys
	}

	allocatableSpace := pFs.DiskSize - pFs.DataStartAddr
	logging.Debug(fmt.Sprintf("Allocatable space calculated: %d", allocatableSpace))
	clusterCount := allocatableSpace / uint32(pFs.ClusterSize)
	logging.Debug(fmt.Sprintf("Cluster count calculated: %d", clusterCount))
	if clusterCount != pFs.FatCount {
		logging.Info(fmt.Sprintf("Cluster count does not match FAT count (clusterCount: %d, fatCount: %d)", clusterCount, pFs.FatCount))
		return custom_errors.ErrInvalidFileSys
	}

	return nil
}

// GetFileSystem reads the file system from the file and returns it along with the FAT tables and data region.
//
// If the file is not a valid file system, an uninitialized file system is returned that can be used to format the file system.
// If IO error occurs, it is returned.
func GetFileSystem(file *os.File) (*pseudo_fat.FileSystem, *[][]int32, *[]byte, error) {
	// sanity check
	if file == nil {
		return nil, nil, nil, custom_errors.ErrNilPointer
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, nil, nil, err
	}

	// prepare uninitialized variables
	pUninitFs := pseudo_fat.GetUninitializedFileSystem()
	var uninitFatsRef [][]int32 = nil
	var uninitDataRef []byte = nil

	if fileInfo.Size() == int64(0) {
		// if the file is empty, return an uninitialized file system
		logging.Info("File is empty")
		return pUninitFs, &uninitFatsRef, &uninitDataRef, nil

	} else if fileInfo.Size() < int64(pseudo_fat.GetSizeOfFileSystem()) {
		// if the file is smaller than it does not contain a file system or is corrupted
		// inform the user and return an uninitialized file system
		logging.Info("File is too small")
		fmt.Println(consts.FileNotFilesys)
		return pUninitFs, &uninitFatsRef, &uninitDataRef, nil
	}

	// try to read the file system
	pFs := pseudo_fat.FileSystem{}
	fsBytes := make([]byte, pseudo_fat.GetSizeOfFileSystem())
	_, err = file.ReadAt(fsBytes, io.SeekStart)
	if err != nil {
		return nil, nil, nil, err
	}

	// try to convert the bytes to a file system
	err = BytesToStruct(fsBytes, &pFs)
	if err != nil {
		return nil, nil, nil, err
	}

	// check if the file system is valid
	err = validateFileSystem(&pFs)
	if err != nil {
		logging.Info("File system is invalid")
		return pseudo_fat.GetUninitializedFileSystem(), &uninitFatsRef, &uninitDataRef, err
	}

	// if all checks passed, file system is valid
	logging.Info("File system is valid")

	fats := make([][]int32, consts.FATableCount)
	for i := 0; i < int(consts.FATableCount); i++ {
		fats[i] = make([]int32, pFs.FatCount)
	}

	// load the FAT tables
	fatsSize := uint32(consts.FATableCount) * pFs.FatCount * uint32(unsafe.Sizeof(int32(0)))
	fatsBytes := make([]byte, fatsSize)
	_, err = file.ReadAt(fatsBytes, int64(pseudo_fat.GetSizeOfFileSystem()))
	if err != nil {
		return nil, nil, nil, err
	}

	// convert the bytes to the FAT tables
	for i := 0; i < int(consts.FATableCount); i++ {
		offset := i * int(pFs.FatCount) * int(unsafe.Sizeof(int32(0)))
		err = binary.Read(bytes.NewReader(fatsBytes[offset:]), binary.LittleEndian, &fats[i])
		if err != nil {
			return nil, nil, nil, err
		}
	}

	logging.Debug(fmt.Sprintf("FAT tables loaded: \n%s", PFormatFats(fats)))

	// load the data region
	dataSize := pFs.DiskSize - pFs.DataStartAddr
	data := make([]byte, dataSize)
	_, err = file.ReadAt(data, int64(pFs.DataStartAddr))
	if err != nil {
		return nil, nil, nil, err
	}

	return &pFs, &fats, &data, nil
}
