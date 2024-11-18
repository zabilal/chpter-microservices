package main

import (
    "context"
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/viper"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"

    pb "github.com/zabilal/microservices/pkg/genproto/order/v1"
    userpb "github.com/zabilal/microservices/pkg/genproto/user/v1"
    "github.com/zabilal/microservices/internal/pkg/config"
    "github.com/zabilal/microservices/internal/pkg/database"
    "github.com/zabilal/microservices/internal/pkg/logger"
    "github.com/zabilal/microservices/internal/pkg/metrics"
    "github.com/zabilal/microservices/internal/pkg/tracing"
    "github.com/zabilal/microservices/internal/order/repository"
    "github.com/zabilal/microservices/internal/order/service"
)

func main() {
    // Initialize context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Load configuration
    cfg, err := config.Load("config/order-service")
    if err != nil {
        fmt.Printf("Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Initialize logger
    log, err := logger.NewLogger(cfg.GetString("log.level"))
    if err != nil {
        fmt.Printf("Failed to initialize logger: %v\n", err)
        os.Exit(1)
    }
    defer log.Sync()

    // Initialize tracer
    tp, err := tracing.InitTracer(ctx, "order-service", cfg.GetString("jaeger.endpoint"))
    if err != nil {
        log.Fatal("Failed to initialize tracer", zap.Error(err))
    }
    defer tp.Shutdown(ctx)

    // Initialize metrics
    m := metrics.NewMetrics("order_service")
    go metrics.StartMetricsServer(ctx, cfg.GetString("metrics.address"))

    // Initialize database connection
    db, err := database.NewPostgresDB(database.Config{
        Host:     cfg.GetString("database.host"),
        Port:     cfg.GetInt("database.port"),
        Username: cfg.GetString("database.username"),
        Password: cfg.GetString("database.password"),
        DBName:   cfg.GetString("database.dbname"),
        SSLMode:  cfg.GetString("database.sslmode"),
    })
    if err != nil {
        log.Fatal("Failed to connect to database", zap.Error(err))
    }
    defer db.Close()

    // Initialize repository
    repo := repository.NewOrderRepository(db)

    // Initialize user service client
    userConn, err := grpc.Dial(
        cfg.GetString("user_service.address"),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(tracing.UnaryClientInterceptor()),
        grpc.WithStreamInterceptor(tracing.StreamClientInterceptor()),
    )
    if err != nil {
        log.Fatal("Failed to connect to user service", zap.Error(err))
    }
    defer userConn.Close()

    userClient := userpb.NewUserServiceClient(userConn)

    // Initialize service
    orderService := service.NewOrderService(repo, userClient, log)

    // Initialize gRPC server
    lis, err := net.Listen("tcp", cfg.GetString("server.address"))
    if err != nil {
        log.Fatal("Failed to listen", zap.Error(err))
    }

    // Create gRPC server with interceptors
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(tracing.UnaryServerInterceptor()),
        grpc.StreamInterceptor(tracing.StreamServerInterceptor()),
    )

    // Register services
    pb.RegisterOrderServiceServer(grpcServer, orderService)
    grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
    reflection.Register(grpcServer)

    // Start server
    go func() {
        log.Info("Starting gRPC server",
            zap.String("address", cfg.GetString("server.address")))
        if err := grpcServer.Serve(lis); err != nil {
            log.Fatal("Failed to serve", zap.Error(err))
        }
    }()

    // Wait for interrupt signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    // Graceful shutdown
    log.Info("Shutting down server...")
    grpcServer.GracefulStop()
    log.Info("Server stopped")
}