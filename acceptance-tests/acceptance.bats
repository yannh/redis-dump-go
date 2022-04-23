#!/usr/bin/env bats

@test "Dumping / restoring all databases" {
  run tests/multiple-dbs.sh
  [ "$status" -eq 0 ]
}
