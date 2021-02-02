package upload

import (
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"guptaspi/info"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Upload struct {
	FilePath       string
	FileSize       uint64
	Offset         uint64
	ExpirationDate time.Time
}

var uploadMap = map[uuid.UUID]*Upload{}
var lock = sync.RWMutex{}

func AddUploadRouter(r *mux.Router) {
	r.HandleFunc("/upload/{volume}", startUpload).Methods("POST")
	r.HandleFunc("/upload/{id}", headUpload).Methods("HEAD")
	r.HandleFunc("/upload/{id}", patchUpload).Methods("PATCH")
	r.HandleFunc("/upload/{id}", terminateUpload).Methods("DELETE")
	r.HandleFunc("/upload", options).Methods("OPTIONS")
}

func startUpload(w http.ResponseWriter, r *http.Request) {
	volume := mux.Vars(r)["volume"]

	// Process overwrite query
	overwriteValue := r.FormValue("overwrite")
	var overwrite bool
	if overwriteValue != "" {
		var err error
		overwrite, err = strconv.ParseBool(overwriteValue)

		if err != nil {
			log.Printf("Overwrite query param error, %v", err)
			w.WriteHeader(400)
			return
		}
	} else {
		overwrite = false
	}

	// Tus version header
	if r.Header.Get("Tus-Resumable") != "1.0.0" {
		w.WriteHeader(405)
		return
	}

	metadata := processMetadata(r.Header.Get("Upload-Metadata"))

	// Get filepath
	b64FilePath, ok := metadata["filename"]
	if !ok {
		w.WriteHeader(400)
		return
	}
	filePathBytes, err := base64.StdEncoding.DecodeString(b64FilePath)
	if err != nil {
		log.Printf("B64 Decode error: %v", err)
		w.WriteHeader(400)
		return
	}
	filePath := string(filePathBytes)

	drive := info.GetDrive(volume)

	if drive == nil {
		w.WriteHeader(404)
		return
	}

	filePath = filepath.Join(drive.Path, filePath)

	var uploadLength uint64

	// Upload length and deferral
	if deferLength := r.Header.Get("Upload-Defer-Length"); deferLength != "" {
		if deferLength != "1" {
			w.WriteHeader(400)
			return
		}
		if r.Header.Get("Upload-Length") != "" {
			w.WriteHeader(400)
			return
		}
		uploadLength = 0
	} else {
		uploadLength, err = strconv.ParseUint(r.Header.Get("Upload-Length"), 10, 64)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		if uploadLength <= 0 {
			w.WriteHeader(400)
			return
		}
	}

	// File creation
	c := make(chan int)
	go createFile(filePath, overwrite, uploadLength, c)

	// UUID generation
	id, err := uuid.NewRandom()
	if err != nil {
		log.Printf("UUID creation error: %v", err)
		w.WriteHeader(500)
		return
	}

	upload := Upload{
		FilePath:       filePath,
		FileSize:       uploadLength,
		Offset:         0,
		ExpirationDate: time.Now().UTC().Add(time.Hour * time.Duration(1)),
	}

	result := <-c
	if result != 201 {
		w.WriteHeader(result)
		return
	}

	lock.Lock()
	uploadMap[id] = &upload
	lock.Unlock()

	w.Header().Add("Tus-Removable", "1.0.0")
	w.Header().Add("Upload-Expires", upload.ExpirationDate.Format(time.RFC3339))
	w.Header().Add("Location", "http://localhost:5000/upload/"+id.String())
	w.WriteHeader(201)
}

func headUpload(w http.ResponseWriter, r *http.Request) {
	idString := mux.Vars(r)["id"]

	upload, err := getUploadFromId(idString)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	w.Header().Add("Upload-Offset", strconv.FormatUint(upload.Offset, 10))
	w.Header().Add("Upload-Expires", upload.ExpirationDate.Format(time.RFC3339))
	w.Header().Add("Tus-Resumable", "1.0.0")
	w.Header().Add("Cache-Control", "no-store")

	if upload.FileSize > 0 {
		w.Header().Add("Upload-Length", strconv.FormatUint(upload.FileSize, 10))
	} else {
		w.Header().Add("Upload-Defer-Length", "1")
	}

	w.WriteHeader(200)
}

func patchUpload(w http.ResponseWriter, r *http.Request) {
	idString := mux.Vars(r)["id"]

	upload, err := getUploadFromId(idString)
	if err != nil {
		w.WriteHeader(400)
		return
	}

	var fileSize uint64
	if upload.FileSize == 0 {
		if deferLength := r.Header.Get("Upload-Defer-Length"); deferLength != "" {
			if deferLength != "1" {
				w.WriteHeader(400)
				return
			}
			fileSize = 0
		} else if uploadLength := r.Header.Get("Upload-Length"); uploadLength != "" {
			fileSize, err = strconv.ParseUint(uploadLength, 10, 64)
			if err != nil {
				w.WriteHeader(400)
				return
			}
		} else {
			w.WriteHeader(400)
			return
		}
	} else {
		fileSize = upload.FileSize
	}

	if uploadOffset, err := strconv.ParseUint(r.Header.Get("Upload-Offset"), 10, 64); err != nil {
		w.WriteHeader(400)
		return
	} else if uploadOffset != upload.Offset {
		w.WriteHeader(409)
		return
	}

	if r.ContentLength <= 0 {
		w.Header().Add("Tus-Resumable", "1.0.0")
		w.Header().Add("Upload-Offset", strconv.FormatUint(upload.Offset, 10))
		w.Header().Add("Upload-Expires", upload.ExpirationDate.Format(time.RFC3339))
		w.WriteHeader(204)
		return
	}

	buffer := make([]byte, r.ContentLength)
	_, err = r.Body.Read(buffer)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(500)
		return
	}

	if uploadChecksum := r.Header.Get("Upload-Checksum"); uploadChecksum != "" {
		parts := strings.Split(uploadChecksum, " ")
		if len(parts) != 2 {
			w.WriteHeader(400)
			return
		}
		algorithm := parts[0]
		hash := parts[1]
		if !verifyChecksum(buffer, algorithm, hash) {
			w.WriteHeader(400)
			return
		}
	}

	c := make(chan int)
	go writeToFile(upload.FilePath, buffer, int64(upload.Offset), c)

	if code := <-c; code != 204 {
		w.WriteHeader(code)
		return
	}
	upload.Offset += uint64(len(buffer))
	upload.FileSize = fileSize

	if upload.FileSize != 0 && upload.FileSize == upload.Offset {
		log.Printf("Upload to %s finished", upload.FilePath)
		id, _ := uuid.Parse(idString)
		lock.Lock()
		delete(uploadMap, id)
		lock.Unlock()
	}

	w.Header().Add("Tus-Resumable", "1.0.0")
	w.Header().Add("Upload-Offset", strconv.FormatUint(upload.Offset, 10))
	w.Header().Add("Upload-Expires", upload.ExpirationDate.Format(time.RFC3339))

	w.WriteHeader(204)
	return
}

func terminateUpload(w http.ResponseWriter, r *http.Request) {
	idString := mux.Vars(r)["id"]
	id, err := uuid.Parse(idString)
	if err != nil {
		log.Printf("Error parsing uuid: %v", err)
		w.WriteHeader(400)
		return
	}

	lock.Lock()
	upload, ok := uploadMap[id]
	if !ok {
		w.WriteHeader(404)
		return
	}
	delete(uploadMap, id)
	lock.Unlock()

	err = os.Remove(upload.FilePath)
	if err != nil {
		log.Printf("Error removing file: %v", err)
	}

	w.Header().Add("Tus-Resumable", "1.0.0")
	w.WriteHeader(204)
}

func options(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Tus-Resumable", "1.0.0")
	w.Header().Add("Tus-Version", "1.0.0")
	w.Header().Add("Tus-Extension", "creation,checksum,expiration,termination")
	w.Header().Add("Tus-Checksum-Algorithm", "sha1,md5,crc32")

	w.WriteHeader(204)
}
