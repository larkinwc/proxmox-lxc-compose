FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o lxc-compose ./cmd/lxc-compose

FROM alpine:latest

RUN apk add --no-cache lxc docker-cli

COPY --from=builder /app/lxc-compose /usr/local/bin/

ENTRYPOINT ["lxc-compose"] 