#!/bin/sh -e

echo "-> Filling Redis with Mock Data..."
redis-cli -h redis FLUSHDB
/generator -output resp -type zset -n 1000000 | redis-cli -h redis --pipe
DBSIZE=`redis-cli -h redis dbsize`

echo "-> Dumping DB..."
/redis-dump-go -host redis -output resp >backup

echo "-> Flushing DB and restoring dump..."
redis-cli -h redis FLUSHDB
redis-cli -h redis --pipe <backup
NEWDBSIZE=`redis-cli -h redis dbsize`
echo "Redis has $DBSIZE entries"

echo "-> Comparing DB sizes..."
if [ $DBSIZE -ne $NEWDBSIZE ]; then
  echo "ERROR - restored DB has $NEWDBSIZE elements, expected $DBSIZE"
  exit 1
else
  echo "OK - $NEWDBSIZE elements"
  exit 0
fi