# Multi-Stage Dockerfile for Promotion Service

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19
ARG DISTROLESS_IMAGE=gcr.io/distroless/static-debian12

# Builder Stage
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev linux-headers
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o promotion_service ./src/PromotionController.go
RUN make run-vulnerability-scan

# Production Stage
FROM ${DISTROLESS_IMAGE}
WORKDIR /app
COPY --from=builder /app/promotion_service /app/
COPY --from=builder /app/config.yaml /etc/iaros/promotion_service/config.yaml
EXPOSE 8083
USER 65534:65534
HEALTHCHECK --interval=30s --timeout=5s \
  CMD ["/app/promotion_service", "healthcheck", "--config", "/etc/iaros/promotion_service/config.yaml"]
ENTRYPOINT ["/app/promotion_service"]
