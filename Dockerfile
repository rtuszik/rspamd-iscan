# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown

WORKDIR /build

COPY . .

ENV CGO_ENABLED=0
RUN go build -mod=vendor -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o rspamd-iscan .

FROM alpine:3.24

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates

COPY --from=builder /build/rspamd-iscan /usr/local/bin/rspamd-iscan

RUN mkdir -p /etc/rspamd-iscan

USER nobody

ENTRYPOINT ["/usr/local/bin/rspamd-iscan"]
CMD ["--cfg-file", "/etc/rspamd-iscan/config.toml"]
