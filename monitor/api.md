## HTTP Signature Rules

### Signing Requirements
- Include an `authorization` field in the HTTP headers for every request.
- Include a `tm` (timestamp) field in the request payload or query string.
- Generate the signature by concatenating all request parameters, appending the account's `secret_key`, and the timestamp from the headers, sorting the segments alphabetically, then producing an MD5 hash of the resulting string.

### GET Signing Example
1. Original request: `https://spotapi2.btcccdn.com/test?cc=46&aa=mm&bb=56`
2. Concatenate parameters: `aa=mm&bb=56&cc=46`
3. Append secret key and timestamp: `aa=mm&bb=56&cc=46&secret_key=4b2211c547dc29f777e9804873abc412&tm=1698400838`
4. Sort key-value pairs alphabetically (already sorted in this case).
5. MD5 hash the final string to produce the authorization signature.
6. POST requests follow the same rules; sort the body keys before signing.

## IP Access Restriction
- High-frequency access over long periods may trigger an IP block.
- Restrictions are applied at the IP level, not per API key.
- Blocked requests return HTTP status `429` with a `block_until` timestamp indicating when access is restored.

```json
{
  "block_until": 1724911030,
  "error": "Too many requests, you are blocked."
}
```

## HTTP Error Codes

| Error Code | Error Message             | Description                  |
| ---------- | ------------------------- | ---------------------------- |
| 201        | CodeNormalError           | General error                |
| 202        | CodeParamError            | Parameter error              |
| 203        | CodeDBError               | Database error               |
| 204        | CodeValidSignError        | Signature verification error |
| 205        | CodeValidAccessIdError    | Access ID authentication error |
| 206        | CodeLimitError            | Limit error                  |
| 207        | CodeCloseMarketError      | Market closure error         |
| 429        | StatusTooManyRequests     | Too many requests            |

## Gateway Error Codes

| Error Code | Error Message       | Description             |
| ---------- | ------------------- | ----------------------- |
| 1          | invalid argument    | Parameter error         |
| 2          | internal error      | Network error           |
| 3          | service unavailable | Service unavailable     |
| 4          | unknown command     | Unknown request         |
| 5          | service timeout     | Request timeout         |
| 6          | require auth        | Authentication required |

## Python Signing Examples
```python
import hashlib
import time
import urllib.parse
from typing import Dict, Tuple

import requests


def generate_signature(params: Dict[str, str], secret_key: str) -> str:
    "Return the MD5 signature for the given parameters."
    pairs = [f"{k}={v}" for k, v in params.items()]
    concatenated = "&".join(pairs)
    concatenated_with_key = f"{concatenated}&secret_key={secret_key}"
    sorted_pairs = "&".join(sorted(concatenated_with_key.split("&")))
    return hashlib.md5(sorted_pairs.encode("utf-8")).hexdigest()


def put_market() -> None:
    secret_key = "473740ab-c249-4f5e-803b-f6db21ec97e0"
    payload = {
        "access_id": "a31b51a3-92ec-4ebe-935e-2a9aeadfc268",
        "tm": int(time.time()),
        "market": "BTCUSDT",
        "side": 1,
        "option": 0,
        "amount": "55.90",
        "source": "python example",
    }
    base_url = "https://spotapi2.btcccdn.com/btcc_api_trade/order/market"
    signature = generate_signature(payload, secret_key)
    headers = {"authorization": signature}
    response = requests.post(base_url, headers=headers, json=payload, timeout=10)
    print(response.text)


def build_get_request(base_url: str, params: Dict[str, str], secret_key: str) -> Tuple[str, Dict[str, str]]:
    signature = generate_signature(params, secret_key)
    full_url = f"{base_url}?{urllib.parse.urlencode(params)}"
    headers = {"authorization": signature}
    return full_url, headers


def get_pending() -> None:
    secret_key = "473740ab-c249-4f5e-803b-f6db21ec97e0"
    access_id = "a31b51a3-92ec-4ebe-935e-2a9aeadfc268"
    params = {
        "tm": int(time.time()),
        "access_id": access_id,
        "market": "BTCUSDT",
        "side": 0,
        "offset": 0,
        "limit": 50,
    }
    base_url = "https://spotapi2.btcccdn.com/btcc_api_trade/order/pending"
    full_url, headers = build_get_request(base_url, params, secret_key)
    response = requests.get(full_url, headers=headers, timeout=10)
    print(response.text)


if __name__ == "__main__":
    put_market()
    get_pending()
```

