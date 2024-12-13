// utils package contains utility functions for the project
package utils

import (
	"bytes"
	"encoding/binary"
	"kiv-zos-semestral-work/custom_errors"
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
func BytesToStruct(data []byte, result interface{}) error {
	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.LittleEndian, result)
	if err != nil {
		return custom_errors.ErrBytesToStruct
	}

	return nil
}
