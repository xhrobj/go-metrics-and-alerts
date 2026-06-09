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
	path string
	mu   sync.Mutex
}

// NewFileObserver создаёт Observer для записи событий аудита в файл.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{
		path: path,
	}
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

	file, err := os.OpenFile(o.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}

	if err := json.NewEncoder(file).Encode(event); err != nil {
		_ = file.Close()
		return fmt.Errorf("write audit event: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close audit file: %w", err)
	}

	return nil
}
