package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	baseURL = "http://localhost:8080"
	apiKey  = "test"
)

func main() {
	fmt.Println("Enter a URL to shorten: ")
	var longURL string
	_, err := fmt.Scanln(&longURL)
	if err != nil {
		fmt.Printf("Error reading URL: %v\n", err)
		return
	}
	shortURL, err := shortenURL(longURL, "test", "2h")
	if err != nil {
		fmt.Printf("Error shortening URL: %v\n", err)
		return
	}
	fmt.Printf("Shortened URL: %s\n", shortURL)

	urlInfo, err := getURLInfo(shortURL)
	if err != nil {
		fmt.Printf("Error getting URL info: %v\n", err)
		return
	}
	fmt.Printf("URL Info: %+v\n", urlInfo)
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