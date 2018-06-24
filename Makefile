#!/usr/bin/make

.PHONY: dep test build docker-image

dep:
	go get -v -d ./...

test:
	go test ./...
	go vet ./...

build:
	CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .

docker-image:
	docker build -t yannh/redis-dump-go .
