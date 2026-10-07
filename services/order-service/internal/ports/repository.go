package ports

import (
	"context"

	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/models"
)

// OrderRepository is the PORT for order data storage.
type OrderRepository interface {
	// Create creates a new order with its items in a single transaction.
	Create(ctx context.Context, order *models.Order) (*models.Order, error)

	// GetByID fetches an order by its ID.
	GetByID(ctx context.Context, id int64) (*models.Order, error)

	// ListByUserID fetches all orders for a specific user.
	ListByUserID(ctx context.Context, userID string, limit, offset int64) ([]models.Order, error)

	// UpdateStatus updates the status of an order.
	UpdateStatus(ctx context.Context, orderID int64, status models.OrderStatus) error
}
