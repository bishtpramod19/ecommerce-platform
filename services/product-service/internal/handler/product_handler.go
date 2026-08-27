package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/middleware"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// ProductHandler handles all HTTP requests for product operations.
type ProductHandler struct {
	productService *service.ProductService
	validate       *validator.Validate
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		validate:       validator.New(),
	}
}

// CreateProduct handles POST /v1/products
// Admin only - verified via gRPC call to user-service
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Get user info from context (set by auth middleware)
	userID := r.Context().Value(middleware.UserIDKey).(string)
	userRole := r.Context().Value(middleware.RoleKey).(string)

	// Only admins can create products
	if userRole != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	var req model.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.productService.CreateProduct(r.Context(), &req, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error creating product")
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

// GetProduct handles GET /v1/products/:id
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "product id is required")
		return
	}

	product, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		if err.Error() == "error fetching product: product not found" {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "error fetching product")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// ListProducts handles GET /v1/products
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := model.ProductFilter{
		Category: r.URL.Query().Get("category"),
		Brand:    r.URL.Query().Get("brand"),
		Cursor:   r.URL.Query().Get("cursor"),
	}

	// Parse price filters
	if minPrice := r.URL.Query().Get("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter.MinPrice = price
		}
	}
	if maxPrice := r.URL.Query().Get("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filter.MaxPrice = price
		}
	}

	// Parse limit
	filter.Limit = 20 // default
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.ParseInt(limit, 10, 64); err == nil {
			filter.Limit = l
		}
	}

	result, err := h.productService.ListProducts(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error fetching products")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// SearchProducts handles GET /v1/products/search
func (h *ProductHandler) SearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "search query 'q' is required")
		return
	}

	// Parse limit
	var limit int64 = 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil {
			limit = parsed
		}
	}

	products, err := h.productService.SearchProducts(r.Context(), query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error searching products")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"products": products,
		"count":    len(products),
	})
}

// UpdateProduct handles PUT /v1/products/:id
// Admin only
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	userRole := r.Context().Value(middleware.RoleKey).(string)
	if userRole != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "product id is required")
		return
	}

	var req model.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.productService.UpdateProduct(r.Context(), id, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error updating product")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// DeleteProduct handles DELETE /v1/products/:id
// Admin only
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	userRole := r.Context().Value(middleware.RoleKey).(string)
	if userRole != "admin" {
		writeError(w, http.StatusForbidden, "admin access required")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "product id is required")
		return
	}

	if err := h.productService.DeleteProduct(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "error deleting product")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "product deleted successfully"})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
