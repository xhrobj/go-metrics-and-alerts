package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TrustedSubnetInterceptor(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		rq any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if subnet == nil {
			return handler(ctx, rq)
		}

		realIP, ok := realIPFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}

		ip := net.ParseIP(realIP)
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}

		return handler(ctx, rq)
	}
}
