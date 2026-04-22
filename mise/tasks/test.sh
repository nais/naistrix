#!/usr/bin/env bash
#MISE description="Run tests"
set -euo pipefail

go test -v --race --cover --coverprofile=cover.out ./...

# run tests without race detection, see: https://github.com/atomicgo/keyboard/issues/6
go test ./pkg/input/input_test.go