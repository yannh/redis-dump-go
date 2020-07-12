[![Build status](https://github.com/yannh/redis-dump-go/workflows/build/badge.svg?branch=master)](https://github.com/yannh/redis-dump-go/actions?query=branch%3Amaster)  [![go report card](https://goreportcard.com/badge/github.com/yannh/redis-dump-go)](https://goreportcard.com/report/github.com/yannh/redis-dump-go)

# Redis-dump-go

Dump Redis keys to a file. Similar in spirit to https://www.npmjs.com/package/redis-dump and https://github.com/delano/redis-dump but:

* Will dump keys across **several processes & connections**
* Uses SCAN rather than KEYS * for much **reduced memory footprint** with large databases
* Easy to deploy & containerize - **single binary**.
* Generates a [RESP](https://redis.io/topics/protocol) file rather than a JSON or a list of commands. This is **faster to ingest**, and [recommended by Redis](https://redis.io/topics/mass-insert) for mass-inserts.

Warning: like similar tools, Redis-dump-go does NOT provide Point-in-Time backups. Please use [Redis backups methods](https://redis.io/topics/persistence) when possible.

## Features

* Dumps all databases present on the Redis server
* Keys TTL are preserved by default
* Configurable Output (Redis commands, RESP)
* Redis password-authentication

## Run

```
$ ./bin/redis-dump-go -h
Usage of ./bin/redis-dump-go:
  -db uint
        only dump this database (default: all databases)
  -filter string
        key filter to use (default "*")
  -host string
        Server host (default "127.0.0.1")
  -n int
        Parallel workers (default 10)
  -output string
        Output type - can be resp or commands (default "resp")
  -port int
        Server port (default 6379)
  -s    Silent mode (disable progress bar)
  -ttl
        Preserve Keys TTL (default true)
$ ./bin/redis-dump-go > dump.resp
Database 0: 9 element dumped
Database 1: 1 element dumped
```

For password-protected Redis servers, set the shell variable REDISDUMPGO_AUTH:

```
$ export REDISDUMPGO_AUTH=myRedisPassword
$ redis-dump-go
```

## Build

Given a correctly configured Go environment:

```
$ go get github.com/yannh/redis-dump-go
$ cd ${GOPATH}/src/github.com/yannh/redis-dump-go
$ go test ./...
$ go install
```

## Importing the data

```
redis-cli --pipe < redis-backup.txt
```

## Release Notes & Gotchas

 * By default, no cleanup is performed before inserting data. When importing the resulting file, hashes, sets and queues will be merged with data already present in the Redis.
