package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type Gateway struct {
	router     *gin.Engine
	userConn   *grpc.ClientConn
	orderConn  *grpc.ClientConn
	httpServer *http.Server
}

func NewGateway(userServiceAddr, orderServiceAddr string) (*Gateway, error) {
	userConn, err := grpc.Dial(userServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %v", err)
	}

	orderConn, err := grpc.Dial(orderServiceAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to order service: %v", err)
	}

	router := gin.Default()

	return &Gateway{
		router:    router,
		userConn:  userConn,
		orderConn: orderConn,
	}, nil
}

func (g *Gateway) Start(port int) error {
	g.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: g.router,
	}
	return g.httpServer.ListenAndServe()
}

func (g *Gateway) Stop(ctx context.Context) error {
	if err := g.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown http server: %v", err)
	}
	if err := g.userConn.Close(); err != nil {
		return fmt.Errorf("failed to close user service connection: %v", err)
	}
	if err := g.orderConn.Close(); err != nil {
		return fmt.Errorf("failed to close order service connection: %v", err)
	}
	return nil
}
