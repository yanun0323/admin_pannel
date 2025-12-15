package model

// OrderBookLevel represents a single price level in the order book
type OrderBookLevel struct {
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

// OrderBook represents the order book data
type OrderBook struct {
	Symbol       string           `json:"symbol"`
	LastUpdateID int64            `json:"lastUpdateId"`
	Bids         []OrderBookLevel `json:"bids"`
	Asks         []OrderBookLevel `json:"asks"`
	BestBid      *OrderBookLevel  `json:"bestBid,omitempty"`
	BestAsk      *OrderBookLevel  `json:"bestAsk,omitempty"`
	Spread       string           `json:"spread,omitempty"`
	Timestamp    int64            `json:"timestamp"`
}

// Order represents a user's order
type Order struct {
	OrderID       string   `json:"orderId"`
	Symbol        string   `json:"symbol"`
	Side          string   `json:"side"` // BUY or SELL
	Type          string   `json:"type"` // LIMIT, MARKET, etc.
	Price         string   `json:"price"`
	Quantity      string   `json:"quantity"`
	ExecutedQty   string   `json:"executedQty"`
	Status        string   `json:"status"` // NEW, FILLED, CANCELED, etc.
	TimeInForce   string   `json:"timeInForce"`
	CreateTime    int64    `json:"createTime"`
	UpdateTime    int64    `json:"updateTime"`
	StopPrice     string   `json:"stopPrice,omitempty"`
	Platform      Platform `json:"platform"`
}

// SpreadRecord represents a single spread measurement
type SpreadRecord struct {
	Timestamp int64  `json:"timestamp"` // Unix timestamp in milliseconds
	Spread    string `json:"spread"`    // Spread value as string
	BestBid   string `json:"bestBid"`
	BestAsk   string `json:"bestAsk"`
}

// TradingWebSocketMessage represents messages for the trading WebSocket
type TradingWebSocketMessage struct {
	Action    string                `json:"action"`    // subscribe, unsubscribe, connect
	Type      string                `json:"type"`      // kline, orderbook, orders
	APIKeyID  string                `json:"apiKeyId"`  // API Key ID for private streams
	Symbol    string                `json:"symbol"`    // Trading pair
	Interval  string                `json:"interval"`  // Kline interval (1m, 5m, etc.)
}


// TradingWebSocketResponse represents response messages from the trading WebSocket
type TradingWebSocketResponse struct {
	Type      string      `json:"type"`      // kline, orderbook, orders, spread, error, connected
	Data      interface{} `json:"data"`      // The actual data
	Platform  string      `json:"platform"`  // Exchange platform
	Symbol    string      `json:"symbol"`    // Trading pair
	Timestamp int64       `json:"timestamp"` // Event timestamp
	Error     string      `json:"error,omitempty"`
}

// ExchangeConfig holds exchange-specific configuration
type ExchangeConfig struct {
	Platform     Platform `json:"platform"`
	IsTestnet    bool     `json:"isTestnet"`
	BaseWSURL    string   `json:"baseWsUrl"`
	BaseRESTURL  string   `json:"baseRestUrl"`
}

// GetBinanceConfig returns Binance WebSocket configuration
func GetBinanceConfig(isTestnet bool) ExchangeConfig {
	if isTestnet {
		return ExchangeConfig{
			Platform:    PlatformBinance,
			IsTestnet:   true,
			BaseWSURL:   "wss://testnet.binance.vision/ws",
			BaseRESTURL: "https://testnet.binance.vision/api",
		}
	}
	return ExchangeConfig{
		Platform:    PlatformBinance,
		IsTestnet:   false,
		BaseWSURL:   "wss://stream.binance.com:9443/ws",
		BaseRESTURL: "https://api.binance.com/api",
	}
}

// GetBTCCConfig returns BTCC WebSocket configuration
func GetBTCCConfig(isTestnet bool) ExchangeConfig {
	if isTestnet {
		// BTCC testnet/UAT environment
		return ExchangeConfig{
			Platform:    PlatformBTCC,
			IsTestnet:   true,
			BaseWSURL:   "wss://spot.cryptouat.com:8700/ws",
			BaseRESTURL: "https://spot.cryptouat.com:8700",
		}
	}
	// BTCC production environment
	return ExchangeConfig{
		Platform:    PlatformBTCC,
		IsTestnet:   false,
		BaseWSURL:   "wss://spotprice2.btcccdn.com/ws",
		BaseRESTURL: "https://spotapi2.btcccdn.com",
	}
}

// GetExchangeConfig returns exchange configuration based on platform
func GetExchangeConfig(platform Platform, isTestnet bool) ExchangeConfig {
	switch platform {
	case PlatformBinance:
		return GetBinanceConfig(isTestnet)
	case PlatformBTCC:
		return GetBTCCConfig(isTestnet)
	default:
		return GetBinanceConfig(isTestnet)
	}
}
