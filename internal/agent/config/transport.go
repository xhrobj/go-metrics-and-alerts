package config

// Transport задаёт протокол отправки метрик Агентом.
type Transport string

const (
	// TransportGRPC отправляет метрики через gRPC.
	TransportGRPC Transport = "grpc"

	// TransportHTTP отправляет метрики через HTTP.
	TransportHTTP Transport = "http"
)
