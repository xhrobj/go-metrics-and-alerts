package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileObserver записывает события аудита в файл.
type FileObserver struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewFileObserver создаёт Observer для записи событий аудита в файл.
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}

	return &FileObserver{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// Notify записывает событие аудита отдельной JSON-строкой в конец файла.
func (o *FileObserver) Notify(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.file == nil || o.encoder == nil {
		return fmt.Errorf("audit file is closed")
	}

	if err := o.encoder.Encode(event); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}

	return nil
}

// Close закрывает файл аудита.
func (o *FileObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.file == nil {
		return nil
	}

	if err := o.file.Close(); err != nil {
		return fmt.Errorf("close audit file: %w", err)
	}

	o.file = nil
	o.encoder = nil

	return nil
}
