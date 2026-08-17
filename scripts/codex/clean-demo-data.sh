#!/usr/bin/env bash
#
# 清理演示库中由 smoke 脚本产生的残留数据。
#
# 背景：`make smoke` / `make smoke-e2e` 直接跑在演示库上，每跑一次就往里写一批
# 一次性数据（版本、会话、注册账号、邀请链接）。客户打开演示时看到的是二十多个
# 叫 smoke-v2.txt 的版本和一串「请用一句话确认 smoke 已打通」的会话，属于销售
# 场景的直接减分项。
#
# 本脚本只删「结构上可判定为 smoke 产物」的行，判定依据是稳定标记而不是时间窗：
#   - 账号：email 以 @digidocs.test 结尾（smoke 脚本固定用这个域）
#   - 文档版本：file_name 以 smoke 开头
#   - 助手会话：标题为 smoke 固定提示词
#   - 邀请链接：由上述 smoke 流程创建、且仍处于 active 的记录
#
# 真实演示内容（种子账号、真实文档与版本、人工创建的会话）一律不动。
#
# 用法：
#   scripts/codex/clean-demo-data.sh --dry-run   # 只统计，不删（默认）
#   scripts/codex/clean-demo-data.sh --apply     # 实际执行
#
# 执行 --apply 前会自动 pg_dump 一份备份到 ~/ops/digidocs-backups/。

set -euo pipefail

CONTAINER="${DIGIDOCS_PG_CONTAINER:-digidocs-postgres}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_NAME="${POSTGRES_DB:-digidocs_mgt}"
BACKUP_DIR="${DIGIDOCS_BACKUP_DIR:-$HOME/ops/digidocs-backups}"

MODE="dry-run"
case "${1:-}" in
  --apply) MODE="apply" ;;
  --dry-run|"") MODE="dry-run" ;;
  *) echo "用法: $0 [--dry-run|--apply]" >&2; exit 2 ;;
esac

psql_run() {
  docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"
}

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "[ERR] 容器 $CONTAINER 未运行" >&2
  exit 1
fi

# 判定条件集中在这里，统计与删除共用同一套口径。
read -r -d '' JUNK_CTE <<'SQL' || true
WITH junk_users AS (
  SELECT id FROM users WHERE email LIKE '%@digidocs.test'
), junk_versions AS (
  SELECT id, document_id FROM document_versions WHERE file_name LIKE 'smoke%'
), junk_convs AS (
  SELECT id FROM assistant_conversations WHERE title LIKE '%smoke 已打通%'
)
SQL

echo "== 当前 smoke 残留统计 =="
psql_run <<SQL
$JUNK_CTE
SELECT 'smoke 版本' AS 项目, count(*) AS 行数 FROM junk_versions
UNION ALL SELECT 'smoke 版本关联审计事件', count(*) FROM audit_events
  WHERE version_id IN (SELECT id FROM junk_versions)
UNION ALL SELECT 'smoke 会话', count(*) FROM junk_convs
UNION ALL SELECT 'smoke 会话消息', count(*) FROM assistant_conversation_messages
  WHERE conversation_id IN (SELECT id FROM junk_convs)
UNION ALL SELECT 'smoke 助手请求', count(*) FROM assistant_requests
  WHERE conversation_id IN (SELECT id FROM junk_convs)
UNION ALL SELECT 'smoke 测试账号', count(*) FROM junk_users
UNION ALL SELECT 'smoke 账号的课题成员关系', count(*) FROM project_members
  WHERE user_id IN (SELECT id FROM junk_users)
UNION ALL SELECT '未使用的邀请链接(active)', count(*) FROM project_invites WHERE status = 'active';
SQL

if [ "$MODE" = "dry-run" ]; then
  echo
  echo "[DRY-RUN] 未做任何修改。确认无误后执行：$0 --apply"
  exit 0
fi

mkdir -p "$BACKUP_DIR"
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_pre-clean_$(date +%Y%m%d-%H%M%S).sql"
echo
echo "== 备份到 $BACKUP_FILE =="
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists > "$BACKUP_FILE"
echo "[OK] 备份完成（$(du -h "$BACKUP_FILE" | cut -f1)）"

echo
echo "== 执行清理（单事务） =="
psql_run <<SQL
BEGIN;

-- 1) 文档的 current_version_id 可能指向即将删除的 smoke 版本，先回退到该文档
--    仍然保留的最新版本，否则删除会被外键挡住、或留下悬空指针。
UPDATE documents d
SET current_version_id = (
      SELECT dv.id FROM document_versions dv
      WHERE dv.document_id = d.id AND dv.file_name NOT LIKE 'smoke%'
      ORDER BY dv.version_no DESC LIMIT 1
    )
WHERE d.current_version_id IN (SELECT id FROM document_versions WHERE file_name LIKE 'smoke%');

-- 2) 审计事件引用版本（ON DELETE NO ACTION），必须先删。
DELETE FROM audit_events
WHERE version_id IN (SELECT id FROM document_versions WHERE file_name LIKE 'smoke%');

DELETE FROM document_versions WHERE file_name LIKE 'smoke%';

-- 3) 助手请求引用会话（NO ACTION），消息是 CASCADE，会随会话一起删。
DELETE FROM assistant_requests
WHERE conversation_id IN (SELECT id FROM assistant_conversations WHERE title LIKE '%smoke 已打通%');

DELETE FROM assistant_conversations WHERE title LIKE '%smoke 已打通%';

-- 4) 成员关系先于账号删除。
DELETE FROM project_members
WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@digidocs.test');

DELETE FROM users WHERE email LIKE '%@digidocs.test';

-- 5) smoke 生成的邀请链接。
DELETE FROM project_invites WHERE status = 'active';

COMMIT;
SQL

echo
echo "== 清理后复核 =="
psql_run <<SQL
SELECT 'smoke 版本' AS 项目, count(*) AS 剩余 FROM document_versions WHERE file_name LIKE 'smoke%'
UNION ALL SELECT 'smoke 会话', count(*) FROM assistant_conversations WHERE title LIKE '%smoke 已打通%'
UNION ALL SELECT 'smoke 测试账号', count(*) FROM users WHERE email LIKE '%@digidocs.test'
UNION ALL SELECT 'active 邀请链接', count(*) FROM project_invites WHERE status = 'active'
UNION ALL SELECT 'current_version_id 悬空的文档', count(*) FROM documents d
  WHERE d.current_version_id IS NOT NULL
    AND NOT EXISTS (SELECT 1 FROM document_versions dv WHERE dv.id = d.current_version_id);
SQL

echo
echo "[OK] 清理完成。备份：$BACKUP_FILE"
echo "[提示] smoke 版本在群晖上共用同一个对象键，数据库行删除后该文件会成为孤儿，"
echo "       需要时手动清理：documents/<document_id>/smoke-v2.txt"
