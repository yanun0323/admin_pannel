# Control Page

A simple control panel system with user authentication (RBAC) and real-time Binance K-line charts.

## Tech Stack

- **Frontend**: SolidJS + TypeScript + Vite
- **Backend**: Go + Chi Router + SQLite
- **Real-time**: WebSocket (Binance Kline streams)

## Features

### Authentication System (RBAC)
- User registration and login
- Role-based access control
- JWT authentication
- Default roles: admin, user
- Permissions: view:dashboard, view:kline, manage:users, manage:roles

### Detection System
- Real-time Binance K-line charts
- Multiple trading pairs support (BTC, ETH, BNB, SOL, etc.)
- Multiple time intervals (1m, 5m, 15m, 1h, 4h, 1d, etc.)
- Interactive candlestick charts with lightweight-charts

## Project Structure

```
control_page/
├── backend/                    # Go backend server
│   ├── cmd/server/             # Server entry point
│   ├── config/                 # Configuration
│   ├── database/               # Database migrations
│   ├── internal/
│   │   ├── adaptor/            # Interface definitions
│   │   ├── delivery/http/      # HTTP handlers & middleware
│   │   ├── model/              # Domain models
│   │   ├── repository/         # Data access layer
│   │   └── usecase/            # Business logic
│   ├── infrastructure/         # Docker configuration
│   └── pkg/connection/         # Database connections
├── frontend/                   # SolidJS frontend
│   ├── src/
│   │   ├── components/         # Reusable components
│   │   ├── lib/                # API client & WebSocket
│   │   ├── pages/              # Page components
│   │   └── stores/             # State management
│   └── Dockerfile
├── docker-compose.yml
└── README.md
```

## Getting Started

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker & Docker Compose (optional)

### Development Setup

#### Backend

```bash
cd backend
go mod tidy
go run .
```

The backend server will start at `http://localhost:8887`

#### Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend development server will start at `http://localhost:5173`

### Docker Setup

```bash
# Build and run all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

- Frontend: http://localhost:3000
- Backend API: http://localhost:8887

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login and get JWT token
- `GET /api/auth/me` - Get current user info (protected)

### K-line Data
- `GET /api/kline/symbols` - Get available trading symbols (protected)
- `GET /api/kline/intervals` - Get available time intervals (protected)
- `WS /ws/kline` - WebSocket endpoint for real-time kline data

## Default Users

After starting the server, you can register a new user. The first user will need to manually be assigned the admin role via database if you want admin privileges.

## Configuration

Backend configuration is in `backend/config/config.yaml`:

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

## License

MIT
