package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	query := r.URL.Query()
	testURL := query.Get("url")

	if testURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	log.Printf("Proxying request to: %s", testURL)

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		http.Error(w, fmt.Sprintf("Error creating request: %v", err), http.StatusInternalServerError)
		return
	}

	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching response: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching response: %v", err), http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	log.Printf("Received response with status: %d", resp.StatusCode)

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	w.WriteHeader(resp.StatusCode)

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		fmt.Println("found")
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Error reading line: %v", err)
				return
			}

			line = strings.Replace(line, "href=\"/", fmt.Sprintf("href=\"/proxy?url=%s/", testURL), -1)
			line = strings.Replace(line, "src=\"/", fmt.Sprintf("src=\"/proxy?url=%s/", testURL), -1)

			_, err = w.Write([]byte(line))
			if err != nil {
				log.Printf("Error writing line: %v", err)
				return
			}
		}
	} else {
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("Error streaming response: %v", err)
		}
	}
}

func queryTest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	_, err := fmt.Fprint(w, "Name: ", name)
	if err != nil {
		return
	}
}
