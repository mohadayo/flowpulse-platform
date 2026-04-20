package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "healthy" {
		t.Errorf("expected healthy status")
	}
	if body["service"] != "event-processor" {
		t.Errorf("expected event-processor service name")
	}
}

func TestProcessEvent(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	event := map[string]interface{}{
		"id":   1,
		"type": "click",
		"data": map[string]interface{}{"page": "/home"},
	}
	body, _ := json.Marshal(event)
	resp, err := http.Post(srv.URL+"/process", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result ProcessedEvent
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Type != "click" {
		t.Errorf("expected type click, got %s", result.Type)
	}
	if result.Priority != "low" {
		t.Errorf("expected low priority, got %s", result.Priority)
	}
	if len(result.Tags) < 1 {
		t.Error("expected at least one tag")
	}
}

func TestProcessEventHighPriority(t *testing.T) {
	event := Event{ID: 1, Type: "error_critical", Data: map[string]interface{}{}}
	pe := ProcessEvent(event)
	if pe.Priority != "high" {
		t.Errorf("expected high priority, got %s", pe.Priority)
	}
}

func TestProcessEventMediumPriority(t *testing.T) {
	event := Event{ID: 2, Type: "warning_low_disk", Data: map[string]interface{}{}}
	pe := ProcessEvent(event)
	if pe.Priority != "medium" {
		t.Errorf("expected medium priority, got %s", pe.Priority)
	}
}

func TestProcessInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/process", "application/json", bytes.NewReader([]byte("bad")))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessMissingType(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	body, _ := json.Marshal(map[string]interface{}{"data": map[string]interface{}{}})
	resp, err := http.Post(srv.URL+"/process", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessMethodNotAllowed(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/process")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestStatsEndpoint(t *testing.T) {
	mu.Lock()
	processed = nil
	mu.Unlock()

	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	event := map[string]interface{}{"id": 1, "type": "click", "data": map[string]interface{}{}}
	body, _ := json.Marshal(event)
	http.Post(srv.URL+"/process", "application/json", bytes.NewReader(body))

	resp, err := http.Get(srv.URL + "/stats")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var stats Stats
	json.NewDecoder(resp.Body).Decode(&stats)
	if stats.TotalProcessed < 1 {
		t.Errorf("expected at least 1 processed event")
	}
}

func TestGenerateTags(t *testing.T) {
	e := Event{Type: "click", Data: map[string]interface{}{"page": "/test", "user_id": "u1"}}
	tags := generateTags(e)
	found := map[string]bool{}
	for _, tag := range tags {
		found[tag] = true
	}
	if !found["click"] {
		t.Error("expected click tag")
	}
	if !found["page-event"] {
		t.Error("expected page-event tag")
	}
	if !found["user-event"] {
		t.Error("expected user-event tag")
	}
}
