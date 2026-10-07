-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id           BIGSERIAL PRIMARY KEY,
    order_number VARCHAR(50)  NOT NULL UNIQUE,
    user_id      VARCHAR(255) NOT NULL,
    status       VARCHAR(50)  NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(12, 2) NOT NULL,
    currency     VARCHAR(10)  NOT NULL DEFAULT 'INR',
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP    NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_status CHECK (
        status IN ('pending', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled')
    ),
    CONSTRAINT total_amount_positive CHECK (total_amount >= 0)
);

-- Order items table
CREATE TABLE IF NOT EXISTS order_items (
    id           BIGSERIAL PRIMARY KEY,
    order_id     BIGINT       NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   VARCHAR(255) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    sku          VARCHAR(100) NOT NULL,
    quantity     BIGINT       NOT NULL,
    unit_price   DECIMAL(12, 2) NOT NULL,
    subtotal     DECIMAL(12, 2) NOT NULL,

    CONSTRAINT quantity_positive   CHECK (quantity > 0),
    CONSTRAINT unit_price_positive CHECK (unit_price >= 0)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status  ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);