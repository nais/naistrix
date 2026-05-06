#!/usr/bin/env bash
#MISE description="Serve godocs locally using pkgsite (same renderer as pkg.go.dev)"
set -euo pipefail

go tool pkgsite -open .
