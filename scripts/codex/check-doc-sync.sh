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

require_pattern() {
  local file="$1"
  local pattern="$2"
  local label="$3"

  if rg -q "$pattern" "$file"; then
    printf '[OK] %s\n' "$label"
  else
    printf '[MISS] %s\n' "$label"
    return 1
  fi
}

status=0
readme_phase="$(extract_readme_phase)"
tasks_phase="$(extract_tasks_phase)"

echo '== required sections =='
require_pattern README.md '^## 开发进度$' 'README.md has 开发进度 section' || status=1
require_pattern TASKS.md '^## 当前阶段$' 'TASKS.md has 当前阶段 section' || status=1
require_pattern TASKS.md '^## 已完成$' 'TASKS.md has 已完成 section' || status=1
require_pattern TASKS.md '^## 进行中$' 'TASKS.md has 进行中 section' || status=1
require_pattern TASKS.md '^## 待办$' 'TASKS.md has 待办 section' || status=1

echo '== phase alignment =='
printf 'README phase: %s\n' "${readme_phase:-<missing>}"
printf 'TASKS phase: %s\n' "${tasks_phase:-<missing>}"

if [[ -z "${readme_phase}" || -z "${tasks_phase}" ]]; then
  echo '[MISS] phase text missing'
  status=1
elif [[ "${readme_phase}" == "${tasks_phase}" ]]; then
  echo '[OK] README and TASKS phases match exactly'
elif [[ "${readme_phase}" == "${tasks_phase}"* || "${tasks_phase}" == "${readme_phase}"* ]]; then
  echo '[OK] README and TASKS phases are aligned by prefix'
else
  echo '[MISS] README and TASKS phases are inconsistent'
  status=1
fi

echo '== codex assets mentioned =='
require_pattern README.md 'ops/codex/skills/' 'README mentions project skills path' || status=1
require_pattern README.md './scripts/codex/doctor.sh' 'README mentions doctor entry' || status=1
require_pattern AGENTS.md 'ops/codex/skills/' 'AGENTS mentions project skills path' || status=1
require_pattern AGENTS.md 'TASKS.md.*README.md' 'AGENTS mentions task/readme sync' || status=1

exit "$status"
