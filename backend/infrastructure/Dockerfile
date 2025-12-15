# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN GOOS=linux go build -o server .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/config/config.yaml ./config/

# Create data directory for SQLite
RUN mkdir -p data

EXPOSE 8887

CMD ["./server"]
