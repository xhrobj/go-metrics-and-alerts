package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestWaitRetry(t *testing.T) {
	err := waitRetry(context.Background(), time.Millisecond)
	if err != nil {
		t.Fatalf("waitRetry() error = %v", err)
	}
}

func TestWaitRetryContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitRetry(ctx, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitRetry() error = %v, want %v", err, context.Canceled)
	}
}

func TestWaitRetryDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithDeadline(
		context.Background(),
		time.Now().Add(-time.Second),
	)
	defer cancel()

	err := waitRetry(ctx, time.Hour)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf(
			"waitRetry() error = %v, want %v",
			err,
			context.DeadlineExceeded,
		)
	}
}

func TestRetryDBOperationSuccess(t *testing.T) {
	calls := 0

	err := retryDBOperation(context.Background(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("retryDBOperation() error = %v", err)
	}

	gotCalls := calls
	wantCalls := 1
	if gotCalls != wantCalls {
		t.Fatalf(
			"retryDBOperation() calls = %d, want %d",
			gotCalls,
			wantCalls,
		)
	}
}

func TestRetryDBOperationNonRetriableError(t *testing.T) {
	wantErr := errors.New("database error")
	calls := 0

	err := retryDBOperation(context.Background(), func() error {
		calls++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("retryDBOperation() error = %v, want %v", err, wantErr)
	}

	gotCalls := calls
	wantCalls := 1
	if gotCalls != wantCalls {
		t.Fatalf(
			"retryDBOperation() calls = %d, want %d",
			gotCalls,
			wantCalls,
		)
	}
}

func TestRetryDBOperationContextCanceledBeforeAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0

	err := retryDBOperation(ctx, func() error {
		calls++
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"retryDBOperation() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	gotCalls := calls
	wantCalls := 0
	if gotCalls != wantCalls {
		t.Fatalf(
			"retryDBOperation() calls = %d, want %d",
			gotCalls,
			wantCalls,
		)
	}
}

func TestRetryDBOperationContextCanceledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	err := retryDBOperation(ctx, func() error {
		calls++
		cancel()

		return &pgconn.PgError{
			Code: pgerrcode.ConnectionException,
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"retryDBOperation() error = %v, want %v",
			err,
			context.Canceled,
		)
	}

	gotCalls := calls
	wantCalls := 1
	if gotCalls != wantCalls {
		t.Fatalf(
			"retryDBOperation() calls = %d, want %d",
			gotCalls,
			wantCalls,
		)
	}
}

func TestIsRetriablePGError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection exception",
			err: &pgconn.PgError{
				Code: pgerrcode.ConnectionException,
			},
			want: true,
		},
		{
			name: "wrapped connection exception",
			err: fmt.Errorf(
				"query failed: %w",
				&pgconn.PgError{
					Code: pgerrcode.ConnectionException,
				},
			),
			want: true,
		},
		{
			name: "non-retriable PostgreSQL error",
			err: &pgconn.PgError{
				Code: pgerrcode.UniqueViolation,
			},
			want: false,
		},
		{
			name: "ordinary error",
			err:  errors.New("database error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetriablePGError(tt.err)
			if got != tt.want {
				t.Fatalf(
					"isRetriablePGError() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}
