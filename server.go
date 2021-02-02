package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"guptaspi/filesystem"
	"guptaspi/info"
	"guptaspi/upload"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()

	info.AddInfoRouter(r)
	filesystem.AddFileSystemRouter(r)
	upload.AddUploadRouter(r)

	http.Handle("/", r)

	fmt.Printf("Running on port: %d\n", 5000)
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
