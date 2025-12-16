# Admin Panel Backend API Documentation

**Base URL:** `http://localhost:8887`

**Version:** 1.0.0

**Last Updated:** 2025-12-11

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Common Response Format](#common-response-format)
4. [Permissions](#permissions)
5. [API Endpoints](#api-endpoints)
   - [Health Check](#health-check)
   - [Authentication](#authentication-apis)
   - [Kline](#kline-apis)
   - [RBAC (Role-Based Access Control)](#rbac-apis)
   - [API Keys](#api-keys-apis)
   - [Switchers](#switchers-apis)
   - [Settings](#settings-apis)
   - [WebSocket](#websocket-apis)

---

## Overview

This API provides backend services for the Admin Panel, including user authentication, role-based access control (RBAC), API key management, and trading strategy configuration.

### Technologies
- **Framework:** Go Chi Router
- **Databases:** SQLite (Users, Roles), MongoDB (API Keys, Switchers, Settings)
- **Authentication:** JWT + TOTP (2FA)

---

## Authentication

All protected endpoints require a valid JWT token in the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

### Login Flow

1. Call `POST /api/auth/login` with username and password
2. If `requires_totp` is `true`, call `POST /api/auth/verify-totp` with the TOTP code
3. Use the returned `token` for subsequent requests

---

## Common Response Format

### Success Response
```json
{
  "message": "operation successful",
  "data": { ... }
}
```

### Error Response
```json
{
  "error": "error message"
}
```

---

## Permissions

| Permission | Description |
|------------|-------------|
| `view:dashboard` | View dashboard |
| `view:kline` | View K-line charts |
| `view:api_keys` | View API keys (list, get, platforms) |
| `view:settings` | View settings and switchers |
| `manage:users` | Manage users |
| `manage:roles` | Manage roles and permissions |
| `manage:api_keys` | Create, update, delete API keys |
| `manage:settings` | Create, update, delete settings and switchers |

---

## API Endpoints

---

### Health Check

#### GET /health
Check if the server is running.

**Authentication:** None

**Response:**
```json
{
  "status": "ok"
}
```

---

### Authentication APIs

#### POST /api/auth/register
Register a new user account.

**Authentication:** None

**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response (201):**
```json
{
  "message": "user registered, please setup 2FA to activate your account",
  "data": {
    "user_id": 1,
    "totp_setup": {
      "secret": "BASE32SECRET",
      "qr_code": "data:image/png;base64,..."
    }
  }
}
```

**Errors:**
- `400` - Invalid request / Password too short
- `409` - User already exists

---

#### POST /api/auth/activate
Activate a user account by verifying TOTP code.

**Authentication:** None

**Request Body:**
```json
{
  "user_id": 1,
  "code": "123456"
}
```

**Response (200):**
```json
{
  "message": "account activated successfully"
}
```

**Errors:**
- `400` - Invalid code / 2FA not setup
- `404` - User not found

---

#### POST /api/auth/login
Login with username and password.

**Authentication:** None

**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response (200) - TOTP Required:**
```json
{
  "requires_totp": true,
  "requires_totp_setup": false,
  "temp_user_id": 1
}
```

**Response (200) - TOTP Setup Required:**
```json
{
  "requires_totp": false,
  "requires_totp_setup": true,
  "temp_user_id": 1,
  "totp_setup": {
    "secret": "BASE32SECRET",
    "qr_code": "data:image/png;base64,..."
  }
}
```

**Response (200) - Success (no TOTP):**
```json
{
  "requires_totp": false,
  "requires_totp_setup": false,
  "token": "jwt_token",
  "user": {
    "id": 1,
    "username": "admin",
    "roles": [...],
    "permissions": [...]
  }
}
```

**Errors:**
- `401` - Invalid credentials
- `403` - Account inactive

---

#### POST /api/auth/verify-totp
Verify TOTP code to complete login.

**Authentication:** None

**Request Body:**
```json
{
  "user_id": 1,
  "code": "123456"
}
```

**Response (200):**
```json
{
  "requires_totp": false,
  "token": "jwt_token",
  "user": {
    "id": 1,
    "username": "admin",
    "roles": [...],
    "permissions": [...]
  }
}
```

**Errors:**
- `400` - 2FA not enabled
- `401` - Invalid user / Invalid code

---

#### GET /api/auth/me
Get current user information.

**Authentication:** Required

**Response (200):**
```json
{
  "id": 1,
  "username": "admin",
  "is_active": true,
  "totp_enabled": true,
  "roles": [
    {
      "id": 1,
      "name": "admin",
      "description": "Administrator"
    }
  ],
  "permissions": [
    "view:dashboard",
    "manage:users",
    "manage:roles"
  ]
}
```

---

#### POST /api/auth/change-password
Change user password.

**Authentication:** Required

**Request Body:**
```json
{
  "current_password": "string",
  "new_password": "string"
}
```

**Response (200):**
```json
{
  "message": "password changed successfully"
}
```

**Errors:**
- `400` - Current password incorrect / New password same as old / Password too short

---

#### POST /api/auth/totp/rebind
Initiate 2FA rebind process.

**Authentication:** Required

**Request Body:**
```json
{
  "password": "string"
}
```

**Response (200):**
```json
{
  "message": "2FA rebind initiated, scan the QR code and verify",
  "data": {
    "secret": "BASE32SECRET",
    "qr_code": "data:image/png;base64,..."
  }
}
```

---

#### POST /api/auth/totp/rebind/confirm
Confirm 2FA rebind with new TOTP code.

**Authentication:** Required

**Request Body:**
```json
{
  "code": "123456"
}
```

**Response (200):**
```json
{
  "message": "2FA rebind successful"
}
```

---

#### POST /api/auth/totp/rebind/cancel
Cancel 2FA rebind process.

**Authentication:** Required

**Response (200):**
```json
{
  "message": "2FA rebind cancelled"
}
```

---

### Kline APIs

#### GET /api/kline/symbols
Get available trading symbols.

**Authentication:** Required  
**Permission:** `view:kline`

**Response (200):**
```json
{
  "data": ["BTCUSDT", "ETHUSDT", "SOLUSDT"]
}
```

---

#### GET /api/kline/intervals
Get available K-line intervals.

**Authentication:** Required  
**Permission:** `view:kline`

**Response (200):**
```json
{
  "data": ["1m", "5m", "15m", "1h", "4h", "1d"]
}
```

---

### RBAC APIs

#### GET /api/rbac/roles
List all roles.

**Authentication:** Required  
**Permission:** `manage:roles`

**Response (200):**
```json
{
  "data": [
    {
      "id": 1,
      "name": "admin",
      "description": "Administrator",
      "permissions": ["view:dashboard", "manage:users", "manage:roles"]
    }
  ]
}
```

---

#### POST /api/rbac/roles
Create a new role.

**Authentication:** Required  
**Permission:** `manage:roles`

**Request Body:**
```json
{
  "name": "operator",
  "description": "System operator",
  "permissions": ["view:dashboard", "view:kline"]
}
```

**Response (201):**
```json
{
  "message": "role created successfully",
  "data": {
    "id": 2,
    "name": "operator",
    "description": "System operator",
    "permissions": ["view:dashboard", "view:kline"]
  }
}
```

---

#### GET /api/rbac/roles/{id}
Get a specific role.

**Authentication:** Required  
**Permission:** `manage:roles`

**Response (200):**
```json
{
  "data": {
    "id": 1,
    "name": "admin",
    "description": "Administrator",
    "permissions": ["view:dashboard", "manage:users", "manage:roles"]
  }
}
```

---

#### PUT /api/rbac/roles/{id}
Update a role.

**Authentication:** Required  
**Permission:** `manage:roles`

**Request Body:**
```json
{
  "name": "admin",
  "description": "Updated description"
}
```

---

#### DELETE /api/rbac/roles/{id}
Delete a role.

**Authentication:** Required  
**Permission:** `manage:roles`

**Response (200):**
```json
{
  "message": "role deleted successfully"
}
```

---

#### PUT /api/rbac/roles/{id}/permissions
Set permissions for a role.

**Authentication:** Required  
**Permission:** `manage:roles`

**Request Body:**
```json
{
  "permissions": ["view:dashboard", "view:kline", "manage:users"]
}
```

---

#### GET /api/rbac/permissions
Get all available permissions.

**Authentication:** Required  
**Permission:** `manage:roles`

**Response (200):**
```json
{
  "data": [
    "view:dashboard",
    "view:kline",
    "view:api_keys",
    "view:settings",
    "manage:users",
    "manage:roles",
    "manage:api_keys",
    "manage:settings"
  ]
}
```

---

#### GET /api/rbac/users
List all users.

**Authentication:** Required  
**Permission:** `manage:roles`

**Response (200):**
```json
{
  "data": [
    {
      "id": 1,
      "username": "admin",
      "is_active": true,
      "roles": [...],
      "permissions": [...]
    }
  ]
}
```

---

#### GET /api/rbac/users/{id}
Get a specific user.

**Authentication:** Required  
**Permission:** `manage:roles`

---

#### POST /api/rbac/users/{id}/roles
Assign a role to a user.

**Authentication:** Required  
**Permission:** `manage:roles`

**Request Body:**
```json
{
  "role_id": 1
}
```

---

#### DELETE /api/rbac/users/{id}/roles/{roleId}
Remove a role from a user.

**Authentication:** Required  
**Permission:** `manage:roles`

---

### API Keys APIs

#### GET /api/api-keys
List all API keys.

**Authentication:** Required  
**Permission:** `view:api_keys`

**Response (200):**
```json
{
  "data": [
    {
      "id": 0,
      "mongo_id": "6937dc0457b5c4ad96495962",
      "user_id": 0,
      "name": "btcc staging",
      "platform": "btcc",
      "api_key_masked": "8e65****00e8",
      "api_secret_masked": "219f****2777",
      "is_testnet": true,
      "is_active": true,
      "created_at": "2024-12-10T10:00:00Z",
      "updated_at": "2024-12-10T10:00:00Z"
    }
  ]
}
```

---

#### GET /api/api-keys/platforms
Get available trading platforms.

**Authentication:** Required  
**Permission:** `view:api_keys`

**Response (200):**
```json
{
  "data": ["binance", "btcc", "okx", "bybit"]
}
```

---

#### GET /api/api-keys/{id}
Get a specific API key.

**Authentication:** Required  
**Permission:** `view:api_keys`

**Parameters:**
- `id` - MongoDB ObjectID string or numeric ID

---

#### POST /api/api-keys
Create a new API key.

**Authentication:** Required  
**Permission:** `manage:api_keys`

**Request Body:**
```json
{
  "name": "my btcc key",
  "platform": "btcc",
  "api_key": "your-api-key",
  "api_secret": "your-api-secret",
  "is_testnet": true
}
```

**Response (201):**
```json
{
  "message": "api key created successfully",
  "data": {
    "mongo_id": "6937dc0457b5c4ad96495962",
    "name": "my btcc key",
    "platform": "btcc",
    "api_key_masked": "your****-key",
    "api_secret_masked": "your****cret",
    "is_testnet": true,
    "is_active": true
  }
}
```

---

#### PUT /api/api-keys/{id}
Update an API key.

**Authentication:** Required  
**Permission:** `manage:api_keys`

**Request Body:**
```json
{
  "name": "updated name",
  "api_key": "new-api-key",
  "api_secret": "new-api-secret",
  "is_testnet": false,
  "is_active": true
}
```

---

#### DELETE /api/api-keys/{id}
Delete an API key.

**Authentication:** Required  
**Permission:** `manage:api_keys`

**Response (200):**
```json
{
  "message": "api key deleted successfully"
}
```

---

### Switchers APIs

Switchers control the enable/disable status of trading pairs.

#### GET /api/switchers
List all switchers.

**Authentication:** Required  
**Permission:** `view:settings`

**Response (200):**
```json
{
  "data": [
    {
      "id": "6937db4e57b5c4ad96495957",
      "pairs": {
        "SOL_USDT": { "enable": true },
        "BTC_USDT": { "enable": false }
      }
    }
  ]
}
```

---

#### GET /api/switchers/{id}
Get a specific switcher.

**Authentication:** Required  
**Permission:** `view:settings`

---

#### POST /api/switchers
Create a new switcher.

**Authentication:** Required  
**Permission:** `manage:settings`

**Request Body:**
```json
{
  "pairs": {
    "SOL_USDT": { "enable": true },
    "BTC_USDT": { "enable": false }
  }
}
```

---

#### PUT /api/switchers/{id}
Update a switcher.

**Authentication:** Required  
**Permission:** `manage:settings`

**Request Body:**
```json
{
  "pairs": {
    "SOL_USDT": { "enable": false },
    "ETH_USDT": { "enable": true }
  }
}
```

---

#### PUT /api/switchers/{id}/pairs/{pair}
Update a single trading pair.

**Authentication:** Required  
**Permission:** `manage:settings`

**Parameters:**
- `id` - Switcher MongoDB ObjectID
- `pair` - Trading pair name (e.g., `SOL_USDT`)

**Request Body:**
```json
{
  "enable": true
}
```

---

#### DELETE /api/switchers/{id}
Delete a switcher.

**Authentication:** Required  
**Permission:** `manage:settings`

---

### Settings APIs

Settings contain strategy configurations for trading pairs.

#### GET /api/settings
List all settings.

**Authentication:** Required  
**Permission:** `view:settings`

**Response (200):**
```json
{
  "data": [
    {
      "id": "6937814057b5c4ad96495953",
      "base": "SOL",
      "quote": "USDT",
      "strategy": "JOE_BIDEN",
      "parameters": {
        "JOE_BIDEN": {
          "DEPTH": 10,
          "DEPTH_PRECISION": "0.01",
          "ORDER_LEVELS": 10
        }
      }
    }
  ]
}
```

---

#### GET /api/settings/search
Search settings by base and quote.

**Authentication:** Required  
**Permission:** `view:settings`

**Query Parameters:**
- `base` - Base currency (e.g., `SOL`)
- `quote` - Quote currency (e.g., `USDT`)

**Example:** `GET /api/settings/search?base=SOL&quote=USDT`

---

#### GET /api/settings/{id}
Get a specific setting.

**Authentication:** Required  
**Permission:** `view:settings`

---

#### POST /api/settings
Create a new setting.

**Authentication:** Required  
**Permission:** `manage:settings`

**Request Body:**
```json
{
  "base": "BTC",
  "quote": "USDT",
  "strategy": "JOE_BIDEN",
  "parameters": {
    "JOE_BIDEN": {
      "DEPTH": 10,
      "ORDER_LEVELS": 5
    }
  }
}
```

---

#### PUT /api/settings/{id}
Update a setting.

**Authentication:** Required  
**Permission:** `manage:settings`

**Request Body:**
```json
{
  "base": "BTC",
  "quote": "USDT",
  "strategy": "NEW_STRATEGY",
  "parameters": {
    "NEW_STRATEGY": { ... }
  }
}
```

---

#### PUT /api/settings/{id}/parameters/{strategy}
Update parameters for a specific strategy.

**Authentication:** Required  
**Permission:** `manage:settings`

**Parameters:**
- `id` - Setting MongoDB ObjectID
- `strategy` - Strategy name (e.g., `JOE_BIDEN`)

**Request Body:**
```json
{
  "parameters": {
    "DEPTH": 20,
    "ORDER_LEVELS": 15,
    "NEW_PARAM": "value"
  }
}
```

---

#### DELETE /api/settings/{id}
Delete a setting.

**Authentication:** Required  
**Permission:** `manage:settings`

---

### WebSocket APIs

WebSocket connections are used for real-time data streaming from exchanges.

---

#### WS /ws/kline
Connect to Binance K-line (candlestick) stream for real-time market data.

**Authentication:** None (public stream)

**URL:** `ws://localhost:8887/ws/kline`

**Description:**  
This WebSocket endpoint proxies K-line data from Binance's public WebSocket API. Clients can subscribe to multiple symbol/interval combinations simultaneously.

---

##### Client → Server Messages

###### Subscribe to K-line
Subscribe to receive K-line updates for a specific symbol and interval.

```json
{
  "action": "subscribe",
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1m"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | string | Yes | Must be `"subscribe"` |
| `data.symbol` | string | Yes | Trading pair (e.g., `BTCUSDT`, `ETHUSDT`) |
| `data.interval` | string | Yes | K-line interval: `1m`, `5m`, `15m`, `30m`, `1h`, `4h`, `1d`, etc. |

###### Unsubscribe from K-line

```json
{
  "action": "unsubscribe",
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1m"
  }
}
```

---

##### Server → Client Messages

K-line data is forwarded directly from Binance in their native format:

```json
{
  "e": "kline",
  "E": 1672531200000,
  "s": "BTCUSDT",
  "k": {
    "t": 1672531200000,
    "T": 1672531259999,
    "s": "BTCUSDT",
    "i": "1m",
    "f": 123456789,
    "L": 123456799,
    "o": "16800.00",
    "c": "16810.50",
    "h": "16815.00",
    "l": "16795.00",
    "v": "100.5",
    "n": 50,
    "x": false,
    "q": "1687552.75",
    "V": "50.25",
    "Q": "843776.38"
  }
}
```

| Field | Description |
|-------|-------------|
| `e` | Event type (`kline`) |
| `E` | Event time (Unix timestamp in ms) |
| `s` | Symbol |
| `k.t` | K-line start time |
| `k.T` | K-line close time |
| `k.i` | Interval |
| `k.o` | Open price |
| `k.c` | Close price |
| `k.h` | High price |
| `k.l` | Low price |
| `k.v` | Base asset volume |
| `k.n` | Number of trades |
| `k.x` | Is this K-line closed? |

---

#### WS /ws/trading
Connect to trading stream for real-time market data and private order updates.

**Authentication:** Required via query parameter `token`

**URL:** `ws://localhost:8887/ws/trading?token=your_jwt_token`

**Supported Platforms:** Binance, BTCC

**Description:**  
This WebSocket endpoint provides authenticated access to exchange data including order books, K-lines, orders, and account assets. Before subscribing to any data streams, clients must first connect to an API key.

---

##### Connection Flow

1. Connect to WebSocket with JWT token
2. Send `connect` action with API key ID
3. Subscribe to desired data streams (kline, orderbook, orders, asset, etc.)

---

##### Client → Server Messages

###### Connect to API Key
Before subscribing to data, you must connect to an API key:

```json
{
  "action": "connect",
  "apiKeyId": 123
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | string | Yes | Must be `"connect"` |
| `apiKeyId` | integer | Yes | The ID of the API key to use |

**Success Response:**
```json
{
  "type": "connected",
  "platform": "btcc",
  "timestamp": 1702300800000,
  "data": {
    "apiKeyId": 123,
    "platform": "btcc",
    "isTestnet": false,
    "name": "My BTCC Key"
  }
}
```

---

###### Subscribe to Data Stream

```json
{
  "action": "subscribe",
  "apiKeyId": 123,
  "type": "orderbook",
  "symbol": "BTCUSDT",
  "interval": "1m"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | string | Yes | Must be `"subscribe"` |
| `apiKeyId` | integer | Yes | API key ID (must match connected key) |
| `type` | string | Yes | Subscription type (see table below) |
| `symbol` | string | Conditional | Trading pair (required for most types) |
| `interval` | string | Conditional | K-line interval (required for `kline` type) |

**Subscription Types:**

| Type | Description | Requires Symbol | Requires Interval | Public/Private |
|------|-------------|-----------------|-------------------|----------------|
| `kline` | Candlestick/K-line data | Yes | Yes | Public |
| `orderbook` or `depth` | Order book depth updates | Yes | No | Public |
| `trades` or `deals` | Recent trades/deals | Yes | No | Public |
| `state` | Market state (BTCC only) | No | No | Public |
| `orders` | User's active orders | Yes | No | Private |
| `asset` | Account balance updates (BTCC only) | No | No | Private |

---

###### Unsubscribe from Data Stream

```json
{
  "action": "unsubscribe",
  "type": "orderbook",
  "symbol": "BTCUSDT"
}
```

---

##### Server → Client Messages

All server responses follow this format:

```json
{
  "type": "orderbook",
  "platform": "btcc",
  "symbol": "BTCUSDT",
  "timestamp": 1702300800000,
  "data": { ... },
  "error": ""
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Response type: `connected`, `kline`, `orderbook`, `orders`, `asset`, `trades`, `state`, `error` |
| `platform` | string | Exchange platform: `binance`, `btcc` |
| `symbol` | string | Trading pair |
| `timestamp` | integer | Event timestamp (Unix ms) |
| `data` | object | The actual data payload |
| `error` | string | Error message (only present on errors) |

---

##### Response Types

###### Order Book Response (`orderbook`)

```json
{
  "type": "orderbook",
  "platform": "btcc",
  "symbol": "BTCUSDT",
  "timestamp": 1702300800000,
  "data": {
    "symbol": "BTCUSDT",
    "lastUpdateId": 123456789,
    "bids": [
      { "price": "42000.50", "quantity": "1.5" },
      { "price": "42000.00", "quantity": "2.3" }
    ],
    "asks": [
      { "price": "42001.00", "quantity": "0.8" },
      { "price": "42001.50", "quantity": "1.2" }
    ],
    "bestBid": { "price": "42000.50", "quantity": "1.5" },
    "bestAsk": { "price": "42001.00", "quantity": "0.8" },
    "spread": "0.50",
    "timestamp": 1702300800000
  }
}
```

###### K-line Response (`kline`)

```json
{
  "type": "kline",
  "platform": "binance",
  "symbol": "BTCUSDT",
  "timestamp": 1702300800000,
  "data": {
    "s": "BTCUSDT",
    "t": 1702300800000,
    "T": 1702300859999,
    "i": "1m",
    "o": "42000.00",
    "c": "42050.00",
    "h": "42060.00",
    "l": "41990.00",
    "v": "150.5",
    "q": "6322575.00",
    "n": 1250,
    "x": false
  }
}
```

###### Orders Response (`orders`)

```json
{
  "type": "orders",
  "platform": "btcc",
  "symbol": "BTCUSDT",
  "timestamp": 1702300800000,
  "data": {
    "orderId": "12345678",
    "symbol": "BTCUSDT",
    "side": "BUY",
    "type": "LIMIT",
    "price": "42000.00",
    "quantity": "0.5",
    "executedQty": "0.0",
    "status": "NEW",
    "timeInForce": "GTC",
    "createTime": 1702300800000,
    "updateTime": 1702300800000
  }
}
```

###### Asset Response (`asset`) - BTCC Only

```json
{
  "type": "asset",
  "platform": "btcc",
  "timestamp": 1702300800000,
  "data": {
    "USDT": {
      "available": "10000.00",
      "freeze": "500.00"
    },
    "BTC": {
      "available": "1.5",
      "freeze": "0.0"
    }
  }
}
```

###### Trades Response (`trades`)

```json
{
  "type": "trades",
  "platform": "btcc",
  "symbol": "BTCUSDT",
  "timestamp": 1702300800000,
  "data": [
    {
      "id": 12345,
      "price": "42000.50",
      "amount": "0.1",
      "type": "buy",
      "time": 1702300800000
    }
  ]
}
```

###### Market State Response (`state`) - BTCC Only

```json
{
  "type": "state",
  "platform": "btcc",
  "timestamp": 1702300800000,
  "data": {
    "BTCUSDT": {
      "period": 86400,
      "last": "42000.00",
      "open": "41500.00",
      "close": "42000.00",
      "high": "42500.00",
      "low": "41000.00",
      "volume": "1500.5",
      "deal": "63007500.00"
    }
  }
}
```

###### Error Response

```json
{
  "type": "error",
  "timestamp": 1702300800000,
  "error": "not connected to any API key, call connect first"
}
```

---

##### Platform-Specific Notes

###### Binance
- Streams are combined into a single WebSocket connection
- Stream names follow format: `{symbol}@{type}` (e.g., `btcusdt@kline_1m`, `btcusdt@depth`)
- Reconnection is automatic when subscription list changes

###### BTCC
- Uses JSON-RPC style messaging with `method` and `params`
- Supports per-message Deflate compression (RFC 7692)
- Requires periodic ping messages (handled automatically)
- Private streams (orders, asset) require authentication with API key
- Authentication uses `server.accessid_auth` with:
  - `access_id`: The API key
  - `authorization`: SHA256 hash of the API secret
  - `tonce`: Unix timestamp in milliseconds

---

##### Complete Example Flow

```javascript
// 1. Connect to WebSocket
const ws = new WebSocket('ws://localhost:8887/ws/trading?token=your_jwt_token');

ws.onopen = () => {
  // 2. Connect to API key
  ws.send(JSON.stringify({
    action: 'connect',
    apiKeyId: 123
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  
  if (msg.type === 'connected' && msg.data?.apiKeyId) {
    // 3. Subscribe to order book
    ws.send(JSON.stringify({
      action: 'subscribe',
      apiKeyId: 123,
      type: 'orderbook',
      symbol: 'BTCUSDT'
    }));
    
    // 4. Subscribe to orders (private)
    ws.send(JSON.stringify({
      action: 'subscribe',
      apiKeyId: 123,
      type: 'orders',
      symbol: 'BTCUSDT'
    }));
  }
  
  // Handle incoming data
  console.log('Received:', msg.type, msg.data);
};
```

---

## Error Codes

| HTTP Code | Description |
|-----------|-------------|
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource not found |
| 409 | Conflict - Resource already exists |
| 500 | Internal Server Error |

---

## MongoDB Collections

| Collection | Database | Description |
|------------|----------|-------------|
| `api_token` | `strategist` | API key storage |
| `switcher` | `strategist` | Trading pair enable/disable status |
| `setting` | `strategist` | Strategy configuration |

---

## Configuration

Configuration is stored in `config/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8887

database:
  driver: "sqlite3"
  dsn: "data/control_page.db"

mongodb:
  uri: "mongodb://localhost:27017"
  database: "strategist"

jwt:
  secret: "your-super-secret-key"
  expiration: 24h

binance:
  websocket_url: "wss://stream.binance.com:9443/ws"
```
