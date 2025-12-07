const WS_BASE_URL = 'ws://localhost:8080/ws';

export interface KlineData {
  s: string;  // Symbol
  t: number;  // Open time
  T: number;  // Close time
  i: string;  // Interval
  o: string;  // Open price
  c: string;  // Close price
  h: string;  // High price
  l: string;  // Low price
  v: string;  // Volume
  q: string;  // Quote volume
  n: number;  // Number of trades
  x: boolean; // Is closed
}

export interface BinanceKlineEvent {
  e: string;  // Event type
  E: number;  // Event time
  s: string;  // Symbol
  k: KlineData;
}

export interface SubscriptionMessage {
  action: 'subscribe' | 'unsubscribe';
  data: {
    symbol: string;
    interval: string;
  };
}

type MessageHandler = (data: BinanceKlineEvent) => void;
type ConnectionHandler = () => void;

class KlineWebSocket {
  private ws: WebSocket | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private connectHandlers: Set<ConnectionHandler> = new Set();
  private disconnectHandlers: Set<ConnectionHandler> = new Set();
  private reconnectTimeout: number | null = null;
  private subscriptions: Set<string> = new Set();

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.ws = new WebSocket(`${WS_BASE_URL}/kline`);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.connectHandlers.forEach(handler => handler());
      
