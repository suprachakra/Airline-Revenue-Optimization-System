# Multi-Stage Dockerfile for Ancillary Service

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev linux-headers
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o ancillary_service ./src/AncillaryController.go

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/ancillary_service .
COPY --from=builder /app/config.yaml /etc/iaros/ancillary_service/config.yaml
EXPOSE 8082
USER 65534:65534
HEALTHCHECK --interval=30s --timeout=5s \
  CMD ["/app/ancillary_service", "healthcheck", "--config", "/etc/iaros/ancillary_service/config.yaml"]
CMD ["/app/ancillary_service", "start", "--config", "/etc/iaros/ancillary_service/config.yaml"]
