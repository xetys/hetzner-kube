#!/usr/bin/env bash

set -ev

go build
VERSION=$(./hetzner-kube version)

mkdir -p dist/${VERSION}

#GOOS=target-OS GOARCH=target-architecture go build -o

# linux
GOOS=linux GOARCH=amd64 go build -o dist/${VERSION}/hetzner-kube-linux-amd64
GOOS=linux GOARCH=386 go build -o dist/${VERSION}/hetzner-kube-linux-386
GOOS=linux GOARCH=arm go build -o dist/${VERSION}/hetzner-kube-linux-arm
GOOS=linux GOARCH=arm64 go build -o dist/${VERSION}/hetzner-kube-linux-arm64

# mac
GOOS=darwin GOARCH=amd64 go build -o dist/${VERSION}/hetzner-kube-darwin-amd64
GOOS=darwin GOARCH=386 go build -o dist/${VERSION}/hetzner-kube-darwin-386

# windows
GOOS=windows GOARCH=amd64 go build -o dist/${VERSION}/hetzner-kube-windows-amd64.exe
GOOS=windows GOARCH=386 go build -o dist/${VERSION}/hetzner-kube-windows-386.exe
