// +build !js

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/pkg/browser"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.Handle("/", http.FileServer(http.Dir(".")))

	log.Println("Opening http://localhost:8080 in the browser")
	browser.OpenURL("http://localhost:8080")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
