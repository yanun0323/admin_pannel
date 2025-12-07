package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/usecase"
)

type APIKeyHandler struct {
	apiKeyUseCase adaptor.APIKeyUseCase
}

func NewAPIKeyHandler(apiKeyUseCase adaptor.APIKeyUseCase) *APIKeyHandler {
	return &APIKeyHandler{apiKeyUseCase: apiKeyUseCase}
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	apiKeys, err := h.apiKeyUseCase.List(r.Context(), user.ID)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list api keys"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: apiKeys})
}

func (h *APIKeyHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	apiKey, err := h.apiKeyUseCase.GetByID(r.Context(), user.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAPIKeyNotFound):
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "api key not found"})
		case errors.Is(err, usecase.ErrAPIKeyUnauthorized):
			WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: "unauthorized to access this api key"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get api key"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: apiKey})
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	var req model.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	apiKey, err := h.apiKeyUseCase.Create(r.Context(), user.ID, &req)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidPlatform):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid platform"})
		case errors.Is(err, usecase.ErrAPIKeyNameEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "api key name is required"})
		case errors.Is(err, usecase.ErrAPIKeyEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "api key is required"})
		case errors.Is(err, usecase.ErrAPISecretEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "api secret is required"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create api key"})
		}
		return
	}

	WriteJSON(w, http.StatusCreated, SuccessResponse{
		Message: "api key created successfully",
		Data:    apiKey,
	})
}

func (h *APIKeyHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	var req model.UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	apiKey, err := h.apiKeyUseCase.Update(r.Context(), user.ID, id, &req)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAPIKeyNotFound):
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "api key not found"})
		case errors.Is(err, usecase.ErrAPIKeyUnauthorized):
			WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: "unauthorized to access this api key"})
		case errors.Is(err, usecase.ErrAPIKeyNameEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "api key name cannot be empty"})
		case errors.Is(err, usecase.ErrAPIKeyEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "api key cannot be empty"})
		case errors.Is(err, usecase.ErrAPISecretEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "api secret cannot be empty"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update api key"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "api key updated successfully",
		Data:    apiKey,
	})
}

func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
		return
	}

	err = h.apiKeyUseCase.Delete(r.Context(), user.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAPIKeyNotFound):
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "api key not found"})
		case errors.Is(err, usecase.ErrAPIKeyUnauthorized):
			WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: "unauthorized to access this api key"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete api key"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "api key deleted successfully"})
}

func (h *APIKeyHandler) GetPlatforms(w http.ResponseWriter, r *http.Request) {
	platforms := h.apiKeyUseCase.GetPlatforms()
	platformStrings := make([]string, len(platforms))
	for i, p := range platforms {
		platformStrings[i] = p.String()
	}
	WriteJSON(w, http.StatusOK, SuccessResponse{Data: platformStrings})
}
