#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

status=0

check_cmd() {
  local name="$1"
  if command -v "$name" >/dev/null 2>&1; then
    printf '[OK] command: %s\n' "$name"
  else
    printf '[MISS] command: %s\n' "$name"
    status=1
  fi
}

check_file() {
  local path="$1"
  if [[ -e "$path" ]]; then
    printf '[OK] file: %s\n' "$path"
  else
    printf '[MISS] file: %s\n' "$path"
    status=1
  fi
}

echo '== commands =='
for cmd in git go uv node npm docker; do
  check_cmd "$cmd"
done

echo '== core docs =='
for file in AGENTS.md README.md TASKS.md docs/项目定义与技术架构.md docs/数据库设计.md docs/API设计.md; do
  check_file "$file"
done

echo '== project codex assets =='
while IFS= read -r file; do
  check_file "$file"
done <<'EOF'
ops/codex/skills/index.yaml
Makefile
scripts/codex/check-doc-sync.sh
scripts/codex/install-hooks.sh
scripts/codex/install-project-skills.sh
scripts/codex/doctor.sh
scripts/codex/report.sh
scripts/codex/smoke-local.sh
.github/INDEX.md
.github/workflows/verify.yml
.githooks/pre-commit
.githooks/pre-push
EOF

echo '== progress sync =='
if grep -nE -- '当前开发进度|当前阶段|已完成|进行中|待办' README.md TASKS.md AGENTS.md >/dev/null 2>&1; then
  echo '[OK] progress markers found in README.md / TASKS.md / AGENTS.md'
else
  echo '[MISS] expected progress markers not found'
  status=1
fi

echo '== doc sync check =='
if ./scripts/codex/check-doc-sync.sh; then
  echo '[OK] check-doc-sync.sh'
else
  echo '[MISS] check-doc-sync.sh'
  status=1
fi

echo '== skill count =='
find ops/codex/skills -mindepth 1 -maxdepth 1 -type d | sort

exit "$status"
