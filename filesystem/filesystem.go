package filesystem

import "github.com/gorilla/mux"

func AddFileSystemRouter(r *mux.Router) {
	r.HandleFunc("/filesystem/{volume}", getFolderChildren).Methods("GET")
}
