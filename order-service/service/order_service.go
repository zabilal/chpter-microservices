// internal/order/service/order_service.go
package service

import (
		"context"
		"sync"
		"time"

		"github.com/golang/protobuf/ptypes"
		"github.com/google/uuid"
		"google.golang.org/grpc/codes"
		"google.golang.org/grpc/status"
		"golang.org/x/sync/errgroup"

		pb "github.com/zabilal/microservices/pkg/genproto/order/v1"
		userpb "github.com/zabilal/microservices/pkg/genproto/user/v1"
		"github.com/zabilal/microservices/internal/pkg/logger"
		"github.com/zabilal/microservices/internal/order/repository"
)

type OrderService struct {
		pb.UnimplementedOrderServiceServer
		repo       repository.OrderRepository
		userClient userpb.UserServiceClient
		logger     *logger.Logger
}

func NewOrderService(
		repo repository.OrderRepository,
		userClient userpb.UserServiceClient,
		logger *logger.Logger,
) *OrderService {
		return &OrderService{
				repo:       repo,
				userClient: userClient,
				logger:     logger,
		}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
		log := s.logger.WithContext(ctx)

		// Validate request
		if err := validateCreateOrderRequest(req); err != nil {
				log.Error("invalid create order request", zap.Error(err))
				return nil, err
		}

		// Use errgroup for concurrent operations
		g, ctx := errgroup.WithContext(ctx)

		var user *userpb.User
		var totalAmount float64

		// Fetch user details concurrently
		g.Go(func() error {
				userResp, err := s.userClient.GetUser(ctx, &userpb.GetUserRequest{
						UserId: req.UserId,
				})
				if err != nil {
						return status.Errorf(codes.Internal, "failed to get user details: %v", err)
				}
				user = userResp.User
				return nil
		})

		// Calculate total amount concurrently
		g.Go(func() error {
				var sum float64
				for _, item := range req.Items {
						sum += float64(item.Quantity) * item.UnitPrice
				}
				totalAmount = sum
				return nil
		})

		// Wait for all concurrent operations to complete
		if err := g.Wait(); err != nil {
				log.Error("failed during concurrent operations", zap.Error(err))
				return nil, err
		}

		// Create order
		order := &pb.Order{
				Id:          uuid.New().String(),
				UserId:      req.UserId,
				Items:       req.Items,
				TotalAmount: totalAmount,
				Status:      pb.OrderStatus_ORDER_STATUS_PENDING,
				User:        user,
				CreatedAt:   ptypes.TimestampNow(),
				UpdatedAt:   ptypes.TimestampNow(),
				PaymentInfo: &pb.PaymentInfo{
						Status: pb.PaymentStatus_PAYMENT_STATUS_PENDING,
						Method: req.PaymentMethod,
				},
				ShippingInfo: req.ShippingInfo,
		}

		// Use transaction to ensure data consistency
		if err := s.repo.CreateOrder(ctx, order); err != nil {
				log.Error("failed to create order", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to create order")
		}

		log.Info("order created successfully",
				zap.String("order_id", order.Id),
				zap.String("user_id", order.UserId),
		)

		return &pb.CreateOrderResponse{
				Order: order,
		}, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.UpdateOrderStatusResponse, error) {
		log := s.logger.WithContext(ctx)

		order, err := s.repo.GetOrder(ctx, req.OrderId)
		if err != nil {
				log.Error("failed to get order", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to get order")
		}

		// Validate status transition
		if !isValidStatusTransition(order.Status, req.Status) {
				return nil, status.Error(codes.InvalidArgument, "invalid status transition")
		}

		order.Status = req.Status
		order.UpdatedAt = ptypes.TimestampNow()

		if err := s.repo.UpdateOrder(ctx, order); err != nil {
				log.Error("failed to update order status", zap.Error(err))
				return nil, status.Error(codes.Internal, "failed to update order status")
		}

		return &pb.UpdateOrderStatusResponse{
				Order: order,
		}, nil
}

func validateCreateOrderRequest(req *pb.CreateOrderRequest) error {
		if req.UserId == "" {
				return status.Error(codes.InvalidArgument, "user_id is required")
		}
		if len(req.Items) == 0 {
				return status.Error(codes.InvalidArgument, "order must contain at least one item")
		}
		for _, item := range req.Items {
				if item.Quantity <= 0 {
						return status.Error(codes.InvalidArgument, "item quantity must be positive")
				}
				if item.UnitPrice <= 0 {
						return status.Error(codes.InvalidArgument, "item unit price must be positive")
				}
		}
		return nil
}

func isValidStatusTransition(current, new pb.OrderStatus) bool {
		// Define valid status transitions
		transitions := map[pb.OrderStatus][]pb.OrderStatus{
				pb.OrderStatus_ORDER_STATUS_PENDING: {
						pb.OrderStatus_ORDER_STATUS_PROCESSING,
						pb.OrderStatus_ORDER_STATUS_CANCELLED,
				},
				pb.OrderStatus_ORDER_STATUS_PROCESSING: {
						pb.OrderStatus_ORDER_STATUS_COMPLETED,
						pb.OrderStatus_ORDER_STATUS_FAILED,
				},
		}

		validTransitions, exists := transitions[current]
		if !exists {
				return false
		}

		for _, validStatus := range validTransitions {
				if new == validStatus {
						return true
				}
		}

		return false
}