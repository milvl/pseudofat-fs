# pseudoFAT — Simplified File System in Go

> Author: Milan Vlachovský
> 
> University project — educational use

A toy, study-oriented file system that simulates key ideas of FAT (File Allocation Table) on top of a single binary file acting as a virtual disk. It lets you create directories, copy/move files, inspect cluster chains, and verify consistency.

## Features

- Virtual disk stored as one binary file
- FAT-style allocation with special markers for free, end-of-file, and bad clusters
- Basic shell-like commands (`mkdir`, `rm`, `mv`, `cp`, `ls`, `pwd`, `cat`, `info`, `load`, `check`, `bug`)
- Cross-platform build (Unix-like & Windows)
- Clear separation of command parsing/execution and FS operations

> Disk size is intentionally capped to 4 GiB for practicality.

## Requirements

- Go 1.22+
- make (optional, for convenient builds)
    - Windows users can install make (e.g., via Chocolatey) or use alternatives.

For more details, see [documentation](docs/doc.md).

## Demo

```bash
./myfs pseudofat_01.fs

10:58:51.893796 [INFO] (main.go:189) - Filesystem file "pseudofat_01.fs" does not exist, creating it...
10:58:51.897721 [INFO] (loader.go:101) - File is empty
pwd
File system is uninitialized. It cannot be used until it is formatted.
format 10MB
Filesystem formatted to 10000000 bytes. Allocatable data space: 9980000 bytes
Filesystem changed, writing to the file...
10:59:12.468367 [INFO] (data_transform.go:184) - Not all bytes were written to the file (written: 9999991, expected: 10000000). Padding the rest with '\0'.
Updated filesystem written to the file.
pwd
/
mkdir a
OK
Filesystem changed, writing to the file...
10:59:26.363556 [INFO] (data_transform.go:184) - Not all bytes were written to the file (written: 9999991, expected: 10000000). Padding the rest with '\0'.
Updated filesystem written to the file.
ls
DIR:    a
mkdir b
OK
Filesystem changed, writing to the file...
10:59:35.166316 [INFO] (data_transform.go:184) - Not all bytes were written to the file (written: 9999991, expected: 10000000). Padding the rest with '\0'.
Updated filesystem written to the file.
mkdir a/a1
OK
Filesystem changed, writing to the file...
10:59:41.231788 [INFO] (data_transform.go:184) - Not all bytes were written to the file (written: 9999991, expected: 10000000). Padding the rest with '\0'.
Updated filesystem written to the file.
ls
DIR:    a
DIR:    b
ls a
DIR:    a1
incp ../file_01.txt file_01.txt
OK
Filesystem changed, writing to the file...
11:00:43.726498 [INFO] (data_transform.go:184) - Not all bytes were written to the file (written: 9999991, expected: 10000000). Padding the rest with '\0'.
Updated filesystem written to the file.
ls
DIR:    a
DIR:    b
FILE:   file_01.txt     26763
incp ../file_02.txt a/a1/file.txt
OK
Filesystem changed, writing to the file...
11:01:29.676691 [INFO] (data_transform.go:184) - Not all bytes were written to the file (written: 9999991, expected: 10000000). Padding the rest with '\0'.
Updated filesystem written to the file.
ls
DIR:    a
DIR:    b
FILE:   file_01.txt     26763
cd a/a1
OK
ls
FILE:   file.txt        23095
exit
Closing file...

ls -l
total 12640
-rwxrwxrwx 1 milvl milvl  2940682 Sep 10 10:53 myfs
-rwxrwxrwx 1 milvl milvl 10000000 Sep 10 11:01 pseudofat_01.fs
```
