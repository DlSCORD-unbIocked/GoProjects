package main

// will use github.com/skip2/go-qrcode for qr implementation
import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/skip2/go-qrcode"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var store = NewURLStore()
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

type URLStore struct {
	mappings map[string]URLRecord
	mutex    sync.RWMutex
}

type URLRecord struct {
	LongURL    string
	ExpiresAt  time.Time
	CustomName string
	Clicks     int
}

func init() {
	fmt.Println("URL Shortener starting...")
}

func main() {
	err := initDB()
	if err != nil {
		fmt.Printf("Error initializing database: %s\n", err)
		return
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Printf("Error closing database: %v\n", err)
		}
	}(db)

	http.HandleFunc("/", handleRedirect)
	http.HandleFunc("/home", handleHome)
	http.HandleFunc("/shorten", referrerCheck(handleShorten))
	http.HandleFunc("/URLShortener.css", serveCSS)
	http.HandleFunc("/qr", handleQRCode)
	http.HandleFunc("/clicks/", handleGetClicks)

	http.HandleFunc("/api/shorten", apiKeyMiddleware(handleAPIShorten))
	http.HandleFunc("/api/url", apiKeyMiddleware(handleAPIGetURL))
	http.HandleFunc("/api/docs", handleAPIDocs)

	port := ":8080"

	fmt.Printf("Server starting on port %s\n", port)

	err = http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
	//prolly want every hour but for testing do every 5 mins
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			store.cleanupExpiredLinks()
		}
	}()
}

