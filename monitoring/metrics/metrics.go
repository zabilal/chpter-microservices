package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "request_duration_seconds",
		Help: "Time spent processing request",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"service", "method"})

	requestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "request_total",
		Help: "Total number of requests",
	}, []string{"service", "method", "status"})

	activeRequests = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "active_requests",
		Help: "Number of requests currently being processed",
	}, []string{"service"})

	databaseOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "database_operation_duration_seconds",
		Help: "Time spent performing database operations",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
	}, []string{"service", "operation"})
)

type MetricsServer struct {
	server *http.Server
	logger *zap.Logger
}

func NewMetricsServer(address string, logger *zap.Logger) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &MetricsServer{
		server: &http.Server{
			Addr:    address,
			Handler: mux,
		},
		logger: logger,
	}
}

func (s *MetricsServer) Start() error {
	s.logger.Info("Starting metrics server", zap.String("address", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *MetricsServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping metrics server")
	return s.server.Shutdown(ctx)
}

// RecordRequest records request duration and updates request counters
func RecordRequest(service, method string, duration time.Duration, err error) {
	requestDuration.WithLabelValues(service, method).Observe(duration.Seconds())
	
	status := "success"
	if err != nil {
		status = "error"
	}
	requestCounter.WithLabelValues(service, method, status).Inc()
}

// TrackActiveRequest tracks active requests for a service
func TrackActiveRequest(service string) func() {
	activeRequests.WithLabelValues(service).Inc()
	return func() {
		activeRequests.WithLabelValues(service).Dec()
	}
}

// RecordDatabaseOperation records database operation duration
func RecordDatabaseOperation(service, operation string, duration time.Duration) {
	databaseOperationDuration.WithLabelValues(service, operation).Observe(duration.Seconds())
}
