package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const (
	btccMarketListProd = "https://spotapi2.btcccdn.com/btcc_api_trade/market/list"
	btccMarketListDev  = "https://spot.cryptouat.com:9910/btcc_api_trade/market/list"
)

type BTCCProxyHandler struct {
	httpClient *http.Client
}

func NewBTCCProxyHandler() *BTCCProxyHandler {
	return &BTCCProxyHandler{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetMarketList proxies the market list request to BTCC API
func (h *BTCCProxyHandler) GetMarketList(w http.ResponseWriter, r *http.Request) {
	// Check if testnet parameter is provided
	testnet := r.URL.Query().Get("testnet") == "true"

	var targetURL string
	if testnet {
		targetURL = btccMarketListDev
	} else {
		targetURL = btccMarketListProd
	}

	resp, err := h.httpClient.Get(targetURL)
	if err != nil {
		WriteJSON(w, http.StatusBadGateway, ErrorResponse{Error: "failed to fetch market list: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		WriteJSON(w, resp.StatusCode, ErrorResponse{Error: "BTCC API returned error"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "failed to read response"})
		return
	}

	// Parse and forward the response
	var result json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		WriteJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "invalid JSON response from BTCC"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
