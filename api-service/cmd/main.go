package main

import (
	"log"
	"net/http"
	"time"

	"cloud-native-platform/api-service/controllers"
)

func main() {
	handler, err := controllers.NewHandler("uploads")
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr: "127.0.0.1:8080", Handler: handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Println("doc-forge: http://localhost:8080")
	log.Fatal(server.ListenAndServe())
}
