import { type Component, createSignal, createEffect, onMount, onCleanup, Show, For } from 'solid-js';
import { createChart, type IChartApi, type ISeriesApi, type CandlestickData, type Time } from 'lightweight-charts';
import { FiRefreshCw, FiWifi, FiWifiOff, FiAlertCircle } from 'solid-icons/fi';
import Layout from '../components/Layout';
import { api } from '../lib/api';
import { klineWs, type BinanceKlineEvent } from '../lib/websocket';

const KLineSimple: Component = () => {
  const [symbols, setSymbols] = createSignal<string[]>([]);
  const [intervals, setIntervals] = createSignal<string[]>([]);
  const [selectedSymbol, setSelectedSymbol] = createSignal('BTCUSDT');
  const [selectedInterval, setSelectedInterval] = createSignal('1m');
  const [isConnected, setIsConnected] = createSignal(false);
  const [isLoading, setIsLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [currentPrice, setCurrentPrice] = createSignal<string | null>(null);
  const [priceChange, setPriceChange] = createSignal<'up' | 'down' | null>(null);

  let chartContainer: HTMLDivElement | undefined;
  let chart: IChartApi | null = null;
  let candlestickSeries: ISeriesApi<'Candlestick'> | null = null;
  let lastPrice: number | null = null;

  onMount(async () => {
    try {
      const [symbolsRes, intervalsRes] = await Promise.all([
        api.getSymbols(),
        api.getIntervals(),
      ]);
      
      if (symbolsRes.data) setSymbols(symbolsRes.data);
      if (intervalsRes.data) setIntervals(intervalsRes.data);
    } catch (e) {
      console.error('Failed to load initial data:', e);
      setError('Failed to load initial data');
    } finally {
      setIsLoading(false);
    }

    // Setup WebSocket
    const unsubMessage = klineWs.onMessage(handleKlineMessage);
    const unsubConnect = klineWs.onConnect(() => setIsConnected(true));
    const unsubDisconnect = klineWs.onDisconnect(() => setIsConnected(false));

    klineWs.connect();
    setIsConnected(klineWs.isConnected());

    onCleanup(() => {
      unsubMessage();
      unsubConnect();
      unsubDisconnect();
      klineWs.disconnect();
      if (chart) {
        chart.remove();
        chart = null;
      }
    });
  });

  // Initialize chart
  createEffect(() => {
    if (!chartContainer || chart) return;

    chart = createChart(chartContainer, {
      width: chartContainer.clientWidth,
      height: 500,
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

  // Handle symbol/interval changes
  createEffect(() => {
    const symbol = selectedSymbol();
    const interval = selectedInterval();

    if (!symbol || !interval) return;

    // Clear existing data
    if (candlestickSeries) {
      candlestickSeries.setData([]);
    }
    lastPrice = null;
    setCurrentPrice(null);

    // Fetch historical data and subscribe to WebSocket
    fetchHistoricalData(symbol, interval);
    
    if (isConnected()) {
      klineWs.subscribe(symbol, interval);
    }
  });

  // Re-subscribe when connection is established
  createEffect(() => {
    if (isConnected()) {
      klineWs.subscribe(selectedSymbol(), selectedInterval());
    }
  });

  const fetchHistoricalData = async (symbol: string, interval: string) => {
    try {
      const response = await fetch(
        `https://api.binance.com/api/v3/klines?symbol=${symbol}&interval=${interval}&limit=500`
      );
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

  const handleKlineMessage = (event: BinanceKlineEvent) => {
    if (!candlestickSeries) return;
    if (event.s !== selectedSymbol()) return;

    const kline = event.k;
    const candle: CandlestickData<Time> = {
      time: (kline.t / 1000) as Time,
      open: parseFloat(kline.o),
      high: parseFloat(kline.h),
      low: parseFloat(kline.l),
      close: parseFloat(kline.c),
    };

    candlestickSeries.update(candle);

    const newPrice = parseFloat(kline.c);
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

  const handleRefresh = () => {
    fetchHistoricalData(selectedSymbol(), selectedInterval());
  };

  return (
    <Layout>
      <div class="kline-simple-page">
        {/* Header */}
        <header class="page-header">
          <div class="header-main">
            <div class="symbol-display">
              <h1>{selectedSymbol()}</h1>
              <Show when={currentPrice()}>
                <span class={`price ${priceChange()}`}>${currentPrice()}</span>
              </Show>
            </div>
            <span class={`status ${isConnected() ? 'connected' : ''}`}>
              {isConnected() ? <FiWifi /> : <FiWifiOff />}
              {isConnected() ? 'Connected' : 'Offline'}
            </span>
          </div>
          
          {/* Controls */}
          <div class="controls">
            <select
              value={selectedSymbol()}
              onChange={(e) => setSelectedSymbol(e.currentTarget.value)}
              disabled={isLoading()}
            >
              <For each={symbols()}>
                {(symbol) => <option value={symbol}>{symbol}</option>}
              </For>
            </select>

            <select
              value={selectedInterval()}
              onChange={(e) => setSelectedInterval(e.currentTarget.value)}
              disabled={isLoading()}
            >
              <For each={intervals()}>
                {(interval) => <option value={interval}>{interval}</option>}
              </For>
            </select>

            <button class="btn-icon" onClick={handleRefresh} title="Refresh">
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

        {/* Chart */}
        <div class="chart-container" ref={chartContainer} />
      </div>

      <style>{`
        .kline-simple-page {
          max-width: 1400px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 24px;
        }

        .header-main {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 16px;
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

        .controls {
          display: flex;
          align-items: center;
          gap: 8px;
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

        .btn-icon:hover {
          color: var(--text);
          border-color: var(--text-muted);
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

        .chart-container {
          width: 100%;
          height: 500px;
          background: var(--surface);
          border-radius: var(--radius-lg);
          overflow: hidden;
        }

        @media (max-width: 768px) {
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

          .chart-container {
            height: 400px;
          }
        }
      `}</style>
    </Layout>
  );
};

export default KLineSimple;
