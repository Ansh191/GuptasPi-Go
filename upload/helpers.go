package upload

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"github.com/google/uuid"
	"hash/crc32"
	"log"
	"os"
	"strings"
)

func processMetadata(metadataText string) map[string]string {
	pairs := strings.Split(metadataText, ",")
	metadata := make(map[string]string, len(pairs))

	for _, pair := range pairs {
		keyValue := strings.Split(pair, " ")
		if len(keyValue) == 2 {
			metadata[keyValue[0]] = keyValue[1]
		} else {
			metadata[keyValue[0]] = ""
		}
	}
	return metadata
}

func createFile(filePath string, overwrite bool, uploadLength uint64, c chan int) {
	var file *os.File
	var err error
	if !overwrite {
		file, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0777)
	} else {
		file, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0777)
	}
	if err != nil {
		log.Printf("File Creation error: %v", err)
		if err.Error() == "open "+filePath+": The file exists." {
			c <- 409
			return
		}
		c <- 500
		return
	}

	_, err = file.Seek(int64(uploadLength-1), 0)
	if err != nil {
		log.Printf("File Seek error: %v", err)
		c <- 500
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
		return
	}

	_, err = file.Write([]byte{0})
	if err != nil {
		log.Printf("File Write error: %v", err)
		c <- 500
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
		return
	}

	if err := file.Close(); err != nil {
		log.Printf("Failed to close file: %v", err)
	}

	c <- 201
}

func getUploadFromId(idString string) (*Upload, error) {
	id, err := uuid.Parse(idString)
	if err != nil {
		log.Printf("Error parsing uuid: %v", err)
		return nil, err
	}

	lock.RLock()
	upload, ok := uploadMap[id]
	lock.RUnlock()

	if !ok {
		return nil, err
	}

	return upload, nil
}

func verifyChecksum(buffer []byte, algorithm string, hash string) bool {
	var hashString string

	switch algorithm {
	case "sha1":
		h := sha1.Sum(buffer)
		hashString = hex.EncodeToString(h[:])
	case "md5":
		h := md5.Sum(buffer)
		hashString = hex.EncodeToString(h[:])
	case "crc32":
		h := make([]byte, 4)
		binary.LittleEndian.PutUint32(h, crc32.ChecksumIEEE(buffer))
		hashString = hex.EncodeToString(h)
	}

	return hash == hashString
}

func writeToFile(filePath string, buffer []byte, offset int64, c chan int) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		c <- 500
		return
	}

	_, err = f.WriteAt(buffer, offset)
	if err != nil {
		log.Printf("Error writing to file: %v", err)
		c <- 500
		return
	}

	c <- 204
	return
}
