package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/middleware"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	userService *service.UserService
	validate    *validator.Validate
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validate:    validator.New(),
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		if err.Error() == "email already registered" {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "error creating user")
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	requestingUserID := r.Context().Value(middleware.UserIDKey).(int64)
	requestingUserRole := r.Context().Value(middleware.RoleKey).(string)

	if requestingUserID != id && requestingUserRole != "admin" {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	user, err := h.userService.GetUser(r.Context(), id)
	if err != nil {
		if err.Error() == "error fetching user: user not found" {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "error fetching user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	requestingUserID := r.Context().Value(middleware.UserIDKey).(int64)
	if requestingUserID != id {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userService.UpdateUser(r.Context(), id, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error updating user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}