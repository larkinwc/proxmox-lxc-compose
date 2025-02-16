FROM alpine:latest

RUN apk add --no-cache lxc docker-cli

COPY lxc-compose /usr/local/bin/

ENTRYPOINT ["lxc-compose"] 