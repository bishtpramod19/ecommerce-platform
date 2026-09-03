package model

import "time"

// Inventory represents stock levels for a product.
// Stored in PostgreSQL for strong consistency.
type Inventory struct {
	ID          int64     `json:"id"`
	ProductID   string    `json:"product_id"`  // references product in product-service
	TotalStock  int64     `json:"total_stock"` // total units available
	Reserved    int64     `json:"reserved"`    // units currently reserved (in pending orders)
	Available   int64     `json:"available"`   // total_stock - reserved
	LastUpdated time.Time `json:"last_updated"`
}

// ReserveRequest is used to reserve stock for an order.
type ReserveRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int64  `json:"quantity"   validate:"required,gt=0"`
	OrderID   string `json:"order_id"   validate:"required"` // for idempotency
}

// ReleaseRequest is used to release reserved stock.
// Called when order is cancelled or payment fails.
type ReleaseRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int64  `json:"quantity"   validate:"required,gt=0"`
	OrderID   string `json:"order_id"   validate:"required"`
}

// InventoryReservation tracks a reservation for idempotency.
// Prevents double-reservation if order-service retries.
type InventoryReservation struct {
	ID        int64     `json:"id"`
	OrderID   string    `json:"order_id"` // unique per order
	ProductID string    `json:"product_id"`
	Quantity  int64     `json:"quantity"`
	Status    string    `json:"status"` // "reserved", "released", "confirmed"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
