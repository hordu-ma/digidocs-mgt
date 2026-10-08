#!/usr/bin/env bash
# PostgreSQL + code_repos backups, with verified offsite delivery and retry.
# NAS document snapshots and secret configuration are managed separately.
set -euo pipefail
umask 077

CONTAINER="${DIGIDOCS_PG_CONTAINER:-digidocs-postgres}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-digidocs_mgt}"
CODE_VOLUME="${DIGIDOCS_CODE_VOLUME:-digidocs-mgt_code_repos}"
BACKUP_DIR="${DIGIDOCS_BACKUP_DIR:-$HOME/ops/digidocs-backups}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"
BACKEND_CONTAINER="${DIGIDOCS_BACKEND_CONTAINER:-digidocs-backend-go}"
OFFSITE_PREFIX="${DIGIDOCS_OFFSITE_PREFIX:-ops-backups}"
OFFSITE_REQUIRED="${DIGIDOCS_OFFSITE_REQUIRED:-1}"
SYNC_ONLY=0
log() { echo "[backup] $*"; }
fail() { echo "[backup][ERR] $*" >&2; exit 1; }
[[ "$#" -eq 0 ]] || { [[ "$#" -eq 1 && "$1" == --sync-only ]] && SYNC_ONLY=1; } \
  || fail "usage: backup.sh [--sync-only]"
[[ "$KEEP_DAYS" =~ ^[0-9]+$ ]] || fail "BACKUP_KEEP_DAYS must be non-negative"
[[ "$OFFSITE_REQUIRED" == 0 || "$OFFSITE_REQUIRED" == 1 ]] || fail "DIGIDOCS_OFFSITE_REQUIRED must be 0 or 1"

