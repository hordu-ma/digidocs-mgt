# 2026-10-08 备份服务修复记录

> 本文记录历史故障和恢复证据。日常操作以[运行参考](部署准备与运行说明.md#备份和恢复)为准。

本机 `digidocs-backup.service` 已恢复成功。共用群晖账号密码轮换由用户负责，尚未纳入本次完成验收。

## 故障与范围

- NAS Tailscale 节点密钥于 2026-10-05 08:01（UTC+8）过期，10 月 6 至 8 日异地同步超时；本地数据库备份仍正常。
- 用户于 10 月 8 日恢复 NAS 身份，实测 Tailscale ping 成功、DSM HTTPS 5001 返回 200。
- 原客户端将含密码或会话 ID 的 HTTP URL 包入错误并写入日志。本轮统一在 Synology 适配层去除 URL 查询串、用户信息和 fragment，同时保留底层错误类型。
- 只重建 `backend-go` 容器；未调整业务 API、数据库结构、网络路由、防火墙、备份 timer 或其他应用部署。

## 实现

- `backend-go/internal/storage/synology/provider.go`：请求构造和 HTTP 执行错误统一脱敏，覆盖登录、列表、上传、下载及重试。
- `backend-go/cmd/backup-sync`：新增只读 `verify`；`upload` 必须在上传后下载核对字节数及 SHA-256 才成功；NAS 清理只匹配备份文件名，保留无关文件。
- `scripts/codex/backup.sh`：`flock` 串行执行；本地文件校验后原子发布；`.offsite/` 记录内容和目的地确认摘要；自动核对并补传未确认的历史文件；任何同步失败均非零退出并暂停清理；`--sync-only` 不生成或删除备份。
- 默认要求异地同步。纯本地部署必须显式设置 `DIGIDOCS_OFFSITE_REQUIRED=0`；后端停止和群晖配置缺失不再被判成备份成功。
- `scripts/codex/test_backup.py` 已纳入 `make verify`，不新增第三方依赖。

## 验收证据

时间均为本机 UTC+8；日志中的 Go 时间为 UTC。

| 验证 | 结果 |
| --- | --- |
| 历史副本核验 | 30 份全部完成下载 SHA-256 校验；22 份直接确认已有副本，8 份重新上传后确认。重新上传不等同于认定此前副本必然不存在 |
| 10 月 6 至 8 日补传 | 数据库与代码仓库归档共 6 份，全部通过 |
| 完整备份服务 | 16:25:18 启动，16:25:26 完成；`Result=success`、`ExecMainStatus=0` |
| 新数据库备份 | `db_digidocs_mgt_20261008-162518.sql.gz`，132540 字节 |
| 新数据库 SHA-256 | `01308edfba8bbef9518d8b11099dbd29cd952a5922ae741e7328fd6d2e6630d3` |
| 新代码仓库归档 | `coderepos_20261008-162518.tar.gz`，104 字节；源卷当前为空，空归档与源卷一致 |
| 本地确认记录 | 本轮结束共 32 份，含 16 份数据库备份及 16 份代码归档 |
| 隔离恢复 | 使用同版本 PostgreSQL、`--network none`、独立 tmpfs；SQL 导入约 0.186 秒，23 张表 / 259 行；逐表行数、内容摘要和序列与源库一致，源库在核验前后保持稳定；演练容器已删除 |
| 脱敏运行态 | 在已部署的工具进程中临时覆盖目标为回环拒绝连接端口，错误不含真实账号、密码、编码密码或 SID 查询参数；未改实际服务配置 |
| 自动化验证 | `go test ./...`、7 项备份故障回归、`make verify`（含 Worker / 前端测试及前端构建）、`git diff --check` 通过 |
| 在线检查 | DigiDocs 四个容器 healthy；本地 `/healthz` 成功，公网入口 HTTP 200；备份 timer active |

恢复演练使用本地最新备份，其字节内容已经与 NAS 下载内容核对一致；未覆盖 NAS 文档对象自身的 DSM 快照恢复，也不代表真机客户验收。

本机详细证据位于 `/home/pgx/ops/digidocs-backup-repair-20261008/`：`backfill.log`、`service.log`、`restore-evidence.json`、`redaction-probe.log`、`verify.log`、`deployment.json`。该目录权限为 0700，未加入凭据或原始未脱敏日志。

## 版本与回滚

- 发布前代码：`cdb32bc6be227d8ee88d15b2601d1c98c83baa97`。
- 发布镜像：`sha256:cde26ab78b290c7e9db2aefef118daab804dc64d4edf63dbe4261b6ddd1384d0`。
- 原镜像已保留为 `digidocs-mgt-backend-go:backup-repair-20261008-before`，ID 为 `sha256:99432716f9b19a4bbe34f08e81fa6498cfd597cf3cf28e15fd261270ab690630`。
- 原备份脚本保留为上述证据目录中的 `backup.sh.before`。

仅在需要撤销本次发布时，于仓库根目录执行：

```bash
cp /home/pgx/ops/digidocs-backup-repair-20261008/backup.sh.before scripts/codex/backup.sh
docker compose -f docker-compose.yml \
  -f /home/pgx/ops/digidocs-backup-repair-20261008/rollback.compose.yml \
  up -d --no-deps --no-build backend-go
```

回滚保留备份及 `.offsite/` 确认标记，不恢复已泄露的旧密码。旧镜像会撤销本次日志脱敏；如需紧急回滚，之后仍需补回脱敏修复。

## 密码轮换收尾

用户已确认该群晖账号共用，DSM 密码及其他设备由用户更新。本机加载位置为 `/home/pgx/code-repos/digidocs-mgt/.env` 的 `SYNOLOGY_PASSWORD`（文件权限 0600），不要把新密码发到聊天或命令行参数。

更新 `.env` 后运行：

```bash
docker compose up -d --no-deps --no-build backend-go
systemctl --user start digidocs-backup.service
systemctl --user show digidocs-backup.service -p Result -p ExecMainStatus
```

需确认新凭据下的异地上传及下载校验成功。已有日志中的旧密码不会因为部署脱敏而自动消失，轮换才会使其失效；本轮没有清空系统历史日志。
