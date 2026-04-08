#!/usr/bin/env bash
#MISE description="Fix go files using go fix"
set -euo pipefail

go fix ./...
