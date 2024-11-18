package handler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/zabilal/microservices/pkg/genproto/order/v1"
	userv1 "github.com/zabilal/microservices/pkg/genproto/user/v1"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrder(ctx context.Context, id string) (*Order, error)
	ListOrders(ctx context.Context, userID string, status orderv1.OrderStatus, pageSize int32, pageToken string) ([]*Order, string, error)
	UpdateOrderStatus(ctx context.Context, id string, status orderv1.OrderStatus) error
}

type Order struct {
	ID            string
	UserID        string
	Items         []OrderItem
	TotalAmount   float64
	Status        orderv1.OrderStatus
	PaymentInfo   PaymentInfo
	ShippingInfo  ShippingInfo
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrderItem struct {
	ProductID   string
	Quantity    int32
	UnitPrice   float64
	ProductName string
}

type PaymentInfo struct {
	PaymentID   string
	Status      orderv1.PaymentStatus
	Method      orderv1.PaymentMethod
	ProcessedAt time.Time
}

type ShippingInfo struct {
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Country      string
	PostalCode   string
	Status       orderv1.ShippingStatus
}

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(config DatabaseConfig) OrderRepository {
	// Initialize database connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}

	return &orderRepository{db: db}
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
}

type OrderHandler struct {
	UnimplementedOrderServiceServer
	repo      OrderRepository
	userConn  *grpc.ClientConn
	userClient userv1.UserServiceClient
	logger    *zap.Logger
}

func NewOrderHandler(repo OrderRepository, userConn *grpc.ClientConn, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		repo:       repo,
		userConn:   userConn,
		userClient: userv1.NewUserServiceClient(userConn),
		logger:     logger,
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	// Validate user exists
	user, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: req.UserId})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, status.Error(codes.InvalidArgument, "user not found")
		}
		h.logger.Error("failed to verify user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to verify user")
	}

	// Create order
	order := &Order{
		ID:     uuid.New().String(),
		UserID: req.UserId,
		Status: orderv1.OrderStatus_ORDER_STATUS_PENDING,
		Items:  make([]OrderItem, len(req.Items)),
		ShippingInfo: ShippingInfo{
			AddressLine1: req.ShippingInfo.AddressLine1,
			AddressLine2: req.ShippingInfo.AddressLine2,
			City:         req.ShippingInfo.City,
			State:        req.ShippingInfo.State,
			Country:      req.ShippingInfo.Country,
			PostalCode:   req.ShippingInfo.PostalCode,
			Status:       orderv1.ShippingStatus_SHIPPING_STATUS_PENDING,
		},
		PaymentInfo: PaymentInfo{
			PaymentID: uuid.New().String(),
			Status:    orderv1.PaymentStatus_PAYMENT_STATUS_PENDING,
			Method:    req.PaymentMethod,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var totalAmount float64
	for i, item := range req.Items {
		order.Items[i] = OrderItem{
			ProductID:   item.ProductId,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			ProductName: item.ProductName,
		}
		totalAmount += item.UnitPrice * float64(item.Quantity)
	}
	order.TotalAmount = totalAmount

	if err := h.repo.CreateOrder(ctx, order); err != nil {
		h.logger.Error("failed to create order", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create order")
	}

	return &orderv1.CreateOrderResponse{
		Order: convertToProtoOrder(order, user),
	}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	order, err := h.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		h.logger.Error("failed to get order", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get order")
	}

	// Get user details
	user, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: order.UserID})
	if err != nil {
		h.logger.Error("failed to get user details", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user details")
	}

	return &orderv1.GetOrderResponse{
		Order: convertToProtoOrder(order, user),
	}, nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	orders, nextPageToken, err := h.repo.ListOrders(ctx, req.UserId, req.Status, req.PageSize, req.PageToken)
	if err != nil {
		h.logger.Error("failed to list orders", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list orders")
	}

	// Get user details if we have orders
	var user *userv1.User
	if len(orders) > 0 {
		user, err = h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: orders[0].UserID})
		if err != nil {
			h.logger.Error("failed to get user details", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to get user details")
		}
	}

	protoOrders := make([]*orderv1.Order, len(orders))
	for i, order := range orders {
		protoOrders[i] = convertToProtoOrder(order, user)
	}

	return &orderv1.ListOrdersResponse{
		Orders:        protoOrders,
		NextPageToken: nextPageToken,
	}, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	order, err := h.repo.GetOrder(ctx, req.OrderId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		h.logger.Error("failed to get order", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get order")
	}

	if err := h.repo.UpdateOrderStatus(ctx, req.OrderId, req.Status); err != nil {
		h.logger.Error("failed to update order status", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update order status")
	}

	order.Status = req.Status
	order.UpdatedAt = time.Now()

	// Get user details
	user, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: order.UserID})
	if err != nil {
		h.logger.Error("failed to get user details", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user details")
	}

	return &orderv1.UpdateOrderStatusResponse{
		Order: convertToProtoOrder(order, user),
	}, nil
}

func convertToProtoOrder(order *Order, user *userv1.User) *orderv1.Order {
	items := make([]*orderv1.OrderItem, len(order.Items))
	for i, item := range order.Items {
		items[i] = &orderv1.OrderItem{
			ProductId:   item.ProductID,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			ProductName: item.ProductName,
		}
	}

	return &orderv1.Order{
		Id:          order.ID,
		UserId:      order.UserID,
		Items:       items,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
		User:        user,
		CreatedAt:   timestamppb.New(order.CreatedAt),
		UpdatedAt:   timestamppb.New(order.UpdatedAt),
		PaymentInfo: &orderv1.PaymentInfo{
			PaymentId:   order.PaymentInfo.PaymentID,
			Status:      order.PaymentInfo.Status,
			Method:      order.PaymentInfo.Method,
			ProcessedAt: timestamppb.New(order.PaymentInfo.ProcessedAt),
		},
		ShippingInfo: &orderv1.ShippingInfo{
			AddressLine1: order.ShippingInfo.AddressLine1,
			AddressLine2: order.ShippingInfo.AddressLine2,
			City:         order.ShippingInfo.City,
			State:        order.ShippingInfo.State,
			Country:      order.ShippingInfo.Country,
			PostalCode:   order.ShippingInfo.PostalCode,
		},
	}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order *Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// Insert order
	query := `
		INSERT INTO orders (id, user_id, status, total_amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`
	
	_, err = tx.ExecContext(ctx, query,
		order.ID,
		order.UserID,
		order.Status,
		order.TotalAmount,
	)
	
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert order items
	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, product_name)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, itemQuery,
			uuid.New().String(),
			order.ID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.ProductName,
		)
		
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Insert payment info
	paymentQuery := `
		INSERT INTO payment_info (id, order_id, payment_id, status, method, processed_at)
		VALUES (?, ?, ?, ?, ?, NOW())
	`
	
	_, err = tx.ExecContext(ctx, paymentQuery,
		uuid.New().String(),
		order.ID,
		order.PaymentInfo.PaymentID,
		order.PaymentInfo.Status,
		order.PaymentInfo.Method,
	)
	
	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert shipping info
	shippingQuery := `
		INSERT INTO shipping_info (id, order_id, address_line1, address_line2, city, state, country, postal_code, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err = tx.ExecContext(ctx, shippingQuery,
		uuid.New().String(),
		order.ID,
		order.ShippingInfo.AddressLine1,
		order.ShippingInfo.AddressLine2,
		order.ShippingInfo.City,
		order.ShippingInfo.State,
		order.ShippingInfo.Country,
		order.ShippingInfo.PostalCode,
		order.ShippingInfo.Status,
	)
	
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *orderRepository) GetOrder(ctx context.Context, id string) (*Order, error) {
	query := `
		SELECT o.id, o.user_id, o.status, o.total_amount, o.created_at, o.updated_at
		FROM orders o
		WHERE o.id = ?
	`
	
	order := &Order{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}

	// Get order items
	itemsQuery := `
		SELECT id, product_id, quantity, unit_price, product_name
		FROM order_items
		WHERE order_id = ?
	`
	
	rows, err := r.db.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := OrderItem{OrderID: id}
		err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.ProductName,
		)
		if err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	// Get payment info
	paymentQuery := `
		SELECT payment_id, status, method, processed_at
		FROM payment_info
		WHERE order_id = ?
	`
	
	err = r.db.QueryRowContext(ctx, paymentQuery, id).Scan(
		&order.PaymentInfo.PaymentID,
		&order.PaymentInfo.Status,
		&order.PaymentInfo.Method,
		&order.PaymentInfo.ProcessedAt,
	)
	
	if err != nil {
		return nil, err
	}

	// Get shipping info
	shippingQuery := `
		SELECT address_line1, address_line2, city, state, country, postal_code, status
		FROM shipping_info
		WHERE order_id = ?
	`
	
	err = r.db.QueryRowContext(ctx, shippingQuery, id).Scan(
		&order.ShippingInfo.AddressLine1,
		&order.ShippingInfo.AddressLine2,
		&order.ShippingInfo.City,
		&order.ShippingInfo.State,
		&order.ShippingInfo.Country,
		&order.ShippingInfo.PostalCode,
		&order.ShippingInfo.Status,
	)
	
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *orderRepository) ListOrders(ctx context.Context, userID string, status orderv1.OrderStatus, pageSize int32, pageToken string) ([]*Order, string, error) {
	query := `
		SELECT o.id, o.user_id, o.status, o.total_amount, o.created_at, o.updated_at
		FROM orders o
		WHERE o.user_id = ? AND o.status = ?
		ORDER BY o.created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.QueryContext(ctx, query, userID, status, pageSize, pageSize*int32(len(pageToken)))
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		order := &Order{}
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Status,
			&order.TotalAmount,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, "", err
		}
		orders = append(orders, order)
	}

	// Get order items
	itemsQuery := `
		SELECT id, product_id, quantity, unit_price, product_name
		FROM order_items
		WHERE order_id = ?
	`
	
	for _, order := range orders {
		rows, err := r.db.QueryContext(ctx, itemsQuery, order.ID)
		if err != nil {
			return nil, "", err
		}
		defer rows.Close()

		for rows.Next() {
			item := OrderItem{OrderID: order.ID}
			err := rows.Scan(
				&item.ID,
				&item.ProductID,
				&item.Quantity,
				&item.UnitPrice,
				&item.ProductName,
			)
			if err != nil {
				return nil, "", err
			}
			order.Items = append(order.Items, item)
		}
	}

	// Get payment info
	paymentQuery := `
		SELECT payment_id, status, method, processed_at
		FROM payment_info
		WHERE order_id = ?
	`
	
	for _, order := range orders {
		err = r.db.QueryRowContext(ctx, paymentQuery, order.ID).Scan(
			&order.PaymentInfo.PaymentID,
			&order.PaymentInfo.Status,
			&order.PaymentInfo.Method,
			&order.PaymentInfo.ProcessedAt,
		)
		if err != nil {
			return nil, "", err
		}
	}

	// Get shipping info
	shippingQuery := `
		SELECT address_line1, address_line2, city, state, country, postal_code, status
		FROM shipping_info
		WHERE order_id = ?
	`
	
	for _, order := range orders {
		err = r.db.QueryRowContext(ctx, shippingQuery, order.ID).Scan(
			&order.ShippingInfo.AddressLine1,
			&order.ShippingInfo.AddressLine2,
			&order.ShippingInfo.City,
			&order.ShippingInfo.State,
			&order.ShippingInfo.Country,
			&order.ShippingInfo.PostalCode,
			&order.ShippingInfo.Status,
		)
		if err != nil {
			return nil, "", err
		}
	}

	return orders, "", nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, id string, status orderv1.OrderStatus) error {
	query := `
		UPDATE orders
		SET status = ?, updated_at = NOW()
		WHERE id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
