#!/bin/sh -e

export DB=2
export REDIS_PORT=6380
export REDIS_USER=test
export REDISDUMPGO_AUTH=testpassword
export REDISCMD="redis-cli -h redis_secure -p $REDIS_PORT --user $REDIS_USER --pass $REDISDUMPGO_AUTH -n 2"
echo $REDISCMD
echo "-> Filling Redis with Mock Data..."
$REDISCMD FLUSHDB
/generator -output resp -type strings -n 100 | $REDISCMD --pipe
DBSIZE=`$REDISCMD dbsize`

echo "-> Dumping DB..."
time /redis-dump-go -host redis_secure -n 250 -port $REDIS_PORT -db $DB -user $REDIS_USER -output resp >backup

echo "-> Flushing DB and restoring dump..."
$REDISCMD FLUSHDB
$REDISCMD --pipe <backup
NEWDBSIZE=`$REDISCMD dbsize`
echo "Redis has $DBSIZE entries"
echo "-> Comparing DB sizes..."
if [ $DBSIZE -ne $NEWDBSIZE ]; then
  echo "ERROR - restored DB has $NEWDBSIZE elements, expected $DBSIZE"
  exit 1
else
  echo "OK - $NEWDBSIZE elements"
  exit 0
fi
