package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultAuditTimeout = time.Second * 5

// RemoteObserver отправляет события аудита на удалённый HTTP-приёмник.
type RemoteObserver struct {
	url    string
	client *http.Client
}

// NewRemoteObserver создаёт Observer для отправки событий аудита по HTTP.
func NewRemoteObserver(url string) *RemoteObserver {
	return &RemoteObserver{
		url: url,
		client: &http.Client{
			Timeout: defaultAuditTimeout,
		},
	}
}

// Notify отправляет событие аудита POST-запросом в формате JSON.
//
// Успешной считается отправка, при которой удалённый приёмник вернул 2xx-статус.
// HTTP-клиент использует внутренний таймаут defaultAuditTimeout.
// Контекст используется для отмены запроса, например при остановке Сервера
// или отмене входящего HTTP-запроса.
func (o *RemoteObserver) Notify(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(event); err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}

	rq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, &body)
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}

	rq.Header.Set("Content-Type", "application/json")

	rs, err := o.client.Do(rq)
	if err != nil {
		return fmt.Errorf("send audit event: %w", err)
	}
	defer rs.Body.Close()

	_, _ = io.Copy(io.Discard, rs.Body)

	if rs.StatusCode < http.StatusOK || rs.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("send audit event: unexpected status %d", rs.StatusCode)
	}

	return nil
}
