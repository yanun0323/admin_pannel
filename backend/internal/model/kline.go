package model

// Kline represents a candlestick/kline data from Binance
type Kline struct {
	Symbol       string  `json:"s"`      // Symbol
	OpenTime     int64   `json:"t"`      // Kline start time
	CloseTime    int64   `json:"T"`      // Kline close time
	Interval     string  `json:"i"`      // Interval
	Open         string  `json:"o"`      // Open price
	Close        string  `json:"c"`      // Close price
	High         string  `json:"h"`      // High price
	Low          string  `json:"l"`      // Low price
	Volume       string  `json:"v"`      // Base asset volume
	QuoteVolume  string  `json:"q"`      // Quote asset volume
	TradeCount   int64   `json:"n"`      // Number of trades
	IsClosed     bool    `json:"x"`      // Is this kline closed?
	FirstTradeID int64   `json:"f"`      // First trade ID
	LastTradeID  int64   `json:"L"`      // Last trade ID
}

// BinanceKlineEvent represents the WebSocket event from Binance
type BinanceKlineEvent struct {
	EventType string `json:"e"` // Event type
	EventTime int64  `json:"E"` // Event time
	Symbol    string `json:"s"` // Symbol
	Kline     Kline  `json:"k"` // Kline data
}

// KlineSubscription represents a client's subscription to a symbol
type KlineSubscription struct {
	Symbol   string `json:"symbol"`
	Interval string `json:"interval"`
}

// WebSocketMessage represents a message from client
type WebSocketMessage struct {
	Action string            `json:"action"` // subscribe, unsubscribe
	Data   KlineSubscription `json:"data"`
}
