#!/usr/bin/env bash
#MISE description="Generate GIFs for usage examples"
set -euo pipefail

go -C ./cmd/generate_examples run .
