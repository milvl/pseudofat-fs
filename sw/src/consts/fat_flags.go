// consts package defines the constants used in the FAT filesystem
package consts

const (
	// FatFree is the value of an unused FAT entry
	FatFree int8 = -1
	// FatFileEnd is the value of the last FAT entry of a file
	FatFileEnd int8 = -2
	// // FatBadCluster is the value of a bad cluster in the FAT - unused
	// FatBadCluster int8 = -3
)
