FROM golang:alpine

ARG GO111MODULE=on
ARG HETZNER_KUBE_VERSION=latest

SHELL [ "/bin/sh", "-xe", "-c" ]
RUN apk add --no-cache --virtual .build-deps git \
    && go get "github.com/xetys/hetzner-kube@${HETZNER_KUBE_VERSION}" \
    && apk del --purge .build-deps \
    && rm -rf /go/pkg \
    && adduser -h /go -D -H -u 1000 go \
    && chown -R go:go /go

USER go

SHELL [ "/bin/sh", "-c" ]

ENTRYPOINT [ "hetzner-kube" ]
