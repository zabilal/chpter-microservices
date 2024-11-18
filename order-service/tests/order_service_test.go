package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/zabilal/microservices/pkg/genproto/order/v1"
	userpb "github.com/zabilal/microservices/pkg/genproto/user/v1"
	"github.com/zabilal/microservices/internal/order/repository"
	"github.com/zabilal/microservices/internal/order/service"
	"github.com/zabilal/microservices/internal/pkg/logger"
)

type mockOrderRepo struct {
	orders map[string]*repository.Order
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders: make(map[string]*repository.Order),
	}
}

func (m *mockOrderRepo) Create(ctx context.Context, order *repository.Order) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockOrderRepo) Get(ctx context.Context, id string) (*repository.Order, error) {
	order, exists := m.orders[id]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return order, nil
}

func (m *mockOrderRepo) List(ctx context.Context, userID string) ([]*repository.Order, error) {
	var orders []*repository.Order
	for _, order := range m.orders {
		if order.UserID == userID {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

func (m *mockOrderRepo) Update(ctx context.Context, order *repository.Order) error {
	if _, exists := m.orders[order.ID]; !exists {
		return sql.ErrNoRows
	}
	m.orders[order.ID] = order
	return nil
}

type mockUserClient struct {
	users map[string]*userpb.User
}

func newMockUserClient() *mockUserClient {
	return &mockUserClient{
		users: make(map[string]*userpb.User),
	}
}

func (m *mockUserClient) GetUser(ctx context.Context, req *userpb.GetUserRequest, opts ...interface{}) (*userpb.GetUserResponse, error) {
	user, exists := m.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &userpb.GetUserResponse{User: user}, nil
}

func TestCreateOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := newMockOrderRepo()
	userClient := newMockUserClient()
	logger := logger.NewLogger("debug")

	// Create test user
	userID := uuid.New().String()
	userClient.users[userID] = &userpb.User{
		Id: userID,
		Username: "testuser",
		Email: "test@example.com",
	}

	orderService := service.NewOrderService(repo, userClient, logger)

	tests := []struct {
		name        string
		req         *pb.CreateOrderRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "valid order",
			req: &pb.CreateOrderRequest{
				UserId: userID,
				Items: []*pb.OrderItem{
					{
						ProductName: "Test Product",
						Quantity:    1,
						Price:      10.99,
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid user",
			req: &pb.CreateOrderRequest{
				UserId: "invalid-user-id",
				Items: []*pb.OrderItem{
					{
						ProductName: "Test Product",
						Quantity:    1,
						Price:      10.99,
					},
				},
			},
			expectError: true,
			errorCode:   codes.NotFound,
		},
		{
			name: "empty items",
			req: &pb.CreateOrderRequest{
				UserId: userID,
				Items:  []*pb.OrderItem{},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := orderService.CreateOrder(context.Background(), tt.req)

			if tt.expectError {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.NotEmpty(t, resp.Order.Id)
			assert.Equal(t, tt.req.UserId, resp.Order.UserId)
			assert.Equal(t, len(tt.req.Items), len(resp.Order.Items))
		})
	}
}

func TestGetOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := newMockOrderRepo()
	userClient := newMockUserClient()
	logger := logger.NewLogger("debug")

	orderService := service.NewOrderService(repo, userClient, logger)

	// Create test order
	orderID := uuid.New().String()
	userID := uuid.New().String()
	testOrder := &repository.Order{
		ID:          orderID,
		UserID:      userID,
		Status:      "PENDING",
		TotalAmount: 10.99,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Items: []*repository.OrderItem{
			{
				ID:          uuid.New().String(),
				OrderID:     orderID,
				ProductName: "Test Product",
				Quantity:    1,
				Price:      10.99,
			},
		},
	}
	repo.orders[orderID] = testOrder

	tests := []struct {
		name        string
		orderID     string
		expectError bool
		errorCode   codes.Code
	}{
		{
			name:        "existing order",
			orderID:     orderID,
			expectError: false,
		},
		{
			name:        "non-existent order",
			orderID:     "non-existent-id",
			expectError: true,
			errorCode:   codes.NotFound,
		},
		{
			name:        "invalid order id",
			orderID:     "",
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := orderService.GetOrder(context.Background(), &pb.GetOrderRequest{
				Id: tt.orderID,
			})

			if tt.expectError {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, testOrder.ID, resp.Order.Id)
			assert.Equal(t, testOrder.UserID, resp.Order.UserId)
			assert.Equal(t, len(testOrder.Items), len(resp.Order.Items))
		})
	}
}

func TestUpdateOrderStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := newMockOrderRepo()
	userClient := newMockUserClient()
	logger := logger.NewLogger("debug")

	orderService := service.NewOrderService(repo, userClient, logger)

	// Create test order
	orderID := uuid.New().String()
	userID := uuid.New().String()
	testOrder := &repository.Order{
		ID:          orderID,
		UserID:      userID,
		Status:      "PENDING",
		TotalAmount: 10.99,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.orders[orderID] = testOrder

	tests := []struct {
		name        string
		req         *pb.UpdateOrderStatusRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "valid status update",
			req: &pb.UpdateOrderStatusRequest{
				Id:     orderID,
				Status: pb.OrderStatus_ORDER_STATUS_PROCESSING,
			},
			expectError: false,
		},
		{
			name: "non-existent order",
			req: &pb.UpdateOrderStatusRequest{
				Id:     "non-existent-id",
				Status: pb.OrderStatus_ORDER_STATUS_PROCESSING,
			},
			expectError: true,
			errorCode:   codes.NotFound,
		},
		{
			name: "invalid status transition",
			req: &pb.UpdateOrderStatusRequest{
				Id:     orderID,
				Status: pb.OrderStatus_ORDER_STATUS_DELIVERED,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := orderService.UpdateOrderStatus(context.Background(), tt.req)

			if tt.expectError {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.errorCode, st.Code())
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.req.Status, resp.Order.Status)
		})
	}
}
