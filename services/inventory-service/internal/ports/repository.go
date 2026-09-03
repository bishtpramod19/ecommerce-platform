package ports

import (
	"context"

	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/model"
)

// InventoryRepository is the PORT for inventory data storage.
// Knows nothing about PostgreSQL or any specific database.
type InventoryRepository interface {
	// GetByProductID fetches inventory for a specific product.
	GetByProductID(ctx context.Context, productID string) (*model.Inventory, error)

	// BulkGetByProductIDs fetches inventory for multiple products.
	BulkGetByProductIDs(ctx context.Context, productIDs []string) ([]model.Inventory, error)

	// Reserve atomically reserves stock for an order.
	// Returns error if insufficient stock.
	Reserve(ctx context.Context, req *model.ReserveRequest) error

	// Release releases previously reserved stock.
	// Called when order is cancelled or payment fails.
	Release(ctx context.Context, req *model.ReleaseRequest) error

	// UpdateStock updates total stock level.
	// Called when new stock arrives in warehouse.
	UpdateStock(ctx context.Context, productID string, quantity int64) error

	// IsAlreadyReserved checks if order already has a reservation.
	// Used for idempotency — prevents double reservation.
	IsAlreadyReserved(ctx context.Context, orderID string, productID string) (bool, error)
}
