# Multi-Stage Dockerfile for Network Planning Service

ARG GO_VERSION=1.22
ARG ALPINE_VERSION=3.19

# Builder Stage
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git make gcc musl-dev linux-headers
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o network_planning_service ./src/ScheduleImporter.go
RUN make run-vulnerability-scan

# Production Stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/network_planning_service /app/
COPY --from=builder /app/config.yaml /etc/iaros/network_planning_service/config.yaml
EXPOSE 8090
USER 65534:65534
HEALTHCHECK --interval=30s --timeout=5s \
  CMD ["/app/network_planning_service", "healthcheck", "--config", "/etc/iaros/network_planning_service/config.yaml"]
CMD ["/app/network_planning_service", "start", "--config", "/etc/iaros/network_planning_service/config.yaml"]
