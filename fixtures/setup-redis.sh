#!/bin/sh

echo "-> Installing Redis-cli and Bats"
apk add redis bats go

echo "-> Waiting for Redis to start..."
timeout 30 sh -c 'until redis-cli -h redis -p 6379 PING >/dev/null; do sleep 1; done'

echo "-> Filling Redis..."
go run generator.go