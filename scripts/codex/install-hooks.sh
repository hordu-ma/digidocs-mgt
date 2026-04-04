#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOOKS_DIR="$ROOT_DIR/.githooks"

if [[ ! -d "$HOOKS_DIR" ]]; then
  echo "missing hooks directory: $HOOKS_DIR" >&2
  exit 1
fi

chmod +x "$HOOKS_DIR"/pre-commit "$HOOKS_DIR"/pre-push
git -C "$ROOT_DIR" config core.hooksPath .githooks

echo "configured git hooks path: .githooks"
echo "active hooks:"
ls -1 "$HOOKS_DIR"
