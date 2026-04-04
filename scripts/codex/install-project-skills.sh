#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PROJECT_SKILLS_DIR="$ROOT_DIR/ops/codex/skills"
TARGET_BASE="${HOME}/.codex/skills"

if [[ ! -d "$PROJECT_SKILLS_DIR" ]]; then
  echo "missing project skills directory: $PROJECT_SKILLS_DIR" >&2
  exit 1
fi

mkdir -p "$TARGET_BASE"

installed=0
for skill_dir in "$PROJECT_SKILLS_DIR"/*; do
  [[ -d "$skill_dir" ]] || continue
  skill_name="$(basename "$skill_dir")"
  target="$TARGET_BASE/$skill_name"

  if [[ -L "$target" || -e "$target" ]]; then
    rm -rf "$target"
  fi

  ln -s "$skill_dir" "$target"
  echo "linked $skill_name -> $target"
  installed=$((installed + 1))
done

echo "installed $installed project skills"
echo "index: $PROJECT_SKILLS_DIR/index.yaml"
