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

# ---------- 3. 保留策略 ----------
log "清理 ${KEEP_DAYS} 天前的备份"
find "$BACKUP_DIR" -maxdepth 1 -type f \( -name 'db_*.sql.gz' -o -name 'coderepos_*.tar.gz' \) \
  -mtime "+${KEEP_DAYS}" -print -delete || true

log "完成。当前保有 $(find "$BACKUP_DIR" -maxdepth 1 -name 'db_*.sql.gz' | wc -l) 份数据库备份，"
log "目录占用 $(du -sh "$BACKUP_DIR" | cut -f1)：$BACKUP_DIR"
