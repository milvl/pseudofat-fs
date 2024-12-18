// utils package contains utility functions that are used across the project
package utils

import (
	"fmt"
	"kiv-zos-semestral-work/consts"
)

// PFormatFats formats the FAT table for pretty printing
func PFormatFats(fat [][]int32) string {
	res := ""

	for i, fatTable := range fat {
		res += fmt.Sprintf("FAT %d:\n", i)
		for j, entry := range fatTable {
			if entry != consts.FatFree {
				res += fmt.Sprintf("\t%d: %d\n", j, entry)
			}
		}
	}

	return res
}
