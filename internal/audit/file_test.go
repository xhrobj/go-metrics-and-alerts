package audit

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readAuditEvents читает файл аудита в формате JSON Lines и возвращает события аудита.
func readAuditEvents(t *testing.T, path string) []Event {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	events := make([]Event, 0, len(lines))

	for _, line := range lines {
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("unmarshal audit event failed: %v", err)
		}

		events = append(events, event)
	}

	return events
}

// Notify создаёт файл аудита и записывает событие отдельной JSON-строкой.
func TestFileObserver_Notify_CreatesFileAndWritesEvent_OK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := NewFileObserver(path)

	event := Event{
		TS:        123,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "127.0.0.1",
	}

	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}

	events := readAuditEvents(t, path)

	if len(events) != 1 {
		t.Fatalf("got events len %d, want %d", len(events), 1)
	}

	got := events[0]
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

// Notify добавляет новое событие в конец существующего файла аудита.
func TestFileObserver_Notify_AppendsEvent_OK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := NewFileObserver(path)

	firstEvent := Event{
		TS:        123,
		Metrics:   []string{"Alloc"},
		IPAddress: "127.0.0.1",
	}
	secondEvent := Event{
		TS:        456,
		Metrics:   []string{"Frees"},
		IPAddress: "127.0.0.2",
	}

	if err := observer.Notify(context.Background(), firstEvent); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if err := observer.Notify(context.Background(), secondEvent); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}

	events := readAuditEvents(t, path)

	if len(events) != 2 {
		t.Fatalf("got events len %d, want %d", len(events), 2)
	}

	if events[0].TS != firstEvent.TS {
		t.Errorf("got first TS %d, want %d", events[0].TS, firstEvent.TS)
	}
	if events[1].TS != secondEvent.TS {
		t.Errorf("got second TS %d, want %d", events[1].TS, secondEvent.TS)
	}
}

// Notify возвращает ошибку, если контекст уже отменён.
func TestFileObserver_Notify_CanceledContext_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := NewFileObserver(path)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := observer.Notify(ctx, Event{})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want %v", err, context.Canceled)
	}
}

// Notify возвращает ошибку, если файл аудита невозможно открыть.
func TestFileObserver_Notify_OpenFileError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "audit.log")
	observer := NewFileObserver(path)

	err := observer.Notify(context.Background(), Event{})

	if err == nil {
		t.Fatal("got nil error, want error")
	}
}
