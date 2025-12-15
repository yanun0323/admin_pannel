package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/usecase"
)

type SettingHandler struct {
	settingUseCase adaptor.SettingUseCase
}

func NewSettingHandler(settingUseCase adaptor.SettingUseCase) *SettingHandler {
	return &SettingHandler{settingUseCase: settingUseCase}
}

func (h *SettingHandler) List(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingUseCase.List(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list settings"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: settings})
}

func (h *SettingHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	setting, err := h.settingUseCase.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrSettingNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "setting not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get setting"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: setting})
}

func (h *SettingHandler) GetByBaseQuote(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")

	if base == "" || quote == "" {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "base and quote are required"})
		return
	}

	setting, err := h.settingUseCase.GetByBaseQuote(r.Context(), base, quote)
	if err != nil {
		if errors.Is(err, usecase.ErrSettingNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "setting not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get setting"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: setting})
}

func (h *SettingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	setting, err := h.settingUseCase.Create(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrSettingBaseEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "base is required"})
		case errors.Is(err, usecase.ErrSettingQuoteEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "quote is required"})
		case errors.Is(err, usecase.ErrSettingStrategyEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "strategy is required"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create setting"})
		}
		return
	}

	WriteJSON(w, http.StatusCreated, SuccessResponse{
		Message: "setting created successfully",
		Data:    setting,
	})
}

func (h *SettingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	setting, err := h.settingUseCase.Update(r.Context(), id, &req)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrSettingNotFound):
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "setting not found"})
		case errors.Is(err, usecase.ErrSettingBaseEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "base cannot be empty"})
		case errors.Is(err, usecase.ErrSettingQuoteEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "quote cannot be empty"})
		case errors.Is(err, usecase.ErrSettingStrategyEmpty):
			WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "strategy cannot be empty"})
		default:
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update setting"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "setting updated successfully",
		Data:    setting,
	})
}

// UpdateParametersRequest is the request structure for updating strategy parameters
type UpdateParametersRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
}

func (h *SettingHandler) UpdateParameters(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	strategy := chi.URLParam(r, "strategy")

	var req UpdateParametersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	setting, err := h.settingUseCase.UpdateParameters(r.Context(), id, strategy, req.Parameters)
	if err != nil {
		if errors.Is(err, usecase.ErrSettingNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "setting not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update parameters"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "parameters updated successfully",
		Data:    setting,
	})
}

func (h *SettingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.settingUseCase.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrSettingNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "setting not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete setting"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "setting deleted successfully"})
}
