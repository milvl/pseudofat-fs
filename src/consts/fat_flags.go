// consts package defines the constants used in the FAT filesystem
package consts

const (
	// FatFree is the value of an unused FAT entry.
	//
	// int32 is used to represent the FAT entries
	// because the FAT entries can be negative and
	// based on experiments and maximum file size,
	// int32 is sufficient (maximum number of
	// clusters is near 1 000 000)
	FatFree int32 = -1

	// FatFileEnd is the value of the last FAT entry of a file.
	//
	// int32 is used to represent the FAT entries
	// because the FAT entries can be negative and
	// based on experiments and maximum file size,
	// int32 is sufficient (maximum number of
	// clusters for 4GB FS is somewhere near 1 000 000).
	FatFileEnd int32 = -2

	// FatBadCluster is the value of a bad cluster in the FAT.
	// int32 is used to represent the FAT entries
	// because the FAT entries can be negative and
	// based on experiments and maximum file size,
	// int32 is sufficient (maximum number of
	// clusters for 4GB FS is somewhere near 1 000 000).
	FatBadCluster int32 = -3
)
