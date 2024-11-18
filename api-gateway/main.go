package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/zabilal/microservices/api-gateway/handler"
	"github.com/zabilal/microservices/monitoring/logger"
	"github.com/zabilal/microservices/monitoring/metrics"
	"github.com/zabilal/microservices/monitoring/tracing"
)

func main() {
	// Initialize logger
	log := logger.NewLogger("debug")
	log = log.WithService("api-gateway")

	// Load configuration
	cfg := loadConfig()

	// Initialize tracer
	tracingCfg := &tracing.Config{
		ServiceName:    "api-gateway",
		ServiceVersion: "1.0.0",
		Environment:    cfg.Environment,
		JaegerEndpoint: cfg.JaegerEndpoint,
	}
	cleanup, err := tracing.InitTracer(tracingCfg)
	if err != nil {
		log.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer cleanup(context.Background())

	// Initialize metrics server
	metricsServer := metrics.NewMetricsServer(cfg.MetricsAddress, log)
	go func() {
		if err := metricsServer.Start(); err != nil {
			log.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Initialize gateway
	gateway, err := handler.NewGateway(cfg.Gateway, log)
	if err != nil {
		log.Fatal("Failed to create gateway", zap.Error(err))
	}
	defer gateway.Close()

	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start gateway
	if err := gateway.Run(ctx); err != nil {
		log.Fatal("Failed to run gateway", zap.Error(err))
	}

	// Wait for interrupt signal
	<-ctx.Done()

	// Shutdown gracefully
	log.Info("Shutting down gateway...")
	if err := metricsServer.Stop(context.Background()); err != nil {
		log.Error("Failed to stop metrics server", zap.Error(err))
	}
}

type Config struct {
	Environment     string
	MetricsAddress  string
	JaegerEndpoint  string
	Gateway         *handler.Config
}

func loadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/microservices")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error loading config file: %w", err))
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("fatal error unmarshaling config: %w", err))
	}

	return &cfg
}