      // Resubscribe to all previous subscriptions
      this.subscriptions.forEach(sub => {
        const [symbol, interval] = sub.split(':');
        this.sendSubscribe(symbol, interval);
      });
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        // Binance combined stream format
        if (data.stream && data.data) {
          this.messageHandlers.forEach(handler => handler(data.data));
        } else if (data.e === 'kline') {
          this.messageHandlers.forEach(handler => handler(data));
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
      }
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.disconnectHandlers.forEach(handler => handler());
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  disconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimeout) {
      return;
    }
    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectTimeout = null;
      this.connect();
    }, 3000);
  }

  subscribe(symbol: string, interval: string): void {
    const key = `${symbol}:${interval}`;
    this.subscriptions.add(key);
    
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.sendSubscribe(symbol, interval);
    }
  }

  unsubscribe(symbol: string, interval: string): void {
    const key = `${symbol}:${interval}`;
    this.subscriptions.delete(key);
    
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.sendUnsubscribe(symbol, interval);
    }
  }

  private sendSubscribe(symbol: string, interval: string): void {
    const message: SubscriptionMessage = {
      action: 'subscribe',
      data: { symbol, interval },
    };
    this.ws?.send(JSON.stringify(message));
  }

  private sendUnsubscribe(symbol: string, interval: string): void {
    const message: SubscriptionMessage = {
      action: 'unsubscribe',
      data: { symbol, interval },
    };
    this.ws?.send(JSON.stringify(message));
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onConnect(handler: ConnectionHandler): () => void {
    this.connectHandlers.add(handler);
    return () => this.connectHandlers.delete(handler);
  }

  onDisconnect(handler: ConnectionHandler): () => void {
    this.disconnectHandlers.add(handler);
    return () => this.disconnectHandlers.delete(handler);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export const klineWs = new KlineWebSocket();

// ========================================
// Trading WebSocket for multi-exchange support
// ========================================

export interface OrderBookLevel {
  price: string;
  quantity: string;
}

export interface OrderBook {
  symbol: string;
  lastUpdateId: number;
  bids: OrderBookLevel[];
  asks: OrderBookLevel[];
  bestBid?: OrderBookLevel;
  bestAsk?: OrderBookLevel;
  spread?: string;
  timestamp: number;
}

export interface Order {
  orderId: string;
  symbol: string;
  side: 'BUY' | 'SELL';
  type: string;
  price: string;
  quantity: string;
  executedQty: string;
  status: string;
  timeInForce: string;
  createTime: number;
  updateTime: number;
  stopPrice?: string;
  platform: string;
}

export interface SpreadRecord {
  timestamp: number;
  spread: string;
  bestBid: string;
  bestAsk: string;
}

export interface TradingMessage {
  action: 'connect' | 'subscribe' | 'unsubscribe';
  type?: 'kline' | 'orderbook' | 'orders';
  apiKeyId?: number;
  symbol?: string;
  interval?: string;
}

export interface TradingResponse {
  type: 'connected' | 'kline' | 'orderbook' | 'order' | 'spread' | 'error';
  data?: unknown;
  platform?: string;
  symbol?: string;
  timestamp: number;
  error?: string;
}

export interface ConnectedData {
  apiKeyId: number;
  platform: string;
  isTestnet: boolean;
  name: string;
}

type TradingMessageHandler = (response: TradingResponse) => void;
type TradingConnectionHandler = () => void;
type TradingErrorHandler = (error: string) => void;

class TradingWebSocket {
  private ws: WebSocket | null = null;
  private messageHandlers: Set<TradingMessageHandler> = new Set();
  private connectHandlers: Set<TradingConnectionHandler> = new Set();
  private disconnectHandlers: Set<TradingConnectionHandler> = new Set();
  private errorHandlers: Set<TradingErrorHandler> = new Set();
  private reconnectTimeout: number | null = null;
  private token: string | null = null;
  private pendingMessages: TradingMessage[] = [];
  private currentApiKeyId: number | null = null;

  connect(token: string): void {
    this.token = token;
    
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.ws = new WebSocket(`${WS_BASE_URL}/trading?token=${token}`);

    this.ws.onopen = () => {
      console.log('Trading WebSocket connected');
      this.connectHandlers.forEach(handler => handler());
      
      // Send pending messages
      this.pendingMessages.forEach(msg => this.send(msg));
      this.pendingMessages = [];
    };

    this.ws.onmessage = (event) => {
      try {
        const response: TradingResponse = JSON.parse(event.data);
        
        if (response.type === 'error' && response.error) {
          this.errorHandlers.forEach(handler => handler(response.error!));
        }
        
        this.messageHandlers.forEach(handler => handler(response));
      } catch (e) {
        console.error('Failed to parse Trading WebSocket message:', e);
      }
    };

    this.ws.onclose = () => {
      console.log('Trading WebSocket disconnected');
      this.disconnectHandlers.forEach(handler => handler());
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('Trading WebSocket error:', error);
    };
  }

  disconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.currentApiKeyId = null;
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimeout || !this.token) {
      return;
    }
    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectTimeout = null;
      if (this.token) {
        this.connect(this.token);
      }
    }, 3000);
  }

  private send(message: TradingMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      this.pendingMessages.push(message);
    }
  }

  connectToApiKey(apiKeyId: number): void {
    this.currentApiKeyId = apiKeyId;
    this.send({
      action: 'connect',
      apiKeyId,
    });
  }

  subscribeKline(symbol: string, interval: string): void {
    this.send({
      action: 'subscribe',
      type: 'kline',
      symbol,
      interval,
    });
  }

  unsubscribeKline(symbol: string, interval: string): void {
    this.send({
      action: 'unsubscribe',
      type: 'kline',
      symbol,
      interval,
    });
  }

  subscribeOrderBook(symbol: string): void {
    this.send({
      action: 'subscribe',
      type: 'orderbook',
      symbol,
    });
  }

  unsubscribeOrderBook(symbol: string): void {
    this.send({
      action: 'unsubscribe',
      type: 'orderbook',
      symbol,
    });
  }

  subscribeOrders(symbol: string): void {
    this.send({
      action: 'subscribe',
      type: 'orders',
      symbol,
    });
  }

  unsubscribeOrders(symbol: string): void {
    this.send({
      action: 'unsubscribe',
      type: 'orders',
      symbol,
    });
  }

  onMessage(handler: TradingMessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onConnect(handler: TradingConnectionHandler): () => void {
    this.connectHandlers.add(handler);
    return () => this.connectHandlers.delete(handler);
  }

  onDisconnect(handler: TradingConnectionHandler): () => void {
    this.disconnectHandlers.add(handler);
    return () => this.disconnectHandlers.delete(handler);
  }

  onError(handler: TradingErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  getCurrentApiKeyId(): number | null {
    return this.currentApiKeyId;
  }
}

export const tradingWs = new TradingWebSocket();
