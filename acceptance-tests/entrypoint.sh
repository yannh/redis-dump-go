#!/bin/sh -e

echo "-> Installing Redis-cli and Bats"
apk add redis bats ncurses

echo "-> Waiting for Redis to start..."
timeout 30 sh -c 'until redis-cli -h redis -p 6379 PING >/dev/null; do sleep 1; done'

echo "-> Running acceptance tests..."
bats --tap acceptance.bats --verbose-run