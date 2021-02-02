// +build linux

package filesystem

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type FileSystem struct {
	Name           string
	FileAttributes os.FileMode
	Extension      string
	CTime          time.Time
	CTimeUtc       time.Time
	ATime          time.Time
	ATimeUtc       time.Time
	MTime          time.Time
	MTimeUtc       time.Time
}

type DirectoryInfo struct {
	Name           string
	FileAttributes os.FileMode
	Extension      string
	CTime          time.Time
	CTimeUtc       time.Time
	ATime          time.Time
	ATimeUtc       time.Time
	MTime          time.Time
	MTimeUtc       time.Time
}

type FileInfo struct {
	Name           string
	FileAttributes os.FileMode
	Extension      string
	CTime          time.Time
	CTimeUtc       time.Time
	ATime          time.Time
	ATimeUtc       time.Time
	MTime          time.Time
	MTimeUtc       time.Time
	length         int64
}

func createFiles(files []os.FileInfo, hidden bool) (di []DirectoryInfo, fi []FileInfo) {
	for _, file := range files {
		if file.IsDir() {
			data := file.Sys().(*syscall.Stat_t)
			if file.Name()[0:1] == "." && !hidden {
				continue
			}
			di = append(di, DirectoryInfo{
				Name:           file.Name(),
				FileAttributes: file.Mode(),
				Extension:      filepath.Ext(file.Name()),
				CTime:          time.Unix(data.Ctim.Unix()),
				CTimeUtc:       time.Unix(data.Ctim.Unix()).UTC(),
				ATime:          time.Unix(data.Atim.Unix()),
				ATimeUtc:       time.Unix(data.Atim.Unix()).UTC(),
				MTime:          time.Unix(data.Mtim.Unix()),
				MTimeUtc:       time.Unix(data.Mtim.Unix()).UTC(),
			})
		} else {
			data := file.Sys().(*syscall.Stat_t)
			if file.Name()[0:1] == "." && !hidden {
				continue
			}
			fi = append(fi, FileInfo{
				Name:           file.Name(),
				FileAttributes: file.Mode(),
				Extension:      filepath.Ext(file.Name()),
				CTime:          time.Unix(data.Ctim.Unix()),
				CTimeUtc:       time.Unix(data.Ctim.Unix()).UTC(),
				ATime:          time.Unix(data.Atim.Unix()),
				ATimeUtc:       time.Unix(data.Atim.Unix()).UTC(),
				MTime:          time.Unix(data.Mtim.Unix()),
				MTimeUtc:       time.Unix(data.Mtim.Unix()).UTC(),
				length:         file.Size(),
			})
		}
	}
	return
}
