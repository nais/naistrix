#!/usr/bin/env bash
#MISE description="Run golangci-lint"
set -euo pipefail

go tool github.com/golangci/golangci-lint/cmd/golangci-lint run
