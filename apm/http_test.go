package apm

import (
	"log"
	"net/http"
	"testing"
	"time"
)

func TestHTTPServer_Handle(t *testing.T) {
	server := NewHTTPServer(":12345")
	server.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Hello, World!")); err != nil {
			log.Println("Error writing response:", err)
		}
	}))
	server.Start()
	for {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:12345/")
		if err != nil {
			continue
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("Failed to close response body: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
	}
	server.Close()
}
