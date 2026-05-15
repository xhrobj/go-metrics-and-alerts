package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type FileObserver struct {
	path string
	mu   sync.Mutex
}

func NewFileObserver(path string) *FileObserver {
	return &FileObserver{
		path: path,
	}
}

func (o *FileObserver) Notify(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// !!!: защитим параллельные записи
	o.mu.Lock()
	defer o.mu.Unlock()

	file, err := os.OpenFile(o.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(event); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}

	return nil
}
