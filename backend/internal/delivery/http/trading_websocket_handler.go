package http

import (
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yanun0323/logs"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

// TradingStreamManager manages WebSocket connections for trading data
type TradingStreamManager struct {
	apiKeyUseCase adaptor.APIKeyUseCase
	authUseCase   adaptor.AuthUseCase
	apiKeyRepo    adaptor.APIKeyRepository

	clients map[*websocket.Conn]*ClientState
	mu      sync.RWMutex

	// Exchange connections per API Key
	exchangeConns map[string]*ExchangeConnection
	exchangeMu    sync.RWMutex

	closed bool
}

// ClientState tracks a client's subscriptions
type ClientState struct {
	UserID        string
	APIKeyID      string
	Subscriptions map[string]bool // subscription key -> active
}

// ExchangeConnection manages connection to an exchange
type ExchangeConnection struct {
	APIKeyID  string
	Platform  model.Platform
	IsTestnet bool
	APIKey    string
	APISecret string
	Config    model.ExchangeConfig

	// Public streams
	PublicWS   *websocket.Conn
	PublicSubs map[string]bool // stream name -> active

	// Private streams (orders)
	PrivateWS   *websocket.Conn
	PrivateSubs map[string]bool

	// Connected clients
	Clients map[*websocket.Conn]bool

	// BTCC specific
	btccMsgID  int64 // atomic counter for BTCC message IDs
	btccAuthed bool  // whether BTCC connection is authenticated

	mu     sync.RWMutex
	done   chan struct{}
	closed int32 // atomic flag to prevent double close
}

// BTCCRequest represents a BTCC WebSocket request message
type BTCCRequest struct {
	ID     int64       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// BTCCResponse represents a BTCC WebSocket response message
type BTCCResponse struct {
	ID     *int64          `json:"id"`
	Method string          `json:"method,omitempty"`
	Error  *BTCCError      `json:"error"`
	Result json.RawMessage `json:"result"`
	Params json.RawMessage `json:"params,omitempty"`
}

// BTCCError represents a BTCC error response
type BTCCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewTradingStreamManager(
	apiKeyUseCase adaptor.APIKeyUseCase,
	authUseCase adaptor.AuthUseCase,
	apiKeyRepo adaptor.APIKeyRepository,
) *TradingStreamManager {
	return &TradingStreamManager{
		apiKeyUseCase: apiKeyUseCase,
		authUseCase:   authUseCase,
		apiKeyRepo:    apiKeyRepo,
		clients:       make(map[*websocket.Conn]*ClientState),
		exchangeConns: make(map[string]*ExchangeConnection),
	}
}

func (m *TradingStreamManager) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *TradingStreamManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if m.isClosed() {
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}

	// Authenticate via token query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := m.authUseCase.ValidateToken(r.Context(), token)
	if err != nil || user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		conn.Close()
		return
	}
	m.clients[conn] = &ClientState{
		UserID:        user.ID,
		Subscriptions: make(map[string]bool),
	}
	m.mu.Unlock()

	log.Printf("new trading client connected: %s, userID=%s", conn.RemoteAddr().String(), user.ID)

	// Send connected confirmation
	m.sendToClient(conn, model.TradingWebSocketResponse{
		Type:      "connected",
		Timestamp: time.Now().UnixMilli(),
	})

	defer func() {
		log.Printf("remove client: %s", conn.RemoteAddr().String())
		m.removeClient(conn)
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		// Check if manager is closed
		if m.isClosed() {
			break
		}

		log.Printf("received raw message: %s", string(message))
		var msg model.TradingWebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("invalid message format: %v", err)
			m.sendError(conn, "invalid message format")
			continue
		}
		log.Printf("parsed message: action=%s, apiKeyID=%s, type=%s, symbol=%s", msg.Action, msg.APIKeyID, msg.Type, msg.Symbol)

		m.handleMessage(conn, user.ID, &msg)
	}
}

func (m *TradingStreamManager) handleMessage(conn *websocket.Conn, userID string, msg *model.TradingWebSocketMessage) {
	log.Printf("handleMessage: action=%s, apiKeyID=%s, type=%s, symbol=%s", msg.Action, msg.APIKeyID, msg.Type, msg.Symbol)
	switch msg.Action {
	case "connect":
		m.handleConnect(conn, userID, msg.APIKeyID)
	case "subscribe":
		m.handleSubscribe(conn, userID, msg)
	case "unsubscribe":
		m.handleUnsubscribe(conn, msg)
	default:
		m.sendError(conn, "unknown action: "+msg.Action)
	}
}

func (m *TradingStreamManager) handleConnect(conn *websocket.Conn, userID string, apiKeyID string) {
	log.Printf("handleConnect: userID=%s, apiKeyID=%s", userID, apiKeyID)

	// Get the API key (full, with secret)
	apiKey, err := m.apiKeyRepo.GetByID(context.Background(), apiKeyID)
	if err != nil {
		log.Printf("handleConnect: GetByID error: %v", err)
		m.sendError(conn, "API key not found")
		return
	}
	if apiKey == nil {
		log.Printf("handleConnect: API key not found for ID: %s", apiKeyID)
		m.sendError(conn, "API key not found")
		return
	}
	log.Printf("handleConnect: found apiKey: ID=%s, Name=%s, Platform=%s", apiKey.ID, apiKey.Name, apiKey.Platform)

	// // Verify ownership
	// if apiKey.UserID != userID {
	// 	m.sendError(conn, "unauthorized: API key does not belong to you")
	// 	return
	// }

	if !apiKey.IsActive {
		m.sendError(conn, "API key is not active")
		return
	}

	// Only support Binance and BTCC
	if apiKey.Platform != model.PlatformBinance && apiKey.Platform != model.PlatformBTCC {
		m.sendError(conn, "unsupported platform: "+apiKey.Platform.String())
		return
	}

	// Update client state
	m.mu.Lock()
	if state, ok := m.clients[conn]; ok {
		state.APIKeyID = apiKeyID
	}
	m.mu.Unlock()

	// Get or create exchange connection
	m.getOrCreateExchangeConn(apiKey)

	// Add client to exchange connection
	m.exchangeMu.RLock()
	if ec, ok := m.exchangeConns[apiKeyID]; ok {
		ec.mu.Lock()
		ec.Clients[conn] = true
		ec.mu.Unlock()
	}
	m.exchangeMu.RUnlock()

	log.Println("connected! send data back")
	// Send confirmation
	m.sendToClient(conn, model.TradingWebSocketResponse{
		Type:      "connected",
		Platform:  apiKey.Platform.String(),
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"apiKeyId":  apiKeyID,
			"platform":  apiKey.Platform.String(),
			"isTestnet": apiKey.IsTestnet,
			"name":      apiKey.Name,
		},
	})
}

