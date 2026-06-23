package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	grpcserver "github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/grpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func newGRPCServer(
	metricsService grpcserver.Service,
	auditor *audit.Auditor,
	trustedSubnet *net.IPNet,
	log *zap.Logger,
) *grpc.Server {
	transport := grpcserver.New(metricsService)
	if auditor != nil {
		transport.EnableAudit(auditor, log)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpcserver.TrustedSubnetInterceptor(trustedSubnet),
		),
	)
	metricspb.RegisterMetricsServer(srv, transport)

	return srv
}

func serveGRPC(
	ctx context.Context,
	srv *grpc.Server,
	listener net.Listener,
	log *zap.Logger,
) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}

		return err

	case <-ctx.Done():
		log.Info("shutdown signal received", zap.String("transport", "gRPC"))
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		shutdownTimeout,
	)
	defer cancel()

	if err := stopGRPC(shutdownCtx, srv); err != nil {
		return err
	}

	if err := <-errCh; err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}

	return nil
}

func stopGRPC(ctx context.Context, srv *grpc.Server) error {
	doneCh := make(chan struct{})

	go func() {
		srv.GracefulStop()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		return nil

	case <-ctx.Done():
		select {
		case <-doneCh:
			return nil
		default:
		}

		srv.Stop()
		<-doneCh

		return fmt.Errorf("shutdown gRPC server: %w", ctx.Err())
	}
}
