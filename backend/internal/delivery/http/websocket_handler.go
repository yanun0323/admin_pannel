package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"control_page/internal/model"
)

const (
	clientPingIntervalKline = 25 * time.Second
	clientPongWaitKline     = 60 * time.Second
	clientWriteWaitKline    = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// BinanceStreamManager manages WebSocket connections to Binance
type BinanceStreamManager struct {
	binanceURL string
	clients    map[*websocket.Conn]map[string]bool // client -> subscriptions
	binanceWS  *websocket.Conn
	mu         sync.RWMutex
	subMu      sync.Mutex
	done       chan struct{}
	closed     bool
}

func NewBinanceStreamManager(binanceURL string) *BinanceStreamManager {
	return &BinanceStreamManager{
		binanceURL: binanceURL,
		clients:    make(map[*websocket.Conn]map[string]bool),
		done:       make(chan struct{}),
	}
}

func (m *BinanceStreamManager) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func (m *BinanceStreamManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if m.isClosed() {
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	// Heartbeat for frontend clients
	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(clientPongWaitKline))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(clientPongWaitKline))
	})
	stopPing := make(chan struct{})
	go clientPingLoopKline(conn, stopPing)

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		conn.Close()
		close(stopPing)
		return
	}
	m.clients[conn] = make(map[string]bool)
	m.mu.Unlock()

	defer func() {
		m.removeClient(conn)
		close(stopPing)
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
		// Refresh read deadline on any inbound frame
		_ = conn.SetReadDeadline(time.Now().Add(clientPongWaitKline))

		// Check if manager is closed
		if m.isClosed() {
			break
		}

		var msg model.WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("invalid message format: %v", err)
			continue
		}

		switch msg.Action {
		case "subscribe":
			m.handleSubscribe(conn, msg.Data)
		case "unsubscribe":
			m.handleUnsubscribe(conn, msg.Data)
		}
	}
}

func clientPingLoopKline(conn *websocket.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(clientPingIntervalKline)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			deadline := time.Now().Add(clientWriteWaitKline)
			if err := conn.SetWriteDeadline(deadline); err != nil {
				return
			}
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, deadline); err != nil {
				return
			}
		case <-stop:
			return
		}
	}
}

func (m *BinanceStreamManager) handleSubscribe(conn *websocket.Conn, sub model.KlineSubscription) {
	streamName := formatStreamName(sub.Symbol, sub.Interval)

	m.mu.Lock()
	if subs, ok := m.clients[conn]; ok {
		subs[streamName] = true
	}
	m.mu.Unlock()

	m.updateBinanceSubscriptions()
}

func (m *BinanceStreamManager) handleUnsubscribe(conn *websocket.Conn, sub model.KlineSubscription) {
	streamName := formatStreamName(sub.Symbol, sub.Interval)

	m.mu.Lock()
	if subs, ok := m.clients[conn]; ok {
		delete(subs, streamName)
	}
	m.mu.Unlock()

	m.updateBinanceSubscriptions()
}

func (m *BinanceStreamManager) removeClient(conn *websocket.Conn) {
	m.mu.Lock()
	delete(m.clients, conn)
	m.mu.Unlock()

	m.updateBinanceSubscriptions()
}

func (m *BinanceStreamManager) updateBinanceSubscriptions() {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	// Collect all unique subscriptions
	allSubs := make(map[string]bool)
	m.mu.RLock()
	for _, subs := range m.clients {
		for sub := range subs {
			allSubs[sub] = true
		}
	}
	m.mu.RUnlock()

	if len(allSubs) == 0 {
		// No subscriptions, close Binance connection
		if m.binanceWS != nil {
			m.binanceWS.Close()
			m.binanceWS = nil
		}
		return
	}

	// Build combined stream URL
	streams := make([]string, 0, len(allSubs))
	for stream := range allSubs {
		streams = append(streams, stream)
	}

	// Close existing connection
	if m.binanceWS != nil {
		m.binanceWS.Close()
	}

	// Create new connection with combined streams
	url := m.binanceURL + "/" + strings.Join(streams, "/")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("binance connection error: %v", err)
		return
	}
	m.binanceWS = ws

	// Start reading from Binance
	go m.readBinanceMessages()
}

func (m *BinanceStreamManager) readBinanceMessages() {
	ws := m.binanceWS
	if ws == nil {
		return
	}

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("binance read error: %v", err)
			}
			return
		}

		// Broadcast to all connected clients
		m.broadcast(message)
	}
}

func (m *BinanceStreamManager) broadcast(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("broadcast error: %v", err)
		}
	}
}

func (m *BinanceStreamManager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true

	// Collect connections to close
	binanceWS := m.binanceWS
	m.binanceWS = nil

	clients := make([]*websocket.Conn, 0, len(m.clients))
	for client := range m.clients {
		clients = append(clients, client)
	}
	m.clients = make(map[*websocket.Conn]map[string]bool)
	m.mu.Unlock()

	close(m.done)

	// Close binance connection outside of lock
	if binanceWS != nil {
		binanceWS.Close()
	}

	// Close all client connections outside of lock
	for _, client := range clients {
		client.Close()
	}

	log.Println("BinanceStreamManager: closed")
}

func formatStreamName(symbol, interval string) string {
	return strings.ToLower(symbol) + "@kline_" + interval
}
