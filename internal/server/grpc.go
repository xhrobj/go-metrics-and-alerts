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
	"google.golang.org/grpc/credentials"
)

func newGRPCServer(
	metricsService grpcserver.MetricsUpdater,
	auditor *audit.Auditor,
	trustedSubnet *net.IPNet,
	tlsCertPath string,
	tlsKeyPath string,
	log *zap.Logger,
) (*grpc.Server, error) {
	hasCert := tlsCertPath != ""
	hasKey := tlsKeyPath != ""

	if hasCert != hasKey {
		return nil, fmt.Errorf(
			"gRPC TLS certificate and private key paths must be specified together",
		)
	}

	transport := grpcserver.New(metricsService, log)

	if auditor != nil {
		transport.EnableAudit(auditor)
	}

	interceptors := []grpc.UnaryServerInterceptor{
		grpcserver.LoggingInterceptor(log),
	}

	if trustedSubnet != nil {
		interceptors = append(
			interceptors,
			grpcserver.TrustedSubnetInterceptor(trustedSubnet),
		)
	}

	options := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(interceptors...),
	}

	if hasCert {
		transportCredentials, err := credentials.NewServerTLSFromFile(
			tlsCertPath,
			tlsKeyPath,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"load gRPC TLS certificate %q and private key %q: %w",
				tlsCertPath,
				tlsKeyPath,
				err,
			)
		}

		options = append(options, grpc.Creds(transportCredentials))
	}

	srv := grpc.NewServer(options...)

	metricspb.RegisterMetricsServer(srv, transport)

	return srv, nil
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
