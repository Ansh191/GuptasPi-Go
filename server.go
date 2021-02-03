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

	r.Use(loggingMiddleware)

	amw := authentication{}
	amw.Initialize()

	r.Use(amw.Middleware)
	r.HandleFunc("/auth/login", amw.Login).Methods("GET")
	r.HandleFunc("/auth/logout", amw.Logout).Methods("GET")
	r.HandleFunc("/auth/createUser", amw.CreateUser).Methods("POST")
	r.HandleFunc("/auth/refresh", amw.Refresh).Methods("POST")

	info.AddInfoRouter(r)
	filesystem.AddFileSystemRouter(r)
	upload.AddUploadRouter(r)

	http.Handle("/", r)

	fmt.Printf("Running on port: %d\n", 5000)
	if err := http.ListenAndServe(":5000", nil); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
