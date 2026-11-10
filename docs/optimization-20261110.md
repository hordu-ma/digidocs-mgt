# 2026-11-10 产品与工程优化记录

本记录跟踪 2026-11-10 评估后启动的一轮优化。每完成一个节点即更新本文件、`TASKS.md` 与 `README.md`，并在节点完成时部署验证、提交推送。

## 背景

对照 `docs/项目定义与技术架构.md` 的设计初衷做了一次全仓库评估。业务骨架和安全整改已到位，但有几块核心承诺尚未兑现，AI 层仍是「把上下文整体塞进 prompt」，用户侧缺少推动流程前进的闭环。

## 优先级与节点

| 节点 | 内容 | 状态 |
| --- | --- | --- |
| M1 | 访问审计：查看、预览、下载写入审计；总览「下载活跃度」；文档「访问记录」 | 已完成并部署 |
| M2 | 待办收件箱与通知：站内「待我处理」、通知表、邮件/Webhook 推送 | 待开始 |
| M3 | 交接候选自动归集；风险规则扩充；转交状态命名拆分 | 待开始 |
| M4 | AI：正文截断与分块、自动摘要、流式输出、结构化输出、建议可执行、向量检索 | 待开始 |
| M5 | 全文检索与服务端分页筛选 | 待开始 |
| M6 | 体验收口：权限可解释、时间与空值统一、版本折叠、命名统一、登录页品牌 | 待开始 |
| M7 | 工程与运维：任务死信与幂等、/metrics、Office 转 PDF 预览、孤儿对象对账、低危安全项 | 待开始 |
| 并行 | TLS、演示默认口令、群晖凭据轮换、main 分支保护 | 需负责人在仓库外执行，见文末 |

## M1 访问审计

### 问题

设计 18.1 要求记录访问、查看、下载。实现中 `AuditService` 只打日志，没有任何代码写入 `view` / `download`。审计摘要的下载计数恒为 0，总览「下载活跃度」没有数据源。

### 设计决定

- 新增 `repository.AuditWriter`，`AuditService` 改为持久化。业务写动作仍在各自事务内写审计，不改。
- 下载与预览 **fail closed**：审计写入失败时返回 `500 audit_failed`，不输出文件。理由是「谁下载了什么」是产品承诺，静默放行等于承诺失效。代价是数据库故障时下载不可用，此时文件读取本身也大概率失败，可接受。
- 查看文档详情为 best effort，失败只记日志。同一用户同一文档 10 分钟内去重，避免页面多次加载刷屏。下载从不去重。
- 新增动作类型 `preview`。`action_type` 实际列类型为 `varchar(32)`，无需迁移。
- 无文档的事件（数据资产下载）在 `extra_data.project_id` 记录课题。审计列表与摘要的项目判定改为「文档所属课题，否则 `extra_data.project_id`」，带 UUID 正则保护避免脏数据导致查询报错。
- 客户端 IP 复用限流的 `TRUSTED_PROXY_DEPTH` 规则；非法 IP 存 NULL。公网链路经 frp 时 IP 仍不可信，见部署说明 3.1.1。

### 改动

- 后端：`AuditService`、`AuditWriter`、`postgres/audit_writer.go`；版本下载/预览、文档详情、数据资产下载四个 handler 接入；审计摘要新增 `view_count`、`preview_count`、`recent_download_count`、`top_downloads`；审计列表新增 `user_display_name`、`document_title`。
- 前端：总览页新增「下载活跃度」面板，风险文档可点击跳转详情；文档详情新增「访问记录」。
- 文档：API 设计 5.4/5.5/8.4/8.5/13.2、数据库设计 2.6/3.10、项目定义 18.1 实现状态。

### 验证

- Go 单测：`AuditService` 委托、view 默认去重窗口、download 不去重、writer 错误上抛。
- 真实 PostgreSQL 集成测试 `TestPostgresIntegrationAccessAudit`：view 三次只落一条；download 两次落两条；非法 IP 存 NULL；数据资产下载按项目计入；非成员看不到任何计数。该测试首次运行发现参数类型推断错误（42P08），已加显式类型转换修复。
- 前端：`vue-tsc`、`vitest` 通过。
- `CI=true make verify` 通过。本机 `doctor` 的「Codex 项目技能已安装」检查在改动前就失败（本机没有 `~/.codex/skills`），属环境项，与本轮无关。
- 隔离烟测栈 `make smoke-e2e` 全绿。烟测脚本新增断言：版本下载与预览后，审计列表里必须能查到该版本的 `download` 与 `preview` 事件。
- 2026-11-10 15:00 部署：重建 `backend-go`、`frontend` 两个容器，均 healthy，`/healthz` 与前端 200，启动日志无错误。Worker 未改动未重建。
- 生产只读核验：在只读事务里对生产审计表执行新的项目归属表达式，无报错。顺带发现 `handover_confirm`、`handover_complete`、`admin_update` 没有课题归属，非管理员不可见，交接事件部分在 M3 修正。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,backend-py-worker,frontend}:before-opt-m1`。
- 部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m1/db.sql.gz`。
- 回滚命令：把两个旧标签重新标记为 `:latest` 后执行 `docker compose --profile app up -d --no-deps --no-build backend-go frontend`。

### 回滚

无 schema 变更。回滚只需恢复上一版三个应用镜像。已写入的 `view` / `preview` / `download` 审计行可保留，旧版本代码不会读取新字段。

## 需负责人在仓库外执行的事项

这些事项无法在仓库内完成，本轮不处理，保留在 `TASKS.md` 待办：

- 公网 TLS：需要证书与 frp/nginx 链路调整。
- 演示环境默认口令 `admin123`：线上账号操作。
- 群晖共用账号凭据轮换：DSM 改密后更新本机 `.env`。
- `main` 分支保护：仓库设置变更。
