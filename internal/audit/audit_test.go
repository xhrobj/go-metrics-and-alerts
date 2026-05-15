package audit

import (
	"context"
	"errors"
	"testing"
)

type mockObserver struct {
	calls  int
	events []Event
	err    error
}

func (m *mockObserver) Notify(ctx context.Context, event Event) error {
	m.calls++
	m.events = append(m.events, event)

	return m.err
}

// Notify рассылает событие всем подписанным Observer-ам.
func TestAuditor_Notify_OK(t *testing.T) {
	event := Event{
		TS:        123,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "127.0.0.1",
	}

	firstObserver := &mockObserver{}
	secondObserver := &mockObserver{}

	auditor := NewAuditor()
	auditor.Subscribe(firstObserver)
	auditor.Subscribe(secondObserver)

	if err := auditor.Notify(context.Background(), event); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if firstObserver.calls != 1 {
		t.Fatalf("expected first observer calls %d, got %d", 1, firstObserver.calls)
	}
	if secondObserver.calls != 1 {
		t.Fatalf("expected second observer calls %d, got %d", 1, secondObserver.calls)
	}

	got := firstObserver.events[0]
	if got.TS != event.TS {
		t.Errorf("expected TS %d, got %d", event.TS, got.TS)
	}
	if got.IPAddress != event.IPAddress {
		t.Errorf("expected IPAddress %q, got %q", event.IPAddress, got.IPAddress)
	}
	if len(got.Metrics) != len(event.Metrics) {
		t.Fatalf("expected metrics len %d, got %d", len(event.Metrics), len(got.Metrics))
	}
	for i := range event.Metrics {
		if got.Metrics[i] != event.Metrics[i] {
			t.Errorf("expected metric %q, got %q", event.Metrics[i], got.Metrics[i])
		}
	}
}

// Notify возвращает ошибку Observer-а, но продолжает рассылку остальным Observer-ам.
func TestAuditor_Notify_ReturnsErrorAndContinues(t *testing.T) {
	observerErr := errors.New("observer failed")

	firstObserver := &mockObserver{err: observerErr}
	secondObserver := &mockObserver{}

	auditor := NewAuditor()
	auditor.Subscribe(firstObserver)
	auditor.Subscribe(secondObserver)

	err := auditor.Notify(context.Background(), Event{})

	if !errors.Is(err, observerErr) {
		t.Fatalf("expected observer error, got %v", err)
	}

	if firstObserver.calls != 1 {
		t.Fatalf("expected first observer calls %d, got %d", 1, firstObserver.calls)
	}
	if secondObserver.calls != 1 {
		t.Fatalf("expected second observer calls %d, got %d", 1, secondObserver.calls)
	}
}

// Notify без подписанных Observer-ов завершается без ошибки.
func TestAuditor_Notify_WithoutObservers_OK(t *testing.T) {
	auditor := NewAuditor()

	if err := auditor.Notify(context.Background(), Event{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// Subscribe игнорирует nil Observer.
func TestAuditor_Subscribe_NilObserver_OK(t *testing.T) {
	auditor := NewAuditor()
	auditor.Subscribe(nil)

	if err := auditor.Notify(context.Background(), Event{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
