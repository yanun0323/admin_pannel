# Spot Trading API (April 2025)

## Table of Contents
- [HTTP Signature Rules](#http-signature-rules)
  - [Signing Requirements](#signing-requirements)
  - [GET Signing Example](#get-signing-example)
- [IP Access Restriction](#ip-access-restriction)
- [HTTP Error Codes](#http-error-codes)
- [Gateway Error Codes](#gateway-error-codes)
- [Python Signing Examples](#python-signing-examples)
- [API Information](#api-information)
  - [Production Environment](#production-environment)
- [General Endpoints](#general-endpoints)
  - [1. Test Connectivity (PING)](#1-test-connectivity-ping)
  - [2. Check Server Time](#2-check-server-time)
- [Asset API](#asset-api)
  - [1. Query User Assets](#1-query-user-assets)v
  - [2. Query Users' Transaction History](#2-query-users-transaction-history)
- [Market Data API](#market-data-api)
  - [3. Query Information for All Trading Pairs](#3-query-information-for-all-trading-pairs)
  - [4. Get Individual Pair Details](#4-get-individual-pair-details)
  - [5. Query Candlestick (Kline) Data](#5-query-candlestick-kline-data)
  - [6. Query Market Depth](#6-query-market-depth)
  - [7. Query Latest Price](#7-query-latest-price)
- [Trading API](#trading-api)
  - [Shared Order Response Fields](#shared-order-response-fields)
  - [1. Place Limit Orders](#1-place-limit-orders)
  - [2. Place Market Orders](#2-place-market-orders)
  - [3. Cancel Orders](#3-cancel-orders)
  - [4. Cancel Orders in Batch](#4-cancel-orders-in-batch)
  - [5. Cancel All Orders](#5-cancel-all-orders)
  - [6. Query Users' Pending Orders](#6-query-users-pending-orders)
  - [7. Query Pending Order Details](#7-query-pending-order-details)
  - [8. Query Order Transaction Details](#8-query-order-transaction-details)
  - [9. Query User Deal History](#9-query-user-deal-history)
  - [10. Query Users' Order History](#10-query-users-order-history)
  - [11. Query Order History Detail](#11-query-order-history-detail)
- [Conditional (Plan) Orders API](#conditional-plan-orders-api)
  - [12. Plan Limit Orders](#12-plan-limit-orders)
  - [13. Plan Market Orders](#13-plan-market-orders)
  - [14. Cancel Plan Orders](#14-cancel-plan-orders)
  - [15. Cancel All Plan Orders](#15-cancel-all-plan-orders)
  - [16. Query Plan Pending Orders](#16-query-plan-pending-orders)
  - [17. Query Plan Order History](#17-query-plan-order-history)
- [WebSocket API](#websocket-api)
  - [1. Description](#1-description)
  - [2. User Authentication (App Users)](#2-user-authentication-app-users)
  - [3. Endpoint Authentication (OpenAPI Users)](#3-endpoint-authentication-openapi-users)
  - [4. Ping](#4-ping)
  - [5. Get Server Time](#5-get-server-time)
  - [6. Query Candlestick Data](#6-query-candlestick-data)
  - [7. Subscribe to Candlestick Data](#7-subscribe-to-candlestick-data)
  - [8. Candlestick Push Notification](#8-candlestick-push-notification)
  - [9. Unsubscribe from Candlestick Data](#9-unsubscribe-from-candlestick-data)
  - [10. Query Depth](#10-query-depth)
  - [11. Subscribe to Depth](#11-subscribe-to-depth)
  - [12. Depth Push Notification](#12-depth-push-notification)
  - [13. Unsubscribe from Depth](#13-unsubscribe-from-depth)
  - [14. Query Market Status](#14-query-market-status)
  - [15. Subscribe to Market Status](#15-subscribe-to-market-status)
  - [16. Market Status Push Notification](#16-market-status-push-notification)
  - [17. Unsubscribe from Market Status](#17-unsubscribe-from-market-status)
  - [18. Query Latest Transactions (Public)](#18-query-latest-transactions-public)
  - [19. Query User's Latest Transactions](#19-query-users-latest-transactions)
  - [20. Subscribe to Latest Transactions](#20-subscribe-to-latest-transactions)
  - [21. Deal Push Notification](#21-deal-push-notification)
  - [22. Unsubscribe from Deal Push](#22-unsubscribe-from-deal-push)
  - [23. Query Orders](#23-query-orders)
  - [24. Query Plan Orders](#24-query-plan-orders)
  - [25. Query User Orders](#25-query-user-orders)
  - [26. Query User Plan Orders](#26-query-user-plan-orders)
  - [27. Subscribe to Orders](#27-subscribe-to-orders)
  - [28. Order Push Notification](#28-order-push-notification)
  - [29. Unsubscribe from Orders](#29-unsubscribe-from-orders)
  - [30. Query Assets](#30-query-assets)
  - [31. Subscribe to Assets](#31-subscribe-to-assets)
  - [32. Asset Push Notification](#32-asset-push-notification)
  - [33. Unsubscribe from Assets](#33-unsubscribe-from-assets)

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

## API Information

### Production Environment
- Base URL: `https://spotapi2.btcccdn.com`

## General Endpoints

### 1. Test Connectivity (PING)
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/ping`

**Response**
```json
{
  "code": 0,
  "data": "PONG",
  "message": "OK"
}
```

### 2. Check Server Time
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/time`

**Response**
```json
{
  "code": 0,
  "data": 1724728663,
  "message": "OK"
}
```

## Asset API

### 1. Query User Assets
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/asset/query`

**Query Parameters**
| Name      | Type    | Required | Description        |
| --------- | ------- | -------- | ------------------ |
| access_id | String  | Yes      | Access identifier  |
| tm        | Integer | Yes      | Unix timestamp (s) |

**Response Fields (per asset)**
| Name       | Type   | Description        |
| ---------- | ------ | ------------------ |
| available  | String | Available balance  |
| frozen     | String | Frozen balance     |

**Sample Response**
```json
{
  "error": null,
  "result": {
    "DOT": {
      "available": "99999999999999999.9999999999",
      "frozen": "0.00000000"
    },
    "BTC": {
      "available": "100000000000000.02087717",
      "frozen": "0.00000000"
    }
  },
  "id": 0
}
```

### 2. Query Users' Transaction History
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/asset/query_history`

**Query Parameters**
| Name       | Type    | Required | Description                                                                 |
| ---------- | ------- | -------- | --------------------------------------------------------------------------- |
| access_id  | String  | Yes      | Access identifier                                                           |
| tm         | Integer | Yes      | Unix timestamp (s)                                                          |
| asset      | String  | No       | Asset symbol to filter (empty for all)                                      |
| business   | String  | No       | Business type (`trade`, `fee`, etc.; empty for all)                         |
| start_time | Integer | No       | Start time (s)                                                              |
| end_time   | Integer | No       | End time (s)                                                                |
| offset     | Integer | Yes      | Pagination offset, starting from 0                                          |
| limit      | Integer | Yes      | Number of records to return (maximum 100)                                   |

**Response Fields**
| Name        | Type     | Description                             |
| ----------- | -------- | --------------------------------------- |
| account     | Integer  | User account ID                         |
| asset       | String   | Asset symbol                            |
| balance     | String   | Balance after change                    |
| business    | String   | Business description                    |
| business_id | String   | Business identifier                     |
| change      | String   | Amount changed                          |
| global_id   | Integer  | Global identifier                       |
| time        | Float    | Timestamp (s)                           |
| user        | Integer  | User ID                                 |
| detail      | Object[] | Additional order details (see below)    |

**Detail Object**
| Field | Type    | Description         |
| ----- | ------- | ------------------- |
| a     | String  | Trade quantity      |
| f     | String  | Trading fee rate    |
| i     | Integer | Order ID            |
| m     | String  | Market name         |
| p     | String  | Price               |

**Sample Response**
```json
{
  "error": null,
  "result": [
    {
      "account": 16087,
      "asset": "BTC",
      "balance": "18811.9744000",
      "business": "trade",
      "business_id": "6996187031",
      "change": "0.32000",
      "global_id": 123456789,
      "time": 1724999537.100992,
      "user": 16087,
      "detail": {
        "a": "0.32000",
        "f": "0.0030",
        "i": 6996187031,
        "m": "BTCUSDT",
        "p": "79368.11"
      }
    }
  ],
  "id": 0
}
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

### 4. Get Individual Pair Details
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/market/detail`

**Query Parameters**
| Name   | Type   | Required | Description   |
| ------ | ------ | -------- | ------------- |
| market | String | Yes      | Trading pair  |

**Response Fields**
Same as *Query Information for All Trading Pairs* for a single market entry.

**Sample Response**
```json
{
  "error": {
    "code": 0,
    "message": ""
  },
  "result": {
    "money": "USDT",
    "stock": "BTC",
    "name": "BTCUSDT",
    "fee_prec": 4,
    "money_prec": 2,
    "stock_prec": 5,
    "min_amount": "0.0003",
    "switch": true
  },
  "id": 0
}
```

### 5. Query Candlestick (Kline) Data
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/market/kline`

**Query Parameters**
| Name       | Type    | Required | Description           |
| ---------- | ------- | -------- | --------------------- |
| market     | String  | Yes      | Trading pair          |
| start_time | Integer | Yes      | Start time (s)        |
| end_time   | Integer | Yes      | End time (s)          |
| interval   | Integer | Yes      | Interval in seconds   |

**Response Structure**
`data.result` is an array where each entry contains:
`[timestamp, open, close, high, low, volume, amount, market]`

**Sample Response**
```json
{
  "code": 0,
  "data": {
    "result": [
      [
        1724119200,
        "60452.83",
        "60504.14",
        "60505.20",
        "60446.02",
        "4.34123",
        "262560.8557142",
        "BTCUSDT"
      ],
      [
        1724119260,
        "60503.99",
        "60475.95",
        "60503.99",
        "60452.83",
        "1.19605",
        "72325.7947210",
        "BTCUSDT"
      ]
    ]
  },
  "message": "OK"
}
```

### 6. Query Market Depth
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/market/depth`

**Query Parameters**
| Name  | Type   | Required | Description                                                                              |
| ----- | ------ | -------- | ---------------------------------------------------------------------------------------- |
| market| String | Yes      | Trading pair                                                                             |
| limit | Integer| Yes      | Depth size; one of `5`, `10`, `20`, `50`                                                 |
| merge | String | Yes      | Depth precision; one of `0`, `0.0000000001`, `0.000000001`, `0.00000001`, ..., `100`     |

**Response Fields**
| Field | Type     | Description                   |
| ----- | -------- | ----------------------------- |
| asks  | String[][] | Ask side price/quantity pairs |
| bids  | String[][] | Bid side price/quantity pairs |
| last  | String   | Last traded price             |
| time  | Integer  | Snapshot timestamp (ms)       |
| ttl   | Integer  | Cache TTL (if provided)       |

### 7. Query Latest Price
- Method: `GET`
- Signature: Not required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/market/ticker`

**Query Parameters**
| Name   | Type   | Required | Description  |
| ------ | ------ | -------- | ------------ |
| market | String | Yes      | Trading pair |

**Sample Response**
```json
{
  "error": null,
  "result": "58883.12",
  "id": 0,
  "ttl": 400
}
```

## Trading API

### Shared Order Response Fields
The following fields appear in responses for order placement, cancellation, and queries:

| Field       | Type    | Description                                  |
| ----------- | ------- | -------------------------------------------- |
| id          | Integer | Order ID                                     |
| type        | Integer | Order type (1 = limit, 2 = market, etc.)     |
| side        | Integer | 1 = buy, 2 = sell                            |
| user        | Integer | User ID                                      |
| account     | Integer | User account ID                              |
| option      | Integer | Time-in-force option                         |
| ctime       | Float   | Creation timestamp (s)                       |
| mtime       | Float   | Last update timestamp (s)                    |
| market      | String  | Trading pair                                 |
| source      | String  | Order source                                 |
| client_id   | String  | Client-provided identifier                   |
| price       | String  | Order price                                  |
| amount      | String  | Order quantity                               |
| taker_fee   | String  | Taker fee rate                               |
| maker_fee   | String  | Maker fee rate                               |
| left        | String  | Remaining quantity                           |
| deal_stock  | String  | Executed base currency quantity              |
| deal_money  | String  | Executed quote currency amount               |
| deal_fee    | String  | Fee charged                                   |
| asset_fee   | String  | Fee amount in fee currency                   |
| fee_discount| String  | Applied fee discount                         |
| fee_asset   | String  | Fee currency                                 |

### 1. Place Limit Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/limit`

**Body Parameters**
| Name      | Type    | Required | Description                                                           |
| --------- | ------- | -------- | --------------------------------------------------------------------- |
| access_id | String  | Yes      | Access identifier                                                     |
| tm        | Integer | Yes      | Unix timestamp (s)                                                    |
| market    | String  | Yes      | Trading pair                                                          |
| side      | Integer | Yes      | `1` for buy, `2` for sell                                             |
| amount    | String  | Yes      | Order quantity                                                        |
| price     | String  | Yes      | Limit price                                                           |
| source    | String  | Yes      | Order source (max 30 characters)                                      |
| option    | Integer | No       | Time-in-force (`0`=GTC, `8`=IOC, `16`=FOK); default `0`               |

**Sample Request Body**
```json
{
  "access_id": "a31b51a3-92ec-4ebe-935e-2a9aeadfc268",
  "tm": 1724999193,
  "market": "BTCUSDT",
  "side": 1,
  "amount": "0.02",
  "price": "59369.11",
  "source": "android"
}
```

### 2. Place Market Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/market`

**Body Parameters**
Same as *Place Limit Orders* except that `option` is optional and `price` is not sent.

### 3. Cancel Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/cancel`

**Body Parameters**
| Name      | Type    | Required | Description      |
| --------- | ------- | -------- | ---------------- |
| access_id | String  | Yes      | Access identifier|
| tm        | Integer | Yes      | Unix timestamp (s)|
| market    | String  | Yes      | Trading pair     |
| id        | Integer | Yes      | Order ID         |

### 4. Cancel Orders in Batch
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/cancel_multi`

**Body Parameters**
| Name      | Type    | Required | Description                                         |
| --------- | ------- | -------- | --------------------------------------------------- |
| access_id | String  | Yes      | Access identifier                                   |
| tm        | Integer | Yes      | Unix timestamp (s)                                  |
| market    | String  | Yes      | Trading pair                                        |
| order_ids | String  | Yes      | Order IDs to cancel separated by `"|"`; max 10 IDs |

**Response Fields**
| Field      | Type    | Description                           |
| ---------- | ------- | ------------------------------------- |
| cancel_cnt | Integer | Number of successfully canceled orders|
| ids        | Integer[]| List of canceled order IDs            |
| noids      | Integer[]| Order IDs that failed to cancel       |

### 5. Cancel All Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/cancel_all`

**Body Parameters**
| Name      | Type    | Required | Description                          |
| --------- | ------- | -------- | ------------------------------------ |
| access_id | String  | Yes      | Access identifier                    |
| tm        | Integer | Yes      | Unix timestamp (s)                   |
| market    | String  | Yes      | Trading pair                         |
| side      | Integer | No       | `0`=all, `1`=buy, `2`=sell (default 0)|

### 6. Query Users' Pending Orders
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/pending`

**Query Parameters**
| Name      | Type    | Required | Description                                   |
| --------- | ------- | -------- | --------------------------------------------- |
| access_id | String  | Yes      | Access identifier                             |
| tm        | Integer | Yes      | Unix timestamp (s)                            |
| market    | String  | Yes      | Trading pair                                  |
| side      | Integer | No       | `0`=all, `1`=buy, `2`=sell                    |
| offset    | Integer | Yes      | Pagination offset (start from 0)              |
| limit     | Integer | Yes      | Page size (max 100)                           |

**Response Fields**
| Field   | Type     | Description           |
| ------- | -------- | --------------------- |
| total   | Integer  | Total pending orders  |
| records | Object[] | List of order objects |

Each record uses the *Shared Order Response Fields* table.

### 7. Query Pending Order Details
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/pending_detail`

**Query Parameters**
| Name      | Type    | Required | Description      |
| --------- | ------- | -------- | ---------------- |
| access_id | String  | Yes      | Access identifier|
| tm        | Integer | Yes      | Unix timestamp (s)|
| market    | String  | Yes      | Trading pair     |
| order_id  | Integer | Yes      | Order ID         |

**Response**
Returns a single order object (see *Shared Order Response Fields*).

### 8. Query Order Transaction Details
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/deal`

**Query Parameters**
| Name      | Type    | Required | Description                             |
| --------- | ------- | -------- | --------------------------------------- |
| access_id | String  | Yes      | Access identifier                       |
| tm        | Integer | Yes      | Unix timestamp (s)                      |
| market    | String  | Yes      | Trading pair                            |
| order_id  | Integer | Yes      | Order ID                                |
| offset    | Integer | Yes      | Pagination offset                       |
| limit     | Integer | Yes      | Page size (max 100)                     |

**Deal Record Fields**
| Field         | Type    | Description                           |
| ------------- | ------- | ------------------------------------- |
| time          | Float   | Deal timestamp (s)                    |
| user          | Integer | User ID                               |
| account       | Integer | User account ID                       |
| deal_user     | Integer | Counterparty user ID                  |
| id            | Integer | Deal ID                               |
| role          | Integer | `1`=maker, `2`=taker                  |
| price         | String  | Execution price                       |
| amount        | String  | Executed quantity                     |
| deal          | String  | Executed amount (quote currency)      |
| fee           | String  | Fee charged                           |
| fee_asset     | String  | Fee currency                          |
| deal_order_id | Integer | Counterparty order ID                 |

### 9. Query User Deal History
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/deal_history`

**Query Parameters**
| Name       | Type    | Required | Description                                  |
| ---------- | ------- | -------- | -------------------------------------------- |
| access_id  | String  | Yes      | Access identifier                            |
| tm         | Integer | Yes      | Unix timestamp (s)                           |
| market     | String  | Yes      | Trading pair                                 |
| side       | Integer | No       | 0=all, 1=buy, 2=sell                         |
| start_time | Integer | No       | Start time (s)                               |
| end_time   | Integer | No       | End time (s)                                 |
| offset     | Integer | Yes      | Pagination offset                            |
| limit      | Integer | Yes      | Page size (max 100)                          |

**Response**
Returns an object with `records` array of deal history entries (same fields as *Deal Record Fields* plus `order_id`).

### 10. Query Users' Order History
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/finished`

**Query Parameters**
Same as *Query User Deal History* (with optional `side`, `start_time`, `end_time`, plus required `offset` and `limit`).

**Response Fields**
Each record includes the *Shared Order Response Fields* along with final execution values.

### 11. Query Order History Detail
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/finish_detail`

**Query Parameters**
| Name      | Type    | Required | Description      |
| --------- | ------- | -------- | ---------------- |
| access_id | String  | Yes      | Access identifier|
| tm        | Integer | Yes      | Unix timestamp (s)|
| order_id  | Integer | Yes      | Order ID         |

**Response**
Returns a single historical order record.

## Conditional (Plan) Orders API

### 12. Plan Limit Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/stop_limit`

**Body Parameters**
| Name       | Type    | Required | Description                         |
| ---------- | ------- | -------- | ----------------------------------- |
| access_id  | String  | Yes      | Access identifier                   |
| tm         | Integer | Yes      | Unix timestamp (s)                  |
| market     | String  | Yes      | Trading pair                        |
| side       | Integer | Yes      | `1`=buy, `2`=sell                   |
| amount     | String  | Yes      | Order quantity                      |
| price      | String  | Yes      | Limit price                         |
| stop_price | String  | Yes      | Trigger price                       |
| source     | String  | Yes      | Order source (max 30 characters)    |

### 13. Plan Market Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/stop_market`

**Body Parameters**
Same as *Plan Limit Orders* except `price` is omitted.

### 14. Cancel Plan Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/cancle_stop`

**Body Parameters**
| Name      | Type    | Required | Description      |
| --------- | ------- | -------- | ---------------- |
| access_id | String  | Yes      | Access identifier|
| tm        | Integer | Yes      | Unix timestamp (s)|
| order_id  | Integer | Yes      | Plan order ID    |

### 15. Cancel All Plan Orders
- Method: `POST`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/cancle_stop_all`

**Body Parameters**
| Name      | Type    | Required | Description                                |
| --------- | ------- | -------- | ------------------------------------------ |
| access_id | String  | Yes      | Access identifier                          |
| tm        | Integer | Yes      | Unix timestamp (s)                         |
| market    | String  | Yes      | Trading pair                               |
| side      | Integer | Yes      | `0`=all, `1`=buy, `2`=sell                 |

### 16. Query Plan Pending Orders
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/pending_stop`

**Query Parameters**
| Name      | Type    | Required | Description                                      |
| --------- | ------- | -------- | ------------------------------------------------ |
| access_id | String  | Yes      | Access identifier                                |
| tm        | Integer | Yes      | Unix timestamp (s)                               |
| market    | String  | Yes      | Trading pair                                     |
| side      | Integer | No       | `0`=all, `1`=buy, `2`=sell                       |
| offset    | Integer | Yes      | Pagination offset                                |
| limit     | Integer | Yes      | Page size (max 100)                              |

**Record Fields**
| Field      | Type    | Description                            |
| ---------- | ------- | -------------------------------------- |
| id         | Integer | Plan order ID                          |
| type       | Integer | Order type                             |
| side       | Integer | 1=buy, 2=sell                         |
| option     | Integer | Time-in-force flag                    |
| state      | Integer | Trigger status                        |
| stop_price | String  | Trigger price                         |
| price      | String  | Order price (limit orders)            |
| amount     | String  | Order quantity                        |
| taker_fee  | String  | Taker fee rate                        |
| maker_fee  | String  | Maker fee rate                        |
| fee_asset  | String  | Fee currency                          |
| fee_discount | String| Fee discount                          |
| ctime      | Float   | Creation timestamp (s)                |
| mtime      | Float   | Update timestamp (s)                  |
| market     | String  | Trading pair                          |
| source     | String  | Order source                          |
| client_id  | String  | Client identifier                     |

### 17. Query Plan Order History
- Method: `GET`
- Signature: Required
- URL: `https://spotapi2.btcccdn.com/btcc_api_trade/order/finished_stop`

**Query Parameters**
Same as *Query Plan Pending Orders* with additional optional time range (`start_time`, `end_time`).

**Record Fields**
Same as *Query Plan Pending Orders* plus:
| Field  | Type  | Description                    |
| ------ | ----- | ------------------------------ |
| status | Integer | Execution status indicator   |
| ftime  | Float | Completion timestamp (s)       |

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

### 11. Subscribe to Depth
- Method: `depth.subscribe`
- Parameters: same as `depth.query`

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
```
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
