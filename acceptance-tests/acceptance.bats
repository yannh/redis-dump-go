#!/usr/bin/env bats

@test "Pass when importing a ZSET with 1M entries" {
  run tests/large-zset.sh
  [ "$status" -eq 0 ]
}

@test "Pass when importing 1M key/values" {
  run tests/1mkeys.sh
  [ "$status" -eq 0 ]
}
