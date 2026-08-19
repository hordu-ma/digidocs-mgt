#!/usr/bin/env bash
#
# DigiDocs 数据备份。
#
# 备份两份东西——它们都是「丢了就没有了」的：
#   1. PostgreSQL 全库（业务事实：文档元数据、流转、交接、审计、账号）
#   2. code_repos 卷里的 Git 裸仓库（成员推送的代码资产的唯一副本）
#
# 群晖上的文件不在这里备份：它由 DSM 自己的快照/备份能力负责，见
# docs/现场部署逐条清单.md。.env 含密钥，也不进本备份，需单独妥善保管。
#
# 本地备份只防「人祸」（误删、误覆盖、坏迁移）：postgres_data 卷和
# $BACKUP_DIR 目前在同一台宿主机的同一块盘上，磁盘物理损坏或整机故障会
# 同时带走源数据和全部备份历史。备份成功后额外同步一份到群晖 NAS（异盘、
# 异机），复用 backend-go 容器已经配置好的 SYNOLOGY_* 凭据——应用本身就
# 用它做文件存储，这里不引入任何新凭据或新挂载方式。未配置群晖存储的部署
# （`STORAGE_BACKEND` 非 `synology`）会自动跳过这一步，只打印提示。
#
# 用法：
#   scripts/codex/backup.sh                 # 执行一次备份
#   BACKUP_KEEP_DAYS=30 scripts/codex/backup.sh
#
# 定时执行见 scripts/codex/install-backup-timer.sh。

set -euo pipefail

CONTAINER="${DIGIDOCS_PG_CONTAINER:-digidocs-postgres}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-digidocs_mgt}"
CODE_VOLUME="${DIGIDOCS_CODE_VOLUME:-digidocs-mgt_code_repos}"
BACKUP_DIR="${DIGIDOCS_BACKUP_DIR:-$HOME/ops/digidocs-backups}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"
BACKEND_CONTAINER="${DIGIDOCS_BACKEND_CONTAINER:-digidocs-backend-go}"
OFFSITE_PREFIX="${DIGIDOCS_OFFSITE_PREFIX:-ops-backups}"

stamp="$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

log() { echo "[backup] $*"; }
fail() { echo "[backup][ERR] $*" >&2; exit 1; }

docker ps --format '{{.Names}}' | grep -qx "$CONTAINER" \
  || fail "容器 $CONTAINER 未运行，无法备份数据库"

# ---------- 1. PostgreSQL ----------
db_file="$BACKUP_DIR/db_${DB_NAME}_${stamp}.sql.gz"
log "导出数据库 $DB_NAME → $(basename "$db_file")"
# pg_dump 的退出码要看 pipefail 前一段，否则 gzip 成功就会掩盖导出失败。
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists \
  | gzip > "$db_file"

# 空档或截断的备份比没有备份更危险——它会让人以为自己有备份。
db_size=$(stat -c %s "$db_file")
[ "$db_size" -gt 1024 ] || fail "数据库备份异常小（${db_size} 字节），判定失败"
gzip -t "$db_file" || fail "数据库备份 gzip 校验失败"
# 恢复时必须能看到建表语句，只校验 gzip 完整性不足以说明内容有效。
# 用 grep -c 而不是 grep -q：-q 命中后立刻退出会让 zcat 收到 SIGPIPE，在
# `set -o pipefail` 下被判成整条管道失败，把一份好备份误报为损坏。
table_count=$(zcat "$db_file" | grep -c "CREATE TABLE" || true)
[ "${table_count:-0}" -gt 0 ] || fail "数据库备份中找不到 CREATE TABLE，内容可疑"
log "数据库备份含 ${table_count} 条建表语句"
log "数据库备份 OK（$(du -h "$db_file" | cut -f1)）"

# ---------- 2. Git 裸仓库 ----------
if docker volume inspect "$CODE_VOLUME" >/dev/null 2>&1; then
  code_file="$BACKUP_DIR/coderepos_${stamp}.tar.gz"
  log "打包代码仓库卷 $CODE_VOLUME → $(basename "$code_file")"
  docker run --rm -v "$CODE_VOLUME":/data:ro -w /data debian:bookworm-slim \
    tar czf - . > "$code_file"
  gzip -t "$code_file" || fail "代码仓库备份 gzip 校验失败"
  log "代码仓库备份 OK（$(du -h "$code_file" | cut -f1)）"
