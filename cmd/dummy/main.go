package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {
	port := "3000"
	log.Printf("Starting Dummy Echo Server on port %s...", port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Dummy] Received Request: %s %s", r.Method, r.URL.Path)

		// Set response headers
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Dummy-Server", "true")

		// Write to body
		fmt.Fprintf(w, "--- ECHO RESPONSE ---\n")
		fmt.Fprintf(w, "Method: %s\n", r.Method)
		fmt.Fprintf(w, "URL: %s\n", r.URL.String())

		fmt.Fprintf(w, "Headers:\n")
		for k, vv := range r.Header {
			fmt.Fprintf(w, "  %s: %s\n", k, strings.Join(vv, ", "))
		}

		fmt.Fprintf(w, "Body:\n")
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				w.Write(body)
				w.Write([]byte("\n"))
			} else {
				fmt.Fprintf(w, "<empty>\n")
			}
			r.Body.Close()
		}

		w.WriteHeader(http.StatusOK)
		log.Printf("[Dummy] Responded to %s %s", r.Method, r.URL.Path)
	})

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
