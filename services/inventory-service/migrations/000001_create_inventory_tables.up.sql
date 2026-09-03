-- Inventory table: tracks stock levels per product
CREATE TABLE IF NOT EXISTS inventory (
    id           BIGSERIAL PRIMARY KEY,
    product_id   VARCHAR(255) NOT NULL UNIQUE,
    total_stock  BIGINT       NOT NULL DEFAULT 0,
    reserved     BIGINT       NOT NULL DEFAULT 0,
    available    BIGINT       NOT NULL DEFAULT 0,
    last_updated TIMESTAMP    NOT NULL DEFAULT NOW(),

    -- Ensure stock levels are never negative
    CONSTRAINT total_stock_non_negative CHECK (total_stock >= 0),
    CONSTRAINT reserved_non_negative    CHECK (reserved >= 0),
    CONSTRAINT available_non_negative   CHECK (available >= 0),
    CONSTRAINT reserved_lte_total       CHECK (reserved <= total_stock)
);

-- Reservations table: tracks individual reservations for idempotency
CREATE TABLE IF NOT EXISTS inventory_reservations (
    id         BIGSERIAL PRIMARY KEY,
    order_id   VARCHAR(255) NOT NULL,
    product_id VARCHAR(255) NOT NULL,
    quantity   BIGINT       NOT NULL,
    status     VARCHAR(50)  NOT NULL DEFAULT 'reserved',
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP    NOT NULL DEFAULT NOW(),

    -- Unique constraint: one reservation per order+product combination
    UNIQUE(order_id, product_id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_inventory_product_id
    ON inventory(product_id);

CREATE INDEX IF NOT EXISTS idx_reservations_order_id
    ON inventory_reservations(order_id);

CREATE INDEX IF NOT EXISTS idx_reservations_product_id
    ON inventory_reservations(product_id);