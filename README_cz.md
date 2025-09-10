# pseudoFAT — Zjednodušený souborový systém v Go

> Autor: Milan Vlachovský 
> 
> Univerzitní projekt — pro studijní účely

Studijní, souborový systém, který simuluje klíčové principy FAT (File Allocation Table) nad jediným binárním souborem sloužícím jako virtuální disk. Umožňuje vytvářet adresáře, kopírovat/přesouvat soubory, zkoumat řetězce clusterů a ověřovat konzistenci.

## Funkce

- Virtuální disk uložený v jednom binárním souboru
- Alokace ve stylu FAT se speciálními značkami pro volné, konec souboru a chybné clustery
- Základní příkazy podobné shellu (`mkdir`, `rm`, `mv`, `cp`, `ls`, `pwd`, `cat`, `info`, `load`, `check`, `bug`)
- Multiplatformní build (Unix-like a Windows)
- Jasné oddělení parsování/vykonávání příkazů a operací nad souborovým systémem

> Velikost disku je z praktických důvodů omezena na 4 GiB.

## Požadavky

- Go 1.22+
- make (volitelné, pro pohodlné buildy)
    - Uživatelé Windows mohou nainstalovat make (např. přes Chocolatey) nebo použít alternativy.

Další podrobnosti viz [dokumentace](docs/doc_cz.md).

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
