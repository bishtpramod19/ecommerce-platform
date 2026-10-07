#!/bin/bash
set -e

echo "Waiting for port-forwards to be ready..."
echo "Make sure these are running in separate terminals:"
echo "  kubectl port-forward -n ecommerce-dev svc/user-service 8001:8001"
echo "  kubectl port-forward -n ecommerce-dev svc/product-service 8002:8002"
echo "  kubectl port-forward -n ecommerce-dev svc/postgres-service 5433:5432"
echo ""
read -p "Press enter once all port-forwards are running..."

# 1. Register user
echo "Registering test user..."
curl -s -X POST http://localhost:8001/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123", "first_name": "pramod", "last_name": "singh"}' > /dev/null

# 2. Promote to admin
echo "Promoting user to admin..."
PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -d users_db \
  -c "UPDATE users SET role='admin' WHERE email='test@example.com';" > /dev/null

# 3. Login to get token
echo "Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8001/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)
echo "Token acquired."

# 4. Create product
echo "Creating product..."
PRODUCT_RESPONSE=$(curl -s -X POST http://localhost:8002/v1/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Cotton T-Shirt", "description": "Premium quality cotton t-shirt", "brand": "Nike", "category": "clothing", "base_price": 299.00, "currency": "INR", "variants": [{"sku": "TSH-S-RED", "size": "S", "color": "Red", "price": 299.00, "images": ["https://cdn.example.com/tsh-s-red.jpg"]}]}')

PRODUCT_ID=$(echo $PRODUCT_RESPONSE | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
echo "Product created: $PRODUCT_ID"

# 5. Add inventory stock
echo "Adding inventory stock..."
PGPASSWORD=postgres psql -h localhost -p 5433 -U postgres -d inventory_db \
  -c "INSERT INTO inventory (product_id, total_stock, available) VALUES ('$PRODUCT_ID', 100, 100) ON CONFLICT (product_id) DO UPDATE SET total_stock=100, available=100;" > /dev/null

echo ""
echo "=== Seed complete ==="
echo "User:       test@example.com / password123 (role: admin)"
echo "Token:      $TOKEN"
echo "Product ID: $PRODUCT_ID"