#!/usr/bin/make

docker-image:
	docker run -e CGO_ENABLED=0 -e GOOS=linux -v ${PWD}:/go/src/github.com/yannh/redis-dump-go -w /go/src/github.com/yannh/redis-dump-go --entrypoint go golang:1.10-alpine build -a -tags netgo -ldflags '-w' .
	docker build -t yannh/redis-dump-go .
