package grpcserver

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor создаёт unary interceptor для логирования gRPC-запросов и ответов.
//
// Interceptor фиксирует:
//   - полное имя gRPC-метода
//   - время выполнения запроса
//   - итоговый gRPC-статус
//
// Логирование выполняется с использованием zap.Logger на уровне Info.
func LoggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
	if log == nil {
		log = zap.NewNop()
	}

	return func(
		ctx context.Context,
		rq any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		rs, err := handler(ctx, rq)

		log.Info("grpc request completed",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", status.Code(err).String()),
		)

		return rs, err
	}
}
