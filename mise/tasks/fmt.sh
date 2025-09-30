#!/usr/bin/env bash
#MISE description="Format all go code using gofumpt"
set -euo pipefail

go tool mvdan.cc/gofumpt -w ./