## Market Data API

### 3. Query Information for All Trading Pairs
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/market/list`

**Response Fields (per market)**
| Field       | Type    | Description                                                      |
| ----------- | ------- | ---------------------------------------------------------------- |
| money       | String  | Quote currency                                                   |
| stock       | String  | Base currency                                                    |
| name        | String  | Trading pair name                                                |
| fee_prec    | Integer | Fee precision                                                    |
| money_prec  | Integer | Quote currency precision                                         |
| stock_prec  | Integer | Base currency precision                                          |
| min_amount  | String  | Minimum tradable amount                                          |
| switch      | Bool    | Trading status (`true` = tradable)                               |
| open_time   | Integer | When `switch` is false, timestamp (s) when the market reopens    |

**Sample Response**
```json
{
  "error": {
    "code": 0,
    "message": ""
  },
  "result": [
    {
      "money": "USDT",
      "stock": "MINA",
      "name": "MINAUSDT",
      "fee_prec": 4,
      "money_prec": 4,
      "stock_prec": 1,
      "min_amount": "50",
      "switch": true
    },
    {
      "money": "USDT",
      "stock": "ZRO",
      "name": "ZROUSDT",
      "fee_prec": 4,
      "money_prec": 3,
      "stock_prec": 2,
      "min_amount": "5",
      "switch": true
    }
  ],
  "id": 0
}
```

## WebSocket API

### 1. Description
- Endpoint: `wss://spotprice2.btcccdn.com/ws`
- Enable per-message Deflate compression (RFC 7692) when connecting.
- Request payloads use JSON objects with keys:
  - `method`: String identifier of the operation
  - `params`: Array of parameters for the operation
  - `id`: Integer request identifier
- Successful responses set `error` to `null` and populate `result`; failures return an error object.
- Push notifications use the same envelope with `id` set to `null`.

**Request Example**
```json
{
  "id": 1,
  "method": "state.subscribe",
  "params": []
}
```

**Successful Response Example**
```json
{
  "error": null,
  "id": 1,
  "result": "success"
}
```

**Error Response Example**
```json
{
  "error": {
    "code": 1,
    "message": "auth failed"
  },
  "id": 1,
  "result": null
}
```

### 2. User Authentication (App Users)
- Method: `server.auth`

**Parameters**
| Index | Type   | Required | Description |
| ----- | ------ | -------- | ----------- |
| 0     | String | Yes      | User token  |

**Response Fields**
| Field  | Type    | Description               |
| ------ | ------- | ------------------------- |
| status | String  | `"success"` on success    |
| flag   | Integer | Authenticated user ID     |

**Sample Request**
```json
{
  "method": "server.auth",
  "params": [
    "00000151741724322824208"
  ],
  "id": 1
}
```

### 3. Endpoint Authentication (OpenAPI Users)
- Method: `server.accessid_auth`

**Parameters**
| Index | Type   | Required | Description                                                  |
| ----- | ------ | -------- | ------------------------------------------------------------ |
| 0     | String | Yes      | `access_id`                                                  |
| 1     | String | Yes      | SHA256 hash of the access key rendered as a 64-char hex string |

### 4. Ping
- Method: `server.ping`
- Parameters: none
- Response: `{ "status": "success" }`

### 5. Get Server Time
- Method: `server.time`
- Parameters: none
- Response: `{ "timestamp": 1724728663 }`


### 6. Query Candlestick Data
- Method: `kline.query`

**Parameters**
| Index | Type    | Required | Description       |
| ----- | ------- | -------- | ----------------- |
| 0     | String  | Yes      | Market name       |
| 1     | Integer | Yes      | Start time (s)    |
| 2     | Integer | Yes      | End time (s)      |
| 3     | Integer | Yes      | Interval (seconds)|

