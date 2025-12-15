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

type SwitcherHandler struct {
	switcherUseCase adaptor.SwitcherUseCase
}

func NewSwitcherHandler(switcherUseCase adaptor.SwitcherUseCase) *SwitcherHandler {
	return &SwitcherHandler{switcherUseCase: switcherUseCase}
}

func (h *SwitcherHandler) List(w http.ResponseWriter, r *http.Request) {
	switchers, err := h.switcherUseCase.List(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to list switchers"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: switchers})
}

func (h *SwitcherHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	switcher, err := h.switcherUseCase.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrSwitcherNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "switcher not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get switcher"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Data: switcher})
}

func (h *SwitcherHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateSwitcherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	switcher, err := h.switcherUseCase.Create(r.Context(), &req)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to create switcher"})
		return
	}

	WriteJSON(w, http.StatusCreated, SuccessResponse{
		Message: "switcher created successfully",
		Data:    switcher,
	})
}

func (h *SwitcherHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateSwitcherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	switcher, err := h.switcherUseCase.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, usecase.ErrSwitcherNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "switcher not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update switcher"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "switcher updated successfully",
		Data:    switcher,
	})
}

// UpdatePairRequest is the request structure for updating a single pair
type UpdatePairRequest struct {
	Enable bool `json:"enable"`
}

func (h *SwitcherHandler) UpdatePair(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pair := chi.URLParam(r, "pair")

	var req UpdatePairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	switcher, err := h.switcherUseCase.UpdatePair(r.Context(), id, pair, req.Enable)
	if err != nil {
		if errors.Is(err, usecase.ErrSwitcherNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "switcher not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to update pair"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Message: "pair updated successfully",
		Data:    switcher,
	})
}

func (h *SwitcherHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.switcherUseCase.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrSwitcherNotFound) {
			WriteJSON(w, http.StatusNotFound, ErrorResponse{Error: "switcher not found"})
		} else {
			WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to delete switcher"})
		}
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{Message: "switcher deleted successfully"})
}
