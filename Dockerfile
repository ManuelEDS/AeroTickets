# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install ca-certificates and tzdata
RUN apk add --no-cache ca-certificates tzdata

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically compiled binary (CGO_ENABLED=0 since we use pure Go modernc.org/sqlite)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o aerobot ./cmd/bot

# Final Minimal Runtime Stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /app/transcripts /app/data

# Copy binary from builder
COPY --from=builder /app/aerobot /app/aerobot

# Expose volumes for persistence
VOLUME ["/app/data", "/app/transcripts"]

ENTRYPOINT ["/app/aerobot"]
CMD ["-config", "/app/config.json"]
