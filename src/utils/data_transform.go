// utils package contains utility functions for the project
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

// StructToBytes converts a struct to bytes. It writes
// the data in little endian encoding. It does ignore
// padding.
func StructToBytes(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)
	if err != nil {
		return nil, custom_errors.ErrStructToBytes
	}

	return buf.Bytes(), nil
}

// BytesToStruct converts bytes to a struct. The data
// are put into the result interface. It expects little
// endian encoding and does not handle padding.
func BytesToStruct(data []byte, pResult interface{}) error {
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, pResult)
	if err != nil {
		return custom_errors.ErrBytesToStruct
	}

	return nil
}

// ParseFSSize parses the size of the filesystem from the string.
//
// No sanity checks are performed here, because the data should
// already come validated from the command parser.
func ParseFSSize(pSize string) (uint32, error) {
	// find the unit
	var unit string
	var parsedSize int
	_, err := fmt.Sscanf(pSize, "%d%s", &parsedSize, &unit)
	if err != nil {
		return 0, custom_errors.ErrParsingUnits
	}

	size := uint32(parsedSize)
	multiplier := uint32(1000)

	// convert the size to bytes
	switch unit {
	case consts.UnitGb:
		size *= multiplier
		fallthrough
	case consts.UnitMb:
		size *= multiplier
		fallthrough
	case consts.UnitKb:
		size *= multiplier
	case consts.UnitB:
		// do nothing

	default:
		return 0, custom_errors.ErrInvalFormatUnits
	}

	if size < uint32(consts.ClusterSize)+uint32(pseudo_fat.GetSizeOfFileSystem())+uint32(2*unsafe.Sizeof(uint32(0))) {
		return 0, custom_errors.ErrDiskTooSmall
	}

	return size, nil
}

// CalculateFSSizes calculates the number of clusters and the size of the data space in bytes.
//
// It starts with fat size of 0 and interatively calculates the size of the data space
// while adjusting the fat size so it fits optimal number of clusters.
func CalculateFSSizes(size uint32) (uint32, uint32, uint32) {
	fsStructSize := uint32(pseudo_fat.GetSizeOfFileSystem())
	fatsSize := uint32(0)
	var clusterCount uint32
	dataSpace := size - fsStructSize - fatsSize

	sizeConverged := false
	for i := 0; i < 1000; i++ {
		clusterCount = dataSpace / uint32(consts.ClusterSize)

		fatsSize = clusterCount * uint32(unsafe.Sizeof(uint32(0)))
		newDataSpace := size - fsStructSize - fatsSize*uint32(consts.FATableCount)

		if newDataSpace == dataSpace {
			sizeConverged = true
			break
		}

		dataSpace = newDataSpace
	}

	if !sizeConverged {
		logging.Warn("The calculation of the file system did not converge. Switching to slower iterative method.")
		dataSpace = size

		for {
			clusterCount = dataSpace / uint32(consts.ClusterSize)
			fatsSize = clusterCount * uint32(unsafe.Sizeof(uint32(0)))

			totalSize := fsStructSize + fatsSize*uint32(consts.FATableCount) + dataSpace
			if totalSize > size {
				dataSpace--
			} else {
				break
			}
		}
	}

	allocatableSpace := clusterCount * uint32(consts.ClusterSize)

	return clusterCount, fatsSize, allocatableSpace
}

