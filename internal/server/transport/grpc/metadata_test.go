package grpcserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"google.golang.org/grpc/metadata"
)

func TestRealIPFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		wantIP string
		wantOK bool
	}{
		{
			name: "without metadata",
			ctx:  context.Background(),
		},
		{
			name: "without real IP",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs("other", "value"),
			),
		},
		{
			name: "empty real IP",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(protocol.MetadataRealIP, ""),
			),
		},
		{
			name: "returns real IP",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(protocol.MetadataRealIP, "192.168.1.42"),
			),
			wantIP: "192.168.1.42",
			wantOK: true,
		},
		{
			name: "returns first real IP",
			ctx: metadata.NewIncomingContext(
				context.Background(),
				metadata.Pairs(
					protocol.MetadataRealIP, "192.168.1.42",
					protocol.MetadataRealIP, "10.0.0.42",
				),
			),
			wantIP: "192.168.1.42",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotOK := realIPFromContext(tt.ctx)

			require.Equal(t, tt.wantIP, gotIP, "got IP %q, want %q", gotIP, tt.wantIP)
			require.Equal(t, tt.wantOK, gotOK, "got ok %t, want %t", gotOK, tt.wantOK)
		})
	}
}
