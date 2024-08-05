package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/proxy", proxy)
	http.HandleFunc("/query", queryTest)
	fmt.Println("Starting server")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprint(w, "Hello, World!")
	if err != nil {
		return
	}
}

func proxy(w http.ResponseWriter, r *http.Request) {
	testURL := "https://www.google.com"

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		_ = fmt.Errorf("error creating request: %v", err)
	}

	req.Header = r.Header
	fmt.Println("Request Headers: ", req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error fetching response", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
}

func queryTest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	_, err := fmt.Fprint(w, "Name: ", name)
	if err != nil {
		return
	}
}
