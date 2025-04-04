# Multi-Stage Build for API Gateway

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

# Stage 1: Build Stage
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev linux-headers
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o api_gateway ./src/main.go
RUN make run-vulnerability-scan

# Stage 2: Production Image
FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata security-ca
WORKDIR /app
COPY --from=builder /app/api_gateway .
COPY --from=builder /app/config.yaml /etc/iaros/gateway/config.yaml
EXPOSE 8443

# Security Hardening: Run as non-root user
USER 65534:65534

HEALTHCHECK --interval=30s --timeout=5s \
  CMD ["/app/api_gateway", "healthcheck", "--config", "/etc/iaros/gateway/config.yaml"]

CMD ["/app/api_gateway", "start", "--config", "/etc/iaros/gateway/config.yaml"]
