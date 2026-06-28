package grpcserver

import (
	"context"

	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"google.golang.org/grpc/metadata"
)

func realIPFromContext(ctx context.Context) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(protocol.MetadataRealIP)
	if len(values) == 0 || values[0] == "" {
		return "", false
	}

	return values[0], true
}
