// +build windows

package filesystem

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type FileSystem struct {
	Name              string
	FileAttributes    uint32
	Extension         string
	CreationTime      time.Time
	CreationTimeUtc   time.Time
	LastAccessTime    time.Time
	LastAccessTimeUtc time.Time
	LastWriteTime     time.Time
	LastWriteTimeUtc  time.Time
}

type DirectoryInfo struct {
	Name              string
	FileAttributes    uint32
	Extension         string
	CreationTime      time.Time
	CreationTimeUtc   time.Time
	LastAccessTime    time.Time
	LastAccessTimeUtc time.Time
	LastWriteTime     time.Time
	LastWriteTimeUtc  time.Time
}

type FileInfo struct {
	Name              string
	FileAttributes    uint32
	Extension         string
	CreationTime      time.Time
	CreationTimeUtc   time.Time
	LastAccessTime    time.Time
	LastAccessTimeUtc time.Time
	LastWriteTime     time.Time
	LastWriteTimeUtc  time.Time
	Length            uint64
}

func createFiles(files []os.FileInfo, hidden bool) (di []DirectoryInfo, fi []FileInfo) {
	for _, file := range files {
		if file.IsDir() {
			data := file.Sys().(*syscall.Win32FileAttributeData)
			if data.FileAttributes&windows.FILE_ATTRIBUTE_HIDDEN > 1 && !hidden {
				continue
			}
			di = append(di, DirectoryInfo{
				Name:              file.Name(),
				FileAttributes:    data.FileAttributes,
				Extension:         filepath.Ext(file.Name()),
				CreationTime:      time.Unix(0, data.CreationTime.Nanoseconds()),
				CreationTimeUtc:   time.Unix(0, data.CreationTime.Nanoseconds()).UTC(),
				LastAccessTime:    time.Unix(0, data.LastAccessTime.Nanoseconds()),
				LastAccessTimeUtc: time.Unix(0, data.LastAccessTime.Nanoseconds()).UTC(),
				LastWriteTime:     time.Unix(0, data.LastWriteTime.Nanoseconds()),
				LastWriteTimeUtc:  time.Unix(0, data.LastWriteTime.Nanoseconds()).UTC(),
			})
		} else {
			data := file.Sys().(*syscall.Win32FileAttributeData)
			if data.FileAttributes&windows.FILE_ATTRIBUTE_HIDDEN > 1 && !hidden {
				continue
			}
			fi = append(fi, FileInfo{
				Name:              file.Name(),
				FileAttributes:    data.FileAttributes,
				Extension:         filepath.Ext(file.Name()),
				CreationTime:      time.Unix(0, data.CreationTime.Nanoseconds()),
				CreationTimeUtc:   time.Unix(0, data.CreationTime.Nanoseconds()).UTC(),
				LastAccessTime:    time.Unix(0, data.LastAccessTime.Nanoseconds()),
				LastAccessTimeUtc: time.Unix(0, data.LastAccessTime.Nanoseconds()).UTC(),
				LastWriteTime:     time.Unix(0, data.LastWriteTime.Nanoseconds()),
				LastWriteTimeUtc:  time.Unix(0, data.LastWriteTime.Nanoseconds()).UTC(),
				Length:            uint64(data.FileSizeHigh)<<32 + uint64(data.FileSizeLow),
			})
		}
	}
	return
}
