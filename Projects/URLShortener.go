package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var store = NewURLStore()
var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

type URLStore struct {
	mappings map[string]string
	mutex    sync.RWMutex
}

func main() {
	http.HandleFunc("/", handleRedirect)
	http.HandleFunc("/home", handleHome)
	http.HandleFunc("/shorten", handleShorten)

	port := ":8080"

	fmt.Printf("Server starting on port %s\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>URL Shortener</title>
    </head>
    <body>
        <h1>URL Shortener</h1>
        <form action="/shorten" method="post">
            <input type="text" name="url" placeholder="Enter URL to shorten" required>
            <input type="submit" value="Shorten">
        </form>
    </body>
    </html>
    `
	_, err := fmt.Fprint(w, html)
	if err != nil {
		http.Error(w, "Error generating response", http.StatusInternalServerError)
	}
}

func handleShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	longURL := r.FormValue("url")
	if longURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	shortCode := store.Save(longURL)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", scheme, r.Host, shortCode)

	html := fmt.Sprintf(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>URL Shortened</title>
    </head>
    <body>
        <h1>URL Shortened</h1>
        <p>Shortened URL: <a href="%s">%s</a></p>
        <a href="/">Go Back</a>
    </body>
    </html>
    `, shortURL, shortURL)
	_, err := fmt.Fprint(w, html)
	if err != nil {
		http.Error(w, "Error generating response", http.StatusInternalServerError)
	}
}

func handleRedirect(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		handleHome(w, r)
		return
	}

	shortCode := r.URL.Path[1:]
	if longURL, exists := store.Get(shortCode); exists {
		http.Redirect(w, r, longURL, http.StatusFound)
	} else {
		html := `
        <!DOCTYPE html>
        <html>
        <head>
            <title>404 Not Found</title>
        </head>
        <body>
            <h1>404 Not Found</h1>
            <p>The URL you are looking for does not exist.</p>
            <a href="/">Go Back</a>
        </body>
        </html>
        `
		w.WriteHeader(http.StatusNotFound)
		_, err := fmt.Fprint(w, html)
		if err != nil {
			http.Error(w, "Error generating response", http.StatusInternalServerError)
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
		shortCode[i] = charset[seededRand.Intn(len(charset))]
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