func (m *TradingStreamManager) getOrCreateExchangeConn(apiKey *model.APIKey) *ExchangeConnection {
	m.exchangeMu.Lock()
	defer m.exchangeMu.Unlock()

	if ec, ok := m.exchangeConns[apiKey.ID]; ok {
		return ec
	}

	config := model.GetExchangeConfig(apiKey.Platform, apiKey.IsTestnet)

	ec := &ExchangeConnection{
		APIKeyID:    apiKey.ID,
		Platform:    apiKey.Platform,
		IsTestnet:   apiKey.IsTestnet,
		APIKey:      apiKey.APIKey,
		APISecret:   apiKey.APISecret,
		Config:      config,
		PublicSubs:  make(map[string]bool),
		PrivateSubs: make(map[string]bool),
		Clients:     make(map[*websocket.Conn]bool),
		done:        make(chan struct{}),
	}

	m.exchangeConns[apiKey.ID] = ec
	return ec
}

func (m *TradingStreamManager) handleSubscribe(conn *websocket.Conn, userID string, msg *model.TradingWebSocketMessage) {
	m.mu.RLock()
	state, ok := m.clients[conn]
	m.mu.RUnlock()

	if !ok || state.APIKeyID == "" {
		m.sendError(conn, "not connected to any API key, call connect first")
		return
	}

	m.exchangeMu.RLock()
	ec, ok := m.exchangeConns[state.APIKeyID]
	m.exchangeMu.RUnlock()

	if !ok {
		m.sendError(conn, "exchange connection not found")
		return
	}

	switch msg.Type {
	case "kline":
		m.subscribeKline(conn, ec, msg.Symbol, msg.Interval)
	case "orderbook", "depth":
		m.subscribeOrderBook(conn, ec, msg.Symbol)
	case "orders":
		m.subscribeOrders(conn, ec, msg.Symbol)
	case "asset":
		// Asset subscription (BTCC specific)
		if ec.Platform == model.PlatformBTCC {
			m.subscribeAsset(conn, ec)
		} else {
			m.sendError(conn, "asset subscription not supported for this platform")
		}
	case "trades", "deals":
		// Trade/deal subscription
		m.subscribeTrades(conn, ec, msg.Symbol)
	case "state":
		// Market state subscription (BTCC specific)
		if ec.Platform == model.PlatformBTCC {
			m.subscribeMarketState(conn, ec)
		} else {
			m.sendError(conn, "state subscription not supported for this platform")
		}
	default:
		m.sendError(conn, "unknown subscription type: "+msg.Type)
	}
}

// subscribeTrades subscribes to trade/deal updates
func (m *TradingStreamManager) subscribeTrades(conn *websocket.Conn, ec *ExchangeConnection, symbol string) {
	streamName := ""
	switch ec.Platform {
	case model.PlatformBinance:
		streamName = strings.ToLower(symbol) + "@trade"
	case model.PlatformBTCC:
		streamName = "deals." + symbol
	default:
		streamName = strings.ToLower(symbol) + "@trade"
	}

	ec.mu.Lock()
	if ec.PublicSubs[streamName] {
		ec.mu.Unlock()
		return
	}
	ec.PublicSubs[streamName] = true
	ec.mu.Unlock()

	if ec.Platform == model.PlatformBTCC {
		m.sendBTCCSubscription(ec, streamName, false)
	} else {
		m.updatePublicConnection(ec)
	}
}

// subscribeMarketState subscribes to market state updates (BTCC specific)
func (m *TradingStreamManager) subscribeMarketState(conn *websocket.Conn, ec *ExchangeConnection) {
	streamName := "state"

	ec.mu.Lock()
	if ec.PublicSubs[streamName] {
		ec.mu.Unlock()
		return
	}
	ec.PublicSubs[streamName] = true
	ec.mu.Unlock()

	m.sendBTCCSubscription(ec, streamName, false)
}

func (m *TradingStreamManager) subscribeKline(conn *websocket.Conn, ec *ExchangeConnection, symbol, interval string) {
	streamName := m.formatKlineStream(ec.Platform, symbol, interval)

	ec.mu.Lock()
	if ec.PublicSubs[streamName] {
		ec.mu.Unlock()
		return
	}
	ec.PublicSubs[streamName] = true
	needConnect := ec.PublicWS == nil
	ec.mu.Unlock()

	if ec.Platform == model.PlatformBTCC {
		if needConnect {
			m.updatePublicConnection(ec)
		} else {
			// BTCC: send subscription on existing connection
			m.sendBTCCSubscription(ec, streamName, false)
		}
	} else {
		// Binance: reconnect with new stream list
		m.updatePublicConnection(ec)
	}
}

func (m *TradingStreamManager) subscribeOrderBook(conn *websocket.Conn, ec *ExchangeConnection, symbol string) {
	streamName := m.formatOrderBookStream(ec.Platform, symbol)

	ec.mu.Lock()
	if ec.PublicSubs[streamName] {
		ec.mu.Unlock()
		return
	}
	ec.PublicSubs[streamName] = true
	needConnect := ec.PublicWS == nil
	ec.mu.Unlock()

	if ec.Platform == model.PlatformBTCC {
		if needConnect {
			m.updatePublicConnection(ec)
		} else {
			// BTCC: send subscription on existing connection
			m.sendBTCCSubscription(ec, streamName, false)
		}
	} else {
		// Binance: reconnect with new stream list
		m.updatePublicConnection(ec)
	}
}

func (m *TradingStreamManager) subscribeOrders(conn *websocket.Conn, ec *ExchangeConnection, symbol string) {
	// Orders require private WebSocket connection
	ec.mu.Lock()
	needConnect := ec.PrivateWS == nil
	streamName := "orders." + symbol
	ec.PrivateSubs[streamName] = true
	ec.mu.Unlock()

	if needConnect {
		m.connectPrivateStream(ec)
	} else if ec.Platform == model.PlatformBTCC && ec.btccAuthed {
		// Already connected and authenticated, just subscribe
		m.sendBTCCSubscription(ec, streamName, true)
	}
}

