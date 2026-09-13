# Multi-stage Dockerfile for Open-Relay Control Plane
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Pre-cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /open-relay-control .

# Minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /open-relay-control /app/open-relay-control

EXPOSE 8090

VOLUME ["/app/pb_data"]

ENTRYPOINT ["/app/open-relay-control", "serve", "--http=0.0.0.0:8090", "--dir=/app/pb_data"]
