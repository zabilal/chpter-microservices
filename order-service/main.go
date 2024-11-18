package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/zabilal/microservices/monitoring/logger"
	"github.com/zabilal/microservices/monitoring/metrics"
	"github.com/zabilal/microservices/monitoring/tracing"
	"github.com/zabilal/microservices/order-service/handler"
)

type Config struct {
	ServerAddress    string
	MetricsAddress   string
	Environment      string
	JaegerEndpoint   string
	UserServiceAddr  string
	DatabaseConfig   DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
}

func main() {
	// Initialize logger
	log := logger.NewLogger("debug")
	log = log.WithService("order-service")

	// Load configuration
	cfg := loadConfig()

	// Initialize tracer
	tracingCfg := &tracing.Config{
		ServiceName:    "order-service",
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

	// Initialize user service client
	userConn, err := grpc.Dial(cfg.UserServiceAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal("Failed to connect to user service", zap.Error(err))
	}
	defer userConn.Close()

	// Initialize repository
	repo := handler.NewOrderRepository(cfg.DatabaseConfig)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", cfg.ServerAddress)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	server := grpc.NewServer()
	orderHandler := handler.NewOrderHandler(repo, userConn, log)
	handler.RegisterOrderServiceServer(server, orderHandler)

	// Start server
	go func() {
		log.Info("Starting gRPC server", zap.String("address", cfg.ServerAddress))
		if err := server.Serve(lis); err != nil {
			log.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Info("Shutting down server...")
	server.GracefulStop()
	if err := metricsServer.Stop(context.Background()); err != nil {
		log.Error("Failed to stop metrics server", zap.Error(err))
	}
}

func loadConfig() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/microservices")
	
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error loading config file: %w", err))
	}

	return &Config{
		ServerAddress:   viper.GetString("server.address"),
		MetricsAddress:  viper.GetString("metrics.address"),
		Environment:     viper.GetString("environment"),
		JaegerEndpoint:  viper.GetString("jaeger.endpoint"),
		UserServiceAddr: viper.GetString("services.user.address"),
		DatabaseConfig: DatabaseConfig{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetInt("database.port"),
			Username: viper.GetString("database.username"),
			Password: viper.GetString("database.password"),
			DBName:   viper.GetString("database.dbname"),
		},
	}
}
