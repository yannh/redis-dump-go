#!/bin/sh

echo "-> Installing Redis-cli"
apk add redis

echo "-> Waiting for Redis to start..."
timeout 30 sh -c 'until redis-cli -h redis -p 6379 PING >/dev/null; do sleep 1; done'

echo "-> Filling Redis..."
echo "
SET foo bar
SET lorem ipsum
" | redis-cli -h redis -p 6379