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
	_, trustedIPv4, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	_, trustedIPv6, err := net.ParseCIDR("2001:db8::/32")
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
			name:        "allows IPv4 from trusted subnet",
			subnet:      trustedIPv4,
			hasMetadata: true,
			realIP:      "192.168.1.42",
			wantCode:    codes.OK,
			wantCalled:  true,
		},
		{
			name:        "allows IPv6 from trusted subnet",
			subnet:      trustedIPv6,
			hasMetadata: true,
			realIP:      "2001:db8::42",
			wantCode:    codes.OK,
			wantCalled:  true,
		},
		{
			name:     "rejects missing metadata",
			subnet:   trustedIPv4,
			wantCode: codes.PermissionDenied,
		},
		{
			name:        "rejects empty IP",
			subnet:      trustedIPv4,
			hasMetadata: true,
			wantCode:    codes.PermissionDenied,
		},
		{
			name:        "rejects invalid IP",
			subnet:      trustedIPv4,
			hasMetadata: true,
			realIP:      "not-an-ip",
			wantCode:    codes.PermissionDenied,
		},
		{
			name:        "rejects IP outside trusted subnet",
			subnet:      trustedIPv4,
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
			handler := func(_ context.Context, rq any) (any, error) {
				called = true
				return rq, nil
			}

			interceptor := TrustedSubnetInterceptor(tt.subnet)
			rq := "request"

			rs, err := interceptor(ctx, rq, &grpc.UnaryServerInfo{}, handler)
			gotCode := status.Code(err)

			require.Equal(t, tt.wantCode, gotCode, "got code %s, want %s", gotCode, tt.wantCode)
			require.Equal(t, tt.wantCalled, called, "got called %t, want %t", called, tt.wantCalled)

			if tt.wantCalled {
				require.Equal(t, rq, rs)
			} else {
				require.Nil(t, rs)
				require.Equal(t, "access denied", status.Convert(err).Message())
			}
		})
	}
}

func TestTrustedSubnetInterceptorPreservesHandlerError(t *testing.T) {
	_, trustedSubnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(protocol.MetadataRealIP, "192.168.1.42"),
	)

	wantErr := status.Error(codes.Internal, "update failed")
	handler := func(context.Context, any) (any, error) {
		return nil, wantErr
	}

	interceptor := TrustedSubnetInterceptor(trustedSubnet)

	rs, err := interceptor(ctx, struct{}{}, &grpc.UnaryServerInfo{}, handler)

	require.Nil(t, rs)
	require.ErrorIs(t, err, wantErr)
}
