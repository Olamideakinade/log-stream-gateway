package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// LogEntry represents a structured application log event.
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Service   string            `json:"service"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// LogBuffer safely stores log entries in memory with a fixed capacity ring buffer.
type LogBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	max     int
}

func NewLogBuffer(max int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, 0, max),
		max:     max,
	}
}

func (b *LogBuffer) Push(entry LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.entries) >= b.max {
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
}

func (b *LogBuffer) List() []LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]LogEntry, len(b.entries))
	copy(out, b.entries)
	return out
}

func main() {
	port := flag.Int("port", 8080, "HTTP listen port")
	capacity := flag.Int("capacity", 1000, "In-memory ring buffer capacity")
	flag.Parse()

	buf := NewLogBuffer(*capacity)

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var entry LogEntry
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&entry);
			err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if entry.Timestamp.IsZero() {
				entry.Timestamp = time.Now()
			}
			buf.Push(entry)
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprintf(w, `{"status":"accepted"}`)

		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(buf.List())

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting log stream gateway on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
