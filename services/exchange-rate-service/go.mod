module github.com/patteeraL/movra/services/exchange-rate-service

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.5.0
	github.com/prometheus/client_golang v1.18.0
	github.com/redis/go-redis/v9 v9.3.0
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.uber.org/zap v1.26.0
	google.golang.org/grpc v1.60.0
	google.golang.org/protobuf v1.31.0
)
