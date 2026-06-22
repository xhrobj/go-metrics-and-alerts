package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
	"github.com/xhrobj/go-metrics-and-alerts/internal/hash"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
)

// HTTPSender отправляет batch метрик на Сервер через HTTP.
type HTTPSender struct {
	baseURL   string
	hashKey   string
	publicKey *rsa.PublicKey
	client    *resty.Client
}

var _ MetricsSender = (*HTTPSender)(nil)

// NewHTTPSender создаёт HTTP-отправитель метрик.
func NewHTTPSender(serverAddr, hashKey, cryptoKey string) (*HTTPSender, error) {
	var publicKey *rsa.PublicKey
	if cryptoKey != "" {
		loadedPublicKey, err := encryption.LoadPublicKey(cryptoKey)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}

		publicKey = loadedPublicKey
	}

	baseURL := serverAddr
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	return &HTTPSender{
		baseURL:   baseURL,
		hashKey:   hashKey,
		publicKey: publicKey,
		client:    resty.New(),
	}, nil
}

// Send отправляет batch метрик на HTTP endpoint /updates.
func (s *HTTPSender) Send(ctx context.Context, metrics []model.Metrics) error {
	body, err := prepareRequestBody(metrics)
	if err != nil {
		return err
	}

	if s.publicKey != nil {
		body, err = encryption.Encrypt(body, s.publicKey)
		if err != nil {
			return fmt.Errorf("encrypt metrics batch: %w", err)
		}
	}

	// Хеш вычисляется от тех же байтов, что будут отправлены Серверу.
	hashValue := hash.CalcHash(body, s.hashKey)

	if err := s.postWithRetry(ctx, "/updates", body, hashValue); err != nil {
		return fmt.Errorf("send metrics batch: %w", err)
	}

	return nil
}

func prepareRequestBody(metrics []model.Metrics) ([]byte, error) {
	body, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("marshal metrics batch: %w", err)
	}

	compressedBody, err := gzipCompress(body)
	if err != nil {
		return nil, fmt.Errorf("gzip compress metrics batch: %w", err)
	}

	return compressedBody, nil
}

func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *HTTPSender) postWithRetry(
	ctx context.Context,
	path string,
	body []byte,
	hashValue string,
) error {
	realIP, err := localIP()
	if err != nil {
		return fmt.Errorf("get local IP: %w", err)
	}

	retryDelays := []time.Duration{
		time.Second * 1,
		time.Second * 3,
		time.Second * 5,
	}

	var lastErr error

	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		rq := s.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader(protocol.HeaderRealIP, realIP).
			SetBody(body)

		if s.publicKey != nil {
			rq.SetHeader(encryption.HeaderContentEncryption, encryption.SchemeRSAOAEPWithAESGCM)
		}

		if hashValue != "" {
			rq.SetHeader("HashSHA256", hashValue)
		}

		rs, err := rq.Post(s.baseURL + path)
		if err == nil {
			if rs.StatusCode() == http.StatusOK {
				return nil
			}

			lastErr = fmt.Errorf("unexpected status code: %d", rs.StatusCode())
			if !isRetriableStatusCode(rs.StatusCode()) || attempt >= len(retryDelays) {
				return lastErr
			}

			if err := waitRetry(ctx, retryDelays[attempt]); err != nil {
				return err
			}
			continue
		}

		lastErr = err
		if !isRetriableHTTPError(err) || attempt >= len(retryDelays) {
			return lastErr
		}

		if err := waitRetry(ctx, retryDelays[attempt]); err != nil {
			return err
		}
	}

	return lastErr
}

func localIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}

		ip := ipNet.IP.To4()
		if ip != nil {
			return ip.String(), nil
		}
	}

	return "", errors.New("local IPv4 address not found")
}

func isRetriableStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode < 600)
}

func isRetriableHTTPError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
