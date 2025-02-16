FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lxc-compose ./cmd/lxc-compose

FROM alpine:latest

RUN apk add --no-cache lxc docker-cli

COPY --from=builder /app/lxc-compose /usr/local/bin/

ENTRYPOINT ["lxc-compose"] 