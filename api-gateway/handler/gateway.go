package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/zabilal/microservices/monitoring/logger"
)

type Gateway struct {
	config     *Config
	logger     *logger.Logger
	limiter    *rate.Limiter
	userConn   *grpc.ClientConn
	orderConn  *grpc.ClientConn
}

type Config struct {
	Listen struct {
		Address string
	}
	Server struct {
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
	}
	Services struct {
		UserService struct {
			Endpoint string
		}
		OrderService struct {
			Endpoint string
		}
	}
	CORS struct {
		AllowedOrigins []string
	}
	RateLimit struct {
		Burst int
	}
}

func NewGateway(config *Config, logger *logger.Logger) (*Gateway, error) {
	// Initialize gRPC connections
	userConn, err := grpc.Dial(config.Services.UserService.Endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	orderConn, err := grpc.Dial(config.Services.OrderService.Endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Gateway{
		config:     config,
		logger:     logger,
		limiter:    rate.NewLimiter(rate.Every(time.Second), config.RateLimit.Burst),
		userConn:   userConn,
		orderConn:  orderConn,
	}, nil
}

func (g *Gateway) Run(ctx context.Context) error {
	// Initialize Gin router
	router := gin.Default()

	// Add middleware
	router.Use(g.corsMiddleware())
	router.Use(g.rateLimitMiddleware())
	router.Use(g.authMiddleware())
	router.Use(g.loggingMiddleware())

	// Register routes
	v1 := router.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.POST("/", g.CreateUser)
			users.GET("/:id", g.GetUser)
		}

		// Order routes
		orders := v1.Group("/orders")
		{
			orders.POST("/", g.CreateOrder)
			orders.GET("/:id", g.GetOrder)
		}
	}

	// Start server
	srv := &http.Server{
		Addr:         g.config.Listen.Address,
		Handler:      router,
		ReadTimeout:  g.config.Server.ReadTimeout,
		WriteTimeout: g.config.Server.WriteTimeout,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			g.logger.Error("Server forced to shutdown", zap.Error(err))
		}
	}()

	g.logger.Info("Starting gateway server", zap.String("address", g.config.Listen.Address))
	return srv.ListenAndServe()
}

func (g *Gateway) Close() error {
	if err := g.userConn.Close(); err != nil {
		return err
	}
	return g.orderConn.Close()
}

// Middleware implementations
func (g *Gateway) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cors.New(cors.Options{
			AllowedOrigins:   g.config.CORS.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
		}).Handler(c.Writer)
		c.Next()
	}
}

func (g *Gateway) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.limiter.Allow() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

func (g *Gateway) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Token validation logic here
		// This is a simplified version, you should implement proper JWT validation
		token := authHeader[7:] // Remove "Bearer " prefix
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}

func (g *Gateway) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		g.logger.Info("Request processed",
			zap.String("path", path),
			zap.String("method", method),
			zap.Int("status", status),
			zap.Duration("latency", latency),
		)
	}
}

// Helper functions
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/api/v1/users/login",
		"/api/v1/users/register",
		"/health",
		"/metrics",
	}

	for _, pp := range publicPaths {
		if pp == path {
			return true
		}
	}
	return false
}

// Route handlers
func (g *Gateway) CreateUser(c *gin.Context) {
	// Implementation
}

func (g *Gateway) GetUser(c *gin.Context) {
	// Implementation
}

func (g *Gateway) CreateOrder(c *gin.Context) {
	// Implementation
}

func (g *Gateway) GetOrder(c *gin.Context) {
	// Implementation
}
