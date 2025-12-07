package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

// TradingStreamManager manages WebSocket connections for trading data
type TradingStreamManager struct {
	apiKeyUseCase adaptor.APIKeyUseCase
	authUseCase   adaptor.AuthUseCase
	apiKeyRepo    adaptor.APIKeyRepository

	clients       map[*websocket.Conn]*ClientState
	mu            sync.RWMutex

	// Exchange connections per API Key
	exchangeConns map[int64]*ExchangeConnection
	exchangeMu    sync.RWMutex
}

// ClientState tracks a client's subscriptions
type ClientState struct {
	UserID        int64
	APIKeyID      int64
	Subscriptions map[string]bool // subscription key -> active
}

// ExchangeConnection manages connection to an exchange
type ExchangeConnection struct {
	APIKeyID      int64
	Platform      model.Platform
	IsTestnet     bool
	APIKey        string
	APISecret     string
	Config        model.ExchangeConfig

	// Public streams
	PublicWS      *websocket.Conn
	PublicSubs    map[string]bool // stream name -> active

	// Private streams (orders)
	PrivateWS     *websocket.Conn
	PrivateSubs   map[string]bool

	// Connected clients
	Clients       map[*websocket.Conn]bool

	mu            sync.RWMutex
	done          chan struct{}
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
		exchangeConns: make(map[int64]*ExchangeConnection),
	}
}

func (m *TradingStreamManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
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
	m.clients[conn] = &ClientState{
		UserID:        user.ID,
		Subscriptions: make(map[string]bool),
	}
	m.mu.Unlock()

	// Send connected confirmation
	m.sendToClient(conn, model.TradingWebSocketResponse{
		Type:      "connected",
		Timestamp: time.Now().UnixMilli(),
	})

	defer func() {
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

		var msg model.TradingWebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("invalid message format: %v", err)
			m.sendError(conn, "invalid message format")
			continue
		}

		m.handleMessage(conn, user.ID, &msg)
	}
}

func (m *TradingStreamManager) handleMessage(conn *websocket.Conn, userID int64, msg *model.TradingWebSocketMessage) {
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

func (m *TradingStreamManager) handleConnect(conn *websocket.Conn, userID int64, apiKeyID int64) {
	// Get the API key (full, with secret)
	apiKey, err := m.apiKeyRepo.GetByID(context.Background(), apiKeyID)
	if err != nil || apiKey == nil {
		m.sendError(conn, "API key not found")
		return
	}

	// Verify ownership
	if apiKey.UserID != userID {
		m.sendError(conn, "unauthorized: API key does not belong to you")
		return
	}

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

func (m *TradingStreamManager) handleSubscribe(conn *websocket.Conn, userID int64, msg *model.TradingWebSocketMessage) {
	m.mu.RLock()
	state, ok := m.clients[conn]
	m.mu.RUnlock()

	if !ok || state.APIKeyID == 0 {
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
	case "orderbook":
		m.subscribeOrderBook(conn, ec, msg.Symbol)
	case "orders":
		m.subscribeOrders(conn, ec, msg.Symbol)
	default:
		m.sendError(conn, "unknown subscription type: "+msg.Type)
	}
}

func (m *TradingStreamManager) subscribeKline(conn *websocket.Conn, ec *ExchangeConnection, symbol, interval string) {
	streamName := m.formatKlineStream(ec.Platform, symbol, interval)

	ec.mu.Lock()
	if ec.PublicSubs[streamName] {
		ec.mu.Unlock()
		return
	}
	ec.PublicSubs[streamName] = true
	ec.mu.Unlock()

	m.updatePublicConnection(ec)
}

func (m *TradingStreamManager) subscribeOrderBook(conn *websocket.Conn, ec *ExchangeConnection, symbol string) {
	streamName := m.formatOrderBookStream(ec.Platform, symbol)

	ec.mu.Lock()
	if ec.PublicSubs[streamName] {
		ec.mu.Unlock()
		return
	}
	ec.PublicSubs[streamName] = true
	ec.mu.Unlock()

	m.updatePublicConnection(ec)
}

func (m *TradingStreamManager) subscribeOrders(conn *websocket.Conn, ec *ExchangeConnection, symbol string) {
	// Orders require private WebSocket connection
	ec.mu.Lock()
	if ec.PrivateWS == nil {
		ec.mu.Unlock()
		m.connectPrivateStream(ec)
		ec.mu.Lock()
	}
	ec.PrivateSubs[symbol] = true
	ec.mu.Unlock()
}

func (m *TradingStreamManager) handleUnsubscribe(conn *websocket.Conn, msg *model.TradingWebSocketMessage) {
	m.mu.RLock()
	state, ok := m.clients[conn]
	m.mu.RUnlock()

	if !ok || state.APIKeyID == 0 {
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
	case "orderbook":
		streamName := m.formatOrderBookStream(ec.Platform, msg.Symbol)
		ec.mu.Lock()
		delete(ec.PublicSubs, streamName)
		ec.mu.Unlock()
	case "orders":
		ec.mu.Lock()
		delete(ec.PrivateSubs, msg.Symbol)
		ec.mu.Unlock()
	}

	m.updatePublicConnection(ec)
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
	var url string
	switch ec.Platform {
	case model.PlatformBinance:
		url = ec.Config.BaseWSURL + "/" + strings.Join(streams, "/")
	case model.PlatformBTCC:
		// BTCC uses a different format - single connection with subscription messages
		url = ec.Config.BaseWSURL
	default:
		url = ec.Config.BaseWSURL + "/" + strings.Join(streams, "/")
	}

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("public ws connection error for %s: %v", ec.Platform, err)
		return
	}
	ec.PublicWS = ws

	// For BTCC, send subscription messages after connecting
	if ec.Platform == model.PlatformBTCC {
		for stream := range ec.PublicSubs {
			subMsg := map[string]interface{}{
				"method": "subscribe",
				"params": []string{stream},
			}
			if err := ws.WriteJSON(subMsg); err != nil {
				log.Printf("BTCC subscribe error: %v", err)
			}
		}
	}

	// Start reading messages
	go m.readPublicMessages(ec)
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
		// Parse BTCC format (adjust based on actual API)
		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			return
		}
		// Handle BTCC-specific message format
		if channel, ok := data["channel"].(string); ok {
			if strings.Contains(channel, "kline") {
				response.Type = "kline"
				response.Data = data["data"]
			} else if strings.Contains(channel, "depth") || strings.Contains(channel, "orderbook") {
				response.Type = "orderbook"
				response.Data = data["data"]
			}
		}
	}

	if response.Type != "" {
		m.broadcastToClients(ec, response)
	}
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
	// BTCC private connection with signature
	ws, _, err := websocket.DefaultDialer.Dial(ec.Config.BaseWSURL, nil)
	if err != nil {
		log.Printf("BTCC private ws connection error: %v", err)
		return
	}
	ec.PrivateWS = ws

	// Send auth message
	timestamp := time.Now().UnixMilli()
	signature := m.signBTCC(ec.APISecret, timestamp)

	authMsg := map[string]interface{}{
		"method": "auth",
		"params": map[string]interface{}{
			"apiKey":    ec.APIKey,
			"timestamp": timestamp,
			"signature": signature,
		},
	}
	if err := ws.WriteJSON(authMsg); err != nil {
		log.Printf("BTCC auth error: %v", err)
		ws.Close()
		ec.PrivateWS = nil
		return
	}

	go m.readPrivateMessages(ec)
}