// subscribeAsset subscribes to asset balance updates (BTCC specific)
func (m *TradingStreamManager) subscribeAsset(conn *websocket.Conn, ec *ExchangeConnection) {
	// Asset updates require private WebSocket connection
	ec.mu.Lock()
	needConnect := ec.PrivateWS == nil
	ec.PrivateSubs["asset"] = true
	ec.mu.Unlock()

	if needConnect {
		m.connectPrivateStream(ec)
	} else if ec.Platform == model.PlatformBTCC && ec.btccAuthed {
		// Already connected and authenticated, just subscribe
		m.sendBTCCSubscription(ec, "asset", true)
	}
}

func (m *TradingStreamManager) handleUnsubscribe(conn *websocket.Conn, msg *model.TradingWebSocketMessage) {
	m.mu.RLock()
	state, ok := m.clients[conn]
	m.mu.RUnlock()

	if !ok || state.APIKeyID == "" {
		return
	}

	m.exchangeMu.RLock()
	ec, ok := m.exchangeConns[state.APIKeyID]
	m.exchangeMu.RUnlock()

	if !ok {
		return
	}

	switch msg.Type {
	case "kline":
		streamName := m.formatKlineStream(ec.Platform, msg.Symbol, msg.Interval)
		ec.mu.Lock()
		delete(ec.PublicSubs, streamName)
		ec.mu.Unlock()

		// For BTCC, send unsubscription message
		if ec.Platform == model.PlatformBTCC {
			m.sendBTCCUnsubscription(ec, streamName, false)
		}

	case "orderbook":
		streamName := m.formatOrderBookStream(ec.Platform, msg.Symbol)
		ec.mu.Lock()
		delete(ec.PublicSubs, streamName)
		ec.mu.Unlock()

		// For BTCC, send unsubscription message
		if ec.Platform == model.PlatformBTCC {
			m.sendBTCCUnsubscription(ec, streamName, false)
		}

	case "orders":
		ec.mu.Lock()
		delete(ec.PrivateSubs, msg.Symbol)
		ec.mu.Unlock()

		// For BTCC, send unsubscription message
		if ec.Platform == model.PlatformBTCC {
			streamName := "orders." + msg.Symbol
			m.sendBTCCUnsubscription(ec, streamName, true)
		}

	case "asset":
		ec.mu.Lock()
		delete(ec.PrivateSubs, "asset")
		ec.mu.Unlock()

		// For BTCC, send unsubscription message
		if ec.Platform == model.PlatformBTCC {
			m.sendBTCCUnsubscription(ec, "asset", true)
		}
	}

	// For non-BTCC platforms, reconnect to update subscriptions
	if ec.Platform != model.PlatformBTCC {
		m.updatePublicConnection(ec)
	}
}

func (m *TradingStreamManager) updatePublicConnection(ec *ExchangeConnection) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Collect all streams
	streams := make([]string, 0, len(ec.PublicSubs))
	for stream := range ec.PublicSubs {
		streams = append(streams, stream)
	}

	if len(streams) == 0 {
		if ec.PublicWS != nil {
			ec.PublicWS.Close()
			ec.PublicWS = nil
		}
		return
	}

	// Close existing connection
	if ec.PublicWS != nil {
		ec.PublicWS.Close()
	}

	// Build URL based on platform
	switch ec.Platform {
	case model.PlatformBinance:
		m.connectBinancePublic(ec, streams)
	case model.PlatformBTCC:
		m.connectBTCCPublic(ec, streams)
	default:
		m.connectBinancePublic(ec, streams)
	}
}

// connectBinancePublic connects to Binance public WebSocket
func (m *TradingStreamManager) connectBinancePublic(ec *ExchangeConnection, streams []string) {
	url := ec.Config.BaseWSURL + "/" + strings.Join(streams, "/")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Binance public ws connection error: %v", err)
		return
	}
	ec.PublicWS = ws

	// Start reading messages
	go m.readPublicMessages(ec)
}

// connectBTCCPublic connects to BTCC public WebSocket with compression support
func (m *TradingStreamManager) connectBTCCPublic(ec *ExchangeConnection, streams []string) {
	// BTCC requires per-message Deflate compression (RFC 7692)
	dialer := websocket.Dialer{
		EnableCompression: true,
	}

	ws, _, err := dialer.Dial(ec.Config.BaseWSURL, nil)
	if err != nil {
		log.Printf("BTCC public ws connection error: %v", err)
		return
	}
	ec.PublicWS = ws

	// Start reading messages first
	go m.readBTCCPublicMessages(ec)

	// Start ping goroutine for BTCC
	go m.btccPingLoop(ec, false)

	// Send subscription messages for each stream
	for stream := range ec.PublicSubs {
		m.sendBTCCSubscription(ec, stream, false)
	}
}

// sendBTCCSubscription sends a subscription message to BTCC
func (m *TradingStreamManager) sendBTCCSubscription(ec *ExchangeConnection, stream string, isPrivate bool) {
	var ws *websocket.Conn
	if isPrivate {
		ws = ec.PrivateWS
	} else {
		ws = ec.PublicWS
	}
	if ws == nil {
		return
	}

	// Parse stream type and parameters
	// Format: kline.BTCUSDT.60 or depth.BTCUSDT.10.0.01
	parts := strings.Split(stream, ".")

	var method string
	var params []interface{}

	switch parts[0] {
	case "kline":
		// kline.subscribe: [market, period]
		method = "kline.subscribe"
		if len(parts) >= 3 {
			interval, _ := strconv.Atoi(parts[2])
			params = []interface{}{parts[1], interval}
		}
	case "depth":
		// depth.subscribe: [market, limit, merge]
		method = "depth.subscribe"
		if len(parts) >= 5 {
			limit, _ := strconv.Atoi(parts[2])
			params = []interface{}{parts[1], limit, parts[3] + "." + parts[4]}
		} else if len(parts) >= 4 {
			limit, _ := strconv.Atoi(parts[2])
			params = []interface{}{parts[1], limit, parts[3]}
		} else if len(parts) >= 3 {
			limit, _ := strconv.Atoi(parts[2])
			params = []interface{}{parts[1], limit, "0.01"}
		} else if len(parts) >= 2 {
			// Default: 20 levels, "0.01" merge
			params = []interface{}{parts[1], 20, "0.01"}
		}
	case "deals":
		// deals.subscribe: [market]
		method = "deals.subscribe"
		if len(parts) >= 2 {
			params = []interface{}{parts[1]}
		}
	case "state":
		// state.subscribe: no params
		method = "state.subscribe"
		params = []interface{}{}
	case "orders":
		// orders.subscribe: [market] (optional)
		method = "orders.subscribe"
		if len(parts) >= 2 && parts[1] != "" {
			params = []interface{}{parts[1]}
		} else {
			params = []interface{}{}
		}
	case "asset":
		// asset.subscribe: no params
		method = "asset.subscribe"
		params = []interface{}{}
	default:
		log.Printf("unknown BTCC stream type: %s", parts[0])
		return
	}

	msgID := atomic.AddInt64(&ec.btccMsgID, 1)
	req := BTCCRequest{
		ID:     msgID,
		Method: method,
		Params: params,
	}

	log.Printf("BTCC subscription: sending method=%s, params=%v, id=%d, isPrivate=%v", method, params, msgID, isPrivate)
	if err := ws.WriteJSON(req); err != nil {
		logs.Errorf("BTCC subscribe error for %s: %v", stream, err)
	}
}

