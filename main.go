package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func receiverHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1 MB is limit for 1 webhook
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error: reading body. Desc: %v\n", err)
		http.Error(w, fmt.Sprintf("Error: reading body. Desc: %v\n", err), http.StatusBadRequest)
		return
	}
	hookId := r.PathValue("id")

	fmt.Printf("receiverHandler: ID пользователя и Body получены, ID: %v,\nBody: %s\n", hookId, body)
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Webhook received\n"))
}

func main() {
	router := initializeRoutes()

	server := http.Server{}

	err := http.ListenAndServe("localhost:8080", router) //TODO: do ":8080", delete "localhost"
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
}

func initializeRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/ingest/{id}", receiverHandler)
	return mux
}
