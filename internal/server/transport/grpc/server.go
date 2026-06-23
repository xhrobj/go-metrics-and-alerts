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

type Service interface {
	UpdateMetrics(context.Context, []model.Metrics) error
}

type Auditor interface {
	Notify(context.Context, audit.Event) error
}

type Server struct {
	metricspb.UnimplementedMetricsServer

	service Service
	auditor Auditor
	log     *zap.Logger
}

func New(service Service) *Server {
	return &Server{
		service: service,
	}
}

// EnableAudit подключает аудит успешной обработки метрик.
func (s *Server) EnableAudit(auditor Auditor, log *zap.Logger) {
	s.auditor = auditor

	if log != nil {
		s.log = log
	}
}

func (s *Server) UpdateMetrics(
	ctx context.Context,
	request *metricspb.UpdateMetricsRequest,
) (*metricspb.UpdateMetricsResponse, error) {
	metrics, err := metricsFromProto(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.service.UpdateMetrics(ctx, metrics); err != nil {
		return nil, serviceError(err)
	}

	return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
}

func serviceError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, context.Canceled.Error())

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(
			codes.DeadlineExceeded,
			context.DeadlineExceeded.Error(),
		)

	default:
		return status.Error(codes.Internal, "update metrics failed")
	}
}
