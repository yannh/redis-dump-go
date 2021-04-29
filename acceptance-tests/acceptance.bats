#!/usr/bin/env bats

@test "Pass when importing a ZSET with 1M entries" {
  run tests/large-zset.sh
  [ "$status" -eq 0 ]
}