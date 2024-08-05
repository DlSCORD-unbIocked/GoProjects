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
	http.HandleFunc("/html", htmlPage)
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

	resp, err := http.Get(testURL)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		http.Error(w, fmt.Sprintf("Error reading response body: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Received content length: %d bytes", len(body))

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(body)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}

	log.Printf("Proxy request completed")
}

func queryTest(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	_, err := fmt.Fprint(w, "Name: ", name)
	if err != nil {
		return
	}
}

func htmlPage(w http.ResponseWriter, r *http.Request) {
	html := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Proxy Redirect</title>
        <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.5.1/jquery.min.js"></script>
    </head>
    <body>
        <h1>Proxy Page</h1>
        <form id="proxyForm">
            <input type="url" id="urlInput" placeholder="Enter URL" required>
            <button type="submit">Load Content</button>
        </form>
        <div id="debug"></div>
        <div id="content"></div>
        <script>
            $(document).ready(function() {
                $("#proxyForm").submit(function(e) {
                    e.preventDefault();
                    var url = $("#urlInput").val();
                    $("#debug").html("Sending request...");
                    $("#content").html("Loading...");
                    $.ajax({
                        url: "/proxy?url=" + encodeURIComponent(url),
                        type: 'GET',
                        success: function(data, textStatus, jqXHR) {
                            $("#debug").html("Request successful. Status: " + jqXHR.status);
                            $("#content").html(data);
                        },
                        error: function(jqXHR, textStatus, errorThrown) {
                            $("#debug").html("Request failed. Status: " + jqXHR.status + errorThrown);
                        }
                    });
                });
            });
        </script>
    </body>
    </html>
    `
	_, err := fmt.Fprint(w, html)
	if err != nil {
		return
	}
}
