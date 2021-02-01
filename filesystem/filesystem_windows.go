// +build windows

package filesystem

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"golang.org/x/sys/windows"
	"guptaspi/info"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

type GetFolderChildrenReturn struct {
	Directories []DirectoryInfo
	Files       []FileInfo
}

func getFolderChildren(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	volume := params["volume"]

	dirPath := r.FormValue("folder")
	hiddenValue := r.FormValue("hidden")
	var hidden bool
	if hiddenValue != "" {
		var err error
		hidden, err = strconv.ParseBool(hiddenValue)

		if err != nil {
			log.Printf("Hidden query param error, %v", err)
			w.WriteHeader(400)
			return
		}
	} else {
		hidden = false
	}

	// Check if volume is valid
	drive := info.GetDrive(volume)

	if drive == nil {
		w.WriteHeader(404)
		return
	}

	dirPath = filepath.Join(drive.Path, dirPath)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Printf("Error when reading path: %v", err)
		w.WriteHeader(500)
		return
	}

	di, fi := createFiles(files, hidden)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(GetFolderChildrenReturn{
		Directories: di,
		Files:       fi,
	})
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
