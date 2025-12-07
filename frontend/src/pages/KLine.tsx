import { type Component, createSignal, createEffect, onMount, onCleanup, Show, For, createMemo } from 'solid-js';
import { createChart, type IChartApi, type ISeriesApi, type CandlestickData, type Time, type IPriceLine } from 'lightweight-charts';
import { FiRefreshCw, FiWifi, FiWifiOff, FiAlertCircle } from 'solid-icons/fi';
import Layout from '../components/Layout';
import { api, type APIKeyResponse } from '../lib/api';
import { tradingWs, type TradingResponse, type OrderBook, type Order, type SpreadRecord, type ConnectedData } from '../lib/websocket';

// Spread history configuration
const SPREAD_HISTORY_INTERVAL = 100; // 100ms
const SPREAD_HISTORY_MAX_POINTS = 600; // 60 seconds of data

const KLine: Component = () => {
  // API Keys state
  const [apiKeys, setApiKeys] = createSignal<APIKeyResponse[]>([]);
  const [selectedApiKeyId, setSelectedApiKeyId] = createSignal<number | null>(null);
  const [connectedApiKey, setConnectedApiKey] = createSignal<ConnectedData | null>(null);
  
  // Market data state
  const [symbols, setSymbols] = createSignal<string[]>([]);
  const [intervals, setIntervals] = createSignal<string[]>([]);
  const [selectedSymbol, setSelectedSymbol] = createSignal('BTCUSDT');
  const [selectedInterval, setSelectedInterval] = createSignal('1m');
  
  // Connection state
  const [isConnected, setIsConnected] = createSignal(false);
  const [isApiKeyConnected, setIsApiKeyConnected] = createSignal(false);
  const [isLoading, setIsLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  
  // Price state
  const [currentPrice, setCurrentPrice] = createSignal<string | null>(null);
  const [priceChange, setPriceChange] = createSignal<'up' | 'down' | null>(null);
  
  // OrderBook state
  const [orderBook, setOrderBook] = createSignal<OrderBook | null>(null);
  
  // Orders state (active orders with price lines on chart)
  const [orders, setOrders] = createSignal<Order[]>([]);
  
  // Spread history for bottom chart
  const [spreadHistory, setSpreadHistory] = createSignal<SpreadRecord[]>([]);
  
  // Pending order count
  const pendingOrderCount = createMemo(() => 
    orders().filter(o => o.status === 'NEW' || o.status === 'PARTIALLY_FILLED').length
  );
  
  // Spread value
  const currentSpread = createMemo(() => {
    const ob = orderBook();
    if (!ob?.bestBid || !ob?.bestAsk) return null;
    return ob.spread;
  });

  // Spread history count for display
  const spreadHistoryCount = createMemo(() => spreadHistory().length);

  let chartContainer: HTMLDivElement | undefined;
  let spreadChartContainer: HTMLDivElement | undefined;
  let chart: IChartApi | null = null;
  let spreadChart: IChartApi | null = null;
  let candlestickSeries: ISeriesApi<'Candlestick'> | null = null;
  let spreadLineSeries: ISeriesApi<'Line'> | null = null;
  let orderPriceLines: Map<string, IPriceLine> = new Map();
  let lastPrice: number | null = null;
  let spreadInterval: number | null = null;

  // Filter API keys to only Binance and BTCC
  const filteredApiKeys = createMemo(() => 
    apiKeys().filter(k => 
      (k.platform === 'binance' || k.platform === 'btcc') && k.is_active
    )
  );

  onMount(async () => {
    try {
      // Load API keys and market data
      const [apiKeysRes, symbolsRes, intervalsRes] = await Promise.all([
        api.listAPIKeys(),
        api.getSymbols(),
        api.getIntervals(),
      ]);
      
      if (apiKeysRes.data) setApiKeys(apiKeysRes.data);
      if (symbolsRes.data) setSymbols(symbolsRes.data);
      if (intervalsRes.data) setIntervals(intervalsRes.data);
    } catch (e) {
      console.error('Failed to load initial data:', e);
      setError('Failed to load initial data');
    } finally {
      setIsLoading(false);
    }

    // Setup WebSocket handlers
    const token = api.getToken();
    if (token) {
      const unsubMessage = tradingWs.onMessage(handleTradingMessage);
      const unsubConnect = tradingWs.onConnect(() => setIsConnected(true));
      const unsubDisconnect = tradingWs.onDisconnect(() => {
        setIsConnected(false);
        setIsApiKeyConnected(false);
      });
      const unsubError = tradingWs.onError((err) => setError(err));

      tradingWs.connect(token);
      setIsConnected(tradingWs.isConnected());

      onCleanup(() => {
        unsubMessage();
        unsubConnect();
        unsubDisconnect();
        unsubError();
        tradingWs.disconnect();
        if (chart) {
          chart.remove();
          chart = null;
        }
        if (spreadChart) {
          spreadChart.remove();
          spreadChart = null;
        }
        if (spreadInterval) {
          clearInterval(spreadInterval);
        }
      });
    }

    // Start spread recording interval
    spreadInterval = window.setInterval(recordSpread, SPREAD_HISTORY_INTERVAL);
  });

  // Initialize main chart
  createEffect(() => {
    if (!chartContainer || chart) return;

    chart = createChart(chartContainer, {
      width: chartContainer.clientWidth,
      height: 400,
      layout: {
        background: { color: '#18181b' },
        textColor: '#71717a',
      },
      grid: {
        vertLines: { color: '#27272a' },
        horzLines: { color: '#27272a' },
      },
      crosshair: {
        mode: 0,
      },
      rightPriceScale: {
        borderColor: '#27272a',
      },
      timeScale: {
        borderColor: '#27272a',
        timeVisible: true,
        secondsVisible: false,
      },
    });

    candlestickSeries = chart.addCandlestickSeries({
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    });

    const handleResize = () => {
      if (chart && chartContainer) {
        chart.applyOptions({ width: chartContainer.clientWidth });
      }
    };

    window.addEventListener('resize', handleResize);

    onCleanup(() => {
      window.removeEventListener('resize', handleResize);
    });
  });

  // Initialize spread chart
  createEffect(() => {
    if (!spreadChartContainer || spreadChart) return;

    spreadChart = createChart(spreadChartContainer, {
      width: spreadChartContainer.clientWidth,
      height: 120,
      layout: {
        background: { color: '#18181b' },
        textColor: '#71717a',
      },
      grid: {
        vertLines: { color: '#27272a' },
        horzLines: { color: '#27272a' },
      },
      rightPriceScale: {
        borderColor: '#27272a',
      },
      timeScale: {
        borderColor: '#27272a',
        timeVisible: true,
        secondsVisible: true,
      },
    });

    spreadLineSeries = spreadChart.addLineSeries({
      color: '#f59e0b',
      lineWidth: 2,
    });

    const handleResize = () => {
      if (spreadChart && spreadChartContainer) {
        spreadChart.applyOptions({ width: spreadChartContainer.clientWidth });
      }
    };

    window.addEventListener('resize', handleResize);

    onCleanup(() => {
      window.removeEventListener('resize', handleResize);
    });
  });

  // Handle API key selection change
  createEffect(() => {
    const apiKeyId = selectedApiKeyId();
    if (apiKeyId && isConnected()) {
      tradingWs.connectToApiKey(apiKeyId);
    }
  });

  // Handle symbol/interval changes
  createEffect(() => {
    const symbol = selectedSymbol();
    const interval = selectedInterval();
    const connected = isApiKeyConnected();

    if (!symbol || !interval || !connected) return;

    // Clear existing data
    if (candlestickSeries) {
      candlestickSeries.setData([]);
    }
    clearOrderPriceLines();
    setOrders([]);
    setOrderBook(null);
    setSpreadHistory([]);
    lastPrice = null;
    setCurrentPrice(null);

    // Unsubscribe from old and subscribe to new
    tradingWs.subscribeKline(symbol, interval);
    tradingWs.subscribeOrderBook(symbol);
    tradingWs.subscribeOrders(symbol);
    
    // Fetch historical data
    fetchHistoricalData(symbol, interval);
  });

  const fetchHistoricalData = async (symbol: string, interval: string) => {
    const connected = connectedApiKey();
    if (!connected) return;

    try {
      let url: string;
      if (connected.platform === 'binance') {
        if (connected.isTestnet) {
          url = `https://testnet.binance.vision/api/v3/klines?symbol=${symbol}&interval=${interval}&limit=500`;
        } else {
          url = `https://api.binance.com/api/v3/klines?symbol=${symbol}&interval=${interval}&limit=500`;
        }
      } else {
        // BTCC - adjust URL as needed
        url = `https://api.btcc.com/api/v3/klines?symbol=${symbol}&interval=${interval}&limit=500`;
      }

      const response = await fetch(url);
      const data = await response.json();

      if (!candlestickSeries) return;

      const candlestickData: CandlestickData<Time>[] = data.map((item: (string | number)[]) => ({
        time: (Number(item[0]) / 1000) as Time,
        open: parseFloat(String(item[1])),
        high: parseFloat(String(item[2])),
        low: parseFloat(String(item[3])),
        close: parseFloat(String(item[4])),
      }));

      candlestickSeries.setData(candlestickData);

      if (candlestickData.length > 0) {
        const lastCandle = candlestickData[candlestickData.length - 1];
        lastPrice = lastCandle.close;
        setCurrentPrice(lastCandle.close.toFixed(2));
      }
    } catch (e) {
      console.error('Failed to fetch historical data:', e);
    }
  };

  const handleTradingMessage = (response: TradingResponse) => {
    switch (response.type) {
      case 'connected':
        setIsApiKeyConnected(true);
        setConnectedApiKey(response.data as ConnectedData);
        break;
      case 'kline':
        handleKlineUpdate(response);
        break;
      case 'orderbook':
        handleOrderBookUpdate(response);
        break;
      case 'order':
        handleOrderUpdate(response);
        break;
      case 'error':
        setError(response.error || 'Unknown error');
        break;
    }
  };

  const handleKlineUpdate = (response: TradingResponse) => {
    if (!candlestickSeries) return;
    if (response.symbol !== selectedSymbol()) return;

    const data = response.data as Record<string, unknown>;
    const kline = data.k as Record<string, unknown>;
    
    if (!kline) return;

    const candle: CandlestickData<Time> = {
      time: (Number(kline.t) / 1000) as Time,
      open: parseFloat(String(kline.o)),
      high: parseFloat(String(kline.h)),
      low: parseFloat(String(kline.l)),
      close: parseFloat(String(kline.c)),
    };

    candlestickSeries.update(candle);

    const newPrice = parseFloat(String(kline.c));
    if (lastPrice !== null) {
      if (newPrice > lastPrice) {
        setPriceChange('up');
      } else if (newPrice < lastPrice) {
        setPriceChange('down');
      }
      setTimeout(() => setPriceChange(null), 300);
    }
    lastPrice = newPrice;
    setCurrentPrice(newPrice.toFixed(2));
  };

  const handleOrderBookUpdate = (response: TradingResponse) => {
    if (response.symbol !== selectedSymbol()) return;
    setOrderBook(response.data as OrderBook);
  };

  const handleOrderUpdate = (response: TradingResponse) => {
    const order = response.data as Order;
    if (order.symbol !== selectedSymbol()) return;

    setOrders(prev => {
      const existing = prev.findIndex(o => o.orderId === order.orderId);
      if (existing >= 0) {
        const updated = [...prev];
        updated[existing] = order;
        return updated;
      }
      return [...prev, order];
    });

    updateOrderPriceLines();
  };

  const updateOrderPriceLines = () => {
    if (!candlestickSeries) return;

    // Remove old lines
    clearOrderPriceLines();

    // Add new lines for active orders
    orders().forEach(order => {
      if (order.status !== 'NEW' && order.status !== 'PARTIALLY_FILLED') return;
      
      const price = parseFloat(order.price);
      if (isNaN(price) || price === 0) return;

      const priceLine = candlestickSeries!.createPriceLine({
        price,
        color: order.side === 'BUY' ? '#22c55e' : '#ef4444',
        lineWidth: 1,
        lineStyle: 2, // Dashed
        axisLabelVisible: true,
        title: `${order.side} ${order.quantity}`,
      });

      orderPriceLines.set(order.orderId, priceLine);
    });
  };

  const clearOrderPriceLines = () => {
    if (!candlestickSeries) return;
    orderPriceLines.forEach((line) => {
      candlestickSeries!.removePriceLine(line);
    });
    orderPriceLines.clear();
  };

  const recordSpread = () => {
    const ob = orderBook();
    if (!ob?.bestBid || !ob?.bestAsk || !ob.spread) return;

    const record: SpreadRecord = {
      timestamp: Date.now(),
      spread: ob.spread,
      bestBid: ob.bestBid.price,
      bestAsk: ob.bestAsk.price,
    };

    setSpreadHistory(prev => {
      const updated = [...prev, record];
      if (updated.length > SPREAD_HISTORY_MAX_POINTS) {
        return updated.slice(-SPREAD_HISTORY_MAX_POINTS);
      }
      return updated;
    });

    // Update spread chart
    if (spreadLineSeries) {
      const spreadValue = parseFloat(ob.spread);
      spreadLineSeries.update({
        time: (record.timestamp / 1000) as Time,
        value: spreadValue,
      });
    }
  };

  const handleRefresh = () => {
    fetchHistoricalData(selectedSymbol(), selectedInterval());
  };

  const handleApiKeyChange = (e: Event) => {
    const value = (e.target as HTMLSelectElement).value;
    setSelectedApiKeyId(value ? parseInt(value) : null);
    setIsApiKeyConnected(false);
    setConnectedApiKey(null);
  };

  const getPlatformDisplay = (platform: string) => {
    const displays: Record<string, string> = {
      binance: 'Binance',
      btcc: 'BTCC',
    };
    return displays[platform] || platform;
  };

  return (
    <Layout>
      <div class="kline-page">
        {/* Header */}
        <header class="page-header">
          <div class="header-main">
            <div class="symbol-display">
              <h1>{selectedSymbol()}</h1>
              <Show when={currentPrice()}>
                <span class={`price ${priceChange()}`}>${currentPrice()}</span>
              </Show>
            </div>
            <div class="status-badges">
              <span class={`status ${isConnected() ? 'connected' : ''}`}>
                {isConnected() ? <FiWifi /> : <FiWifiOff />}
                {isConnected() ? 'Connected' : 'Offline'}
              </span>
              <Show when={isApiKeyConnected() && connectedApiKey()}>
                <span class="badge api-connected">
                  {getPlatformDisplay(connectedApiKey()!.platform)}
                  {connectedApiKey()!.isTestnet && ' (Testnet)'}
                </span>
              </Show>
            </div>
          </div>
          
          {/* Controls */}
          <div class="controls">
            <select
              value={selectedApiKeyId()?.toString() || ''}
              onChange={handleApiKeyChange}
              class="api-key-select"
            >
              <option value="">Select API Key...</option>
              <For each={filteredApiKeys()}>
                {(key) => (
                  <option value={key.id.toString()}>
                    {key.name} - {getPlatformDisplay(key.platform)}
                    {key.is_testnet ? ' (Testnet)' : ''}
                  </option>
                )}
              </For>
            </select>

            <select
              value={selectedSymbol()}
              onChange={(e) => setSelectedSymbol(e.currentTarget.value)}
              disabled={isLoading() || !isApiKeyConnected()}
            >
              <For each={symbols()}>
                {(symbol) => <option value={symbol}>{symbol}</option>}
              </For>
            </select>

            <select
              value={selectedInterval()}
              onChange={(e) => setSelectedInterval(e.currentTarget.value)}
              disabled={isLoading() || !isApiKeyConnected()}
            >
              <For each={intervals()}>
                {(interval) => <option value={interval}>{interval}</option>}
              </For>
            </select>

            <button class="btn-icon" onClick={handleRefresh} title="Refresh" disabled={!isApiKeyConnected()}>
              <FiRefreshCw />
            </button>
          </div>
        </header>

        {/* Error message */}
        <Show when={error()}>
          <div class="error-banner">
            <FiAlertCircle />
            {error()}
            <button onClick={() => setError(null)}>×</button>
          </div>
        </Show>

        {/* Stats bar */}
        <Show when={isApiKeyConnected()}>
          <div class="stats-bar">
            <div class="stat-item">
              <span class="stat-label">Orders</span>
              <span class="stat-value">{pendingOrderCount()}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Spread Samples</span>
              <span class="stat-value">{spreadHistoryCount()}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Best Bid</span>
              <span class="stat-value bid">{orderBook()?.bestBid?.price || '-'}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Best Ask</span>
              <span class="stat-value ask">{orderBook()?.bestAsk?.price || '-'}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Spread</span>
              <span class="stat-value spread">{currentSpread() || '-'}</span>
            </div>
          </div>
        </Show>

        {/* Main content area */}
        <div class="main-content">
          {/* Chart area */}
          <div class="chart-section">
            <Show 
              when={isApiKeyConnected()} 
              fallback={
                <div class="placeholder">
                  <p>Select an API Key to start viewing market data</p>
                </div>
              }
            >
              <div class="chart-wrapper" ref={chartContainer} />
            </Show>
          </div>

          {/* OrderBook panel */}
          <Show when={isApiKeyConnected()}>
            <div class="orderbook-panel">
              <h3>Order Book</h3>
              <div class="orderbook-content">
                <div class="orderbook-section asks">
                  <div class="orderbook-header">
                    <span>Price</span>
                    <span>Quantity</span>
                  </div>
                  <div class="orderbook-levels">
                    <For each={orderBook()?.asks?.slice(0, 10).reverse() || []}>
                      {(level) => (
                        <div class="orderbook-level ask">
                          <span class="price">{parseFloat(level.price).toFixed(2)}</span>
                          <span class="qty">{parseFloat(level.quantity).toFixed(4)}</span>
                        </div>
                      )}
                    </For>
                  </div>
                </div>
                
                <div class="orderbook-mid">
                  <span class="spread-label">Spread</span>
                  <span class="spread-value">{currentSpread() || '-'}</span>
                </div>

                <div class="orderbook-section bids">
                  <div class="orderbook-levels">
                    <For each={orderBook()?.bids?.slice(0, 10) || []}>
                      {(level) => (
                        <div class="orderbook-level bid">
                          <span class="price">{parseFloat(level.price).toFixed(2)}</span>
                          <span class="qty">{parseFloat(level.quantity).toFixed(4)}</span>
                        </div>
                      )}
                    </For>
                  </div>
                </div>
              </div>
            </div>
          </Show>
        </div>

        {/* Spread chart */}
        <Show when={isApiKeyConnected()}>
          <div class="spread-chart-section">
            <h3>Spread History (100ms intervals)</h3>
            <div class="spread-chart-wrapper" ref={spreadChartContainer} />
          </div>
        </Show>
      </div>

      <style>{`
        .kline-page {
          max-width: 100%;
        }

        .page-header {
          margin-bottom: 16px;
        }

        .header-main {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 12px;
        }

        .symbol-display {
          display: flex;
          align-items: baseline;
          gap: 12px;
        }

        .symbol-display h1 {
          font-size: 24px;
          font-weight: 600;
        }

        .price {
          font-size: 20px;
          font-weight: 600;
          transition: color 0.15s;
        }

        .price.up {
          color: var(--success);
        }

        .price.down {
          color: var(--danger);
        }

        .status-badges {
          display: flex;
          gap: 8px;
          align-items: center;
        }

        .status {
          display: flex;
          align-items: center;
          gap: 4px;
          padding: 4px 10px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 500;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          background: rgba(239, 68, 68, 0.1);
          color: var(--danger);
        }

        .status.connected {
          background: rgba(34, 197, 94, 0.1);
          color: var(--success);
        }

        .badge.api-connected {
          padding: 4px 10px;
          border-radius: 999px;
          font-size: 11px;
          font-weight: 500;
          background: rgba(59, 130, 246, 0.1);
          color: #3b82f6;
        }

        .controls {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-wrap: wrap;
        }

        .controls select {
          padding: 8px 12px;
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: var(--radius);
          color: var(--text);
          font-size: 13px;
          cursor: pointer;
          transition: border-color 0.15s;
        }

        .controls select:focus {
          outline: none;
          border-color: var(--primary);
        }

        .controls select:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .api-key-select {
          min-width: 250px;
        }

        .btn-icon {
          width: 34px;
          height: 34px;
          display: flex;
          align-items: center;
          justify-content: center;
          border: 1px solid var(--border);
          border-radius: var(--radius);
          background: var(--surface);
          color: var(--text-muted);
          transition: all 0.15s;
          cursor: pointer;
        }

        .btn-icon:hover:not(:disabled) {
          color: var(--text);
          border-color: var(--text-muted);
        }

        .btn-icon:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .error-banner {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 10px 14px;
          background: rgba(239, 68, 68, 0.1);
          border: 1px solid rgba(239, 68, 68, 0.3);
          border-radius: var(--radius);
          color: var(--danger);
          font-size: 13px;
          margin-bottom: 16px;
        }

        .error-banner button {
          margin-left: auto;
          background: none;
          border: none;
          color: var(--danger);
          cursor: pointer;
          font-size: 18px;
          padding: 0;
          line-height: 1;
        }

        .stats-bar {
          display: flex;
          gap: 24px;
          padding: 12px 16px;
          background: var(--surface);
          border-radius: var(--radius);
          margin-bottom: 16px;
        }

        .stat-item {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }

        .stat-label {
          font-size: 11px;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .stat-value {
          font-size: 14px;
          font-weight: 600;
          font-family: monospace;
        }

        .stat-value.bid {
          color: var(--success);
        }

        .stat-value.ask {
          color: var(--danger);
        }

        .stat-value.spread {
          color: #f59e0b;
        }

        .main-content {
          display: flex;
          gap: 16px;
          margin-bottom: 16px;
        }

        .chart-section {
          flex: 1;
          min-width: 0;
        }

        .chart-wrapper {
          width: 100%;
          height: 400px;
          background: var(--surface);
          border-radius: var(--radius-lg);
          overflow: hidden;
        }

        .placeholder {
          height: 400px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: var(--surface);
          border-radius: var(--radius-lg);
          color: var(--text-muted);
        }

        .orderbook-panel {
          width: 280px;
          flex-shrink: 0;
          background: var(--surface);
          border-radius: var(--radius-lg);
          padding: 12px;
        }

        .orderbook-panel h3 {
          font-size: 13px;
          font-weight: 500;
          margin-bottom: 12px;
          color: var(--text-secondary);
        }

        .orderbook-content {
          display: flex;
          flex-direction: column;
        }

        .orderbook-header {
          display: flex;
          justify-content: space-between;
          padding: 4px 0;
          font-size: 10px;
          color: var(--text-muted);
          text-transform: uppercase;
          letter-spacing: 0.5px;
        }

        .orderbook-levels {
          display: flex;
          flex-direction: column;
        }

        .orderbook-level {
          display: flex;
          justify-content: space-between;
          padding: 2px 0;
          font-size: 12px;
          font-family: monospace;
        }

        .orderbook-level.ask .price {
          color: var(--danger);
        }

        .orderbook-level.bid .price {
          color: var(--success);
        }

        .orderbook-level .qty {
          color: var(--text-muted);
        }

        .orderbook-mid {
          display: flex;
          justify-content: space-between;
          padding: 8px 0;
          margin: 4px 0;
          border-top: 1px solid var(--border);
          border-bottom: 1px solid var(--border);
        }

        .spread-label {
          font-size: 11px;
          color: var(--text-muted);
        }

        .spread-value {
          font-size: 12px;
          font-weight: 600;
          color: #f59e0b;
          font-family: monospace;
        }

        .spread-chart-section {
          background: var(--surface);
          border-radius: var(--radius-lg);
          padding: 12px;
        }

        .spread-chart-section h3 {
          font-size: 13px;
          font-weight: 500;
          margin-bottom: 12px;
          color: var(--text-secondary);
        }

        .spread-chart-wrapper {
          width: 100%;
          height: 120px;
          border-radius: var(--radius);
          overflow: hidden;
        }

        @media (max-width: 900px) {
          .main-content {
            flex-direction: column;
          }

          .orderbook-panel {
            width: 100%;
          }

          .header-main {
            flex-direction: column;
            align-items: flex-start;
            gap: 12px;
          }

          .controls {
            width: 100%;
          }

          .controls select {
            flex: 1;
          }

          .api-key-select {
            min-width: 0;
          }
        }
      `}</style>
    </Layout>
  );
};

export default KLine;