// sendBTCCUnsubscription sends an unsubscription message to BTCC
func (m *TradingStreamManager) sendBTCCUnsubscription(ec *ExchangeConnection, stream string, isPrivate bool) {
	var ws *websocket.Conn
	if isPrivate {
		ws = ec.PrivateWS
	} else {
		ws = ec.PublicWS
	}
	if ws == nil {
		return
	}

	parts := strings.Split(stream, ".")
	var method string

	switch parts[0] {
	case "kline":
		method = "kline.unsubscribe"
	case "depth":
		method = "depth.unsubscribe"
	case "deals":
		method = "deals.unsubscribe"
	case "state":
		method = "state.unsubscribe"
	case "orders":
		method = "orders.unsubscribe"
	case "asset":
		method = "asset.unsubscribe"
	default:
		return
	}

	msgID := atomic.AddInt64(&ec.btccMsgID, 1)
	req := BTCCRequest{
		ID:     msgID,
		Method: method,
		Params: []interface{}{},
	}

	if err := ws.WriteJSON(req); err != nil {
		log.Printf("BTCC unsubscribe error for %s: %v", stream, err)
	}
}

// btccPingLoop sends periodic ping messages to BTCC
func (m *TradingStreamManager) btccPingLoop(ec *ExchangeConnection, isPrivate bool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ec.mu.RLock()
			var ws *websocket.Conn
			if isPrivate {
				ws = ec.PrivateWS
			} else {
				ws = ec.PublicWS
			}
			ec.mu.RUnlock()

			if ws == nil {
				return
			}

			msgID := atomic.AddInt64(&ec.btccMsgID, 1)
			req := BTCCRequest{
				ID:     msgID,
				Method: "server.ping",
				Params: []interface{}{},
			}
			if err := ws.WriteJSON(req); err != nil {
				log.Printf("BTCC ping error: %v", err)
				return
			}
		case <-ec.done:
			return
		}
	}
}

func (m *TradingStreamManager) readPublicMessages(ec *ExchangeConnection) {
	ec.mu.RLock()
	ws := ec.PublicWS
	ec.mu.RUnlock()

	if ws == nil {
		return
	}

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("public ws read error: %v", err)
			}
			return
		}

		// Parse and broadcast based on platform
		m.handlePublicMessage(ec, message)
	}
}

// readBTCCPublicMessages reads messages from BTCC public WebSocket
// BTCC may send compressed messages, so we handle decompression
func (m *TradingStreamManager) readBTCCPublicMessages(ec *ExchangeConnection) {
	ec.mu.RLock()
	ws := ec.PublicWS
	ec.mu.RUnlock()

	if ws == nil {
		return
	}

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("BTCC public ws read error: %v", err)
			}
			return
		}

		// Handle binary messages (compressed)
		if messageType == websocket.BinaryMessage {
			decompressed, err := m.decompressFlate(message)
			if err != nil {
				log.Printf("BTCC decompress error: %v", err)
				continue
			}
			message = decompressed
		}

		m.handleBTCCPublicMessage(ec, message)
	}
}

// decompressFlate decompresses a flate-compressed message
func (m *TradingStreamManager) decompressFlate(data []byte) ([]byte, error) {
	reader := flate.NewReader(io.NopCloser(strings.NewReader(string(data))))
	defer reader.Close()

	var result strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return []byte(result.String()), nil
}

// handleBTCCPublicMessage handles messages from BTCC public WebSocket
func (m *TradingStreamManager) handleBTCCPublicMessage(ec *ExchangeConnection, message []byte) {
	var btccResp BTCCResponse
	if err := json.Unmarshal(message, &btccResp); err != nil {
		log.Printf("BTCC parse error: %v, message: %s", err, string(message))
		return
	}

	// Handle error responses
	if btccResp.Error != nil {
		logs.Errorf("BTCC error: code=%d, message=%s", btccResp.Error.Code, btccResp.Error.Message)
		return
	}

	// Handle push notifications (id is null)
	if btccResp.ID == nil && btccResp.Method != "" {
		m.handleBTCCPushNotification(ec, btccResp.Method, btccResp.Params)
		return
	}

	// Handle request responses (id is not null)
	// These are typically subscription confirmations, we can log them
	if btccResp.ID != nil {
		logs.Infof("BTCC response for id %d: %s", *btccResp.ID, string(btccResp.Result))
	}
}

