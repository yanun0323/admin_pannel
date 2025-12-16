package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/model/enum"
	"control_page/internal/usecase"
)

type RBACHandler struct {
	roleUseCase adaptor.RoleUseCase
	userUseCase adaptor.UserUseCase
}

func NewRBACHandler(roleUseCase adaptor.RoleUseCase, userUseCase adaptor.UserUseCase) *RBACHandler {
	return &RBACHandler{
		roleUseCase: roleUseCase,
		userUseCase: userUseCase,
	}
}

// Role handlers

type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SetPermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.roleUseCase.ListRoles(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list roles"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: roles})
}

func (h *RBACHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid role id"})
		return
	}

	role, err := h.roleUseCase.GetRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrRoleNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "role not found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get role"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: role})
}

func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}

	permissions := make([]enum.Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		permissions[i] = enum.Permission(p)
	}

	role, err := h.roleUseCase.CreateRole(r.Context(), req.Name, req.Description, permissions)
	if err != nil {
		if errors.Is(err, usecase.ErrRoleAlreadyExists) {
			WriteJSON(w, http.StatusConflict, ErrorResponse{Error: "role already exists"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create role"})
		return
	}

	WriteJSON(w, http.StatusCreated, SuccessResponse{
		Message: "role created successfully",
		Data:    role,
	})
}

func (h *RBACHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid role id"})
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "name is required"})
		return
	}

	role, err := h.roleUseCase.UpdateRole(r.Context(), id, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrRoleNotFound):
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "role not found"})
		case errors.Is(err, usecase.ErrRoleAlreadyExists):
			WriteJSON(w, http.StatusConflict, ErrorResponse{Error: "role name already exists"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update role"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "role updated successfully",
		Data:    role,
	})
}

func (h *RBACHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid role id"})
		return
	}

	if err := h.roleUseCase.DeleteRole(r.Context(), id); err != nil {
		if errors.Is(err, usecase.ErrRoleNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "role not found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete role"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "role deleted successfully"})
}

func (h *RBACHandler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid role id"})
		return
	}

	var req SetPermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	permissions := make([]enum.Permission, len(req.Permissions))
	for i, p := range req.Permissions {
		permissions[i] = enum.Permission(p)
	}

	if err := h.roleUseCase.SetPermissions(r.Context(), id, permissions); err != nil {
		if errors.Is(err, usecase.ErrRoleNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "role not found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to set permissions"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "permissions updated successfully"})
}

func (h *RBACHandler) GetAllPermissions(w http.ResponseWriter, r *http.Request) {
	permissions := h.roleUseCase.GetAllPermissions()
	WriteJSON(w, http.StatusOK, SuccessResponse{Data: permissions})
}

// User handlers

type AssignRoleRequest struct {
	RoleID string `json:"role_id"`
}

type CreateUserRequest struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	IsActive bool     `json:"is_active"`
	Roles    []string `json:"roles"`
}

type UpdateUserRequest struct {
	Username string   `json:"username"`
	IsActive bool     `json:"is_active"`
	Roles    []string `json:"roles"`
}

func (h *RBACHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userUseCase.ListUsers(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list users"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: users})
}

func (h *RBACHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	user, err := h.userUseCase.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get user"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: user})
}

func (h *RBACHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "username and password are required"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to hash password"})
		return
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashed),
		IsActive: req.IsActive,
	}

	created, err := h.userUseCase.CreateUser(r.Context(), user, req.Roles)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	WriteJSON(w, http.StatusCreated, SuccessResponse{Data: created})
}

func (h *RBACHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	user, err := h.userUseCase.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get user"})
		return
	}

	user.Username = req.Username
	user.IsActive = req.IsActive

	if err := h.userUseCase.UpdateUser(r.Context(), &user.User); err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update user"})
		return
	}

	// Sync roles
	currentRoleIDs := make(map[string]struct{})
	for _, role := range user.Roles {
		currentRoleIDs[role.ID] = struct{}{}
	}
	targetRoleIDs := make(map[string]struct{})
	for _, id := range req.Roles {
		targetRoleIDs[id] = struct{}{}
		if _, ok := currentRoleIDs[id]; !ok {
			if err := h.userUseCase.AssignRole(r.Context(), user.ID, id); err != nil {
				WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to assign role"})
				return
			}
		}
	}
	for roleID := range currentRoleIDs {
		if _, ok := targetRoleIDs[roleID]; !ok {
			if err := h.userUseCase.RemoveRole(r.Context(), user.ID, roleID); err != nil {
				WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to remove role"})
				return
			}
		}
	}

	updated, err := h.userUseCase.GetUser(r.Context(), id)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch updated user"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: updated})
}

func (h *RBACHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	if err := h.userUseCase.DeleteUser(r.Context(), id); err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete user"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "user deleted successfully"})
}

func (h *RBACHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.RoleID == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "role_id is required"})
		return
	}

	if err := h.userUseCase.AssignRole(r.Context(), id, req.RoleID); err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to assign role"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "role assigned successfully"})
}

func (h *RBACHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	roleID := chi.URLParam(r, "roleId")
	if roleID == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid role id"})
		return
	}

	if err := h.userUseCase.RemoveRole(r.Context(), userID, roleID); err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to remove role"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "role removed successfully"})
}

func (h *RBACHandler) ResetUserTOTP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid user id"})
		return
	}

	setup, err := h.userUseCase.ResetUserTOTP(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to reset 2FA"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: setup})
}
