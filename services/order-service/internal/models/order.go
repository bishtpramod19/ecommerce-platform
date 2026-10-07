package models

import "time"

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusConfirmed  OrderStatus = "confirmed"
	StatusProcessing OrderStatus = "processing"
	StatusShipped    OrderStatus = "shipped"
	StatusDelivered  OrderStatus = "delivered"
	StatusCancelled  OrderStatus = "cancelled"
)

// Order represents a customer order.
type Order struct {
	ID          int64       `json:"id"`
	OrderNumber string      `json:"order_number"` // human-readable, e.g. "ORD-20260115-0001"
	UserID      string      `json:"user_id"`
	Status      OrderStatus `json:"status"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Currency    string      `json:"currency"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// OrderItem represents a single product line item within an order.
// Snapshot of product details AT TIME OF ORDER (price could change later).
type OrderItem struct {
	ID          int64   `json:"id"`
	OrderID     int64   `json:"order_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"` // snapshot, not live lookup
	SKU         string  `json:"sku"`
	Quantity    int64   `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"` // snapshot of price at order time
	Subtotal    float64 `json:"subtotal"`   // unit_price * quantity
}

// CreateOrderRequest is what the client sends to place an order.
type CreateOrderRequest struct {
	Items []CreateOrderItemRequest `json:"items" validate:"required,min=1"`
}

// CreateOrderItemRequest represents one item the user wants to order.
type CreateOrderItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	SKU       string `json:"sku"        validate:"required"`
	Quantity  int64  `json:"quantity"   validate:"required,gt=0"`
}
