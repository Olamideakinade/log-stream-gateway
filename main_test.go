package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogBufferRing(t *testing.T) {
	buf := NewLogBuffer(2)
	buf.Push(LogEntry{Message: "one", Level: "INFO"})
	buf.Push(LogEntry{Message: "two", Level: "INFO"})
	buf.Push(LogEntry{Message: "three", Level: "INFO"})

	entries := buf.List()
	if len(entries) != 2 {
		t.Fatalf("expected length 2, got %d", len(entries))
	}
	if entries[0].Message != "two" {
		t.Errorf("expected first message to be 'two', got '%s'", entries[0].Message)
	}
	if entries[1].Message != "three" {
		t.Errorf("expected second message to be 'three', got '%s'", entries[1].Message)
	}
}

func TestHTTPHandlerIngestAndFetch(t *testing.T) {
	buf := NewLogBuffer(10)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var entry LogEntry
			_ = json.NewDecoder(r.Body).Decode(&entry)
			buf.Push(entry)
			w.WriteHeader(http.StatusAccepted)
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(buf.List())
		}
	})

	// Post a log
	payload, _ := json.Marshal(LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Service:   "payment-service",
		Message:   "database timeout",
	})

	reqPost := httptest.NewRequest(http.MethodPost, "/logs", bytes.NewBuffer(payload))
	recPost := httptest.NewRecorder()
	mux.ServeHTTP(recPost, reqPost)

	if recPost.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", recPost.Code)
	}

	// Get logs
	reqGet := httptest.NewRequest(http.MethodGet, "/logs", nil)
	recGet := httptest.NewRecorder()
	mux.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recGet.Code)
	}

	var fetched []LogEntry
	_ = json.NewDecoder(recGet.Body).Decode(&fetched)
	if len(fetched) != 1 {
		Unt: t.Fatalf("expected 1 log entry returned, got %d", len(fetched))
	}
	if fetched[0].Service != "payment-service" {
		t.Errorf("expected service payment-service, got %s", fetched[0].Service)
	}
}
