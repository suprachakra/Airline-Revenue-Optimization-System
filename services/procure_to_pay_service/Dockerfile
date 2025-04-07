# Multi-Stage Docker Build for Procure-to-Pay Service
ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

# Builder Stage
FROM golang:${GO_VERSION}-alpine AS builder
RUN apk add --no-cache git gcc musl-dev
WORKDIR /app
COPY . .
RUN go mod download && \
    go build -o procure-to-pay-service ./src

# Production Stage
FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/procure-to-pay-service .
COPY --from=builder /app/config /etc/iaros/procure_to_pay
USER 65534:65534
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD ["/app/procure-to-pay-service", "healthcheck"]
CMD ["/app/procure-to-pay-service", "start", "--config", "/etc/iaros/procure_to_pay/config.yaml"]
