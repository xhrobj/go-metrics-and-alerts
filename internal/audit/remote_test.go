package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Notify отправляет событие аудита POST-запросом в формате JSON.
func TestRemoteObserver_Notify_SendsEvent_OK(t *testing.T) {
	event := Event{
		TS:        123,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "127.0.0.1",
	}

	events := make(chan Event, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("got method %s, want %s", r.Method, http.MethodPost)
		}

		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("got Content-Type %q, want %q", got, "application/json")
		}

		var got Event
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode audit event failed: %v", err)
		}

		events <- got
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)

	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}

	got := <-events

	if got.TS != event.TS {
		t.Errorf("got TS %d, want %d", got.TS, event.TS)
	}
	if got.IPAddress != event.IPAddress {
		t.Errorf("got IPAddress %q, want %q", got.IPAddress, event.IPAddress)
	}
	if len(got.Metrics) != len(event.Metrics) {
		t.Fatalf("got metrics len %d, want %d", len(got.Metrics), len(event.Metrics))
	}

	for i := range event.Metrics {
		if got.Metrics[i] != event.Metrics[i] {
			t.Errorf("got metric %q, want %q", got.Metrics[i], event.Metrics[i])
		}
	}
}

// Notify считает успешной отправку аудита, если удалённый приёмник вернул 2xx-статус.
func TestRemoteObserver_Notify_Accepts2xxStatus_OK(t *testing.T) {
	statuses := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
	}

	for _, status := range statuses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))

		observer := NewRemoteObserver(server.URL)

		if err := observer.Notify(context.Background(), Event{}); err != nil {
			t.Fatalf("got error %v, want nil for status %d", err, status)
		}

		server.Close()
	}
}

// Notify возвращает ошибку, если удалённый приёмник вернул не 2xx-статус.
func TestRemoteObserver_Notify_Non2xxStatus_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	observer := NewRemoteObserver(server.URL)

	err := observer.Notify(context.Background(), Event{})

	if err == nil {
		t.Fatal("got nil error, want error")
	}
}

// Notify возвращает ошибку, если контекст уже отменён.
func TestRemoteObserver_Notify_CanceledContext_ReturnsError(t *testing.T) {
	observer := NewRemoteObserver("http://example.com/audit")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := observer.Notify(ctx, Event{})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want %v", err, context.Canceled)
	}
}

// Notify возвращает ошибку, если URL удалённого приёмника некорректный.
func TestRemoteObserver_Notify_InvalidURL_ReturnsError(t *testing.T) {
	observer := NewRemoteObserver("://bad-url")

	err := observer.Notify(context.Background(), Event{})

	if err == nil {
		t.Fatal("got nil error, want error")
	}
}
