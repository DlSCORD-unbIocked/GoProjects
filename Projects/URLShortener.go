package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
)

type URLStore struct {
	mappings map[string]string
	mutex    sync.RWMutex
}

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

func NewURLStore() *URLStore {
	return &URLStore{
		mappings: make(map[string]string),
	}
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	shortCode := make([]byte, 6)
	for i := range shortCode {
		shortCode[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortCode)
}

func (store *URLStore) Get(shortCode string) (string, bool) {
	store.mutex.RLock()
	defer store.mutex.RUnlock()

	longURL, exists := store.mappings[shortCode]
	return longURL, exists
}

func (store *URLStore) Save(longURL string) string {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	var shortCode string
	for {
		shortCode = generateShortCode()
		if _, exists := store.mappings[shortCode]; !exists {
			break
		}
	}

	store.mappings[shortCode] = longURL
	return shortCode
}
