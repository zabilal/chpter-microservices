# Monitoring Package

This package provides centralized monitoring, logging, and tracing capabilities for the microservices architecture.

## Components

### Metrics

The metrics package provides Prometheus-based monitoring with the following features:
- Request duration tracking
- Request counting with status
- Active request tracking
- Database operation duration tracking

Usage:
```go
import "github.com/zabilal/microservices/monitoring/metrics"

// Start metrics server
metricsServer := metrics.NewMetricsServer(":9090", logger)
go metricsServer.Start()

// Record request metrics
defer metrics.RecordRequest("user-service", "CreateUser", time.Since(start), err)

// Track active requests
defer metrics.TrackActiveRequest("user-service")()

// Record database operation
defer metrics.RecordDatabaseOperation("user-service", "CreateUser", time.Since(start))
```

### Logger

The logger package provides structured logging using zap with the following features:
- Multiple log levels
- JSON and console output formats
- Custom field support
- Error tracking

Usage:
```go
import "github.com/zabilal/microservices/monitoring/logger"

// Create a new logger
log := logger.NewLogger("debug")

// Add fields
log = log.WithFields(
    zap.String("service", "user-service"),
    zap.String("version", "1.0.0"),
)

// Log messages
log.Info("Starting service")
log.Error("Failed to process request", zap.Error(err))
```

### Tracing

The tracing package provides distributed tracing using OpenTelemetry and Jaeger with the following features:
- Request tracing
- Error tracking
- Custom tags and events
- Service dependencies visualization

Usage:
```go
import "github.com/zabilal/microservices/monitoring/tracing"

// Initialize tracer
cfg := &tracing.Config{
    ServiceName:    "user-service",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    JaegerEndpoint: "http://jaeger:14268/api/traces",
}
cleanup, err := tracing.InitTracer(cfg)
defer cleanup(context.Background())

// Create spans
ctx, span := tracing.StartSpan(ctx, "CreateUser")
defer span.End()

// Add tags
tracing.AddSpanTags(ctx, map[string]string{
    "user_id": userID,
    "status": "success",
})

// Record errors
tracing.AddSpanError(ctx, err)
```

## Configuration

Each component can be configured through environment variables or configuration files. See individual package documentation for specific configuration options.

## Dependencies

- Prometheus client_golang
- Uber zap
- OpenTelemetry
- Jaeger client

## Best Practices

1. Always use structured logging with relevant context
2. Use appropriate log levels
3. Add meaningful metrics labels
4. Create spans for significant operations
5. Add relevant tags to spans
6. Handle cleanup properly