mkdir -p "$BACKUP_DIR/.offsite"
exec 9>"$BACKUP_DIR/.backup.lock"
flock -n 9 || fail "another backup/sync is running"
partial=""
staged=""
cleanup() {
  if [[ -n "$partial" ]]; then rm -f -- "$partial"; fi
  if [[ -n "$staged" ]]; then docker exec "$BACKEND_CONTAINER" rm -f "$staged" >/dev/null 2>&1 || true; fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
running() { docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null | grep -qx true; }

if [[ "$SYNC_ONLY" == 0 ]]; then
  running "$CONTAINER" || fail "database container is not running"
  stamp="$(date +%Y%m%d-%H%M%S)"
  db_file="$BACKUP_DIR/db_${DB_NAME}_${stamp}.sql.gz"
  [[ ! -e "$db_file" ]] || fail "backup timestamp already exists"
  partial="$db_file.partial"
  log "export database -> $(basename "$db_file")"
  docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists | gzip > "$partial"
  [[ $(stat -c %s "$partial") -gt 1024 ]] || fail "database dump is unexpectedly small"
  gzip -t "$partial" || fail "database gzip validation failed"
  table_count=$(zcat "$partial" | grep -c 'CREATE TABLE' || true)
  [[ "${table_count:-0}" -gt 0 ]] || fail "database dump has no CREATE TABLE"
  mv -- "$partial" "$db_file"
  partial=""
  log "database backup OK: tables=$table_count"

  if docker volume inspect "$CODE_VOLUME" >/dev/null 2>&1; then
    code_file="$BACKUP_DIR/coderepos_${stamp}.tar.gz"
    partial="$code_file.partial"
    docker run --rm -v "$CODE_VOLUME":/data:ro -w /data debian:bookworm-slim tar czf - . > "$partial"
    tar tzf "$partial" >/dev/null || fail "code repository archive validation failed"
    mv -- "$partial" "$code_file"
    partial=""
    log "code repository backup OK: $(basename "$code_file")"
  else
    fail "code repository volume is missing: $CODE_VOLUME (database backup retained)"
  fi
fi

# A receipt is bound to both the content and the destination. Credential changes
# do not invalidate receipts; changing NAS/share/account/prefix does.
if [[ "$OFFSITE_REQUIRED" == 1 ]]; then
  running "$BACKEND_CONTAINER" || fail "backend container unavailable; local backups retained, cleanup skipped"
  destination=$(docker exec "$BACKEND_CONTAINER" sh -c '
    [ -n "$SYNOLOGY_HOST" ] || exit 1
    printf "%s\n" "$SYNOLOGY_HOST" "$SYNOLOGY_PORT" "$SYNOLOGY_HTTPS" "$SYNOLOGY_SHARE_PATH" "$SYNOLOGY_ACCOUNT" | sha256sum
  ') || fail "offsite configuration missing; local backups retained, cleanup skipped"
  destination=$(printf '%s\n' "$destination" "$OFFSITE_PREFIX" | sha256sum | cut -d ' ' -f1)
fi

sync_offsite() {
  local local_file="$1" remote_name receipt expected sync_output
  remote_name="$(basename "$local_file")"
  receipt="$BACKUP_DIR/.offsite/$remote_name.sha256"
  expected="$destination $(sha256sum "$local_file" | cut -d ' ' -f1)"
  if [[ -f "$receipt" && "$(cat "$receipt")" == "$expected" ]]; then return 0; fi
  gzip -t "$local_file" || { log "[ERR] invalid local gzip: $remote_name"; return 1; }
  staged=$(docker exec "$BACKEND_CONTAINER" mktemp /tmp/digidocs-backup-sync.XXXXXX) || return 1
  docker cp "$local_file" "$BACKEND_CONTAINER:$staged" || return 1
  # Existing copies are verified first, so migration does not overwrite good
  # backups or refresh their NAS retention time unnecessarily.
  if sync_output=$(docker exec "$BACKEND_CONTAINER" /usr/local/bin/digidocs-backup-sync verify "$staged" "$OFFSITE_PREFIX/$remote_name" 2>&1); then
    log "existing offsite copy verified: $remote_name"
  elif sync_output=$(docker exec "$BACKEND_CONTAINER" /usr/local/bin/digidocs-backup-sync upload "$staged" "$OFFSITE_PREFIX/$remote_name" 2>&1); then
    log "offsite upload and verification OK: $remote_name"
  else
    log "[ERR] offsite delivery failed: $remote_name"
    echo "$sync_output" >&2
    return 1
  fi
  echo "$sync_output"
  docker exec "$BACKEND_CONTAINER" rm -f "$staged" || return 1
  staged=""
  printf '%s\n' "$expected" > "$receipt.tmp"
  mv -- "$receipt.tmp" "$receipt"
}

if [[ "$OFFSITE_REQUIRED" == 1 ]]; then
  # Most recent database backups first, then all pending historical archives.
  while IFS= read -r -d '' local_file; do
    sync_offsite "$local_file" || fail "offsite delivery incomplete; all local backups retained, cleanup skipped"
  done < <(find "$BACKUP_DIR" -maxdepth 1 -type f \( -name 'db_*.sql.gz' -o -name 'coderepos_*.tar.gz' \) -print0 | sort -zr)
else
  log "offsite delivery explicitly disabled (DIGIDOCS_OFFSITE_REQUIRED=0)"
fi

if [[ "$SYNC_ONLY" == 1 ]]; then
  log "sync-only complete; no backups deleted"
  exit 0
fi

# Only a fully successful run reaches retention. Pending files survive outages.
while IFS= read -r -d '' old_file; do
  if [[ "$OFFSITE_REQUIRED" == 0 || -f "$BACKUP_DIR/.offsite/$(basename "$old_file").sha256" ]]; then
    rm -- "$old_file"
    rm -f -- "$BACKUP_DIR/.offsite/$(basename "$old_file").sha256"
    log "pruned local backup: $(basename "$old_file")"
  fi
done < <(find "$BACKUP_DIR" -maxdepth 1 -type f \( -name 'db_*.sql.gz' -o -name 'coderepos_*.tar.gz' \) -mtime "+${KEEP_DAYS}" -print0)

if [[ "$OFFSITE_REQUIRED" == 1 ]]; then
  if prune_output=$(docker exec "$BACKEND_CONTAINER" /usr/local/bin/digidocs-backup-sync prune "$OFFSITE_PREFIX" "$KEEP_DAYS" 2>&1); then
    log "offsite retention complete"
  else
    log "[WARN] offsite retention failed (verified backups retained)"
    echo "$prune_output" >&2
  fi
fi
log "complete: database_backups=$(find "$BACKUP_DIR" -maxdepth 1 -name 'db_*.sql.gz' | wc -l) size=$(du -sh "$BACKUP_DIR" | cut -f1)"
