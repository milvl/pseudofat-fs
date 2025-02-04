// consts contains all constants used in the application
package consts

// MaxInputBufferSize is the maximum size of the input buffer
const MaxInputBufferSize uint16 = 1024

// MaxFileNameLength is the maximum length of a file name
const MaxFileNameLength = 11 // 8 characters + 3 characters for the extension

// MaxFilesystemSize is the maximum size of the file system
const MaxFilesystemSize uint32 = 4294967295 // circa 4 GB (2^32 - 1)

// MaxClusterSize is the maximum size of a cluster
const ClusterSize uint16 = 4000 // 4 KB

// MaxClusterCount is the maximum number of clusters.
// It was calculated iteratively:
//
// 1. AllocatableSpace = MaxFilesystemSize - FS structure size (omit FAT table sizes in the beginning)
//
// 2. ClusterCount = AllocatableSpace / ClusterSize
//
// 3. FATSize = ClusterCount * 4 (4 bytes per FAT entry as int32 is used - see fat_flags.go)
//
// 4. NewAllocatableSpace = MaxFilesystemSize - FS structure size - FATSize
//
// 5. if NewAllocatableSpace == AllocatableSpace, return ClusterCount
const MaxClusterCount uint32 = 1070530

// MaxFATSize is the maximum size of the FAT table.
// It was calculated iteratively:
//
// 1. AllocatableSpace = MaxFilesystemSize - FS structure size (omit FAT table sizes in the beginning)
//
// 2. ClusterCount = AllocatableSpace / ClusterSize
//
// 3. FATSize = ClusterCount * 4 (4 bytes per FAT entry as int32 is used - see fat_flags.go)
//
// 4. NewAllocatableSpace = MaxFilesystemSize - FS structure size - FATSize
//
// 5. if NewAllocatableSpace == AllocatableSpace, return ClusterCount
const MaxFATSize uint32 = 4282120

// StudentNumLen is the length of the Orion login
const StudentNumLen uint8 = 9

// FATableCount is the number of FAT tables
const FATableCount uint8 = 2

// ByteSizeInt is the size of a byte
const ByteSizeInt = 256
