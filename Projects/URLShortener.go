package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/shorten", handleShorten)

	port := ":8080"

	fmt.Printf("Server starting on port %s\n", port)

	err := http.ListenAndServe(port, nil)

	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprint(w, "Shorten your urls")
	if err != nil {
		return
	}
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		_, err := fmt.Fprint(w, "test")
		if err != nil {
			return
		}
	} else {
		_, err := fmt.Fprint(w, "Send a post request to shorten a url")
		if err != nil {
			return
		}
	}
}
