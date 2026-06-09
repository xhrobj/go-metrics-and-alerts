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

func newFileObserverForTest(t *testing.T, path string) *FileObserver {
	t.Helper()

	observer, err := NewFileObserver(path)
	if err != nil {
		t.Fatalf("create file observer failed: %v", err)
	}

	t.Cleanup(func() {
		if err := observer.Close(); err != nil {
			t.Fatalf("close file observer failed: %v", err)
		}
	})

	return observer
}

// Notify создаёт файл аудита и записывает событие отдельной JSON-строкой.
func TestFileObserver_Notify_CreatesFileAndWritesEvent_OK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := newFileObserverForTest(t, path)

	event := Event{
		TS:        42,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "127.0.0.1",
	}

	if err := observer.Notify(context.Background(), event); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}

	if err := observer.Close(); err != nil {
		t.Fatalf("close file observer failed: %v", err)
	}

	events := readAuditEvents(t, path)
	if len(events) != 1 {
		t.Fatalf("got events len %d, want %d", len(events), 1)
	}

	assertAuditEvent(t, events[0], event)
}

// Notify записывает несколько событий через один открытый FileObserver.
func TestFileObserver_Notify_WritesSeveralEvents_OK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := newFileObserverForTest(t, path)

	firstEvent := Event{
		TS:        42,
		Metrics:   []string{"Alloc"},
		IPAddress: "127.0.0.1",
	}

	secondEvent := Event{
		TS:        43,
		Metrics:   []string{"Frees"},
		IPAddress: "127.0.0.2",
	}

	if err := observer.Notify(context.Background(), firstEvent); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}

	if err := observer.Notify(context.Background(), secondEvent); err != nil {
		t.Fatalf("got error %v, want nil", err)
	}

	if err := observer.Close(); err != nil {
		t.Fatalf("close file observer failed: %v", err)
	}

	events := readAuditEvents(t, path)
	if len(events) != 2 {
		t.Fatalf("got events len %d, want %d", len(events), 2)
	}

	assertAuditEvent(t, events[0], firstEvent)
	assertAuditEvent(t, events[1], secondEvent)
}

// Notify возвращает ошибку, если контекст уже отменён.
func TestFileObserver_Notify_CanceledContext_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	observer := newFileObserverForTest(t, path)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := observer.Notify(ctx, Event{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want %v", err, context.Canceled)
	}
}

// NewFileObserver возвращает ошибку, если файл аудита невозможно открыть.
func TestNewFileObserver_OpenFileError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "audit.log")

	observer, err := NewFileObserver(path)
	if err == nil {
		t.Fatal("got nil error, want error")
	}

	if observer != nil {
		t.Fatal("got observer, want nil")
	}
}

// Close можно вызвать повторно без ошибки.
func TestFileObserver_Close_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	observer, err := NewFileObserver(path)
	if err != nil {
		t.Fatalf("create file observer failed: %v", err)
	}

	if err := observer.Close(); err != nil {
		t.Fatalf("close file observer failed: %v", err)
	}

	if err := observer.Close(); err != nil {
		t.Fatalf("second close file observer failed: %v", err)
	}
}

// Notify возвращает ошибку после закрытия FileObserver.
func TestFileObserver_Notify_AfterClose_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")

	observer, err := NewFileObserver(path)
	if err != nil {
		t.Fatalf("create file observer failed: %v", err)
	}

	if err := observer.Close(); err != nil {
		t.Fatalf("close file observer failed: %v", err)
	}

	err = observer.Notify(context.Background(), Event{})
	if err == nil {
		t.Fatal("got nil error, want error")
	}
}