**Result Entry**: `[timestamp, open, close, high, low, volume, amount, market]`

### 7. Subscribe to Candlestick Data
- Method: `kline.subscribe`

**Parameters**
| Index | Type    | Required | Description               |
| ----- | ------- | -------- | ------------------------- |
| 0     | String  | Yes      | Market name               |
| 1     | Integer | Yes      | Period identifier         |

### 8. Candlestick Push Notification
- Method: `kline.update`
- Parameters: array of kline rows; each row follows the structure described in section 6.

**Sample Push**
```json
{
  "method": "kline.update",
  "params": [
    [
      1724384031,
      "60621.24",
      "60621.24",
      "60621.24",
      "60621.24",
      "0.00069",
      "41.8286556",
      "BTCUSDT"
    ]
  ],
  "id": null
}
```

### 9. Unsubscribe from Candlestick Data
- Method: `kline.unsubscribe`
- Parameters: none

### 10. Query Depth
- Method: `depth.query`

**Parameters**
| Index | Type    | Required | Description                                                      |
| ----- | ------- | -------- | ---------------------------------------------------------------- |
| 0     | String  | Yes      | Market name                                                      |
| 1     | Integer | Yes      | Depth size (`5`, `10`, `20`, `50`)                               |
| 2     | String  | Yes      | Merge precision (e.g., `"0.01"`)                                 |

**Result Fields**
| Field | Type       | Description                  |
| ----- | ---------- | ---------------------------- |
| asks  | String[][] | Ask price/quantity ladder    |
| bids  | String[][] | Bid price/quantity ladder    |
| last  | String     | Latest traded price          |
| time  | Integer    | Snapshot timestamp (ms)      |
| checksum | Integer | Depth checksum               |

### 12. Depth Push Notification
- Method: `depth.update`
- Parameters:
  - `params[0]`: Boolean, `true` for a full snapshot, `false` for incremental update
  - `params[1]`: Object with `asks`, `bids`, `last`, `time`, `checksum`
  - `params[2]`: Market name (when subscribed to multiple markets)

```json
{
    "id": null,
    "method": "depth.update",
    "params": [
        false,
        {
            "asks": [
                [
                    "60849.26",
                    "0"
                ],
                [
                    "60849.27",
                    "0.05260"
                ]
            ],
            "bids": [
                [
                    "60846.85",
                    "0"
                ],
                [
                    "60846.61",
                    "0.07007"
                ]
            ],
            "checksum": 436966352,
            "last": "60848.00",
            "time": 1724392426713
        },
        "BTCUSDT"
    ]
}
```

### 11. Subscribe to Depth
- Method: `depth.subscribe`
- Parameters: same as `depth.query`

### 12. Depth Push Notification
- Method: `depth.update`
- Parameters:
  - `params[0]`: Boolean, `true` for a full snapshot, `false` for incremental update
  - `params[1]`: Object with `asks`, `bids`, `last`, `time`, `checksum`
  - `params[2]`: Market name (when subscribed to multiple markets)

### 13. Unsubscribe from Depth
- Method: `depth.unsubscribe`
- Parameters: none

### 14. Query Market Status
- Method: `state.query`
- Parameters: none

### 15. Subscribe to Market Status
- Method: `state.subscribe`
- Parameters: none

### 16. Market Status Push Notification
- Method: `state.update`
- Push parameters include overall trading status for all pairs.

### 17. Unsubscribe from Market Status
- Method: `state.unsubscribe`
- Parameters: none

### 18. Query Latest Transactions (Public)
- Method: `deals.query`

**Parameters**
| Index | Type    | Required | Description                    |
| ----- | ------- | -------- | ------------------------------ |
| 0     | String  | Yes      | Market name                    |
| 1     | Integer | Yes      | Result limit (up to 100 items) |

### 19. Query User's Latest Transactions
- Method: `user.deals.query`

**Parameters**
| Index | Type    | Required | Description         |
| ----- | ------- | -------- | ------------------- |
| 0     | String  | Yes      | Authenticated token |
| 1     | Integer | Yes      | Result limit        |

