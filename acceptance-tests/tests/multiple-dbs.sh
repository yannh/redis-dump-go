#!/bin/sh -e

NDBS=3

echo "-> Filling Redis with Mock Data..."
redis-cli -h redis FLUSHALL
for DB in `seq 1 $NDBS`; do
  redis-cli -h redis -n $DB SET thisdb $DB
done

echo "-> Dumping DB..."
time /redis-dump-go -host redis -n 250 -output commands >backup

echo "-> Flushing DB and restoring dump..."
redis-cli -h redis FLUSHALL
redis-cli -h redis -n $DB --pipe <backup

NEWNDBS=`redis-cli -h redis info keyspace | grep keys | wc -l`

echo "-> Expecting right amount of DBS..."
if [ $NDBS -ne $NEWNDBS ]; then
  echo "ERROR - only $NEWNDBS found, expected $NDBS"
  exit 1
else
  echo "OK - $NEWNDBS dbs"
  exit 0
fi
