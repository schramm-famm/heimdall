package main

import (
	"github.com/gorilla/mux"
	"heimdall/handlers"
	"log"
	"net/http"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/token", handlers.PostTokenHandler).Methods("POST")
	r.PathPrefix("/").Handler(http.HandlerFunc(handlers.ReqHandler))

	log.Fatal(http.ListenAndServe(":8080", r))
}
