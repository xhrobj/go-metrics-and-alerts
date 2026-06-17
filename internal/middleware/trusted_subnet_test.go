package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
)

func TestWithTrustedSubnet(t *testing.T) {
	_, trustedSubnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("net.ParseCIDR() error = %v", err)
	}

	tests := []struct {
		name       string
		subnet     *net.IPNet
		realIP     string
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "disabled without subnet",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "allows IP from trusted subnet",
			subnet:     trustedSubnet,
			realIP:     "192.168.1.42",
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
		{
			name:       "rejects missing IP",
			subnet:     trustedSubnet,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "rejects IP outside trusted subnet",
			subnet:     trustedSubnet,
			realIP:     "10.0.0.42",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			handler := WithTrustedSubnet(tt.subnet)(next)
			rq := httptest.NewRequest(http.MethodPost, "/updates", nil)
			if tt.realIP != "" {
				rq.Header.Set(protocol.HeaderRealIP, tt.realIP)
			}
			rs := httptest.NewRecorder()

			handler.ServeHTTP(rs, rq)

			if got, want := rs.Code, tt.wantStatus; got != want {
				t.Fatalf("status = %d, want %d", got, want)
			}
			if got, want := called, tt.wantCalled; got != want {
				t.Fatalf("next handler called = %t, want %t", got, want)
			}
		})
	}
}
