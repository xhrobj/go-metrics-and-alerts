package grpcserver

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor создаёт unary interceptor для логирования gRPC-запросов и ответов.
//
// Interceptor фиксирует:
//   - полное имя gRPC-метода
//   - время выполнения запроса
//   - итоговый gRPC-статус
//
// Успешные запросы и клиентские ошибки логируются на уровне Debug,
// серверные ошибки — на уровне Error.
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
		code := status.Code(err)
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", code.String()),
		}

		if isServerErrorCode(code) {
			log.Error("grpc request completed", fields...)
		} else {
			log.Debug("grpc request completed", fields...)
		}

		return rs, err
	}
}

func isServerErrorCode(code codes.Code) bool {
	switch code {
	case codes.Unknown,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DataLoss:
		return true

	default:
		return false
	}
}
