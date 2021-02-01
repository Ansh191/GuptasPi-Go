package main

import (
	"github.com/gorilla/mux"
	"guptaspi/filesystem"
	"guptaspi/info"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()

	info.AddInfoRouter(r)
	filesystem.AddFileSystemRouter(r)

	http.Handle("/", r)

	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