func (m *TradingStreamManager) handlePublicMessage(ec *ExchangeConnection, message []byte) {
	var response model.TradingWebSocketResponse
	response.Platform = ec.Platform.String()
	response.Timestamp = time.Now().UnixMilli()

	switch ec.Platform {
	case model.PlatformBinance:
		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			return
		}

		// Handle combined stream format
		if stream, ok := data["stream"].(string); ok {
			if streamData, ok := data["data"].(map[string]interface{}); ok {
				if strings.Contains(stream, "@kline") {
					response.Type = "kline"
					response.Data = streamData
					if s, ok := streamData["s"].(string); ok {
						response.Symbol = s
					}
				} else if strings.Contains(stream, "@depth") {
					response.Type = "orderbook"
					response.Data = m.parseOrderBookData(streamData)
					parts := strings.Split(stream, "@")
					if len(parts) > 0 {
						response.Symbol = strings.ToUpper(parts[0])
					}
				}
			}
		} else if eventType, ok := data["e"].(string); ok {
			// Direct event format
			switch eventType {
			case "kline":
				response.Type = "kline"
				response.Data = data
				if s, ok := data["s"].(string); ok {
					response.Symbol = s
				}
			case "depthUpdate":
				response.Type = "orderbook"
				response.Data = m.parseOrderBookData(data)
				if s, ok := data["s"].(string); ok {
					response.Symbol = s
				}
			}
		}

	case model.PlatformBTCC:
		// BTCC messages are handled by handleBTCCPublicMessage
		return
	}

	if response.Type != "" {
		m.broadcastToClients(ec, response)
	}
}

// handleBTCCPushNotification handles BTCC push notifications
func (m *TradingStreamManager) handleBTCCPushNotification(ec *ExchangeConnection, method string, params json.RawMessage) {
	var response model.TradingWebSocketResponse
	response.Platform = ec.Platform.String()
	response.Timestamp = time.Now().UnixMilli()

	switch method {
	case "kline.update":
		// Params: array of kline rows [[timestamp, open, close, high, low, volume, amount, market], ...]
		var klines [][]interface{}
		if err := json.Unmarshal(params, &klines); err != nil {
			log.Printf("BTCC kline.update parse error: %v", err)
			return
		}
		for _, kline := range klines {
			response.Type = "kline"
			response.Data = m.parseBTCCKline(kline)
			if len(kline) >= 8 {
				if market, ok := kline[7].(string); ok {
					response.Symbol = market
				}
			}
			m.broadcastToClients(ec, response)
		}
		return

	case "depth.update":
		// Params: [isFullSnapshot (bool), depthData (object), market (string)]
		var depthParams []json.RawMessage
		if err := json.Unmarshal(params, &depthParams); err != nil {
			log.Printf("BTCC depth.update parse error: %v", err)
			return
		}
		if len(depthParams) < 2 {
			return
		}

		var isFullSnapshot bool
		if err := json.Unmarshal(depthParams[0], &isFullSnapshot); err != nil {
			return
		}

		var depthData map[string]interface{}
		if err := json.Unmarshal(depthParams[1], &depthData); err != nil {
			return
		}

		var market string
		if len(depthParams) >= 3 {
			if err := json.Unmarshal(depthParams[2], &market); err == nil {
				response.Symbol = market
			}
		}

		response.Type = "orderbook"
		response.Data = m.parseBTCCDepth(depthData, isFullSnapshot)

	case "deals.update":
		// Params: [market, [deals...]]
		var dealParams []json.RawMessage
		if err := json.Unmarshal(params, &dealParams); err != nil {
			log.Printf("BTCC deals.update parse error: %v", err)
			return
		}
		if len(dealParams) < 2 {
			return
		}

		var market string
		if err := json.Unmarshal(dealParams[0], &market); err == nil {
			response.Symbol = market
		}

		response.Type = "trades"
		response.Data = dealParams[1]

	case "state.update":
		// Market status update
		response.Type = "state"
		response.Data = params

	case "order.update":
		// Order update (private)
		var orderParams []json.RawMessage
		if err := json.Unmarshal(params, &orderParams); err != nil {
			log.Printf("BTCC order.update parse error: %v", err)
			return
		}
		if len(orderParams) < 1 {
			return
		}

		var orderData map[string]interface{}
		if err := json.Unmarshal(orderParams[0], &orderData); err != nil {
			return
		}

		response.Type = "order"
		response.Data = m.parseBTCCOrder(orderData)
		if market, ok := orderData["market"].(string); ok {
			response.Symbol = market
		}

	case "asset.update":
		// Asset balance update (private)
		response.Type = "asset"
		response.Data = params

	default:
		// Ignore unknown methods
		return
	}

	if response.Type != "" {
		m.broadcastToClients(ec, response)
	}
}

// parseBTCCKline parses a BTCC kline array into a structured format
// Format: [timestamp, open, close, high, low, volume, amount, market]
func (m *TradingStreamManager) parseBTCCKline(kline []interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if len(kline) >= 8 {
		result["timestamp"] = kline[0]
		result["open"] = kline[1]
		result["close"] = kline[2]
		result["high"] = kline[3]
		result["low"] = kline[4]
		result["volume"] = kline[5]
		result["amount"] = kline[6]
		result["market"] = kline[7]
	}
	return result
}

// parseBTCCDepth parses BTCC depth data into OrderBook format
func (m *TradingStreamManager) parseBTCCDepth(data map[string]interface{}, isFullSnapshot bool) *model.OrderBook {
	ob := &model.OrderBook{
		Timestamp: time.Now().UnixMilli(),
	}

	// Parse last price
	if last, ok := data["last"].(string); ok {
		// Store last price in a custom field if needed
		_ = last
	}

	// Parse timestamp
	if ts, ok := data["time"].(float64); ok {
		ob.Timestamp = int64(ts)
	}

	// Parse bids
	if bids, ok := data["bids"].([]interface{}); ok {
		ob.Bids = make([]model.OrderBookLevel, 0, len(bids))
		for _, bid := range bids {
			if level, ok := bid.([]interface{}); ok && len(level) >= 2 {
				ob.Bids = append(ob.Bids, model.OrderBookLevel{
					Price:    fmt.Sprint(level[0]),
					Quantity: fmt.Sprint(level[1]),
				})
			}
		}
		if len(ob.Bids) > 0 {
			ob.BestBid = &ob.Bids[0]
		}
	}

	// Parse asks
	if asks, ok := data["asks"].([]interface{}); ok {
		ob.Asks = make([]model.OrderBookLevel, 0, len(asks))
		for _, ask := range asks {
			if level, ok := ask.([]interface{}); ok && len(level) >= 2 {
				ob.Asks = append(ob.Asks, model.OrderBookLevel{
					Price:    fmt.Sprint(level[0]),
					Quantity: fmt.Sprint(level[1]),
				})
			}
		}
		if len(ob.Asks) > 0 {
			ob.BestAsk = &ob.Asks[0]
		}
	}

	// Calculate spread
	if ob.BestBid != nil && ob.BestAsk != nil {
		bidPrice, _ := strconv.ParseFloat(ob.BestBid.Price, 64)
		askPrice, _ := strconv.ParseFloat(ob.BestAsk.Price, 64)
		spread := askPrice - bidPrice
		ob.Spread = fmt.Sprintf("%.8f", spread)
	}

	return ob
}

