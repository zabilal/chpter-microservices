// internal/order/repository/postgres/order_repository.go
package postgres

import (
		"context"
		"encoding/json"
		"errors"
		"time"

		"github.com/jackc/pgx/v4"
		"github.com/jackc/pgx/v4/pgxpool"
		"google.golang.org/protobuf/types/known/timestamppb"

		pb "github.com/yourusername/microservices/pkg/genproto/order/v1"
		"github.com/yourusername/microservices/internal/pkg/logger"
)

type OrderRepository struct {
		pool   *pgxpool.Pool
		logger *logger.Logger
}

func NewOrderRepository(pool *pgxpool.Pool, logger *logger.Logger) *OrderRepository {
		return &OrderRepository{
				pool:   pool,
				logger: logger,
		}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *pb.Order) error {
		tx, err := r.pool.Begin(ctx)
		if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		// Insert order
		query := `
				INSERT INTO orders (
						id, user_id, total_amount, status,
						payment_info, shipping_info, created_at, updated_at
				)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		paymentInfoJSON, err := json.Marshal(order.PaymentInfo)
		if err != nil {
				return fmt.Errorf("failed to marshal payment info: %w", err)
		}

		shippingInfoJSON, err := json.Marshal(order.ShippingInfo)
		if err != nil {
				return fmt.Errorf("failed to marshal shipping info: %w", err)
		}

		_, err = tx.Exec(ctx, query,
				order.Id,
				order.UserId,
				order.TotalAmount,
				order.Status,
				paymentInfoJSON,
				shippingInfoJSON,
				time.Now(),
				time.Now(),
		)

		if err != nil {
				return fmt.Errorf("failed to create order: %w", err)
		}

		// Insert order items
		itemQuery := `
				INSERT INTO order_items (
						order_id, product_id, quantity, unit_price, product_name
				)
				VALUES ($1, $2, $3, $4, $5)
		`

		for _, item := range order.Items {
				_, err = tx.Exec(ctx, itemQuery,
						order.Id,
						item.ProductId,
						item.Quantity,
						item.UnitPrice,
						item.ProductName,
				)
				if err != nil {
						return fmt.Errorf("failed to create order item: %w", err)
				}
		}

		if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, id string) (*pb.Order, error) {
		query := `
				SELECT o.id, o.user_id, o.total_amount, o.status,
							 o.payment_info, o.shipping_info, o.created_at, o.updated_at
				FROM orders o
				WHERE o.id = $1
		`

		var order pb.Order
		var paymentInfoJSON, shippingInfoJSON []byte
		var createdAt, updatedAt time.Time

		err := r.pool.QueryRow(ctx, query, id).Scan(
				&order.Id,
				&order.UserId,
				&order.TotalAmount,
				&order.Status,
				&paymentInfoJSON,
				&shippingInfoJSON,
				&createdAt,
				&updatedAt,
		)

		if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
						return nil, nil
				}
				return nil, fmt.Errorf("failed to get order: %w", err)
		}

		// Unmarshal payment info
		var paymentInfo pb.PaymentInfo
		if err := json.Unmarshal(paymentInfoJSON, &paymentInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal payment info: %w", err)
		}
		order.PaymentInfo = &paymentInfo

		// Unmarshal shipping info
		var shippingInfo pb.ShippingInfo
		if err := json.Unmarshal(shippingInfoJSON, &shippingInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal shipping info: %w", err)
		}
		order.ShippingInfo = &shippingInfo

		// Get order items
		itemsQuery := `
				SELECT product_id, quantity, unit_price, product_name
				FROM order_items
				WHERE order_id = $1
		`

		rows, err := r.pool.Query(ctx, itemsQuery, id)
		if err != nil {
				return nil, fmt.Errorf("failed to get order items: %w", err)
		}
		defer rows.Close()

		var items []*pb.OrderItem
		for rows.Next() {
				var item pb.OrderItem
				err := rows.Scan(
						&item.ProductId,
						&item.Quantity,
						&item.UnitPrice,
						&item.ProductName,
				)
				if err != nil {
						return nil, fmt.Errorf("failed to scan order item: %w", err)
				}
				items = append(items, &item)
		}

		if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("error iterating order items: %w", err)
		}

		order.Items = items
		order.CreatedAt = timestamppb.New(createdAt)
		order.UpdatedAt = timestamppb.New(updatedAt)

		return &order, nil
}
