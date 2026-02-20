# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO enabled (required for go-sqlite3)
RUN CGO_ENABLED=1 go build -o /app/focusd ./cmd/main.go

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/focusd .

# Create an empty .env so godotenv.Load() doesn't fail
# All config is passed via environment variables instead
RUN touch .env

ENV PORT=8089

EXPOSE 8089

# TURSO_CONNECTION_PATH and TURSO_CONNECTION_TOKEN must be provided at runtime
# e.g. docker run -e TURSO_CONNECTION_PATH=... -e TURSO_CONNECTION_TOKEN=... focusd
CMD ["./focusd", "serve"]
