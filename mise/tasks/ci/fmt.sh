#!/usr/bin/env bash
#MISE description="Format all go code using gofumpt, and fail if any files are not formatted correctly."
#MISE depends=["fmt"]
set -euo pipefail

if ! git diff --exit-code --name-only; then
  echo "The file(s) listed above are not formatted correctly. Please run 'mise run fmt' and commit the changes."
  exit 1
fi
