package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	_listenAddr               = ":9999"
	_urlUpstreamMarketList    = "https://spotapi2.btcccdn.com/btcc_api_trade/market/list"
	_urlUpstreamMarketListDev = "https://spot.cryptouat.com:9910/btcc_api_trade/market/list"
	_urlUpstreamWebSocket     = "wss://spotprice2.btcccdn.com/ws"
	_urlUpstreamWebSocketDev  = "wss://spot.cryptouat.com:8700/ws"
	_indexContentType         = "text/html; charset=utf-8"

	prodAccessID  = "fb41aca3-9146-4e92-a2db-39e6387009e0"
	prodSecretKey = "dc12f77c-3021-4cc0-a3c4-f3cc60dd2081"

	// heng
	// devAccessID  = "0e179f65-db50-4af7-a688-4c93dec5683c"
	// devSecretKey = "4ad35511-01d9-440e-bdf8-6ec3e9eb4de5"

	// yanun
	devAccessID  = "8e650365-1853-4e86-a64b-6bd989a100e8"
	devSecretKey = "219ff50a-87f5-4e88-a75c-d247f4112777"
)

var marketHTTPClient = &http.Client{Timeout: 10 * time.Second}

var (
	devMode      = flag.Lookup("dev") != nil || os.Getenv("DEV") != ""
	envAccessID  = os.Getenv("ACCESS_ID")
	envSecretKey = os.Getenv("SECRET_KEY")
)

func urlUpstreamMarketList() string {
	if devMode {
		return _urlUpstreamMarketListDev
	}
	return _urlUpstreamMarketList
}

func urlUpstreamWebSocket() string {
	if devMode {
		return _urlUpstreamWebSocketDev
	}

	return _urlUpstreamWebSocket
}

func main() {
	wsProxy, err := newWebSocketProxy(urlUpstreamWebSocket())
	if err != nil {
		log.Fatalf("configure websocket proxy: %v", err)
	}

	addr := os.Getenv("ADDR")
	if len(addr) == 0 {
		addr = _listenAddr
	}

	mux := http.NewServeMux()
	mux.Handle("/", serveIndex())
	mux.HandleFunc("/api/markets", marketListHandler)
	mux.Handle("/ws", wsProxy)

	log.Printf("\n\n\tmonitor webpage: http://localhost%s\n\n", addr)
	server := &http.Server{Addr: addr, Handler: mux}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}

func serveIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		body, err := os.ReadFile("./index.html")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		accessID := prodAccessID
		secretKey := prodSecretKey
		env := "production"
		if devMode {
			accessID = devAccessID
			secretKey = devSecretKey
			env = "dev"
		}

		if len(envAccessID) != 0 && len(envSecretKey) != 0 {
			accessID = envAccessID
			secretKey = envSecretKey
		}

		w.Header().Set("Content-Type", _indexContentType)

		replacer := strings.NewReplacer(
			"{{ACCESS_ID}}", accessID,
			"{{SECRET_KEY}}", secretKey,
			"{{ENVIRONMENT}}", env,
		)

		if _, err := replacer.WriteString(w, string(body)); err != nil {
			log.Printf("write index: %v", err)
		}
	})
}

func marketListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, urlUpstreamMarketList(), nil)
	if err != nil {
		http.Error(w, "create upstream request", http.StatusInternalServerError)
		return
	}

	resp, err := marketHTTPClient.Do(upstreamReq)
	if err != nil {
		log.Printf("market list proxy request: %v", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if resp.Header.Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("write market list body: %v", err)
	}
}

func newWebSocketProxy(rawURL string) (http.Handler, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse upstream websocket url: %w", err)
	}

	originalScheme := parsed.Scheme
	switch parsed.Scheme {
	case "wss":
		parsed.Scheme = "https"
	case "ws":
		parsed.Scheme = "http"
	case "http", "https":
		// acceptable as-is
	default:
		return nil, fmt.Errorf("unsupported upstream websocket scheme: %s", parsed.Scheme)
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	proxy := httputil.NewSingleHostReverseProxy(parsed)
	proxy.FlushInterval = 0
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = parsed.Path
		req.URL.RawPath = parsed.RawPath
		req.Host = parsed.Host
		originScheme := "https"
		if originalScheme == "ws" {
			originScheme = "http"
		}
		if originalScheme == "http" || originalScheme == "https" {
			originScheme = originalScheme
		}
		req.Header.Set("Origin", originScheme+"://"+parsed.Host)
		req.Header.Del("Accept-Encoding")
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		log.Printf("websocket proxy error: %v", proxyErr)
		http.Error(w, "upstream websocket failure", http.StatusBadGateway)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection := strings.ToLower(r.Header.Get("Connection"))
		if !strings.Contains(connection, "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "upgrade required", http.StatusBadRequest)
			return
		}
		proxy.ServeHTTP(w, r)
	}), nil
}
