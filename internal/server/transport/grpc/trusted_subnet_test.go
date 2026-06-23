package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTrustedSubnetInterceptor(t *testing.T) {
	_, trustedSubnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	tests := []struct {
		name        string
		subnet      *net.IPNet
		hasMetadata bool
		realIP      string
		wantCode    codes.Code
		wantCalled  bool
	}{
		{
			name:       "disabled without subnet",
			wantCode:   codes.OK,
			wantCalled: true,
		},
		{
			name:        "allows IP from trusted subnet",
			subnet:      trustedSubnet,
			hasMetadata: true,
			realIP:      "192.168.1.42",
			wantCode:    codes.OK,
			wantCalled:  true,
		},
		{
			name:     "rejects missing metadata",
			subnet:   trustedSubnet,
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "rejects empty IP",
			subnet:      trustedSubnet,
			hasMetadata: true,
			wantCode:    codes.PermissionDenied,
		},
		{
			name:        "rejects invalid IP",
			subnet:      trustedSubnet,
			hasMetadata: true,
			realIP:      "not-an-ip",
			wantCode:    codes.PermissionDenied,
		},
		{
			name:        "rejects IP outside trusted subnet",
			subnet:      trustedSubnet,
			hasMetadata: true,
			realIP:      "10.0.0.42",
			wantCode:    codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.hasMetadata {
				ctx = metadata.NewIncomingContext(
					ctx,
					metadata.Pairs(protocol.MetadataRealIP, tt.realIP),
				)
			}

			called := false

			handler := func(context.Context, any) (any, error) {
				called = true

				return "ok", nil
			}

			interceptor := TrustedSubnetInterceptor(tt.subnet)

			rq := struct{}{}
			rs, err := interceptor(
				ctx,
				rq,
				&grpc.UnaryServerInfo{},
				handler,
			)

			gotCode := status.Code(err)
			require.Equal(
				t,
				tt.wantCode,
				gotCode,
				"got code %s, want %s",
				gotCode,
				tt.wantCode,
			)

			require.Equal(
				t,
				tt.wantCalled,
				called,
				"got called %t, want %t",
				called,
				tt.wantCalled,
			)

			if tt.wantCalled {
				require.Equal(t, "ok", rs)
			} else {
				require.Nil(t, rs)
			}
		})
	}
}
