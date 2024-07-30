#!/usr/bin/make

.PHONY: test build build-static build-generator-static docker-image save-image push-image docker-test docker-build-static docker-build-generator-static release acceptance-tests

RELEASE_VERSION ?= latest

export GOFLAGS=-mod=vendor

all: test build

test:
	go test -race ./...
	go vet ./...

build:
	go build -o bin/redis-dump-go

build-static:
	git config --global --add safe.directory $$PWD
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/redis-dump-go

build-generator-static:
	git config --global --add safe.directory $$PWD
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/generator ./utils/generator/main.go

docker-image:
	docker build -t redis-dump-go:${RELEASE_VERSION} .

save-image:
	docker save --output redis-dump-go-image.tar redis-dump-go:${RELEASE_VERSION}

push-image:
	docker tag redis-dump-go:latest ghcr.io/yannh/redis-dump-go:${RELEASE_VERSION}
	docker push ghcr.io/yannh/redis-dump-go:${RELEASE_VERSION}

docker-test:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.22.5 make test

docker-build-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.22.5 make build-static

docker-build-generator-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.22.5 make build-generator-static

goreleaser-build-static:
	docker run -t -e GOOS=linux -e GOARCH=amd64 -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go goreleaser/goreleaser:v2.1.0 build --single-target --clean --snapshot
	cp dist/redis-dump-go_linux_amd64_v1/redis-dump-go bin/

release:
	docker run -e GITHUB_TOKEN -e GIT_OWNER -t -v /var/run/docker.sock:/var/run/docker.sock -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go goreleaser/goreleaser:v1.22.1 release --clean

acceptance-tests: docker-build-static docker-build-generator-static
	docker-compose run tests
