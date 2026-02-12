# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o travelmate-api ./cmd/api

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies (e.g. ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/travelmate-api .

# Copy necessary assets (migrations, seeds, html if any)
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/seeds ./seeds

# Expose port
EXPOSE 8080

# Command to run
CMD ["./travelmate-api"]
