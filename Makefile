#!/usr/bin/make

.PHONY: test build build-static docker-image docker-test docker-build-static

export GOFLAGS=-mod=vendor

all: test build

test:
	go test ./...
	go vet ./...

build:
	go build -o bin/redis-dump-go

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/redis-dump-go

docker-image:
	docker build -t yannh/redis-dump-go .

docker-test:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.14 make test

docker-build-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.14 make build-static

release:
	docker run -e GITHUB_TOKEN -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go goreleaser/goreleaser:v0.138 goreleaser release --rm-dist