package gobutton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Event JSON marshaling/unmarshaling tests ---

func TestEventMarshalJSON(t *testing.T) {
	battery := uint8(85)
	evt := Event{
		ButtonID:  "btn-001",
		Action:    ActionSingle,
		Battery:   &battery,
		Timestamp: "2026-04-17T10:30:00Z",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal of Marshal output failed: %v", err)
	}

	if got["button_id"] != "btn-001" {
		t.Errorf("button_id: got %v, want btn-001", got["button_id"])
	}
	if got["action"] != "Single" {
		t.Errorf("action: got %v, want Single", got["action"])
	}
	if got["battery"] != float64(85) {
		t.Errorf("battery: got %v, want 85", got["battery"])
	}
	if got["timestamp"] != "2026-04-17T10:30:00Z" {
		t.Errorf("timestamp: got %v, want 2026-04-17T10:30:00Z", got["timestamp"])
	}
}

func TestEventMarshalJSONWithoutBattery(t *testing.T) {
	evt := Event{
		ButtonID:  "btn-002",
		Action:    ActionDouble,
		Timestamp: "2026-04-17T11:00:00Z",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal of Marshal output failed: %v", err)
	}

	if _, ok := got["battery"]; ok {
		t.Errorf("battery should be omitted when nil, but got: %v", got["battery"])
	}
}

func TestEventUnmarshalJSON(t *testing.T) {
	jsonData := `{"button_id":"btn-003","action":"LongPress","battery":100,"timestamp":"2026-04-17T12:00:00Z"}`

	var evt Event
	if err := json.Unmarshal([]byte(jsonData), &evt); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if evt.ButtonID != "btn-003" {
		t.Errorf("ButtonID: got %q, want btn-003", evt.ButtonID)
	}
	if evt.Action != ActionLongPress {
		t.Errorf("Action: got %q, want LongPress", evt.Action)
	}
	if evt.Battery == nil || *evt.Battery != 100 {
		t.Errorf("Battery: got %v, want *100", evt.Battery)
	}
	if evt.Timestamp != "2026-04-17T12:00:00Z" {
		t.Errorf("Timestamp: got %q, want 2026-04-17T12:00:00Z", evt.Timestamp)
	}
}

func TestEventUnmarshalJSONWithoutBattery(t *testing.T) {
	jsonData := `{"button_id":"btn-004","action":"Single","timestamp":"2026-04-17T12:30:00Z"}`

	var evt Event
	if err := json.Unmarshal([]byte(jsonData), &evt); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if evt.Battery != nil {
		t.Errorf("Battery: got %v, want nil", evt.Battery)
	}
}

func TestActionTypes(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected string
	}{
		{ActionSingle, "Single"},
		{ActionDouble, "Double"},
		{ActionLongPress, "LongPress"},
	}

	for _, tt := range tests {
		if string(tt.action) != tt.expected {
			t.Errorf("ActionType %v: got %q, want %q", tt.action, string(tt.action), tt.expected)
		}
	}
}

// --- SSE data parsing tests ---

func TestClientListenFiltersByButtonID(t *testing.T) {
	// Create test events
	events := []Event{
		{ButtonID: "btn-A", Action: ActionSingle, Timestamp: "2026-04-17T10:00:00Z"},
		{ButtonID: "btn-B", Action: ActionDouble, Timestamp: "2026-04-17T10:01:00Z"},
		{ButtonID: "btn-A", Action: ActionLongPress, Timestamp: "2026-04-17T10:02:00Z"},
		{ButtonID: "btn-C", Action: ActionSingle, Timestamp: "2026-04-17T10:03:00Z"},
		{ButtonID: "btn-A", Action: ActionDouble, Timestamp: "2026-04-17T10:04:00Z"},
	}

	var receivedEvents []Event
	handler := func(evt Event) {
		receivedEvents = append(receivedEvents, evt)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}

		for _, evt := range events {
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data:%s\n\n", string(data))
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
		// Don't close here - let context cancellation handle it
	}))
	defer server.Close()

	// Create client and listen for btn-A events only
	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_ = client.Listen(ctx, "btn-A", handler)

	// Verify we received only btn-A events
	if len(receivedEvents) != 3 {
		t.Errorf("received %d events for btn-A, want 3: %+v", len(receivedEvents), receivedEvents)
	}

	for _, evt := range receivedEvents {
		if evt.ButtonID != "btn-A" {
			t.Errorf("received event with ButtonID %q, want btn-A", evt.ButtonID)
		}
	}
}

