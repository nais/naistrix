#!/usr/bin/env bash
#MISE description="Run golangci-lint"
set -euo pipefail

golangci-lint run
