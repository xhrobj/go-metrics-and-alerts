package repository_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

type snapshotterFunc func(context.Context) (map[string]float64, map[string]int64, error)

func TestFileStore_SaveAndLoad(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metrics.json")

	source := repository.NewMemStorage()

	if err := source.UpdateGauge(ctx, "TestGauge", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	if err := source.UpdateCounter(ctx, "PollCount", 42); err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	store := repository.NewFileStore(path)

	if err := store.Save(source); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	target := repository.NewMemStorage()

	if err := store.Load(target); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	gotGauge, err := target.GetGauge(ctx, "TestGauge")
	if err != nil {
		t.Fatalf("GetGauge() error = %v", err)
	}

	if wantGauge := 5.11; gotGauge != wantGauge {
		t.Fatalf("GetGauge() got = %v, want %v", gotGauge, wantGauge)
	}

	gotCounter, err := target.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("GetCounter() error = %v", err)
	}

	if wantCounter := int64(42); gotCounter != wantCounter {
		t.Fatalf("GetCounter() got = %v, want %v", gotCounter, wantCounter)
	}
}

func TestFileStore_LoadMissingFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "missing.json")

	store := repository.NewFileStore(path)
	target := repository.NewMemStorage()

	if err := store.Load(target); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	_, err := target.GetGauge(ctx, "TestGauge")
	if !errors.Is(err, repository.ErrMetricNotFound) {
		t.Fatalf("GetGauge() error = %v, want %v", err, repository.ErrMetricNotFound)
	}
}

func TestFileStore_LoadEmptyFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metrics.json")

	if err := os.WriteFile(path, nil, 0o666); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := repository.NewFileStore(path)
	target := repository.NewMemStorage()

	if err := store.Load(target); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	_, err := target.GetCounter(ctx, "PollCount")
	if !errors.Is(err, repository.ErrMetricNotFound) {
		t.Fatalf("GetCounter() error = %v, want %v", err, repository.ErrMetricNotFound)
	}
}

func TestFileStore_LoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	if err := os.WriteFile(path, []byte("{"), 0o666); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := repository.NewFileStore(path)
	target := repository.NewMemStorage()

	if err := store.Load(target); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestFileStore_SaveSnapshotError(t *testing.T) {
	wantErr := errors.New("snapshot failed")
	store := repository.NewFileStore(filepath.Join(t.TempDir(), "metrics.json"))

	err := store.Save(snapshotterFunc(func(context.Context) (map[string]float64, map[string]int64, error) {
		return nil, nil, wantErr
	}))

	if !errors.Is(err, wantErr) {
		t.Fatalf("Save() error = %v, want %v", err, wantErr)
	}
}

func (f snapshotterFunc) Snapshot(ctx context.Context) (map[string]float64, map[string]int64, error) {
	return f(ctx)
}
