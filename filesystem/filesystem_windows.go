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
	Name              string    `json:"name"`
	FileAttributes    uint32    `json:"file_attributes"`
	Extension         string    `json:"extension"`
	CreationTime      time.Time `json:"creation_time"`
	CreationTimeUtc   time.Time `json:"creation_time_utc"`
	LastAccessTime    time.Time `json:"last_access_time"`
	LastAccessTimeUtc time.Time `json:"last_access_time_utc"`
	LastWriteTime     time.Time `json:"last_write_time"`
	LastWriteTimeUtc  time.Time `json:"last_write_time_utc"`
}

type DirectoryInfo struct {
	Name              string    `json:"name"`
	FileAttributes    uint32    `json:"file_attributes"`
	Extension         string    `json:"extension"`
	CreationTime      time.Time `json:"creation_time"`
	CreationTimeUtc   time.Time `json:"creation_time_utc"`
	LastAccessTime    time.Time `json:"last_access_time"`
	LastAccessTimeUtc time.Time `json:"last_access_time_utc"`
	LastWriteTime     time.Time `json:"last_write_time"`
	LastWriteTimeUtc  time.Time `json:"last_write_time_utc"`
}

type FileInfo struct {
	Name              string    `json:"name"`
	FileAttributes    uint32    `json:"file_attributes"`
	Extension         string    `json:"extension"`
	CreationTime      time.Time `json:"creation_time"`
	CreationTimeUtc   time.Time `json:"creation_time_utc"`
	LastAccessTime    time.Time `json:"last_access_time"`
	LastAccessTimeUtc time.Time `json:"last_access_time_utc"`
	LastWriteTime     time.Time `json:"last_write_time"`
	LastWriteTimeUtc  time.Time `json:"last_write_time_utc"`
	Length            uint64    `json:"length"`
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
