package grpcserver

import (
	"context"
	"errors"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsUpdater описывает бизнес-логику обновления метрик,
// используемую gRPC-Сервером.
type MetricsUpdater interface {
	// UpdateMetrics сохраняет набор метрик за одну операцию.
	UpdateMetrics(context.Context, []model.Metrics) error
}

// Auditor описывает диспетчер событий аудита.
type Auditor interface {
	// Notify обрабатывает событие аудита.
	Notify(context.Context, audit.Event) error
}

// Server реализует gRPC-сервис Metrics.
type Server struct {
	metricspb.UnimplementedMetricsServer

	service MetricsUpdater
	auditor Auditor
	log     *zap.Logger
}

var _ metricspb.MetricsServer = (*Server)(nil)

// New создаёт gRPC-Сервер, использующий переданный сервис метрик.
func New(service MetricsUpdater, log *zap.Logger) *Server {
	if log == nil {
		log = zap.NewNop()
	}

	return &Server{
		service: service,
		log:     log,
	}
}

// EnableAudit подключает аудит успешной обработки метрик.
func (s *Server) EnableAudit(auditor Auditor) {
	s.auditor = auditor
}

// UpdateMetrics принимает и сохраняет батч метрик.
func (s *Server) UpdateMetrics(
	ctx context.Context,
	rq *metricspb.UpdateMetricsRequest,
) (*metricspb.UpdateMetricsResponse, error) {
	metrics, err := metricsFromProto(rq)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.service.UpdateMetrics(ctx, metrics); err != nil {
		grpcErr := serviceError(err)
		fields := []zap.Field{zap.Error(err)}

		if isServerErrorCode(status.Code(grpcErr)) {
			s.log.Error("update metrics failed", fields...)
		} else {
			s.log.Debug("update metrics failed", fields...)
		}

		return nil, grpcErr
	}

	s.notifyAudit(ctx, metrics)

	return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
}

func serviceError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, context.Canceled.Error())

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())

	default:
		return status.Error(codes.Internal, "update metrics failed")
	}
}