// WriteFileSystem writes the file system to the file.
func WriteFileSystem(pFile *os.File, pFs *pseudo_fat.FileSystem, fatsRef [][]int32, dataRef []byte) error {
	// rewind the file
	_, err := pFile.Seek(0, 0)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error rewinding the file: %s", err))
		return err
	}

	// if new size is smaller than the old size, truncate the file
	err = pFile.Truncate(int64(pFs.DiskSize))
	if err != nil {
		return err
	}

	writtenBytes := uint32(0)

	// write the file system structure
	fsBytes, err := StructToBytes(pFs)
	if err != nil {
		return err
	}
	_, err = pFile.Write(fsBytes)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error writing the file system structure: %s", err))
		return err
	}
	writtenBytes += uint32(len(fsBytes))

	// write the FATs
	for i := range fatsRef {
		fatBytes, err := StructToBytes(fatsRef[i])
		if err != nil {
			return err
		}

		_, err = pFile.Write(fatBytes)
		if err != nil {
			logging.Critical(fmt.Sprintf("Error writing the FATs: %s", err))
			return err
		}
		writtenBytes += uint32(len(fatBytes))
	}

	// write the data region
	_, err = pFile.Write(dataRef)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error writing the data region: %s", err))
		return err
	}
	writtenBytes += uint32(len(dataRef))

	// pad if necessary
	if writtenBytes < pFs.DiskSize {
		logging.Info(fmt.Sprintf("Not all bytes were written to the file (written: %d, expected: %d). Padding the rest with '\\0'.", writtenBytes, pFs.DiskSize))
		padding := make([]byte, pFs.DiskSize-writtenBytes)
		_, err = pFile.Write(padding)
		if err != nil {
			logging.Critical(fmt.Sprintf("Error writing the padding: %s", err))
			return err
		}
	}

	return nil
}

// ReadFileSystem reads the file system from the file.
func ReadFileSystem(pFile *os.File, pFs *pseudo_fat.FileSystem, fatsRef *[][]int32, dataRef *[]byte) error {
	// rewind the file
	_, err := pFile.Seek(0, 0)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error rewinding the file: %s", err))
		return err
	}

	// read the file system structure
	fsSize := pseudo_fat.GetSizeOfFileSystem()
	fsBytes := make([]byte, fsSize)
	_, err = pFile.Read(fsBytes)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error reading the file system structure: %s", err))
		return err
	}

	// convert the bytes to the file system structure
	err = BytesToStruct(fsBytes, pFs)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error converting the bytes to the file system structure: %s", err))
		return err
	}

	// read the FATs
	fatCount := pFs.FatCount
	if fatCount <= 0 {
		return custom_errors.ErrInvalidFatCount
	}

	fatSize := fatCount * uint32(unsafe.Sizeof(int32(0)))
	*fatsRef = make([][]int32, consts.FATableCount)

	for i := 0; i < int(consts.FATableCount); i++ {
		(*fatsRef)[i] = make([]int32, fatCount)

		fatBytes := make([]byte, int(fatSize))
		_, err = io.ReadFull(pFile, fatBytes)
		if err != nil {
			logging.Critical(fmt.Sprintf("Error reading FAT %d: %s", i, err))
			return custom_errors.ErrReadingFat
		}

		err = BytesToStruct(fatBytes, &(*fatsRef)[i])
		if err != nil {
			logging.Critical(fmt.Sprintf("Error converting bytes to FAT %d: %s", i, err))
			return custom_errors.ErrConvertingFat
		}
	}

	// calculate the size of the data region
	dataSize := int(pFs.DiskSize) - int(fsSize) - (int(consts.FATableCount) * int(fatSize))
	if dataSize < 0 {
		return custom_errors.ErrDataTooSmall
	}

	*dataRef = make([]byte, dataSize)
	_, err = io.ReadFull(pFile, *dataRef)
	if err != nil {
		logging.Critical(fmt.Sprintf("Error reading the data region: %s", err))
		return err
	}

	return nil
}

// NewDirectoryEntry creates a new directory entry.
//
// size is the size of the file in bytes (irelevant for directories).
// TODO: add error handling.
func NewDirectoryEntry(isFile bool, size uint32, startCluster uint32, parentCluster uint32, name string) pseudo_fat.DirectoryEntry {
	res := pseudo_fat.DirectoryEntry{
		IsFile:        isFile,
		Size:          size,
		StartCluster:  startCluster,
		ParentCluster: parentCluster,
	}
	copy(res.Name[:], []byte(name))

	return res
}

// GetNormalizedStrFromMem converts the byte slice to a string and trims the null bytes.
func GetNormalizedStrFromMem(data []byte) string {
	return string(bytes.Trim(data, "\x00"))
}

// IsClusterEmpty checks if the cluster data are empty.
func IsClusterEmpty(clusterData []byte) bool {
	for _, b := range clusterData {
		if b != 0 {
			return false
		}
	}

	return true
}