func TestClientListenNoMatchingButtonID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}
		evt := Event{ButtonID: "other-btn", Action: ActionSingle, Timestamp: "2026-04-17T10:00:00Z"}
		data, _ := json.Marshal(evt)
		fmt.Fprintf(w, "data:%s\n\n", string(data))
		flusher.Flush()
		// Close connection after sending data
	}))
	defer server.Close()

	var receivedEvents []Event
	handler := func(evt Event) {
		receivedEvents = append(receivedEvents, evt)
	}

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = client.Listen(ctx, "nonexistent-btn", handler)

	if len(receivedEvents) != 0 {
		t.Errorf("received %d events, want 0: %+v", len(receivedEvents), receivedEvents)
	}
}

func TestClientListenEmptyStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Send empty data
		fmt.Fprintf(w, "data:\n\n")
	}))
	defer server.Close()

	var receivedEvents []Event
	handler := func(evt Event) {
		receivedEvents = append(receivedEvents, evt)
	}

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := client.Listen(ctx, "any-btn", handler)
	if err == nil {
		t.Error("expected an error from empty stream")
	}
}

func TestClientListenHandlesMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}

		// Send valid event
		fmt.Fprintf(w, "data:{\"button_id\":\"valid-btn\",\"action\":\"Single\",\"timestamp\":\"2026-04-17T10:00:00Z\"}\n\n")
		flusher.Flush()

		// Send malformed JSON - should be skipped
		fmt.Fprintf(w, "data:invalid json here\n\n")
		flusher.Flush()

		// Send another valid event for a different button (filtered out)
		fmt.Fprintf(w, "data:{\"button_id\":\"another-valid\",\"action\":\"Double\",\"timestamp\":\"2026-04-17T10:01:00Z\"}\n\n")
		flusher.Flush()

		// Close connection to signal end of stream
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	var receivedEvents []Event
	handler := func(evt Event) {
		receivedEvents = append(receivedEvents, evt)
	}

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = client.Listen(ctx, "valid-btn", handler)

	// Should receive the valid event and ignore the malformed one
	if len(receivedEvents) != 1 {
		t.Errorf("received %d events, want 1: %+v", len(receivedEvents), receivedEvents)
	}
}

// --- Registry Register/Unregister tests ---

func TestRegisterSuccess(t *testing.T) {
	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/buttons" {
			t.Errorf("Path: got %s, want /buttons", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type: got %s, want application/json", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Register(context.Background(), server.URL, "test-btn-001")
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if receivedBody["button_id"] != "test-btn-001" {
		t.Errorf("button_id: got %q, want test-btn-001", receivedBody["button_id"])
	}
}

func TestRegisterFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	err := Register(context.Background(), server.URL, "test-btn-002")
	if err == nil {
		t.Error("Register should return error on failure")
	}
}

func TestRegisterNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	err := Register(context.Background(), server.URL, "bad-btn")
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestUnregisterSuccess(t *testing.T) {
	var receivedMethod, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Unregister(context.Background(), server.URL, "test-btn-003")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	if receivedMethod != http.MethodDelete {
		t.Errorf("Method: got %s, want DELETE", receivedMethod)
	}
	if receivedPath != "/buttons/test-btn-003" {
		t.Errorf("Path: got %s, want /buttons/test-btn-003", receivedPath)
	}
}

func TestUnregisterFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := Unregister(context.Background(), server.URL, "nonexistent-btn")
	if err == nil {
		t.Error("Unregister should return error on failure")
	}
}

func TestUnregisterNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	err := Unregister(context.Background(), server.URL, "forbidden-btn")
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestRegisterURLWithTrailingSlash(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Pass URL with trailing slash
	err := Register(context.Background(), server.URL+"/", "trailing-slash-btn")
	if err != nil {
		t.Fatalf("Register with trailing slash failed: %v", err)
	}

	// Path should not have double slashes
	if receivedPath != "/buttons" {
		t.Errorf("Path: got %s, want /buttons (no double slash)", receivedPath)
	}
}

func TestUnregisterURLWithTrailingSlash(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Unregister(context.Background(), server.URL+"/", "trailing-slash-btn")
	if err != nil {
		t.Fatalf("Unregister with trailing slash failed: %v", err)
	}

	if receivedPath != "/buttons/trailing-slash-btn" {
		t.Errorf("Path: got %s, want /buttons/trailing-slash-btn", receivedPath)
	}
}

func TestRegisterContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow handler to allow context cancellation
		select {
		case <-done:
			w.WriteHeader(http.StatusOK)
		case <-time.After(5 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	defer close(done)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Register(ctx, server.URL, "slow-btn")
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

// --- NewClient tests ---

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client := NewClient("http://example.com/api/")

	if client.apiURL != "http://example.com/api" {
		t.Errorf("apiURL: got %q, want http://example.com/api", client.apiURL)
	}
}

func TestNewClientNoTrailingSlash(t *testing.T) {
	client := NewClient("http://example.com/api")

	if client.apiURL != "http://example.com/api" {
		t.Errorf("apiURL: got %q, want http://example.com/api", client.apiURL)
	}
}

func TestClientListenInvalidURL(t *testing.T) {
	client := NewClient("://invalid-url")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.Listen(ctx, "any-btn", func(evt Event) {})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

// --- SSE stream format tests ---

func TestClientListenSSEMultiLineData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not implement http.Flusher")
		}

		// Send two events in sequence
		fmt.Fprintf(w, "data:{\"button_id\":\"ml-btn\",\"action\":\"Single\",\"timestamp\":\"2026-04-17T10:00:00Z\"}\n\n")
		flusher.Flush()

		fmt.Fprintf(w, "data:{\"button_id\":\"ml-btn\",\"action\":\"Double\",\"timestamp\":\"2026-04-17T10:01:00Z\"}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	var receivedEvents []Event
	handler := func(evt Event) {
		receivedEvents = append(receivedEvents, evt)
	}

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = client.Listen(ctx, "ml-btn", handler)

	if len(receivedEvents) != 2 {
		t.Errorf("received %d events, want 2: %+v", len(receivedEvents), receivedEvents)
	}
}

func TestClientListenHTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "service unavailable")
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := client.Listen(ctx, "any-btn", func(evt Event) {})
	if err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestSSELineParsing(t *testing.T) {
	// Test various SSE line formats
	tests := []struct {
		name    string
		line    string
		isData  bool
		payload string
	}{
		{"simple data", "data:hello", true, "hello"},
		{"data with space", "data: hello", true, "hello"},
		{"data with leading spaces", "data:   hello", true, "hello"},
		{"event line", "event:message", false, ""},
		{"id line", "id:123", false, ""},
		{"empty line", "", false, ""},
		{"comment", ":comment", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := []byte(tt.line)
			if bytes.HasPrefix(line, []byte("data:")) {
				if !tt.isData {
					t.Errorf("expected %q not to be data line", tt.line)
				}
				payload := string(bytes.TrimPrefix(line, []byte("data:")))
				payload = string(bytes.TrimLeft([]byte(payload), " "))
				if payload != tt.payload {
					t.Errorf("payload: got %q, want %q", payload, tt.payload)
				}
			}
		})
	}
}