// parseBTCCOrder parses BTCC order data
func (m *TradingStreamManager) parseBTCCOrder(data map[string]interface{}) *model.Order {
	order := &model.Order{
		Platform: model.PlatformBTCC,
	}

	if v, ok := data["id"].(float64); ok {
		order.OrderID = fmt.Sprintf("%.0f", v)
	}
	if v, ok := data["market"].(string); ok {
		order.Symbol = v
	}
	if v, ok := data["side"].(float64); ok {
		// BTCC: 1=buy, 2=sell
		if int(v) == 1 {
			order.Side = "BUY"
		} else {
			order.Side = "SELL"
		}
	}
	if v, ok := data["type"].(float64); ok {
		// BTCC: 1=limit, 2=market
		if int(v) == 1 {
			order.Type = "LIMIT"
		} else {
			order.Type = "MARKET"
		}
	}
	if v, ok := data["price"].(string); ok {
		order.Price = v
	}
	if v, ok := data["amount"].(string); ok {
		order.Quantity = v
	}
	if v, ok := data["deal_stock"].(string); ok {
		order.ExecutedQty = v
	}
	if v, ok := data["left"].(string); ok {
		// Calculate status based on left amount
		leftQty, _ := strconv.ParseFloat(v, 64)
		if leftQty == 0 {
			order.Status = "FILLED"
		} else {
			order.Status = "PARTIALLY_FILLED"
		}
	}
	if v, ok := data["option"].(float64); ok {
		// BTCC: 0=GTC, 8=IOC, 16=FOK
		switch int(v) {
		case 0:
			order.TimeInForce = "GTC"
		case 8:
			order.TimeInForce = "IOC"
		case 16:
			order.TimeInForce = "FOK"
		}
	}
	if v, ok := data["ctime"].(float64); ok {
		order.CreateTime = int64(v * 1000) // Convert to milliseconds
	}
	if v, ok := data["mtime"].(float64); ok {
		order.UpdateTime = int64(v * 1000)
	}

	return order
}

func (m *TradingStreamManager) parseOrderBookData(data map[string]interface{}) *model.OrderBook {
	ob := &model.OrderBook{
		Timestamp: time.Now().UnixMilli(),
	}

	if s, ok := data["s"].(string); ok {
		ob.Symbol = s
	}
	if u, ok := data["u"].(float64); ok {
		ob.LastUpdateID = int64(u)
	}

	// Parse bids
	if bids, ok := data["b"].([]interface{}); ok {
		ob.Bids = make([]model.OrderBookLevel, 0, len(bids))
		for _, bid := range bids {
			if level, ok := bid.([]interface{}); ok && len(level) >= 2 {
				ob.Bids = append(ob.Bids, model.OrderBookLevel{
					Price:    fmt.Sprint(level[0]),
					Quantity: fmt.Sprint(level[1]),
				})
			}
		}
		if len(ob.Bids) > 0 {
			ob.BestBid = &ob.Bids[0]
		}
	}

	// Parse asks
	if asks, ok := data["a"].([]interface{}); ok {
		ob.Asks = make([]model.OrderBookLevel, 0, len(asks))
		for _, ask := range asks {
			if level, ok := ask.([]interface{}); ok && len(level) >= 2 {
				ob.Asks = append(ob.Asks, model.OrderBookLevel{
					Price:    fmt.Sprint(level[0]),
					Quantity: fmt.Sprint(level[1]),
				})
			}
		}
		if len(ob.Asks) > 0 {
			ob.BestAsk = &ob.Asks[0]
		}
	}

	// Calculate spread
	if ob.BestBid != nil && ob.BestAsk != nil {
		bidPrice, _ := strconv.ParseFloat(ob.BestBid.Price, 64)
		askPrice, _ := strconv.ParseFloat(ob.BestAsk.Price, 64)
		spread := askPrice - bidPrice
		ob.Spread = fmt.Sprintf("%.8f", spread)
	}

	return ob
}

func (m *TradingStreamManager) connectPrivateStream(ec *ExchangeConnection) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if ec.PrivateWS != nil {
		return
	}

	switch ec.Platform {
	case model.PlatformBinance:
		m.connectBinancePrivate(ec)
	case model.PlatformBTCC:
		m.connectBTCCPrivate(ec)
	}
}

func (m *TradingStreamManager) connectBinancePrivate(ec *ExchangeConnection) {
	// Binance requires a listen key for user data stream
	listenKey, err := m.getBinanceListenKey(ec)
	if err != nil {
		log.Printf("failed to get Binance listen key: %v", err)
		return
	}

	url := ec.Config.BaseWSURL + "/" + listenKey
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Binance private ws connection error: %v", err)
		return
	}
	ec.PrivateWS = ws

	go m.readPrivateMessages(ec)
	go m.keepAliveListenKey(ec, listenKey)
}

func (m *TradingStreamManager) getBinanceListenKey(ec *ExchangeConnection) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	var url string
	if ec.IsTestnet {
		url = "https://testnet.binance.vision/api/v3/userDataStream"
	} else {
		url = "https://api.binance.com/api/v3/userDataStream"
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-MBX-APIKEY", ec.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ListenKey string `json:"listenKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ListenKey, nil
}

func (m *TradingStreamManager) keepAliveListenKey(ec *ExchangeConnection, listenKey string) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.pingBinanceListenKey(ec, listenKey)
		case <-ec.done:
			return
		}
	}
}

