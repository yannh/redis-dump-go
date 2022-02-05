#!/usr/bin/env bats

@test "Prints help on -h" {
  run /redis-dump-go -h
  [ "${lines[0]}" == 'Usage: /redis-dump-go [OPTION]...' ]
  [ "$status" -eq 0 ]
}

@test "fail on incorrect parameters" {
  run /redis-dump-go -fail
  [ "$status" -eq 1 ]
}

@test "Pass when using a non-default db" {
  run tests/select-db.sh
  [ "$status" -eq 0 ]
}

@test "Pass when using a non-default db, and a password" {
  run tests/select-db-with-password.sh
  [ "$status" -eq 0 ]
}

# https://github.com/yannh/redis-dump-go/issues/11
# https://github.com/yannh/redis-dump-go/issues/18
@test "Pass when importing a ZSET with 1M entries" {
  run tests/large-zset.sh
  [ "$status" -eq 0 ]
}

@test "Pass when importing 1M key/values" {
  run tests/1mkeys.sh
  [ "$status" -eq 0 ]
}