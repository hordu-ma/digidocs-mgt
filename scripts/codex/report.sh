#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

extract_readme_phase() {
  awk '
    /## 开发进度/ { in_section=1; next }
    in_section && /^当前阶段已进入 / {
      line=$0
      sub(/^当前阶段已进入 /, "", line)
      sub(/：$/, "", line)
      print line
      exit
    }
  ' README.md
}

extract_tasks_phase() {
  awk '
    /## 当前阶段/ { getline; getline; print; exit }
  ' TASKS.md
}

current_branch="$(git branch --show-current 2>/dev/null || true)"
readme_phase="$(extract_readme_phase)"
tasks_phase="$(extract_tasks_phase)"
project_skill_count="$(find ops/codex/skills -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
installed_skill_count="$(find "${HOME}/.codex/skills" -mindepth 1 -maxdepth 1 -type l -lname "${ROOT_DIR}/ops/codex/skills/*" 2>/dev/null | wc -l | tr -d ' ')"
hooks_path="$(git config --get core.hooksPath 2>/dev/null || true)"

printf 'repo: %s\n' "$ROOT_DIR"
printf 'branch: %s\n' "${current_branch:-<detached>}"
printf 'readme_phase: %s\n' "${readme_phase:-<missing>}"
printf 'tasks_phase: %s\n' "${tasks_phase:-<missing>}"
printf 'project_skill_count: %s\n' "$project_skill_count"
printf 'installed_project_skill_count: %s\n' "$installed_skill_count"
printf 'git_hooks_path: %s\n' "${hooks_path:-<default>}"
printf 'verify_workflow: %s\n' ".github/workflows/verify.yml"
