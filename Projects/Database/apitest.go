package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	baseURL = "http://localhost:8080"
)

var (
	apiKey = "test"
)

func main() {
	operationPtr := flag.String("op", "shorten", "Choose: shorten, info, test")
	urlPtr := flag.String("url", "", "URL to shorten or get info for")
	customNamePtr := flag.String("custom", "", "(Optional) Custom name for shortened URL")
	expiresInPtr := flag.String("expires", "24h", "(Optional) Expiration time for shortened URL")

	flag.Parse()

	if *operationPtr == "test" {
		testErrorCases()
		return
	}
	if *urlPtr == "" {
		log.Fatal("URL is required")
	}

	switch *operationPtr {
	case "shorten":
		shortURL, err := shortenURL(*urlPtr, *customNamePtr, *expiresInPtr)
		if err != nil {
			log.Fatalf("Error shortening URL: %v", err)
		}
		fmt.Printf("Shortened URL: %s\n", shortURL)

	case "info":
		urlInfo, err := getURLInfo(*urlPtr)
		if err != nil {
			log.Fatalf("Error getting URL info: %v", err)
		}
		fmt.Printf("URL Info: %+v\n", urlInfo)

	default:
		log.Fatalf("Unknown operation: %s", *operationPtr)
	}
}

func shortenURL(longURL, customName, expiresIn string) (string, error) {
	payload := map[string]string{
		"long_url":    longURL,
		"custom_name": customName,
		"expires_in":  expiresIn,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/shorten", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	var result struct {
		ShortURL string `json:"short_url"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	return result.ShortURL, nil
}

func getURLInfo(shortURL string) (map[string]interface{}, error) {
	shortCode := shortURL[len(baseURL)+1:]
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/url?code=%s", baseURL, shortCode), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func testErrorCases() {
	fmt.Println("Running error case tests...")

	fmt.Println("\nTesting invalid URL:")
	_, err := shortenURL("not-a-valid-url", "", "1h")
	if err == nil {
		fmt.Println("Expected error for invalid URL, but got none")
	} else {
		fmt.Printf("Correctly received error for invalid URL: %v\n", err)
	}

	fmt.Println("\nTesting non-existent short code:")
	_, err = getURLInfo("http://localhost:8080/nonexistent")
	if err == nil {
		fmt.Println("Expected error for non-existent short code, but got none")
	} else {
		fmt.Printf("Correctly received error for non-existent short code: %v\n", err)
	}

	fmt.Println("\nTesting invalid expiration time:")
	_, err = shortenURL("https://example.com", "", "invalid-time")
	if err == nil {
		fmt.Println("Expected error for invalid expiration time, but got none")
	} else {
		fmt.Printf("Correctly received error for invalid expiration time: %v\n", err)
	}

	fmt.Println("\nTesting invalid API key:")
	originalAPIKey := apiKey
	apiKey = "invalid-key"
	_, err = shortenURL("https://example.com", "", "1h")
	if err == nil {
		fmt.Println("Expected error for invalid API key, but got none")
	} else {
		fmt.Printf("Correctly received error for invalid API key: %v\n", err)
	}
	apiKey = originalAPIKey

	fmt.Println("\nError case tests completed.")
}