func (m *TradingStreamManager) pingBinanceListenKey(ec *ExchangeConnection, listenKey string) {
	client := &http.Client{Timeout: 10 * time.Second}

	var url string
	if ec.IsTestnet {
		url = "https://testnet.binance.vision/api/v3/userDataStream?listenKey=" + listenKey
	} else {
		url = "https://api.binance.com/api/v3/userDataStream?listenKey=" + listenKey
	}

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("X-MBX-APIKEY", ec.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (m *TradingStreamManager) connectBTCCPrivate(ec *ExchangeConnection) {
	// BTCC private connection with compression support
	dialer := websocket.Dialer{
		EnableCompression: true,
	}

	log.Printf("BTCC private: connecting to %s", ec.Config.BaseWSURL)
	ws, _, err := dialer.Dial(ec.Config.BaseWSURL, nil)
	if err != nil {
		log.Printf("BTCC private ws connection error: %v", err)
		return
	}
	ec.PrivateWS = ws
	log.Printf("BTCC private: connected successfully")

	// BTCC uses server.accessid_auth for OpenAPI authentication
	// Parameters: [access_id, sha256_of_secret_key]
	// The secret key should be hashed with SHA256 and rendered as 64-char hex string
	signature := m.signBTCCAccessKey(ec.APISecret)

	// Debug logging for auth troubleshooting
	secretLen := len(ec.APISecret)
	sigPrefix := signature
	if len(sigPrefix) > 8 {
		sigPrefix = sigPrefix[:8] + "..."
	}
	log.Printf("BTCC private: auth debug - access_id=%s, secret_len=%d, sig_prefix=%s", ec.APIKey, secretLen, sigPrefix)

	msgID := atomic.AddInt64(&ec.btccMsgID, 1)
	authReq := BTCCRequest{
		ID:     msgID,
		Method: "server.accessid_auth",
		Params: []interface{}{ec.APIKey, signature},
	}

	log.Printf("BTCC private: sending auth request, id=%d, access_id=%s", msgID, ec.APIKey)
	if err := ws.WriteJSON(authReq); err != nil {
		log.Printf("BTCC auth error: %v", err)
		ws.Close()
		ec.PrivateWS = nil
		return
	}
	log.Printf("BTCC private: auth request sent")

	// Mark as authenticated (will be confirmed by response)
	ec.btccAuthed = false

	// Start reading private messages
	go m.readBTCCPrivateMessages(ec)

	// Start ping loop for private connection
	go m.btccPingLoop(ec, true)
}

// signBTCCAccessKey generates SHA256 hash of the secret key for BTCC authentication
// Returns a 64-character hex string
func (m *TradingStreamManager) signBTCCAccessKey(secretKey string) string {
	hash := sha256.Sum256([]byte(secretKey))
	return hex.EncodeToString(hash[:])
}

// readBTCCPrivateMessages reads messages from BTCC private WebSocket
func (m *TradingStreamManager) readBTCCPrivateMessages(ec *ExchangeConnection) {
	ec.mu.RLock()
	ws := ec.PrivateWS
	ec.mu.RUnlock()

	if ws == nil {
		return
	}

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("BTCC private ws read error: %v", err)
			}
			return
		}

		// Handle binary messages (compressed)
		if messageType == websocket.BinaryMessage {
			decompressed, err := m.decompressFlate(message)
			if err != nil {
				log.Printf("BTCC decompress error: %v", err)
				continue
			}
			message = decompressed
		}

		m.handleBTCCPrivateMessage(ec, message)
	}
}

// handleBTCCPrivateMessage handles messages from BTCC private WebSocket
func (m *TradingStreamManager) handleBTCCPrivateMessage(ec *ExchangeConnection, message []byte) {
	log.Printf("BTCC private raw message: %s", string(message))

	var btccResp BTCCResponse
	if err := json.Unmarshal(message, &btccResp); err != nil {
		log.Printf("BTCC private parse error: %v", err)
		return
	}

	// Handle error responses
	if btccResp.Error != nil {
		logs.Errorf("BTCC private error: code=%d, message=%s, id=%v, method=%s", btccResp.Error.Code, btccResp.Error.Message, btccResp.ID, btccResp.Method)

		// Broadcast error to clients
		m.broadcastToClients(ec, model.TradingWebSocketResponse{
			Type:      "error",
			Platform:  ec.Platform.String(),
			Error:     btccResp.Error.Message,
			Timestamp: time.Now().UnixMilli(),
		})
		return
	}

	// Handle authentication response
	if btccResp.ID != nil && btccResp.Result != nil {
		var authResult struct {
			Status string `json:"status"`
			Flag   int64  `json:"flag"`
		}
		if err := json.Unmarshal(btccResp.Result, &authResult); err == nil {
			if authResult.Status == "success" {
				ec.mu.Lock()
				ec.btccAuthed = true
				ec.mu.Unlock()
				log.Printf("BTCC authentication successful, user flag: %d", authResult.Flag)

				// Subscribe to private channels after authentication
				for sub := range ec.PrivateSubs {
					m.sendBTCCSubscription(ec, sub, true)
				}
				return
			}
		}
	}

	// Handle push notifications (id is null)
	if btccResp.ID == nil && btccResp.Method != "" {
		m.handleBTCCPushNotification(ec, btccResp.Method, btccResp.Params)
		return
	}
}

func (m *TradingStreamManager) readPrivateMessages(ec *ExchangeConnection) {
	ec.mu.RLock()
	ws := ec.PrivateWS
	ec.mu.RUnlock()

	if ws == nil {
		return
	}

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("private ws read error: %v", err)
			}
			return
		}

		m.handlePrivateMessage(ec, message)
	}
}

func (m *TradingStreamManager) handlePrivateMessage(ec *ExchangeConnection, message []byte) {
	var response model.TradingWebSocketResponse
	response.Platform = ec.Platform.String()
	response.Timestamp = time.Now().UnixMilli()

	switch ec.Platform {
	case model.PlatformBinance:
		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			return
		}

		if eventType, ok := data["e"].(string); ok {
			switch eventType {
			case "executionReport":
				response.Type = "order"
				order := m.parseBinanceOrder(data)
				response.Data = order
				response.Symbol = order.Symbol
			case "outboundAccountPosition":
				response.Type = "account"
				response.Data = data
			}
		}

	case model.PlatformBTCC:
		// BTCC private messages are handled by readBTCCPrivateMessages
		return
	}

	if response.Type != "" {
		m.broadcastToClients(ec, response)
	}
}

