#!/usr/bin/make

.PHONY: test build build-static docker-image docker-test docker-build-static

RELEASE_VERSION ?= latest

export GOFLAGS=-mod=vendor

all: test build

test:
	go test ./...
	go vet ./...

integration-test:
	docker-compose run

build:
	go build -o bin/redis-dump-go

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/redis-dump-go

build-generator-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/generator ./utils/generator/main.go

docker-image:
	docker build -t redis-dump-go:${RELEASE_VERSION} .

save-image:
	docker save --output redis-dump-go-image.tar redis-dump-go:${RELEASE_VERSION}

push-image:
	docker tag redis-dump-go:latest ghcr.io/yannh/redis-dump-go:${RELEASE_VERSION}
	docker push ghcr.io/yannh/redis-dump-go:${RELEASE_VERSION}

docker-test:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.16 make test

docker-build-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.16 make build-static

docker-build-generator-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.16 make build-generator-static

release:
	docker run -e GITHUB_TOKEN -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go goreleaser/goreleaser:v0.164.0-amd64 --rm-dist

integration-tests:
	docker-compose run tests