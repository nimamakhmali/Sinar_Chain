# Sinar Chain Dockerfile
# =====================

# Build stage
FROM golang:1.20-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sinar_chain main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S sinar && \
    adduser -u 1001 -S sinar -G sinar

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/sinar_chain .

# Change ownership to non-root user
RUN chown -R sinar:sinar /app

# Switch to non-root user
USER sinar

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/blockchain/info || exit 1

# Run the application
CMD ["./sinar_chain"] 