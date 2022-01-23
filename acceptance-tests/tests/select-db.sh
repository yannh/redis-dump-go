#!/bin/sh -e

export DB=2

echo "-> Filling Redis with Mock Data..."
redis-cli -h redis -n $DB FLUSHDB
/generator -output resp -type strings -n 100 | redis-cli -h redis -n $DB --pipe
DBSIZE=`redis-cli -h redis -n $DB dbsize`

echo "-> Dumping DB..."
time /redis-dump-go -host redis -n 250 -db $DB -output resp >backup

echo "-> Flushing DB and restoring dump..."
redis-cli -h redis -n $DB FLUSHDB
redis-cli -h redis -n $DB --pipe <backup
NEWDBSIZE=`redis-cli -h redis -n $DB dbsize`
echo "Redis has $DBSIZE entries"

echo "-> Comparing DB sizes..."
if [ $DBSIZE -ne $NEWDBSIZE ]; then
  echo "ERROR - restored DB has $NEWDBSIZE elements, expected $DBSIZE"
  exit 1
else
  echo "OK - $NEWDBSIZE elements"
  exit 0
fi
