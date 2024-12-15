// utils package contains utility functions for the project
package utils

import (
	"bytes"
	"encoding/binary"
	"kiv-zos-semestral-work/custom_errors"
	"reflect"
)

// StructToBytes converts a struct to bytes. It writes
// the data in little endian encoding.
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
// endian encoding.
func BytesToStruct(data []byte, pResult interface{}) error {
	// sanity checks
	if reflect.ValueOf(pResult).Kind() != reflect.Ptr {
		return custom_errors.ErrNotPtr
	} else if pResult == nil {
		return custom_errors.ErrNilPointer
	} else if len(data) == 0 {
		return custom_errors.ErrEmptySlice
	}

	buff := bytes.NewReader(data)
	err := binary.Read(buff, binary.LittleEndian, pResult)
	if err != nil {
		return custom_errors.ErrBytesToStruct
	}

	return nil
}
