package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLoggingUsesLevelByStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantLevel  zapcore.Level
	}{
		{
			name:       "successful response",
			statusCode: http.StatusOK,
			body:       "ok",
			wantLevel:  zap.DebugLevel,
		},
		{
			name:       "client error",
			statusCode: http.StatusBadRequest,
			body:       "bad request",
			wantLevel:  zap.DebugLevel,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       "internal error",
			wantLevel:  zap.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, observedLogs := observer.New(zap.DebugLevel)
			log := zap.New(core)

			handler := WithLogging(log)(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.statusCode)
					_, err := w.Write([]byte(tt.body))
					require.NoError(t, err)
				},
			))

			rq := httptest.NewRequest(http.MethodPost, "/update", nil)
			rs := httptest.NewRecorder()

			handler.ServeHTTP(rs, rq)

			entries := observedLogs.FilterMessage("http request completed").All()
			require.Len(t, entries, 1)
			require.Equal(t, tt.wantLevel, entries[0].Level)

			fields := entries[0].ContextMap()
			require.Equal(t, "/update", fields["uri"])
			require.Equal(t, http.MethodPost, fields["method"])
			require.EqualValues(t, tt.statusCode, fields["status"])
			require.EqualValues(t, len(tt.body), fields["size"])
			require.Contains(t, fields, "duration")
		})
	}
}