func (m *TradingStreamManager) signBTCC(secret string, timestamp int64) string {
	message := fmt.Sprintf("%d", timestamp)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
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
		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			return
		}
		if channel, ok := data["channel"].(string); ok {
			if strings.Contains(channel, "order") {
				response.Type = "order"
				response.Data = data["data"]
			}
		}
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
		return "kline." + symbol + "." + interval
	default:
		return strings.ToLower(symbol) + "@kline_" + interval
	}
}

func (m *TradingStreamManager) formatOrderBookStream(platform model.Platform, symbol string) string {
	switch platform {
	case model.PlatformBinance:
		return strings.ToLower(symbol) + "@depth@100ms"
	case model.PlatformBTCC:
		return "depth." + symbol
	default:
		return strings.ToLower(symbol) + "@depth@100ms"
	}
}

func (m *TradingStreamManager) removeClient(conn *websocket.Conn) {
	m.mu.Lock()
	state := m.clients[conn]
	delete(m.clients, conn)
	m.mu.Unlock()

	if state != nil && state.APIKeyID != 0 {
		m.exchangeMu.RLock()
		if ec, ok := m.exchangeConns[state.APIKeyID]; ok {
			ec.mu.Lock()
			delete(ec.Clients, conn)
			clientCount := len(ec.Clients)
			ec.mu.Unlock()

			// If no more clients, cleanup exchange connection
			if clientCount == 0 {
				m.cleanupExchangeConn(state.APIKeyID)
			}
		}
		m.exchangeMu.RUnlock()
	}
}

func (m *TradingStreamManager) cleanupExchangeConn(apiKeyID int64) {
	m.exchangeMu.Lock()
	defer m.exchangeMu.Unlock()

	if ec, ok := m.exchangeConns[apiKeyID]; ok {
		close(ec.done)
		if ec.PublicWS != nil {
			ec.PublicWS.Close()
		}
		if ec.PrivateWS != nil {
			ec.PrivateWS.Close()
		}
		delete(m.exchangeConns, apiKeyID)
	}
}

func (m *TradingStreamManager) Close() {
	m.mu.Lock()
	for conn := range m.clients {
		conn.Close()
	}
	m.clients = make(map[*websocket.Conn]*ClientState)
	m.mu.Unlock()

	m.exchangeMu.Lock()
	for _, ec := range m.exchangeConns {
		close(ec.done)
		if ec.PublicWS != nil {
			ec.PublicWS.Close()
		}
		if ec.PrivateWS != nil {
			ec.PrivateWS.Close()
		}
	}
	m.exchangeConns = make(map[int64]*ExchangeConnection)
	m.exchangeMu.Unlock()
}
