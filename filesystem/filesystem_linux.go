// +build linux

package filesystem

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type FileSystem struct {
	Name           string      `json:"name"`
	FileAttributes os.FileMode `json:"file_attributes"`
	Extension      string      `json:"extension"`
	CTime          time.Time   `json:"c_time"`
	CTimeUtc       time.Time   `json:"c_time_utc"`
	ATime          time.Time   `json:"a_time"`
	ATimeUtc       time.Time   `json:"a_time_utc"`
	MTime          time.Time   `json:"m_time"`
	MTimeUtc       time.Time   `json:"m_time_utc"`
}

type DirectoryInfo struct {
	Name           string      `json:"name"`
	FileAttributes os.FileMode `json:"file_attributes"`
	Extension      string      `json:"extension"`
	CTime          time.Time   `json:"c_time"`
	CTimeUtc       time.Time   `json:"c_time_utc"`
	ATime          time.Time   `json:"a_time"`
	ATimeUtc       time.Time   `json:"a_time_utc"`
	MTime          time.Time   `json:"m_time"`
	MTimeUtc       time.Time   `json:"m_time_utc"`
}

type FileInfo struct {
	Name           string      `json:"name"`
	FileAttributes os.FileMode `json:"file_attributes"`
	Extension      string      `json:"extension"`
	CTime          time.Time   `json:"c_time"`
	CTimeUtc       time.Time   `json:"c_time_utc"`
	ATime          time.Time   `json:"a_time"`
	ATimeUtc       time.Time   `json:"a_time_utc"`
	MTime          time.Time   `json:"m_time"`
	MTimeUtc       time.Time   `json:"m_time_utc"`
	length         int64       `json:"length"`
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
