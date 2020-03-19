// +build !js

package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Println("Serving on HTTP port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
