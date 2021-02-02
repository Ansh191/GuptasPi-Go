// +build linux

package info

import (
	"golang.org/x/sys/unix"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

type MountData struct {
	Device         string
	MountPoint     string
	FileSystemType string
	ReadWrite      string
}

func createDrive(mountPoint string) (*Drive, error) {
	volumeName := getVolumeName(mountPoint)
	diskSpace, err := getDiskSpace(mountPoint)
	if err != nil {
		return nil, err
	}

	return &Drive{
		Path:               mountPoint,
		VolumeLabel:        volumeName,
		AvailableFreeSpace: diskSpace[0],
		TotalFreeSpace:     diskSpace[1],
		TotalSize:          diskSpace[2],
	}, nil
}

func getDiskSpace(mountPoint string) ([3]uint64, error) {
	var stat unix.Statfs_t

	err := unix.Statfs(mountPoint, &stat)
	if err != nil {
		log.Printf("Error getting drive stats: %v", err)
		return [3]uint64{0, 0, 0}, err
	}

	bSize := uint64(stat.Bsize)

	return [3]uint64{bSize * stat.Bavail, bSize * stat.Bfree, bSize * stat.Blocks}, nil
}

func getVolumeName(mountPoint string) string {
	return filepath.Base(mountPoint)
}

func getNetworkDrives() []string {
	mounts := getDrives()

	var networkDrives []string

	for _, mount := range mounts {
		// Get NTFS drives
		if mount.FileSystemType == "fuseblk" {
			networkDrives = append(networkDrives, mount.MountPoint)
		}
	}

	return networkDrives
}

func getDrives() []*MountData {
	data, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		log.Printf("Error reading mounts: %v", err)
		return nil
	}

	mountStrings := strings.Split(string(data), "\n")

	var mounts []*MountData

	for _, mountString := range mountStrings {
		mountData := strings.Split(mountString, " ")
		if len(mountData) >= 4 {
			mounts = append(mounts, &MountData{
				Device:         mountData[0],
				MountPoint:     mountData[1],
				FileSystemType: mountData[2],
				ReadWrite:      mountData[3],
			})
		}
	}

	return mounts
}
