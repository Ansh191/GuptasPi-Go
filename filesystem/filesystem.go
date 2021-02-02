package filesystem

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"guptaspi/info"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
)

type GetFolderChildrenReturn struct {
	Directories []DirectoryInfo
	Files       []FileInfo
}

func AddFileSystemRouter(r *mux.Router) {
	r.HandleFunc("/filesystem/{volume}", getFolderChildren).Methods("GET")
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
