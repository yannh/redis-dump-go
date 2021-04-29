#!/bin/sh -e

echo "-> Filling Redis with Mock Data..."
redis-cli -h redis FLUSHDB
/generator -output resp -type zset -n 1000000 | redis-cli -h redis --pipe
KEYNAME=`redis-cli -h redis KEYS '*'`
COUNT=`redis-cli -h redis ZCOUNT $KEYNAME -inf +inf`

echo "-> Dumping DB..."
/redis-dump-go -host redis -output resp >backup

echo "-> Flushing DB and restoring dump..."
redis-cli -h redis FLUSHDB
redis-cli -h redis --pipe <backup
NEWCOUNT=`redis-cli -h redis ZCOUNT $KEYNAME -inf +inf`
echo "Redis has $COUNT entries"

echo "-> Comparing ZSET sizes..."
if [ $COUNT -ne $NEWCOUNT ]; then
  echo "ERROR - restored ZSET $KEYNAME has $NEWCOUNT elements, expected $COUNT"
  exit 1
else
  echo "OK - ZSET $KEYNAME has $NEWCOUNT elements"
  exit 0
fi