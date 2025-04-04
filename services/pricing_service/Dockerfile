# Multi-Stage Dockerfile for Pricing Service

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

# Stage 1: Build Stage
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev linux-headers
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o pricing_service ./src/main.go
RUN make run-vulnerability-scan

# Stage 2: Production Image
FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/pricing_service .
COPY --from=builder /app/config.yaml /etc/iaros/pricing_service/config.yaml
COPY --from=builder /app/config/fallback_rules.yaml /etc/iaros/pricing_service/fallback_rules.yaml
EXPOSE 8080
USER 65534:65534
HEALTHCHECK --interval=30s --timeout=5s \
  CMD ["/app/pricing_service", "healthcheck", "--config", "/etc/iaros/pricing_service/config.yaml"]
CMD ["/app/pricing_service", "start", "--config", "/etc/iaros/pricing_service/config.yaml"]
