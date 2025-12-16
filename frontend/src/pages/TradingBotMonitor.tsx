import { createEffect, createResource, createSignal, For, onCleanup, onMount, Show, untrack, type Component } from 'solid-js';
import Layout from '../components/Layout';
import { api } from '../lib/api';
import { tradingWs, type Order, type OrderBook, type TradingResponse } from '../lib/websocket';

type Candle = {
    time: number; // seconds
    open: number;
    high: number;
    low: number;
    close: number;
};

const TradingBotMonitor: Component = () => {
    const LS_SYMBOL_KEY = 'tradingBotMonitor:selectedSymbol';
    const LS_INTERVAL_KEY = 'tradingBotMonitor:selectedInterval';

    let chartContainer!: HTMLDivElement;
    let canvasEl: HTMLCanvasElement | null = null;
    let resizeObserver: ResizeObserver | null = null;

    const candleByTime = new Map<number, Candle>();
    let flushScheduled = false;
    let lastCandleTime = 0;
    let panPx = 0; // how many pixels we moved to older candles (0 = latest aligned)
    let pixelsPerBar = 10; // zoom level (px per candle)
    let isDragging = false;
    let dragLastX = 0;
    let hoverX: number | null = null;
    let hoverY: number | null = null;

    const [apiKeys] = createResource(async () => {
        const response = await api.listAPIKeys();
        const keys = response.data || [];
        return keys.filter((key) => key.is_active);
    });

    const [selectedKeyId, setSelectedKeyId] = createSignal<string>('');
    const [selectedSymbol, setSelectedSymbol] = createSignal<string>('BTCUSDT');
    const [selectedInterval, setSelectedInterval] = createSignal<string>('1m');
    const [currentPrice, setCurrentPrice] = createSignal<number>(0);
    const [lastPrice, setLastPrice] = createSignal<number>(0);
    const [orders, setOrders] = createSignal<Order[]>([]);
    const [orderBook, setOrderBook] = createSignal<OrderBook | null>(null);
    const [maxAskQty, setMaxAskQty] = createSignal<number>(1);
    const [maxBidQty, setMaxBidQty] = createSignal<number>(1);
    const [connected, setConnected] = createSignal<boolean>(false);
    const [error, setError] = createSignal<string>('');
    const [hasCandles, setHasCandles] = createSignal<boolean>(false);
    const [candles, setCandles] = createSignal<Candle[]>([]);

    // Default to first active API key once loaded
    createEffect(() => {
        const keys = apiKeys();
        if (keys && keys.length > 0 && !selectedKeyId()) {
            setSelectedKeyId(keys[0]!.id);
        }
    });

    // Restore last selected trading pair from localStorage (if any)
    onMount(() => {
        if (typeof window === 'undefined') return;
        const savedSymbol = localStorage.getItem(LS_SYMBOL_KEY);
        const savedInterval = localStorage.getItem(LS_INTERVAL_KEY);
        if (savedSymbol) {
            setSelectedSymbol(savedSymbol);
        }
        if (savedInterval) {
            setSelectedInterval(savedInterval);
        }
    });

    // Persist trading pair whenever it changes
    createEffect(() => {
        const symbol = selectedSymbol();
        if (typeof window === 'undefined') return;
        try {
            localStorage.setItem(LS_SYMBOL_KEY, symbol);
        } catch {
            // ignore storage errors (quota/denied)
        }
    });

    // Persist interval whenever it changes
    createEffect(() => {
        const interval = selectedInterval();
        if (typeof window === 'undefined') return;
        try {
            localStorage.setItem(LS_INTERVAL_KEY, interval);
        } catch {
            // ignore storage errors
        }
    });

    const ensureCanvas = () => {
        if (!chartContainer) return;
        if (canvasEl) {
            // If the container node was re-created (e.g. re-mount/reconcile),
            // re-attach the existing canvas so drawing is visible.
            if (canvasEl.parentElement !== chartContainer) {
                chartContainer.appendChild(canvasEl);
            }
            return;
        }

        const canvas = document.createElement('canvas');
        canvas.style.width = '100%';
        canvas.style.height = '100%';
        canvas.style.display = 'block';
        canvasEl = canvas;
        chartContainer.appendChild(canvas);

        const onWheel = (e: WheelEvent) => {
            e.preventDefault();
            const data = candles();
            if (!data.length || !canvasEl || !chartContainer) return;

            const dpr = window.devicePixelRatio || 1;

            const paddingLeft = Math.floor(64 * dpr);
            const paddingRight = Math.floor(64 * dpr);
            const paddingTop = Math.floor(64 * dpr);
            const paddingBottom = Math.floor(64 * dpr);

            const plotW = Math.max(0, canvasEl.width - paddingLeft - paddingRight);
            const plotH = Math.max(0, canvasEl.height - paddingTop - paddingBottom);
            if (plotW <= 0 || plotH <= 0) return;

            const absDeltaX = Math.abs(e.deltaX);
            // const absDeltaY = Math.abs(e.deltaY);

            const barsFit = Math.max(1, plotW / pixelsPerBar);
            const offsetBars = panPx / pixelsPerBar;
            const startIdx = data.length - barsFit - offsetBars;

            // Anchor zoom at chart center (not cursor)
            const centerRatio = 0.5;
            const targetIndex = startIdx + centerRatio * barsFit;

            if (absDeltaX > 0.1) { //absDeltaY) {
                // Horizontal scroll -> pan (invert direction)
                panPx = panPx - e.deltaX; //Math.max(0, panPx - e.deltaX);
                drawChart('pan');
                return;
            }

            // Vertical scroll -> zoom with cursor anchor (less sensitive)
            const zoomStep = 0.03; // low sensitivity
            const zoomFactor = e.deltaY > 0 ? (1 - zoomStep) : (1 + zoomStep);
            const nextPixelsPerBar = Math.min(40, Math.max(2, pixelsPerBar * zoomFactor));
            const nextBarsFit = Math.max(1, plotW / nextPixelsPerBar);

            // Keep targetIndex at center after zoom
            const desiredStartIdx = targetIndex - centerRatio * nextBarsFit;
            const nextOffsetBars = data.length - nextBarsFit - desiredStartIdx;

            pixelsPerBar = nextPixelsPerBar;
            const maxPan = (data.length + nextBarsFit) * pixelsPerBar;
            const minPan = -plotW; // allow a screen of blank space on the right
            panPx = Math.min(maxPan, Math.max(minPan, nextOffsetBars * pixelsPerBar));
            drawChart('zoom');
        };

        const onMouseDown = (e: MouseEvent) => {
            isDragging = true;
            dragLastX = e.clientX;
        };

        const onMouseUp = () => {
            isDragging = false;
        };

        const onMouseLeave = () => {
            isDragging = false;
            hoverX = null;
            hoverY = null;
            drawChart('hover-leave');
        };

        const onMouseMove = (e: MouseEvent) => {
            const dpr = window.devicePixelRatio || 1;
            hoverX = e.offsetX * dpr;
            hoverY = e.offsetY * dpr;

            if (!isDragging) {
                drawChart('hover');
                return;
            }

            const dx = e.clientX - dragLastX;
            dragLastX = e.clientX;
            // Invert: dragging left moves to older data (panPx increases)
            // allow pan both directions with blank space, invert direction
            panPx = panPx - dx;
            drawChart('pan');
        };

        canvas.addEventListener('wheel', onWheel, { passive: false });
        canvas.addEventListener('mousedown', onMouseDown);
        canvas.addEventListener('mouseup', onMouseUp);
        canvas.addEventListener('mouseleave', onMouseLeave);
        canvas.addEventListener('mousemove', onMouseMove);

        const applySize = () => {
            if (!canvasEl || !chartContainer) return;
            const width = Math.max(0, Math.floor(chartContainer.clientWidth));
            const height = Math.max(0, Math.floor(chartContainer.clientHeight));
            if (width <= 0 || height <= 0) return;

            const dpr = window.devicePixelRatio || 1;
            const nextW = Math.floor(width * dpr);
            const nextH = Math.floor(height * dpr);
            if (canvasEl.width !== nextW) canvasEl.width = nextW;
            if (canvasEl.height !== nextH) canvasEl.height = nextH;
            drawChart('resize/applySize');
        };

        resizeObserver = new ResizeObserver(() => applySize());
        resizeObserver.observe(chartContainer);
        applySize();
    };

    const clearChart = () => {
        candleByTime.clear();
        flushScheduled = false;
        lastCandleTime = 0;
        panPx = 0;
        pixelsPerBar = 10;
        setCurrentPrice(0);
        setHasCandles(false);
        setCandles([]);
        drawChart('clearChart');
    };

    const normalizeCandleTimestampSeconds = (raw: unknown): number => {
        let ts = Number(raw);
        if (!Number.isFinite(ts)) return 0;
        if (ts > 9_999_999_999) ts = Math.floor(ts / 1000);
        return ts;
    };

    const computeOBWidth = (qty: number, side: 'ask' | 'bid') => {
        if (!Number.isFinite(qty) || qty <= 0) return 0;
        const maxQty = side === 'ask' ? maxAskQty() : maxBidQty();
        if (maxQty <= 0) return 0;
        // const ratio = Math.min(1, Math.log1p(qty) / Math.log1p(maxQty));
        const ratio = qty / maxQty
        const scaled = ratio * 0.6; // cap at 60% of row width
        return Math.max(0.01, scaled) * 100;
    };

    const computeOrderWidth = (qty: number, maxQty: number) => {
        if (!Number.isFinite(qty) || qty <= 0 || maxQty <= 0) return 0;
        // const ratio = Math.min(1, Math.log1p(qty) / Math.log1p(maxQty));
        const ratio = qty / maxQty
        const scaled = ratio * 0.6; // cap at 60% of row width
        return Math.max(0.01, scaled) * 100;
    };

    const parseIncomingCandle = (payload: any): Candle | null => {
        if (!payload) return null;
        const time = normalizeCandleTimestampSeconds(payload.time ?? payload[0] ?? payload.t);
        if (!time) return null;

        const open = Number(payload.open ?? payload[1] ?? payload.o);
        const close = Number(payload.close ?? payload[2] ?? payload.c);
        const high = Number(payload.high ?? payload[3] ?? payload.h);
        const low = Number(payload.low ?? payload[4] ?? payload.l);

        if (![open, close, high, low].every(Number.isFinite)) return null;

        return {
            time,
            open,
            high,
            low,
            close,
        };
    };

    const drawChart = (_?: string) => {
        if (!canvasEl) return;
        const ctx = canvasEl.getContext('2d');
        if (!ctx) return;

        const dpr = window.devicePixelRatio || 1;
        const width = canvasEl.width;
        const height = canvasEl.height;

        // Theme aligned to monitor/index.html
        const bg = '#11151aff';
        // const text = '#e2e8f0';
        const grid = 'rgba(148, 163, 184, 0.10)';
        const up = '#10b981';
        const down = '#ef4444';
        const axis = 'rgba(148, 163, 184, 0.55)';

        ctx.setTransform(1, 0, 0, 1, 0, 0);
        ctx.clearRect(0, 0, width, height);
        ctx.fillStyle = bg;
        ctx.fillRect(0, 0, width, height);

        const data = candles();
        const activeOrders = orders();
        if (!data.length) return;

        const paddingLeft = Math.floor(12 * dpr);
        const paddingRight = Math.floor(62 * dpr);
        const paddingTop = Math.floor(12 * dpr);
        const paddingBottom = Math.floor(18 * dpr);

        const plotW = Math.max(0, width - paddingLeft - paddingRight);
        const plotH = Math.max(0, height - paddingTop - paddingBottom);
        if (plotW <= 0 || plotH <= 0) return;

        const total = data.length;
        const barsFit = Math.max(1, Math.floor(plotW / pixelsPerBar));
        const offsetBars = panPx / pixelsPerBar;
        const startIdxFloat = total - barsFit - offsetBars;

        const padLeftBars = Math.max(0, -Math.floor(startIdxFloat));
        const startIdx = Math.max(0, Math.floor(startIdxFloat));
        const endIdxRaw = startIdx + barsFit - padLeftBars;
        const padRightBars = Math.max(0, Math.ceil(endIdxRaw - total));
        const endIdx = Math.min(total, Math.floor(endIdxRaw));

        const view = data.slice(startIdx, endIdx);
        const viewCount = view.length;
        if (!viewCount && padLeftBars === 0 && padRightBars === 0) return;

        let minLow = Number.POSITIVE_INFINITY;
        let maxHigh = Number.NEGATIVE_INFINITY;
        for (const c of view) {
            if (c.low < minLow) minLow = c.low;
            if (c.high > maxHigh) maxHigh = c.high;
        }
        if (!Number.isFinite(minLow) || !Number.isFinite(maxHigh) || minLow === maxHigh) {
            minLow = minLow - 1;
            maxHigh = maxHigh + 1;
        }

        const yFor = (price: number) => {
            const t = (price - minLow) / (maxHigh - minLow);
            return paddingTop + Math.round((1 - t) * plotH);
        };

        // Grid
        ctx.strokeStyle = grid;
        ctx.lineWidth = Math.max(1, Math.floor(1 * dpr));
        const gridRows = 4;
        for (let i = 0; i <= gridRows; i++) {
            const y = paddingTop + Math.round((plotH * i) / gridRows);
            ctx.beginPath();
            ctx.moveTo(paddingLeft, y);
            ctx.lineTo(paddingLeft + plotW, y);
            ctx.stroke();
        }

        const step = Math.max(1, pixelsPerBar);
        const bodyW = Math.max(1, Math.floor(step * 0.65));

        // Candles
        // Draw left padding as empty space and shift view accordingly
        for (let i = 0; i < viewCount; i++) {
            const c = view[i]!;
            const xCenter = paddingLeft + Math.floor(step * (padLeftBars + i + 0.5));
            const x0 = xCenter - Math.floor(bodyW / 2);
            const x1 = xCenter + Math.floor(bodyW / 2);

            const yOpen = yFor(c.open);
            const yClose = yFor(c.close);
            const yHigh = yFor(c.high);
            const yLow = yFor(c.low);
            const isUp = c.close >= c.open;
            ctx.strokeStyle = isUp ? up : down;
            ctx.fillStyle = isUp ? up : down;

            // Wick
            ctx.beginPath();
            ctx.moveTo(xCenter, yHigh);
            ctx.lineTo(xCenter, yLow);
            ctx.stroke();

            // Body
            const top = Math.min(yOpen, yClose);
            const bottom = Math.max(yOpen, yClose);
            const bodyH = Math.max(Math.floor(1 * dpr), bottom - top);
            ctx.fillRect(x0, top, Math.max(1, x1 - x0), bodyH);
        }

        // Right axis price labels (min/max)
        ctx.fillStyle = axis;
        ctx.font = `${Math.floor(11 * dpr)}px ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif`;
        ctx.textAlign = 'left';
        ctx.textBaseline = 'middle';

        const formatPrice = (v: number) => {
            if (!Number.isFinite(v)) return '—';
            const abs = Math.abs(v);
            if (abs >= 1000) return v.toFixed(2);
            if (abs >= 1) return v.toFixed(4);
            return v.toFixed(6);
        };

        const yMax = yFor(maxHigh);
        const yMin = yFor(minLow);
        ctx.fillText(formatPrice(maxHigh), paddingLeft + plotW + Math.floor(6 * dpr), yMax);
        ctx.fillText(formatPrice(minLow), paddingLeft + plotW + Math.floor(6 * dpr), yMin);

        // X-axis time labels (4 ticks)
        ctx.textAlign = 'center';
        ctx.textBaseline = 'top';
        const ticks = 4;
        for (let i = 0; i <= ticks; i++) {
            const frac = i / ticks;
            const idx = Math.min(viewCount - 1, Math.max(0, Math.round((padLeftBars + frac * (viewCount - 1)))));
            const candle = view[idx];
            const x = paddingLeft + Math.floor(step * (padLeftBars + idx + 0.5));
            const timeStr = new Date(candle.time * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            ctx.fillStyle = axis;
            ctx.fillText(timeStr, x, paddingTop + plotH + 4 * dpr);
        }
        ctx.textAlign = 'left';
        ctx.textBaseline = 'middle';

        // Active orders dashed lines (BUY=green, SELL=red) with labels (deduped by side+price)
        if (activeOrders.length) {
            ctx.setLineDash([4 * dpr, 2 * dpr]); // denser dashes
            ctx.lineWidth = 1; // keep stroke at 1px
            const seen = new Set<string>();
            for (const order of activeOrders) {
                const priceNum = Number(order.price);
                if (!Number.isFinite(priceNum) || priceNum <= 0) continue;
                const key = `${order.side}:${priceNum.toFixed(8)}`;
                if (seen.has(key)) continue;
                seen.add(key);
                const y = yFor(priceNum);
                const isBuy = order.side === 'BUY';
                const color = isBuy ? 'rgba(59, 130, 246, 0.7)' : 'rgba(244, 114, 182, 0.7)';
                ctx.strokeStyle = color;
                ctx.beginPath();
                ctx.moveTo(paddingLeft, y);
                ctx.lineTo(paddingLeft + plotW, y);
                ctx.stroke();
            }
            ctx.setLineDash([]);
        }

        // Best Bid/Ask first level lines + tags
        const drawTag = (y: number, label: string, value: number, labelColor: string) => {
            ctx.setLineDash([4 * dpr, 2 * dpr]);
            ctx.strokeStyle = labelColor;
            ctx.lineWidth = 1;
            ctx.beginPath();
            ctx.moveTo(paddingLeft, y);
            ctx.lineTo(paddingLeft + plotW, y);
            ctx.stroke();
            ctx.setLineDash([]);

            const priceTxt = value.toFixed(8);
            const labelTxt = label;
            ctx.font = `${Math.floor(11 * dpr)}px ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif`;

            const labelW = 30 * dpr;
            const priceW = ctx.measureText(priceTxt).width + 12 * dpr;
            const boxH = 18 * dpr;
            const boxY = y - boxH / 2;
            let boxX = paddingLeft + plotW - priceW;//+ Math.floor(6 * dpr);

            // Label box
            ctx.fillStyle = labelColor;
            ctx.beginPath();
            if (typeof ctx.roundRect === 'function') {
                ctx.roundRect(boxX, boxY, labelW, boxH, 5);
                ctx.fill();
            } else {
                ctx.fillRect(boxX, boxY, labelW, boxH);
            }
            ctx.fillStyle = '#f8fafc';
            ctx.fillText(labelTxt, boxX + 6 * dpr, y);

            // Price box
            boxX += labelW + 1 * dpr;
            ctx.fillStyle = labelColor;
            ctx.beginPath();
            if (typeof ctx.roundRect === 'function') {
                ctx.roundRect(boxX, boxY, priceW, boxH, 5);
                ctx.fill();
            } else {
                ctx.fillRect(boxX, boxY, priceW, boxH);
            }
            ctx.fillStyle = '#f8fafc';
            ctx.fillText(priceTxt, boxX + 6 * dpr, y);
        };

        const price = currentPrice();
        const book = orderBook();
        if (book) {
            let bestBid = Number.NEGATIVE_INFINITY;
            let bestAsk = Number.POSITIVE_INFINITY;
            for (const b of book.bids || []) {
                const v = Number(b.price);
                if (Number.isFinite(v) && v > bestBid) bestBid = v;
            }
            for (const a of book.asks || []) {
                const v = Number(a.price);
                if (Number.isFinite(v) && v < bestAsk) bestAsk = v;
            }
            const priceY = Number.isFinite(price) ? yFor(price) : null;
            const tagGap = 22 * dpr; // gap to avoid overlapping labels
            const halfBoxH = 9 * dpr; // from drawTag height 18*dpr

            let askY = Number.isFinite(bestAsk) ? yFor(bestAsk) : null;
            let bidY = (Number.isFinite(bestBid) && bestBid > Number.NEGATIVE_INFINITY) ? yFor(bestBid) : null;

            if (askY !== null && priceY !== null && Math.abs(askY - priceY) < tagGap) {
                askY = priceY - tagGap;
            }
            if (bidY !== null && priceY !== null && Math.abs(bidY - priceY) < tagGap) {
                bidY = priceY + tagGap;
            }

            // keep order: ask above bid
            if (askY !== null && bidY !== null && (askY - bidY) > -tagGap) {
                askY = bidY - tagGap;
            }

            // clamp within plot
            const minY = paddingTop + halfBoxH;
            const maxY = paddingTop + plotH - halfBoxH;

            if (askY !== null) {
                askY = Math.min(Math.max(askY, minY), maxY);
                drawTag(askY, 'Ask', bestAsk, 'rgba(236, 72, 153, 0.95)');
            }
            if (bidY !== null) {
                bidY = Math.min(Math.max(bidY, minY), maxY);
                drawTag(bidY, 'Bid', bestBid, 'rgba(37, 99, 235, 0.95)');
            }
        }

        // Current price line with label, color by up/down
        if (Number.isFinite(price) && price > 0) {
            const y = yFor(price);
            const priceColor = price >= lastPrice() ? '#22c55e' : '#ef4444';
            ctx.strokeStyle = priceColor;
            ctx.lineWidth = Math.max(1, Math.floor(1 * dpr));
            ctx.beginPath();
            ctx.setLineDash([6 * dpr, 4 * dpr]);
            ctx.moveTo(paddingLeft, y);
            ctx.lineTo(paddingLeft + plotW, y);
            ctx.stroke();
            ctx.setLineDash([]);

            const label = `${price.toFixed(8)}`;
            const boxW = ctx.measureText(label).width + 12 * dpr;
            const boxH = 20 * dpr;
            const boxX = paddingLeft + plotW - boxW + 31 * dpr;
            const boxY = y - boxH / 2;
            ctx.fillStyle = priceColor;
            ctx.beginPath();
            if (typeof ctx.roundRect === 'function') {
                ctx.roundRect(boxX, boxY, boxW, boxH, 5);
                ctx.fill();
            } else {
                ctx.fillRect(boxX, boxY, boxW, boxH);
            }
            ctx.fillStyle = '#ecececff';
            ctx.fillText(label, boxX + 4 * dpr, y);
        }

        // Hover crosshair and tooltips
        if (hoverX !== null && hoverY !== null) {
            const x = hoverX;
            const y = hoverY;
            const inPlot = x >= paddingLeft && x <= paddingLeft + plotW && y >= paddingTop && y <= paddingTop + plotH;
            if (inPlot) {
                ctx.save();
                ctx.setLineDash([4 * dpr, 2 * dpr]);
                ctx.strokeStyle = 'rgba(148, 163, 184, 0.7)';
                ctx.lineWidth = 1;
                ctx.beginPath();
                ctx.moveTo(x, paddingTop);
                ctx.lineTo(x, paddingTop + plotH);
                ctx.moveTo(paddingLeft, y);
                ctx.lineTo(paddingLeft + plotW, y);
                ctx.stroke();
                ctx.restore();
            }
        }
    };

    const scheduleFlush = () => {
        if (flushScheduled) return;
        flushScheduled = true;
        const schedule =
            typeof window.requestAnimationFrame === 'function'
                ? window.requestAnimationFrame.bind(window)
                : (cb: FrameRequestCallback) => window.setTimeout(cb, 16);
        schedule(() => {
            flushScheduled = false;
            if (candleByTime.size === 0) return;

            const nextCandles = Array.from(candleByTime.entries())
                .sort((a, b) => a[0] - b[0])
                .slice(-200)
                .map(([, candle]) => candle);
            setCandles(nextCandles);
            lastCandleTime = nextCandles.length ? nextCandles[nextCandles.length - 1]!.time : 0;
            drawChart('flush');
        });
    };

    // Initialize chart on mount
    onMount(() => {
        ensureCanvas();
        onCleanup(() => {
            if (resizeObserver) {
                resizeObserver.disconnect();
                resizeObserver = null;
            }
            if (canvasEl) {
                canvasEl.remove();
                canvasEl = null;
            }
        });
    });

    // Clear chart when symbol or interval changes
    createEffect(() => {
        // Avoid tracking dependencies from clearChart() -> drawChart() (candles/currentPrice),
        // otherwise this effect re-runs on every candle update and keeps clearing the chart.
        untrack(() => clearChart());
    });

    // Redraw on price change (e.g., when only close updates)
    createEffect(() => {
        currentPrice();
        drawChart('priceEffect');
    });

    // WebSocket connection effect
    createEffect(() => {
        const keyId = selectedKeyId();
        const symbol = selectedSymbol();
        const interval = selectedInterval();

        if (!keyId || !symbol || !interval) return;

        const token = api.getToken();
        if (!token) return;

        setError('');
        setConnected(false);
        setOrders([]);
        untrack(() => clearChart()); // ensure fresh chart before resubscribe

        // Connect to WebSocket
        tradingWs.connect(token);

        const sendSubscriptions = () => {
            tradingWs.connectToApiKey(keyId);
            tradingWs.subscribeKline(symbol, interval);
            tradingWs.subscribeOrderBook(symbol);
            tradingWs.subscribeOrder(symbol);
        };

        // Send initial subscriptions
        sendSubscriptions();

        // Handle messages
        const handleMessage = (data: TradingResponse) => {
            if (data.type === 'kline' && data.data) {
                ensureCanvas();

                const candle = parseIncomingCandle(data.data as any);
                if (!candle) return;


                candleByTime.set(candle.time, candle);

                // Fast-path: append/update tail in-order; otherwise flush (handles out-of-order/backfill).
                if (candle.time >= lastCandleTime) {
                    setCandles((prev) => {
                        if (prev.length === 0) return [candle];
                        const last = prev[prev.length - 1]!;
                        if (last.time === candle.time) {
                            const next = prev.slice(0, prev.length - 1);
                            next.push(candle);
                            return next;
                        }

                        const next = prev.concat(candle);
                        if (next.length > 200) next.splice(0, next.length - 200);
                        return next;
                    });
                    lastCandleTime = candle.time;
                    drawChart('kline/fastPath');
                } else {
                    scheduleFlush();
                }

                setLastPrice(currentPrice());
                setCurrentPrice(candle.close);
                setHasCandles(true);
                setConnected(true);
            } else if (data.type === 'orderbook' && data.data) {
                const book = data.data as OrderBook;
                const asks = (book.asks || []).slice(0, 50);
                const bids = (book.bids || []).slice(0, 50);
                const maxAsk = asks.reduce((m, l) => Math.max(m, Number(l.quantity) || 0), 1);
                const maxBid = bids.reduce((m, l) => Math.max(m, Number(l.quantity) || 0), 1);
                setMaxAskQty(maxAsk || 1);
                setMaxBidQty(maxBid || 1);
                setOrderBook({
                    ...book,
                    bids: (book.bids || []).slice(0, 50),
                    asks: (book.asks || []).slice(0, 50),
                });
                setConnected(true);
            } else if (data.type === 'order' && data.data) {
                const orderData = data.data as Order;
                const isTerminal =
                    orderData.status === 'FILLED' ||
                    orderData.status === 'CANCELED' ||
                    orderData.status === 'UNKNOWN';
                setOrders(prev => {
                    const next = [...prev];
                    const idx = next.findIndex(o => o.orderId === orderData.orderId);
                    if (isTerminal) {
                        if (idx !== -1) next.splice(idx, 1);
                        return next;
                    }
                    if (idx === -1) {
                        next.push(orderData);
                    } else {
                        next[idx] = orderData;
                    }
                    return next;
                });
                setConnected(true);
            } else if (data.type === 'error') {
                setError(data.error || 'Unknown error');
                setConnected(false);
            }
        };

        const unsubscribeMessage = tradingWs.onMessage(handleMessage);
        const unsubscribeConnect = tradingWs.onConnect(() => {
            // Re-run subscriptions after auto-reconnect
            sendSubscriptions();
        });

        // Cleanup
        onCleanup(() => {
            unsubscribeMessage();
            unsubscribeConnect();
            tradingWs.unsubscribeKline(symbol, interval);
            tradingWs.unsubscribeOrderBook(symbol);
            tradingWs.unsubscribeOrders(symbol);
            tradingWs.disconnect();
        });
    });

    return (
        <Layout>
            <div class="monitor-page">
                <div class="page-header">
                    <div class="page-header-content">
                        <h1>Trading Bot Monitor</h1>
                        <p>Real-time K-line charts and order tracking</p>
                    </div>
                    <div class="connection-status">
                        <div class={`status-indicator ${connected() ? 'connected' : 'disconnected'}`}>
                            <div class={`status-dot ${connected() ? 'success' : 'danger'}`} />
                            <span>{connected() ? 'Connected' : 'Disconnected'}</span>
                        </div>
                    </div>
                </div>

                {/* Controls */}
                <div class="controls-section">
                    <div class="control-group">
                        <label>API Key</label>
                        <select
                            value={selectedKeyId()}
                            onChange={(e) => setSelectedKeyId(e.currentTarget.value)}
                            disabled={!apiKeys() || apiKeys()!.length === 0}
                        >
                            <option value="">Select API Key</option>
                            <For each={apiKeys()}>
                                {(key) => (
                                    <option value={key.id}>
                                        {key.name || key.id} {key.is_testnet ? '(Testnet)' : ''}
                                    </option>
                                )}
                            </For>
                        </select>
                    </div>

                    <div class="control-group">
                        <label>Trading Pair</label>
                        <select
                            value={selectedSymbol()}
                            onChange={(e) => setSelectedSymbol(e.currentTarget.value)}
                        >
                            <option value="BTCUSDT">BTC/USDT</option>
                            <option value="SOLUSDT">SOL/USDT</option>
                            <option value="XRPUSDT">XRP/USDT</option>
                            <option value="DOGEUSDT">DOGE/USDT</option>
                        </select>
                    </div>

                    <div class="control-group">
                        <label>Interval</label>
                        <select
                            value={selectedInterval()}
                            onChange={(e) => setSelectedInterval(e.currentTarget.value)}
                        >
                            <option value="1m">1 minute</option>
                            <option value="5m">5 minutes</option>
                            <option value="15m">15 minutes</option>
                            <option value="1h">1 hour</option>
                            <option value="4h">4 hours</option>
                            <option value="1d">1 day</option>
                        </select>
                    </div>
                </div>

                {/* Error Message */}
                <Show when={error()}>
                    <div class="error-message">
                        <p>{error()}</p>
                    </div>
                </Show>

                {/* Monitor Grid */}
                <div class="monitor-grid">
                    {/* K-line Chart */}
                    <div class="chart-section">
                        <div class="section-header">
                            <h3>K-Line Chart - {selectedSymbol()}</h3>
                            <div class="current-price">
                                <span>Current Price:</span>
                                <span class={`price-value ${currentPrice() > 0 ? 'active' : ''}`}>
                                    ${currentPrice().toFixed(8)}
                                </span>
                            </div>
                        </div>
                        <div class="chart-wrapper" ref={(el) => { chartContainer = el; ensureCanvas(); }}>
                            <Show when={!connected() && !hasCandles()}>
                                <div class="chart-empty">
                                    Select an API Key and Trading Pair to view chart
                                </div>
                            </Show>
                        </div>
                    </div>

                    {/* Order Book */}
                    <div class="orders-section">
                        <div class="section-header">
                            <h3>Order Book</h3>
                            <div class="order-count">
                                <span>{orders().filter(o => { o.side === 'SELL' }).length || 0} / {orders().filter(o => { o.side === 'BUY' }).length || 0}</span>
                            </div>
                        </div>
                        <div class="orderbook-grid">
                            <div class="orderbook-side asks">
                                {/* <div class="orderbook-head">Asks (20)</div> */}
                                <Show when={orderBook()} fallback={<div class="empty-orders">No data</div>}>
                                    <For each={(orderBook()?.asks || [])
                                        .map(l => ({ ...l, qtyNum: Number(l.quantity) }))
                                        .sort((a, b) => Number(a.price) - Number(b.price))
                                        .slice(-15)
                                        .reverse()}>
                                        {(level, _) => (
                                            <div class="ob-row">
                                                <div
                                                    class="ob-bar ask"
                                                    style={{
                                                        width: `${computeOBWidth(level.qtyNum, 'ask')}%`,
                                                    }}
                                                />
                                                <For each={orders().filter(o => o.side === 'SELL' && Number(o.price) === Number(level.price))}>
                                                    {(order) => (
                                                        <div
                                                            class="ob-order-bar ask"
                                                            style={{
                                                                width: `${computeOrderWidth(Number(order.quantity), maxAskQty())}%`,
                                                            }}
                                                        />
                                                    )}
                                                </For>
                                                <span class="price ask">{Number(level.price).toFixed(8)}</span>
                                                <span class="qty">{level.quantity}</span>
                                            </div>
                                        )}
                                    </For>
                                </Show>
                            </div>
                            <div class="orderbook-side bids">
                                <Show when={orderBook()} fallback={<div class="empty-orders">No data</div>}>
                                    <For each={(orderBook()?.bids || [])
                                        .map(l => ({ ...l, qtyNum: Number(l.quantity) }))
                                        .sort((a, b) => Number(b.price) - Number(a.price))
                                        .slice(0, 15)}>
                                        {(level) => (
                                            <div class="ob-row">
                                                <div
                                                    class="ob-bar bid"
                                                    style={{
                                                        width: `${computeOBWidth(level.qtyNum, 'bid')}%`,
                                                    }}
                                                />
                                                <For each={orders().filter(o => o.side === 'BUY' && Number(o.price) === Number(level.price))}>
                                                    {(order) => (
                                                        <div
                                                            class="ob-order-bar bid"
                                                            style={{
                                                                width: `${computeOrderWidth(Number(order.quantity), maxBidQty())}%`,
                                                            }}
                                                        />
                                                    )}
                                                </For>
                                                <span class="price bid">{Number(level.price).toFixed(8)}</span>
                                                <span class="qty">{level.quantity}</span>
                                            </div>
                                        )}
                                    </For>
                                </Show>
                                {/* <div class="orderbook-head">Bids (20)</div> */}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <style>{`
                .monitor-page {
                    max-width: 1400px;
                    margin: 0 auto;
                }

                .page-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: flex-start;
                    margin-bottom: 24px;
                }

                .page-header h1 {
                    font-size: 28px;
                    font-weight: 600;
                    margin-bottom: 4px;
                }

                .page-header p {
                    color: var(--text-secondary);
                    font-size: 14px;
                }

                .connection-status {
                    display: flex;
                    align-items: center;
                }

                .status-indicator {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    padding: 8px 16px;
                    background: var(--surface);
                    border-radius: var(--radius);
                    font-size: 13px;
                }

                .status-dot {
                    width: 8px;
                    height: 8px;
                    border-radius: 50%;
                }

                .status-dot.success {
                    background: var(--success);
                    box-shadow: 0 0 8px var(--success);
                    animation: pulse 2s infinite;
                }

                .status-dot.danger {
                    background: var(--danger);
                }

                @keyframes pulse {
                    0%, 100% { opacity: 1; }
                    50% { opacity: 0.5; }
                }

                .controls-section {
                    display: flex;
                    flex-wrap: wrap;
                    gap: 16px;
                    margin-bottom: 24px;
                    padding: 20px;
                    background: var(--surface);
                    border-radius: var(--radius-lg);
                    border: 1px solid var(--border);
                }

                .control-group {
                    display: flex;
                    flex-direction: column;
                    gap: 6px;
                    min-width: 180px;
                }

                .control-group label {
                    font-size: 13px;
                    font-weight: 500;
                    color: var(--text-secondary);
                }

                .control-group select {
                    padding: 10px 12px;
                    background: var(--background);
                    border: 1px solid var(--border);
                    border-radius: var(--radius);
                    color: var(--text);
                    font-size: 14px;
                    cursor: pointer;
                    transition: all 0.15s;
                }

                .control-group select:hover {
                    border-color: var(--primary);
                }

                .control-group select:focus {
                    outline: none;
                    border-color: var(--primary);
                    box-shadow: 0 0 0 3px var(--primary-light);
                }

                .control-group select:disabled {
                    opacity: 0.5;
                    cursor: not-allowed;
                }

                .error-message {
                    padding: 12px 16px;
                    background: rgba(239, 68, 68, 0.1);
                    border: 1px solid var(--danger);
                    border-radius: var(--radius);
                    margin-bottom: 24px;
                }

                .error-message p {
                    color: var(--danger);
                    font-size: 14px;
                    margin: 0;
                }

                .monitor-grid {
                    display: grid;
                    grid-template-columns: 1fr 350px;
                    gap: 24px;
                }

                @media (max-width: 1024px) {
                    .monitor-grid {
                        grid-template-columns: 1fr;
                    }
                }

                .chart-section,
                .orders-section {
                    background: var(--surface);
                    border-radius: var(--radius-lg);
                    border: 1px solid var(--border);
                    overflow: hidden;
                }

                .section-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 16px 20px;
                    border-bottom: 1px solid var(--border);
                }

                .section-header h3 {
                    font-size: 16px;
                    font-weight: 600;
                    margin: 0;
                }

                .current-price {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    font-size: 14px;
                }

                .current-price span:first-child {
                    color: var(--text-secondary);
                }

                .price-value {
                    font-weight: 600;
                    color: var(--text-muted);
                    font-family: monospace;
                    font-size: 16px;
                    transition: color 0.3s;
                }

                .price-value.active {
                    color: var(--primary);
                }

                .chart-wrapper {
                    height: 400px;
                    position: relative;
                }

                .chart-empty {
                    position: absolute;
                    top: 0;
                    left: 0;
                    right: 0;
                    bottom: 0;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    color: var(--text-muted);
                    font-size: 14px;
                    background: var(--background);
                }

                .order-count {
                    padding: 4px 12px;
                    background: var(--primary-light);
                    color: var(--primary);
                    border-radius: 999px;
                    font-size: 13px;
                    font-weight: 500;
                }

                .orderbook-grid {
                    display: grid;
                    grid-template-rows: 1fr 1fr;
                    height: 600px;
                    border-top: 1px solid var(--border);
                }

                .orderbook-side {
                    padding: 8px 10px;
                    display: flex;
                    flex-direction: column;
                    gap: 0px;
                    overflow: hidden; /* avoid scroll, fit all */
                }

                .orderbook-side.asks {
                    background: transparent;
                    justify-content: flex-end; /* align asks to bottom */
                }

                .orderbook-side.bids {
                    background: transparent;
                }

                .orderbook-head {
                    font-size: 11px;
                    text-transform: uppercase;
                    letter-spacing: 0.5px;
                    color: var(--text-secondary);
                    margin-bottom: 2px;
                }

                .ob-row {
                    display: flex;
                    justify-content: flex-start;
                    font-family: monospace;
                    font-size: 11px;
                    line-height: 0;
                    padding: 0;
                    position: relative;
                    min-height: 15px;
                }

                .price {
                    width: 50%;
                    z-index: 3;
                    position: absolute;
                    left: 0;
                    top: 50%;
                    transform: translateY(-50%);
                    text-align: left;
                }

                .price.bid {
                    color: var(--success);
                }

                .price.ask {
                    color: var(--danger);
                }

                .qty {
                    color: var(--text-secondary);
                    text-align: left;
                    z-index: 3;
                    position: absolute;
                    left: 50%;
                    width: 50%;
                    top: 50%;
                    transform: translateY(-50%);
                }

                .ob-bar {
                    position: absolute;
                    top: 0;
                    bottom: 0;
                    right: 0;
                    z-index: 1;
                    opacity: 0.35;
                }

                .ob-bar.ask {
                    background: rgba(244, 114, 114, 0.15);
                }

                .ob-bar.bid {
                    background: rgba(59, 246, 162, 0.15);
                }

                .ob-order-bar {
                    position: absolute;
                    top: 0px;
                    bottom: 0px;
                    right: 0;
                    z-index: 2;
                    opacity: 0.55;
                }

                .ob-order-bar.ask {
                    background: rgba(244, 114, 114, 0.35);
                }

                .ob-order-bar.bid {
                    background: rgba(59, 246, 162, 0.35);
                }
                .empty-orders {
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    padding: 24px 12px;
                    color: var(--text-muted);
                    font-size: 13px;
                }
            `}</style>
        </Layout>
    );
};

export default TradingBotMonitor;
