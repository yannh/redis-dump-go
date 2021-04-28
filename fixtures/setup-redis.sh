#!/bin/sh -e

echo "-> Installing Redis-cli and Bats"
apk add redis bats

echo "-> Waiting for Redis to start..."
timeout 30 sh -c 'until redis-cli -h redis -p 6379 PING >/dev/null; do sleep 1; done'

echo "-> Run Bats tests"

echo "-> Filling Redis with Mock Data..."
chmod a+x /generator

redis-cli -h redis FLUSHDB
/generator -n 1000 | redis-cli -h redis --pipe
DBSIZE=`redis-cli -h redis dbsize`
echo "Redis has $DBSIZE entries"


echo "-> Dumping DB..."
/redis-dump-go -host redis -output resp >backup

echo "-> Flushing DB and restoring dump..."
redis-cli -h redis FLUSHDB
redis-cli -h redis --pipe <backup
NEWDBSIZE=`redis-cli -h redis dbsize`
echo "Redis has $DBSIZE entries"

if [ $DBSIZE -eq $NEWDBSIZE ]; then
  false
fi