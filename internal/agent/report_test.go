package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption/testkeys"
	"github.com/xhrobj/go-metrics-and-alerts/internal/hash"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
)

// report() ставит задачу в очередь, а воркер отправляет POST с нужными заголовками.
func TestAgent_Report_SendsPOSTWithContentType(t *testing.T) {
	requests := 0
	done := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/updates" {
			t.Fatalf("expected path /updates, got %q", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}

		ce := r.Header.Get("Content-Encoding")
		if ce != "gzip" {
			t.Fatalf("expected Content-Encoding gzip, got %q", ce)
		}

		realIP := r.Header.Get(realIPHeader)
		if net.ParseIP(realIP) == nil {
			t.Fatalf("X-Real-IP = %q, want valid IP address", realIP)
		}

		w.WriteHeader(http.StatusOK)
		done <- struct{}{}
	}))
	defer server.Close()

	lg := zap.NewNop()
	cfg := config.AgentConfig{
		ServerAddr:          server.URL,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
		Key:                 "secret-key",
	}

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("failed to prepare test gauge metric: %v", err)
	}

	a, err := New(repo, cfg, lg)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	a.pollSinceReport.Store(1)
	a.report()

	var task reportTask
	select {
	case task = <-a.recvQueue:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for task in sendQueue")
	}

	if err := a.sendMetrics(context.Background(), task.metrics); err != nil {
		t.Fatalf("failed to send metrics: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for request")
	}

	if requests != 1 {
		t.Fatalf("expected 1 requests, got %d", requests)
	}
}

// report() ставит в очередь batch, а воркер отправляет корректные JSON-метрики на /updates.
func TestAgent_Report_SendsCorrectJSONMetrics(t *testing.T) {
	var metrics []model.Metrics
	done := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { done <- struct{}{} }()

		if r.URL.Path != "/updates" {
			t.Fatalf("expected path /updates, got %q", r.URL.Path)
		}

		if got := r.Header.Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("expected Content-Encoding gzip, got %q", got)
		}

		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer func() {
			_ = zr.Close()
		}()

		if err := json.NewDecoder(zr).Decode(&metrics); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	lg := zap.NewNop()
	cfg := config.AgentConfig{
		ServerAddr:          server.URL,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
		Key:                 "secret-key",
	}

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("failed to prepare test gauge metric: %v", err)
	}

	a, err := New(repo, cfg, lg)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	a.pollSinceReport.Store(3)
	a.report()

	var task reportTask
	select {
	case task = <-a.recvQueue:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for task in sendQueue")
	}

	if err := a.sendMetrics(context.Background(), task.metrics); err != nil {
		t.Fatalf("failed to send metrics: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for request")
	}

	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}

	var foundGauge bool
	var foundCounter bool

	for _, m := range metrics {
		switch {
		case m.ID == "Alloc" && m.MType == model.Gauge:
			foundGauge = true
			if m.Value == nil {
				t.Fatal("expected gauge value to be set")
			}
			if *m.Value != 5.11 {
				t.Fatalf("expected gauge value 5.11, got %v", *m.Value)
			}
			if m.Delta != nil {
				t.Fatal("expected gauge delta to be nil")
			}

		case m.ID == "PollCount" && m.MType == model.Counter:
			foundCounter = true
			if m.Delta == nil {
				t.Fatal("expected counter delta to be set")
			}
			if *m.Delta != 3 {
				t.Fatalf("expected counter delta 3, got %v", *m.Delta)
			}
			if m.Value != nil {
				t.Fatal("expected counter value to be nil")
			}
		}
	}

	if !foundGauge {
		t.Error("expected gauge metric Alloc to be sent")
	}

	if !foundCounter {
		t.Error("expected counter metric PollCount to be sent")
	}

	if a.pollSinceReport.Load() != 0 {
		t.Errorf("expected pollSinceReport to be reset to 0, got %d", a.pollSinceReport.Load())
	}
}

// TestAgent_SendMetrics_EncryptsBody проверяет, что при заданном публичном ключе
// Агент шифрует gzip-body и вычисляет хеш от отправляемых зашифрованных байтов.
func TestAgent_SendMetrics_EncryptsBody(t *testing.T) {
	const hashKey = "secret-key"

	pair := testkeys.Generate(t)

	type capturedRequest struct {
		body              []byte
		hash              string
		contentEncoding   string
		contentEncryption string
	}

	requestCh := make(chan capturedRequest, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		requestCh <- capturedRequest{
			body:              body,
			hash:              r.Header.Get("HashSHA256"),
			contentEncoding:   r.Header.Get("Content-Encoding"),
			contentEncryption: r.Header.Get(encryption.HeaderContentEncryption),
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.AgentConfig{
		ServerAddr:          server.URL,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
		Key:                 hashKey,
		CryptoKey:           pair.PublicKeyPath,
	}

	a, err := New(repository.NewMemStorage(), cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	value := 5.11
	delta := int64(42)

	want := []model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &delta,
		},
	}

	if err := a.sendMetrics(context.Background(), want); err != nil {
		t.Fatalf("sendMetrics() error = %v", err)
	}

	var gotRequest capturedRequest
	select {
	case gotRequest = <-requestCh:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for encrypted request")
	}

	if got, want := gotRequest.contentEncoding, "gzip"; got != want {
		t.Fatalf("Content-Encoding = %q, want %q", got, want)
	}

	if got, want := gotRequest.contentEncryption,
		encryption.SchemeRSAOAEPWithAESGCM; got != want {
		t.Fatalf("Content-Encryption = %q, want %q", got, want)
	}

	wantHash := hash.CalcHash(gotRequest.body, hashKey)
	if gotRequest.hash != wantHash {
		t.Fatalf("HashSHA256 = %q, want %q", gotRequest.hash, wantHash)
	}

	privateKey, err := encryption.LoadPrivateKey(pair.PrivateKeyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	compressedBody, err := encryption.Decrypt(gotRequest.body, privateKey)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	zr, err := gzip.NewReader(bytes.NewReader(compressedBody))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() {
		_ = zr.Close()
	}()

	var got []model.Metrics
	if err := json.NewDecoder(zr).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sendMetrics() metrics = %+v, want %+v", got, want)
	}
}

func TestPrepareRequestBody(t *testing.T) {
	value := 5.11
	delta := int64(42)

	want := []model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &delta,
		},
	}

	body, err := prepareRequestBody(want)
	if err != nil {
		t.Fatalf("prepareRequestBody() error = %v", err)
	}

	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() {
		_ = zr.Close()
	}()

	var got []model.Metrics
	if err := json.NewDecoder(zr).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prepareRequestBody() = %+v, want %+v", got, want)
	}
}