func (m *TradingStreamManager) parseBinanceOrder(data map[string]interface{}) *model.Order {
	order := &model.Order{
		Platform: model.PlatformBinance,
	}

	if v, ok := data["i"].(float64); ok {
		order.OrderID = fmt.Sprintf("%.0f", v)
	}
	if v, ok := data["s"].(string); ok {
		order.Symbol = v
	}
	if v, ok := data["S"].(string); ok {
		order.Side = v
	}
	if v, ok := data["o"].(string); ok {
		order.Type = v
	}
	if v, ok := data["p"].(string); ok {
		order.Price = v
	}
	if v, ok := data["q"].(string); ok {
		order.Quantity = v
	}
	if v, ok := data["z"].(string); ok {
		order.ExecutedQty = v
	}
	if v, ok := data["X"].(string); ok {
		order.Status = v
	}
	if v, ok := data["f"].(string); ok {
		order.TimeInForce = v
	}
	if v, ok := data["T"].(float64); ok {
		order.CreateTime = int64(v)
	}

	return order
}

func (m *TradingStreamManager) broadcastToClients(ec *ExchangeConnection, response model.TradingWebSocketResponse) {
	ec.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(ec.Clients))
	for client := range ec.Clients {
		clients = append(clients, client)
	}
	ec.mu.RUnlock()

	for _, client := range clients {
		m.sendToClient(client, response)
	}
}

func (m *TradingStreamManager) sendToClient(conn *websocket.Conn, response model.TradingWebSocketResponse) {
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("send to client error: %v", err)
	}
}

func (m *TradingStreamManager) sendError(conn *websocket.Conn, message string) {
	m.sendToClient(conn, model.TradingWebSocketResponse{
		Type:      "error",
		Error:     message,
		Timestamp: time.Now().UnixMilli(),
	})
}

func (m *TradingStreamManager) formatKlineStream(platform model.Platform, symbol, interval string) string {
	switch platform {
	case model.PlatformBinance:
		return strings.ToLower(symbol) + "@kline_" + interval
	case model.PlatformBTCC:
		// BTCC kline interval is in seconds
		// Convert common intervals: 1m=60, 5m=300, 15m=900, 1h=3600, 4h=14400, 1d=86400
		intervalSeconds := m.convertIntervalToSeconds(interval)
		return fmt.Sprintf("kline.%s.%d", symbol, intervalSeconds)
	default:
		return strings.ToLower(symbol) + "@kline_" + interval
	}
}

// convertIntervalToSeconds converts interval string to seconds for BTCC
func (m *TradingStreamManager) convertIntervalToSeconds(interval string) int {
	// Handle numeric-only input (assume seconds)
	if sec, err := strconv.Atoi(interval); err == nil {
		return sec
	}

	// Parse format like "1m", "5m", "1h", "1d"
	if len(interval) < 2 {
		return 60 // default 1 minute
	}

	value, err := strconv.Atoi(interval[:len(interval)-1])
	if err != nil {
		return 60
	}

	unit := interval[len(interval)-1]
	switch unit {
	case 's', 'S':
		return value
	case 'm', 'M':
		return value * 60
	case 'h', 'H':
		return value * 3600
	case 'd', 'D':
		return value * 86400
	case 'w', 'W':
		return value * 604800
	default:
		return 60
	}
}

func (m *TradingStreamManager) formatOrderBookStream(platform model.Platform, symbol string) string {
	switch platform {
	case model.PlatformBinance:
		return strings.ToLower(symbol) + "@depth@100ms"
	case model.PlatformBTCC:
		// BTCC depth format: depth.MARKET.LIMIT.MERGE
		// Default: 20 levels, 0.01 merge precision
		return fmt.Sprintf("depth.%s.20.0.01", symbol)
	default:
		return strings.ToLower(symbol) + "@depth@100ms"
	}
}

func (m *TradingStreamManager) removeClient(conn *websocket.Conn) {
	m.mu.Lock()
	state := m.clients[conn]
	delete(m.clients, conn)
	m.mu.Unlock()

	if state != nil && state.APIKeyID != "" {
		var shouldCleanup bool
		var apiKeyID string

		m.exchangeMu.RLock()
		if ec, ok := m.exchangeConns[state.APIKeyID]; ok {
			ec.mu.Lock()
			delete(ec.Clients, conn)
			clientCount := len(ec.Clients)
			ec.mu.Unlock()

			// Mark for cleanup if no more clients
			if clientCount == 0 {
				shouldCleanup = true
				apiKeyID = state.APIKeyID
			}
		}
		m.exchangeMu.RUnlock()

		// Cleanup outside of RLock to avoid deadlock
		if shouldCleanup {
			m.cleanupExchangeConn(apiKeyID)
		}
	}
}

func (m *TradingStreamManager) cleanupExchangeConn(apiKeyID string) {
	m.exchangeMu.Lock()
	ec, ok := m.exchangeConns[apiKeyID]
	if !ok {
		m.exchangeMu.Unlock()
		return
	}
	delete(m.exchangeConns, apiKeyID)
	m.exchangeMu.Unlock()

	// Close connections outside of lock
	if atomic.CompareAndSwapInt32(&ec.closed, 0, 1) {
		close(ec.done)
	}
	if ec.PublicWS != nil {
		ec.PublicWS.Close()
	}
	if ec.PrivateWS != nil {
		ec.PrivateWS.Close()
	}
}

func (m *TradingStreamManager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	m.mu.Unlock()

	log.Println("TradingStreamManager: closing all connections...")

	// Collect all exchange connections to close
	m.exchangeMu.Lock()
	exchangeConns := make([]*ExchangeConnection, 0, len(m.exchangeConns))
	for apiKeyID, ec := range m.exchangeConns {
		log.Printf("TradingStreamManager: will close exchange connection for apiKeyID=%s", apiKeyID)
		exchangeConns = append(exchangeConns, ec)
	}
	m.exchangeConns = make(map[string]*ExchangeConnection)
	m.exchangeMu.Unlock()

	// Close exchange connections outside of lock
	for _, ec := range exchangeConns {
		if atomic.CompareAndSwapInt32(&ec.closed, 0, 1) {
			close(ec.done)
		}
		if ec.PublicWS != nil {
			ec.PublicWS.Close()
		}
		if ec.PrivateWS != nil {
			ec.PrivateWS.Close()
		}
	}
	log.Println("TradingStreamManager: exchange connections closed")

	// Collect all client connections to close
	m.mu.Lock()
	clients := make([]*websocket.Conn, 0, len(m.clients))
	for conn := range m.clients {
		clients = append(clients, conn)
	}
	m.clients = make(map[*websocket.Conn]*ClientState)
	m.mu.Unlock()

	// Close client connections outside of lock
	for _, conn := range clients {
		conn.Close()
	}

	log.Println("TradingStreamManager: all connections closed")
}
