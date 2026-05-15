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

type RemoteObserver struct {
	url    string
	client *http.Client
}

func NewRemoteObserver(url string) *RemoteObserver {
	return &RemoteObserver{
		url: url,
		client: &http.Client{
			Timeout: defaultAuditTimeout,
		},
	}
}

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
