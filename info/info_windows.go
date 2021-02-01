// +build windows

package info

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"golang.org/x/sys/windows"
	"net/http"
	"sync"
	"syscall"
)

type Drive struct {
	Path               string
	VolumeLabel        string
	AvailableFreeSpace uint64
	TotalFreeSpace     uint64
	TotalSize          uint64
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

// createDrive creates a Drive pointer from a drive letter.
// Returns error in second argument if there was an error getting information
func createDrive(letter string) (*Drive, error) {
	rootPath := letter + ":\\"
	volumeName, err := getVolumeName(rootPath)
	if err != nil {
		return nil, err
	}
	diskSpace, err := getDiskSpace(rootPath)

	return &Drive{
		Path:               rootPath,
		VolumeLabel:        volumeName,
		AvailableFreeSpace: diskSpace[0],
		TotalFreeSpace:     diskSpace[1],
		TotalSize:          diskSpace[2],
	}, nil
}

// getDiskSpace returns an array of three items in the following order:
// [available free space to server in drive, total free space in drive, total space in drive].
// Returns error in second argument if unable to get information.
func getDiskSpace(rootPathName string) ([3]uint64, error) {
	rootPathNamePtr, err := syscall.UTF16PtrFromString(rootPathName)
	if err != nil {
		return [3]uint64{0, 0, 0}, err
	}

	var availableFreeSpace uint64
	var totalSpace uint64
	var totalFreeSpace uint64

	err = windows.GetDiskFreeSpaceEx(
		rootPathNamePtr,
		&availableFreeSpace,
		&totalSpace,
		&totalFreeSpace,
	)

	if err != nil {
		return [3]uint64{0, 0, 0}, err
	}

	return [3]uint64{availableFreeSpace, totalFreeSpace, totalSpace}, nil
}

// getVolumeName gets the Volume Name of the drive located at the root path.
// Returns error in second argument if unable to get information.
func getVolumeName(rootPathName string) (string, error) {
	rootPathNamePtr, err := syscall.UTF16PtrFromString(rootPathName)
	if err != nil {
		return "", err
	}

	var volumeNameBuffer = make([]uint16, syscall.MAX_PATH+1)
	var nVolumeNameSize = uint32(len(volumeNameBuffer))
	var volumeSerialNumber uint32
	var maximumComponentLength uint32
	var fileSystemFlags uint32
	var fileSystemNameBuffer = make([]uint16, syscall.MAX_PATH+1)
	var nFileSystemNameBuffer uint32 = syscall.MAX_PATH + 1

	err = windows.GetVolumeInformation(
		rootPathNamePtr,
		&volumeNameBuffer[0],
		nVolumeNameSize,
		&volumeSerialNumber,
		&maximumComponentLength,
		&fileSystemFlags,
		&fileSystemNameBuffer[0],
		nFileSystemNameBuffer,
	)

	if err != nil {
		return "", err
	}

	//fmt.Printf("%s\n", syscall.UTF16ToString(volumeNameBuffer))
	//fmt.Printf("%d\n", volumeSerialNumber)
	//fmt.Printf("%d\n", maximumComponentLength)
	//fmt.Printf("%b\n", fileSystemFlags)
	//fmt.Printf("%s\n", syscall.UTF16ToString(fileSystemNameBuffer))

	return syscall.UTF16ToString(volumeNameBuffer), nil
}

// getNetworkDrives returns list of drive letters of all network drives.
func getNetworkDrives() []string {
	drives := getDrives()
	var networkDrives []string

	for _, drive := range drives {
		if driveLetter, err := syscall.UTF16PtrFromString(drive + ":\\"); err != nil {
			continue
		} else {
			driveType := windows.GetDriveType(driveLetter)
			if driveType == windows.DRIVE_REMOTE {
				networkDrives = append(networkDrives, drive)
			}
		}
	}

	return networkDrives
}

// getDrives returns list of all drive letters on server
func getDrives() []string {
	var drives []string

	if bitMask, err := windows.GetLogicalDrives(); err != nil {
		return nil
	} else {
		drives = bitsToDrives(bitMask)
	}

	return drives
}

func bitsToDrives(bitMap uint32) (drives []string) {
	availableDrives := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

	for i := range availableDrives {
		if bitMap&1 == 1 {
			drives = append(drives, availableDrives[i])
		}
		bitMap >>= 1
	}

	return
}
