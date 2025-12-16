# Control Page Backend

Go backend server for the Control Page system.

## Tech Stack

- Go 1.23
- Chi Router (HTTP routing)
- SQLite (Database)
- Gorilla WebSocket (Real-time communication)
- golang-jwt/jwt (JWT authentication)
- sqlx (SQL extensions)

## Project Structure

```
backend/
├── main.go                     # Application entry point
├── cmd/
│   └── server/                 # Server initialization
├── config/                     # Configuration files
├── database/
│   └── migration/              # SQL migration files
├── internal/
│   ├── adaptor/                # Interface definitions
│   ├── delivery/http/          # HTTP handlers, middleware, router
│   ├── model/
│   │   └── enum/               # Domain enums
│   ├── repository/             # Data access implementations
│   └── usecase/                # Business logic implementations
├── infrastructure/             # Docker configuration
└── pkg/
    └── connection/             # Database connection utilities
```

## Getting Started

### Prerequisites

- Go 1.23+

### Installation

```bash
# Download dependencies
go mod tidy

# Run the server
go run .
```

### Build

```bash
# Build binary
go build -o server .

# Run binary
./server
```

## Configuration

Configuration file: `config/config.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8887

database:
  driver: "sqlite3"
  dsn: "data/control_page.db"

jwt:
  secret: "your-super-secret-key-change-in-production"
  expiration: 24h

binance:
  websocket_url: "wss://stream.binance.com:9443/ws"
```

## API Documentation

### Public Endpoints

#### Register User
```
POST /api/auth/register
Content-Type: application/json

{
  "username": "user1",
  "password": "password123"
}
```

#### Login
```
POST /api/auth/login
Content-Type: application/json

{
  "username": "user1",
  "password": "password123"
}

Response:
{
  "token": "jwt-token-here",
  "user": { ... }
}
```

### Protected Endpoints

All protected endpoints require the `Authorization: Bearer <token>` header.

#### Get Current User
```
GET /api/auth/me
Authorization: Bearer <token>
```

#### Get Trading Symbols
```
GET /api/kline/symbols
Authorization: Bearer <token>
```

#### Get Time Intervals
```
GET /api/kline/intervals
Authorization: Bearer <token>
```

### WebSocket Endpoints

#### K-line Stream
```
WS /ws/kline

// Subscribe to a symbol
{
  "action": "subscribe",
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1m"
  }
}

// Unsubscribe from a symbol
{
  "action": "unsubscribe",
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1m"
  }
}
```

## Database Schema

### Users
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| username | TEXT | Unique username |
| password | TEXT | Hashed password |
| is_active | INTEGER | Account status |
| created_at | DATETIME | Creation timestamp |
| updated_at | DATETIME | Last update timestamp |

### Roles
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| name | TEXT | Unique role name |
| description | TEXT | Role description |
| created_at | DATETIME | Creation timestamp |
| updated_at | DATETIME | Last update timestamp |

### Role Permissions
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| role_id | INTEGER | Foreign key to roles |
| permission | TEXT | Permission string |

### User Roles
| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| user_id | INTEGER | Foreign key to users |
| role_id | INTEGER | Foreign key to roles |

## Default Roles and Permissions

| Role | Permissions |
|------|-------------|
| admin | view:dashboard, view:kline, manage:users, manage:roles |
| user | view:dashboard, view:kline |
