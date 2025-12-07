package http

import (
	"net/http"

	"control_page/internal/adaptor"
)

type KlineHandler struct {
	klineUseCase adaptor.KlineUseCase
}

func NewKlineHandler(klineUseCase adaptor.KlineUseCase) *KlineHandler {
	return &KlineHandler{klineUseCase: klineUseCase}
}

func (h *KlineHandler) GetSymbols(w http.ResponseWriter, r *http.Request) {
	symbols, err := h.klineUseCase.GetAvailableSymbols(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get symbols"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Data: symbols,
	})
}

func (h *KlineHandler) GetIntervals(w http.ResponseWriter, r *http.Request) {
	intervals, err := h.klineUseCase.GetAvailableIntervals(r.Context())
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to get intervals"})
		return
	}

	WriteJSON(w, http.StatusOK, SuccessResponse{
		Data: intervals,
	})
}
