package info

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"sync"
)

type Drive struct {
	Path               string `json:"path"`
	VolumeLabel        string `json:"volume_label"`
	AvailableFreeSpace uint64 `json:"available_free_space"`
	TotalFreeSpace     uint64 `json:"total_free_space"`
	TotalSize          uint64 `json:"total_size"`
}

var driveMap = map[string]*Drive{}
var lock = sync.RWMutex{}

func GetDrive(volumeLabel string) *Drive {
	lock.RLock()
	if len(driveMap) == 0 {
		lock.RUnlock()
		populateDriveMap()
	}
	lock.RLock()
	defer lock.RUnlock()
	return driveMap[volumeLabel]
}

func populateDriveMap() {
	lock.Lock()
	defer lock.Unlock()
	for _, letter := range getNetworkDrives() {
		if drive, err := createDrive(letter); err != nil {
			continue
		} else {
			driveMap[drive.VolumeLabel] = drive
		}
	}
}

// AddInfoRouter installs endpoints into main router located in server.go.
// r is a pointer to that router
func AddInfoRouter(r *mux.Router) {
	r.HandleFunc("/info", getInfo).Methods("GET")
}

// getInfo corresponds to the GET /info endpoint.
// This endpoint returns information on the network drives available to the server.
func getInfo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	networkDrives := getNetworkDrives()

	lock.RLock()
	if len(driveMap) != len(networkDrives) {
		lock.RUnlock()
		populateDriveMap()
	}

	lock.RLock()
	defer lock.RUnlock()
	var drives []*Drive
	for _, drive := range driveMap {
		drives = append(drives, drive)
	}
	_ = json.NewEncoder(w).Encode(drives)
}
