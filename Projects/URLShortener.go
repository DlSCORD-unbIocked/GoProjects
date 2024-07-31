package main

import (
	"embed"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

var store = NewURLStore()
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
var cssFile embed.FS

type URLStore struct {
	mappings map[string]URLRecord
	mutex    sync.RWMutex
}

type URLRecord struct {
	LongURL    string
	ExpiresAt  time.Time
	CustomName string
}

func main() {
	http.HandleFunc("/", handleRedirect)
	http.HandleFunc("/home", handleHome)
	http.HandleFunc("/shorten", handleShorten)
	http.HandleFunc("/URLShortener.css", serveCSS)

	port := ":8080"

	fmt.Printf("Server starting on port %s\n", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

func serveCSS(w http.ResponseWriter, r *http.Request) {
	//ccurrently hardcoded change later
	path := ""
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading CSS file: %v\n", err)
		http.Error(w, "Could not read CSS file", http.StatusInternalServerError)
		return
	}
	fmt.Printf("CSS file size: %d bytes\n", len(content))
	w.Header().Set("Content-Type", "text/css")
	_, err = w.Write(content)
	if err != nil {
		fmt.Printf("Error writing CSS response: %v\n", err)
		http.Error(w, "Error writing CSS response", http.StatusInternalServerError)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// :heart: jetbrains mono
	html := `
    <!DOCTYPE html>
	<html>
	<head>
		<title>URL Shortener</title>
		<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=JetBrains+Mono&display=swap">
		<link rel="stylesheet" href="/URLShortener.css">
	</head>
	<body>
		<h1>URL Shortener</h1>
		<form action="/shorten" method="post">
			<input type="text" name="url" placeholder="Enter URL to shorten" required>
			<input type="text" name="expires_in" placeholder="Expiration (e.g., 24h)" optional>
			<input type="text" name="custom_name" placeholder="Custom name (optional)">
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
	customName := r.FormValue("custom_name")

	if longURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	expirationStr := r.FormValue("expires_in")
	fmt.Println(expirationStr)
	var expiresIn = 24 * time.Hour //for testing change ltr to 24hrs maybe
	if expirationStr != "" {
		var err error
		expiresIn, err = time.ParseDuration(expirationStr)
		if err != nil {
			http.Error(w, "Invalid expiration duration", http.StatusBadRequest)
			return
		}
	}

	var shortCode string
	if customName != "" {
		if store.IsCustomNameAvailable(customName) {
			shortCode = customName
		} else {
			http.Error(w, "Custom name already in use", http.StatusBadRequest)
			return
		}
	}
	shortCode = store.Save(longURL, expiresIn, customName)

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
		<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=JetBrains+Mono&display=swap">
		<link rel="stylesheet" href="/URLShortener.css">
    </head>
    <body>
        <h1>URL Shortened</h1>
        <p>Shortened URL: <a href="%s">%s</a></p>
        <a id="back" href="/">Go Back</a>
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
	store.mutex.RLock()
	record, exists := store.mappings[shortCode]
	store.mutex.RUnlock()

	if !exists || time.Now().After(record.ExpiresAt) {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, record.LongURL, http.StatusFound)
}

func NewURLStore() *URLStore {
	return &URLStore{
		mappings: make(map[string]URLRecord),
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

	record, exists := store.mappings[shortCode]
	if !exists || time.Now().After(record.ExpiresAt) {
		return "", false
	}
	return record.LongURL, true
}

func (store *URLStore) Save(longURL string, expiresIn time.Duration, customName string) string {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	var shortCode string
	if customName != "" {
		shortCode = customName
	} else {
		for {
			shortCode = generateShortCode()
			if _, exists := store.mappings[shortCode]; !exists {
				break
			}
		}
	}

	store.mappings[shortCode] = URLRecord{
		LongURL:    longURL,
		ExpiresAt:  time.Now().Add(expiresIn),
		CustomName: customName,
	}
	return shortCode
}

func (store *URLStore) IsCustomNameAvailable(name string) bool {
	store.mutex.RLock()
	defer store.mutex.RUnlock()
	_, exists := store.mappings[name]
	return !exists
}
