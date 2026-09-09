package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Request received", r.URL.Path)
	fmt.Println("Hello from origin!")
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Server running...")
	log.Fatal(http.ListenAndServe(":4000", nil))
}
