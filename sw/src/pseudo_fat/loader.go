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

func validateFileSystem(pFs *FileSystem) error {
	// check if the file system is valid
	// TODO: Here

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

	// if the file is empty, return an uninitialized file system
	if fileInfo.Size() == int64(0) {
		return GetUninitializedFileSystem(), nil, nil

		// if the file is smaller than it does not contain a file system or is corrupted
		// inform the user and return an uninitialized file system
	} else if fileInfo.Size() < int64(unsafe.Sizeof(FileSystem{})) {
		fmt.Println(consts.FileNotFilesys)
		return GetUninitializedFileSystem(), nil, nil
	}

	// try to read the file system
	var pFs FileSystem
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
