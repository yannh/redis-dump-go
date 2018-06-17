# Redis-dump-go

Dump Redis keys to a file. Similar in spirit to https://www.npmjs.com/package/redis-dump and https://github.com/delano/redis-dump but:

 * Will dump keys across several processes & connections
 * Easy to deploy & containerize - single binary.
 * Generates a [RESP](https://redis.io/topics/protocol) file rather than a JSON or a list of commands. This is faster to ingest, and [recommended by Redis](https://redis.io/topics/mass-insert) for mass-inserts.

Warning: like similar tools, Redis-dump-go does NOT provide Point-in-Time backups. Please use [Redis backups methods](https://redis.io/topics/persistence) when possible.

## Build

Given a correctly configured Go environment:

```
$ go get github.com/yannh/redis-dump-go
$ cd ${GOPATH}/src/github.com/yannh/redis-dump-go
$ go test ./...
$ go install
```

## Usage

```
$ redis-dump-go -h
Usage of redis-dump-go:
  -host string
    Server host (default "127.0.0.1")
  -port int
    Server port (default 6379)
  -s  Silent mode (disable progress bar)
$ redis-dump-go > redis-backup.txt
[==================================================] 100% [5/5]
```

## Importing the data

```
redis-cli --pipe < redis-backup.txt
```

## Release Notes & Gotchas

 * By default, no cleanup is performed before inserting data. When importing the resulting file, hashes, sets and queues will be merged with data already present in the Redis.
 * Key expiration is currently not supported, and ignored.
