#!/usr/bin/env bash
#MISE description="Run tests"
set -euo pipefail

go test --race --cover --coverprofile=cover.out ./...
