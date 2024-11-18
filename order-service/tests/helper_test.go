package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zabilal/microservices/internal/order/repository"
)

func createTestOrder(t *testing.T, repo repository.OrderRepository) *repository.Order {
	ctx := context.Background()
	order := &repository.Order{
		ID:          uuid.New().String(),
		UserID:      uuid.New().String(),
		Status:      "PENDING",
		TotalAmount: 99.99,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Items: []*repository.OrderItem{
			{
				ID:          uuid.New().String(),
				ProductName: "Test Product",
				Quantity:    2,
				Price:      49.99,
			},
		},
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)
	return order
}

func requireOrderEqual(t *testing.T, expected, actual *repository.Order) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.UserID, actual.UserID)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.TotalAmount, actual.TotalAmount)
	require.Len(t, actual.Items, len(expected.Items))

	for i := range expected.Items {
		requireOrderItemEqual(t, expected.Items[i], actual.Items[i])
	}
}

func requireOrderItemEqual(t *testing.T, expected, actual *repository.OrderItem) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.ProductName, actual.ProductName)
	require.Equal(t, expected.Quantity, actual.Quantity)
	require.Equal(t, expected.Price, actual.Price)
}