else
  log "跳过代码仓库：卷 $CODE_VOLUME 不存在"
fi

# ---------- 3. 异地同步（群晖）----------
# 把刚生成、已校验过的备份文件复制进 backend-go 容器，再用同容器里的
# digidocs-backup-sync（复用 internal/storage/synology 客户端）推到群晖。
# 本地文件已经是校验通过的有效备份，这一步失败不影响本地备份继续保留。
sync_offsite() {
  local local_file="$1"
  local remote_name staged sync_output
  remote_name="$(basename "$local_file")"
  staged="/tmp/digidocs-backup-sync-payload"

  if ! docker ps --format '{{.Names}}' | grep -qx "$BACKEND_CONTAINER"; then
    log "跳过群晖同步：容器 $BACKEND_CONTAINER 未运行"
    return 0
  fi
  if ! docker cp "$local_file" "$BACKEND_CONTAINER:$staged" 2>/dev/null; then
    log "[ERR] 群晖同步失败：无法把 $remote_name 复制进 $BACKEND_CONTAINER"
    return 1
  fi

  if sync_output=$(docker exec "$BACKEND_CONTAINER" /usr/local/bin/digidocs-backup-sync upload "$staged" "$OFFSITE_PREFIX/$remote_name" 2>&1); then
    docker exec "$BACKEND_CONTAINER" rm -f "$staged" 2>/dev/null || true
    log "群晖同步 OK：$remote_name"
    return 0
  fi

  docker exec "$BACKEND_CONTAINER" rm -f "$staged" 2>/dev/null || true
  if grep -q "SYNOLOGY_HOST is not set" <<<"$sync_output"; then
    log "跳过群晖同步：本部署未配置 Synology 存储（STORAGE_BACKEND != synology）"
    return 0
  fi
  log "[ERR] 群晖同步失败：$remote_name"
  echo "$sync_output" >&2
  return 1
}

offsite_ok=1
sync_offsite "$db_file" || offsite_ok=0
if [ -n "${code_file:-}" ] && [ -f "$code_file" ]; then
  sync_offsite "$code_file" || offsite_ok=0
fi

# ---------- 4. 保留策略 ----------
log "清理 ${KEEP_DAYS} 天前的备份"
find "$BACKUP_DIR" -maxdepth 1 -type f \( -name 'db_*.sql.gz' -o -name 'coderepos_*.tar.gz' \) \
  -mtime "+${KEEP_DAYS}" -print -delete || true

log "完成。当前保有 $(find "$BACKUP_DIR" -maxdepth 1 -name 'db_*.sql.gz' | wc -l) 份数据库备份，"
log "目录占用 $(du -sh "$BACKUP_DIR" | cut -f1)：$BACKUP_DIR"

# 群晖侧同步过去之后不会自动过期，每天一次不清理会一直堆积。这一步只是
# 「省磁盘」，不是安全网，失败只警告、不影响本次备份的成败判定。
if docker ps --format '{{.Names}}' | grep -qx "$BACKEND_CONTAINER"; then
  if prune_output=$(docker exec "$BACKEND_CONTAINER" /usr/local/bin/digidocs-backup-sync prune "$OFFSITE_PREFIX" "$KEEP_DAYS" 2>&1); then
    log "群晖侧清理完成"
  elif ! grep -q "SYNOLOGY_HOST is not set" <<<"$prune_output"; then
    log "[WARN] 群晖侧清理失败（不影响本次备份结果）："
    echo "$prune_output" >&2
  fi
fi

# 本地备份已经落盘且校验通过，不会被下面这一步撤销——只是让"群晖同步失败"
# 这件事在 systemctl --user list-timers / journalctl 里能被看见，而不是被
# 本地备份的成功日志盖过去，逼着有人真去 NAS 侧查一下。
[ "$offsite_ok" -eq 1 ] || fail "本地备份已完成，但群晖异地同步失败，请检查群晖连通性/凭据后重跑"
