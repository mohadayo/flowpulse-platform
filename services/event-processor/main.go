package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Event struct {
	ID         int                    `json:"id"`
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data"`
	ReceivedAt float64                `json:"received_at,omitempty"`
}

type ProcessedEvent struct {
	Event
	ProcessedAt string   `json:"processed_at"`
	Tags        []string `json:"tags"`
	Priority    string   `json:"priority"`
}

type Stats struct {
	TotalProcessed int            `json:"total_processed"`
	ByType         map[string]int `json:"by_type"`
	ByPriority     map[string]int `json:"by_priority"`
}

var (
	processed []ProcessedEvent
	mu        sync.RWMutex
	logger    *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "[event-processor] ", log.LstdFlags)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func classifyPriority(eventType string) string {
	switch {
	case strings.Contains(eventType, "error") || strings.Contains(eventType, "alert"):
		return "high"
	case strings.Contains(eventType, "warning"):
		return "medium"
	default:
		return "low"
	}
}

func generateTags(e Event) []string {
	tags := []string{e.Type}
	if _, ok := e.Data["page"]; ok {
		tags = append(tags, "page-event")
	}
	if _, ok := e.Data["user_id"]; ok {
		tags = append(tags, "user-event")
	}
	return tags
}

func ProcessEvent(e Event) ProcessedEvent {
	return ProcessedEvent{
		Event:       e,
		ProcessedAt: time.Now().UTC().Format(time.RFC3339),
		Tags:        generateTags(e),
		Priority:    classifyPriority(e.Type),
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "event-processor",
		"timestamp": time.Now().Unix(),
	})
}

func processHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	if e.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "event type is required"})
		return
	}
	pe := ProcessEvent(e)
	mu.Lock()
	processed = append(processed, pe)
	mu.Unlock()
	logger.Printf("Processed event id=%d type=%s priority=%s", pe.ID, pe.Type, pe.Priority)
	writeJSON(w, http.StatusOK, pe)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	stats := Stats{
		TotalProcessed: len(processed),
		ByType:         make(map[string]int),
		ByPriority:     make(map[string]int),
	}
	for _, pe := range processed {
		stats.ByType[pe.Type]++
		stats.ByPriority[pe.Priority]++
	}
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/process", processHandler)
	mux.HandleFunc("/stats", statsHandler)
	return mux
}

func main() {
	port := getEnv("PROCESSOR_PORT", "8002")
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	logger.Printf("Event Processor starting on %s", addr)
	if err := http.ListenAndServe(addr, NewMux()); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}