### 20. Subscribe to Latest Transactions
- Method: `deals.subscribe`

**Parameters**
| Index | Type   | Required | Description |
| ----- | ------ | -------- | ----------- |
| 0     | String | Yes      | Market name |

### 21. Deal Push Notification
- Method: `deals.update`
- Parameters: Array of recent trades where each entry contains `[id, price, amount, side, timestamp]`.

### 22. Unsubscribe from Deal Push
- Method: `deals.unsubscribe`
- Parameters: none

### 23. Query Orders
- Method: `orders.query`

**Parameters**
| Index | Type    | Required | Description                         |
| ----- | ------- | -------- | ----------------------------------- |
| 0     | String  | Yes      | Market name                         |
| 1     | Integer | Yes      | Result limit                        |
| 2     | Integer | Yes      | Offset                              |
| 3     | Integer | Yes      | Filter by side (`0`, `1`, `2`)      |

### 24. Query Plan Orders
- Method: `orders.stop.query`
- Parameters similar to `orders.query` with additional trigger filters.

### 25. Query User Orders
- Method: `user.orders.query`
- Requires authentication; parameters include `market`, `offset`, `limit`, optional `side`, and time range.

### 26. Query User Plan Orders
- Method: `user.orders.stop.query`
- Same filters as `user.orders.query` with additional trigger columns.

### 27. Subscribe to Orders
- Method: `orders.subscribe`
- Parameters: market name (optional to receive all markets).

### 28. Order Push Notification
- Method: `orders.update`
- Provides incremental order-book updates aligned with REST order structure.
```json
{
    "id": null,
    "method": "order.update",
    "params": [
        {
            "account": 0,
            "amount": "61.10000",
            "asset_fee": "0",
            "client_id": "",
            "ctime": 1724399839.3365309,
            "deal_fee": "0.000002970",
            "deal_money": "60.4913760",
            "deal_stock": "0.00099",
            "fee_asset": null,
            "fee_discount": "1",
            "id": 6297762817,
            "last_deal_amount": "0.00099",
            "last_deal_id": 212194073,
            "last_deal_price": "61102.40",
            "last_deal_time": 1724399839.336551,
            "last_role": 2,
            "left": "0.6086240",
            "maker_fee": "0",
            "market": "BTCUSDT",
            "mtime": 1724399839.336551,
            "option": 0,
            "price": "0",
            "side": 1,
            "source": "btcusdt_market_1",
            "taker_fee": "0.0030",
            "type": 2,
            "user": 15174
        }
    ]
}
```

### 29. Unsubscribe from Orders
- Method: `orders.unsubscribe`
- Parameters: none

### 30. Query Assets
- Method: `asset.query`
- Requires authentication token; returns current balances.

### 31. Subscribe to Assets
- Method: `asset.subscribe`
- Streams balance updates for the authenticated user.
```json
{
    "id": 1,
    "error": null,
    "result": {
        "ALGO": {
            "available": "0.00000000",
            "frozen": "0.00000000"
        },
        "AVAX": {
            "available": "0.00000000",
            "frozen": "0.00000000"
        },
        "BONK": {
            "available": "0.00000000",
            "frozen": "0.00000000"
        },
        "BTC": {
            "available": "0.26700128",
            "frozen": "0.64000000"
        },
        "ETH": {
            "available": "320.21674797",
            "frozen": "0.00000000"
        },
        "GALA": {
            "available": "0.00000000",
            "frozen": "0.00000000"
        },
        "TON": {
            "available": "0.00000000",
            "frozen": "0.00000000"
        },
        "WLD": {
            "available": "0.00000000",
            "frozen": "0.00000000"
        }
    }
}
```

### 32. Asset Push Notification
- Method: `asset.update`
- Sends changed asset balances; each entry matches the REST asset format.

```json
{
    "id": null,
    "method": "asset.update",
    "params": [
        {
            "USDT": {
                "available": "265377.80514156",
                "flag": 15174,
                "frozen": "0.00000000"
            }
        }
    ]
}
```

### 33. Unsubscribe from Assets
- Method: `asset.unsubscribe`
- Parameters: none