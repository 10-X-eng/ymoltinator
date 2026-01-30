# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy source code first
COPY . .

# Download dependencies and generate go.sum
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o ainews-api .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates and create non-root user
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -g '' appuser

# Copy binary from builder
COPY --from=builder /app/ainews-api .

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run the application
CMD ["./ainews-api"]
