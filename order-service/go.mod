module github.com/zabilal/microservices/order-service

go 1.21

require (
    google.golang.org/grpc v1.59.0
    google.golang.org/protobuf v1.31.0
    go.uber.org/zap v1.26.0
    github.com/spf13/viper v1.18.0
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/exporters/jaeger v1.21.0
    github.com/lib/pq v1.10.9
    github.com/go-sql-driver/mysql v1.7.1
)
