#!/bin/sh -e

echo "-> Installing Redis-cli and Bats"
apk add redis bats ncurses

echo "-> Waiting for Redis to start..."
timeout 30 sh -c 'until redis-cli -h redis -p 6379 PING >/dev/null; do sleep 1; done'

chmod a+x /generator

bats -p acceptance.bats