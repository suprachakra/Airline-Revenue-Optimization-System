# Multi-stage build for Customer Intelligence Platform
FROM golang:1.19-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (for downloading dependencies)
RUN apk update && apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o customer-intelligence-platform ./main.go

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S iaros && \
    adduser -u 1001 -S iaros -G iaros

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/customer-intelligence-platform .

# Copy configuration files
COPY --from=builder /app/config.yaml .

# Change ownership to non-root user
RUN chown -R iaros:iaros /root/

# Switch to non-root user
USER iaros

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./customer-intelligence-platform"] 