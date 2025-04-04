# Multi-Stage Dockerfile for Forecasting Service

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev linux-headers
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o forecasting_service ./src/ForecastController.go
RUN make run-vulnerability-scan

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/forecasting_service .
COPY --from=builder /app/config.yaml /etc/iaros/forecasting_service/config.yaml
EXPOSE 8081
USER 65534:65534
HEALTHCHECK --interval=30s --timeout=5s \
  CMD ["/app/forecasting_service", "healthcheck", "--config", "/etc/iaros/forecasting_service/config.yaml"]
CMD ["/app/forecasting_service", "start", "--config", "/etc/iaros/forecasting_service/config.yaml"]
