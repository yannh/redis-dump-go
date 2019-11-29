#!/usr/bin/make

.PHONY: test build docker-image

export GOFLAGS=-mod=vendor

test:
	go test ./...
	go vet ./...

build:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .

docker-image:
	docker build -t yannh/redis-dump-go .