func serveCSS(w http.ResponseWriter, r *http.Request) {
	path, _ := filepath.Abs("Projects/static/URLShortener.css")
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading CSS file: %v\n", err)
		http.Error(w, "Could not read CSS file", http.StatusInternalServerError)
		return
	}
	//fmt.Printf("CSS file size: %d bytes\n", len(content))
	w.Header().Set("Content-Type", "text/css")
	_, err = w.Write(content)
	if err != nil {
		fmt.Printf("Error writing CSS response: %v\n", err)
		http.Error(w, "Error writing CSS response", http.StatusInternalServerError)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// :heart: jetbrains mono
	store.mutex.RLock()
	urlList := make([]struct {
		ShortCode string
		LongURL   string
		Clicks    int
		ExpiresAt time.Time
	}, 0, len(store.mappings))

	now := time.Now()
	for shortCode, record := range store.mappings {
		if now.Before(record.ExpiresAt) {
			urlList = append(urlList, struct {
				ShortCode string
				LongURL   string
				Clicks    int
				ExpiresAt time.Time
			}{shortCode, record.LongURL, record.Clicks, record.ExpiresAt})
		}
	}
	store.mutex.RUnlock()

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
        <h2>Shortened URLs</h2>
        <table>
            <tr>
                <th>Short URL</th>
                <th>Original URL</th>
                <th>Clicks</th>
                <th>QR Code</th>
            </tr>
    `

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	for _, url := range urlList {
		qrURL := fmt.Sprintf("%s://%s/qr?code=%s", scheme, r.Host, url.ShortCode)
		html += fmt.Sprintf(`
            <tr>
                <td><a href="/%s">%s</a></td>
                <td>%s</td>
                <td>%d</td>
                <td><a href="%s" target="_blank">View QR</a></td>
            </tr>
        `, url.ShortCode, url.ShortCode, url.LongURL, url.Clicks, qrURL)
	}

	html += `
        </table>
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

	if !isValidURL(longURL) {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	expirationStr := r.FormValue("expires_in")
	//fmt.Println(expirationStr)
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
	qrURL := fmt.Sprintf("%s://%s/qr?code=%s", scheme, r.Host, shortCode)

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
        <p>Clicks: <span id="clicks">0</span></p>
        <p>QR Code: <a href="%s" target="_blank">View QR Code</a></p>
        <img src="%s" alt="QR Code" width="200" height="200">
        <br>
        <a href="/">Go Back</a>
        <script>
        function updateClicks() {
            fetch('/clicks/%s')
                .then(response => response.json())
                .then(data => {
                    document.getElementById('clicks').textContent = data.clicks;
                });
        }
        setInterval(updateClicks, 5000);
        updateClicks(); 
        </script>
    </body>
    </html>
    `, shortURL, shortURL, qrURL, qrURL, shortCode)
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

	if !exists {
		http.NotFound(w, r)
		return
	}

	if time.Now().After(record.ExpiresAt) {
		store.mutex.Lock()
		delete(store.mappings, shortCode)
		store.mutex.Unlock()
		http.NotFound(w, r)
		return
	}

	store.IncrementClicks(shortCode)
	http.Redirect(w, r, record.LongURL, http.StatusFound)
}

func handleGetClicks(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Path[len("/clicks/"):]
	store.mutex.RLock()
	record, exists := store.mappings[shortCode]
	store.mutex.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]int{"clicks": record.Clicks})
	if err != nil {
		return
	}
}

func handleAPIShorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		LongURL    string `json:"long_url"`
		CustomName string `json:"custom_name,omitempty"`
		ExpiresIn  string `json:"expires_in,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.LongURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	if !isValidURL(input.LongURL) {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	expiresIn, err := time.ParseDuration(input.ExpiresIn)
	if err != nil {
		expiresIn = 24 * time.Hour
	}

	shortCode := store.Save(input.LongURL, expiresIn, input.CustomName)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", scheme, r.Host, shortCode)

	response := struct {
		ShortURL string `json:"short_url"`
	}{
		ShortURL: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

func handleAPIGetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shortCode := r.URL.Query().Get("code")
	if shortCode == "" {
		http.Error(w, "Missing short code", http.StatusBadRequest)
		return
	}

	store.mutex.RLock()
	record, exists := store.mappings[shortCode]
	store.mutex.RUnlock()

	if !exists || time.Now().After(record.ExpiresAt) {
		http.NotFound(w, r)
		return
	}

	response := struct {
		LongURL    string `json:"long_url"`
		Clicks     int    `json:"clicks"`
		ExpiresAt  string `json:"expires_at"`
		CustomName string `json:"custom_name,omitempty"`
	}{
		LongURL:    record.LongURL,
		Clicks:     record.Clicks,
		ExpiresAt:  record.ExpiresAt.Format(time.RFC3339),
		CustomName: record.CustomName,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

func handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	docs := `
    API Documentation:

    1. Shorten URL
       Endpoint: POST /api/shorten
       Headers: X-API-Key: your-secret-api-key
       Body: {
           "long_url": "https://example.com",
           "custom_name": "optional-custom-name",
           "expires_in": "24h" // put in any time ie. 24h, 1h, 30m, 1s
       }

    2. Get URL Info
       Endpoint: GET /api/url?code=<short_code>
       Headers: X-API-Key: your-secret-api-key

    `

	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte(docs))
	if err != nil {
		return
	}
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

func (store *URLStore) IncrementClicks(shortCode string) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if record, exists := store.mappings[shortCode]; exists {
		record.Clicks++
		store.mappings[shortCode] = record
	}
}

func (store *URLStore) cleanupExpiredLinks() {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	now := time.Now()
	for shortCode, record := range store.mappings {
		if now.After(record.ExpiresAt) {
			delete(store.mappings, shortCode)
		}
	}
}

func isValidURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func handleQRCode(w http.ResponseWriter, r *http.Request) {
	shortCode := r.URL.Query().Get("code")
	if shortCode == "" {
		http.Error(w, "Missing short code", http.StatusBadRequest)
		return
	}

	store.mutex.RLock()
	_, exists := store.mappings[shortCode]
	store.mutex.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	fullURL := fmt.Sprintf("%s://%s/%s", scheme, r.Host, shortCode)

	qr, err := qrcode.Encode(fullURL, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	_, err = w.Write(qr)
	if err != nil {
		return
	}
}

// use in /api/... maybe
func apiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test" { // temp for testing
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func referrerCheck(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/shorten" {
			referer := r.Header.Get("Referer")
			if !strings.HasPrefix(referer, "http://localhost:8080/") &&
				!strings.HasPrefix(referer, "https://localhost:8080/") {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	}
}
