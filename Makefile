#!/usr/bin/make

.PHONY: test build build-static build-generator-static docker-image save-image push-image docker-test docker-build-static docker-build-generator-static release acceptance-tests

RELEASE_VERSION ?= latest

export GOFLAGS=-mod=vendor

all: test build

test:
	go test ./...
	go vet ./...

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
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.18 make test

docker-build-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.18 make build-static

docker-build-generator-static:
	docker run -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go golang:1.18 make build-generator-static

goreleaser-build-static:
	docker run -e GOCACHE=/tmp -v $$PWD/.gitconfig:/root/.gitconfig -t -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go goreleaser/goreleaser:v1.8.3 build --single-target --skip-post-hooks --rm-dist --snapshot
	cp dist/redis-dump-go_linux_amd64_v1/redis-dump-go bin/

release:
	docker run -e GITHUB_TOKEN -t -v $$PWD/.gitconfig:/root/.gitconfig -v /var/run/docker.sock:/var/run/docker.sock -v $$PWD:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go goreleaser/goreleaser:v1.8.3 release --rm-dist

acceptance-tests: docker-build-static docker-build-generator-static
	docker-compose run tests
