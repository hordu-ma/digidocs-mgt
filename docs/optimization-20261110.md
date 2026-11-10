# 2026-11-10 产品与工程优化记录

本记录跟踪 2026-11-10 评估后启动的一轮优化。每完成一个节点即更新本文件、`TASKS.md` 与 `README.md`，并在节点完成时部署验证、提交推送。

## 背景

对照 `docs/项目定义与技术架构.md` 的设计初衷做了一次全仓库评估。业务骨架和安全整改已到位，但有几块核心承诺尚未兑现，AI 层仍是「把上下文整体塞进 prompt」，用户侧缺少推动流程前进的闭环。

## 优先级与节点

| 节点 | 内容 | 状态 |
| --- | --- | --- |
| M1 | 访问审计：查看、预览、下载写入审计；总览「下载活跃度」；文档「访问记录」 | 已完成并部署 |
| M2 | 待办收件箱与通知：站内「待我处理」、通知表、邮件/Webhook 推送 | 已完成并部署 |
| M3 | 交接候选自动归集；风险规则扩充；转交状态命名拆分 | 已完成并部署 |
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

## M2 待办收件箱与通知

### 问题

转交后接收人不知道，交接单等确认没人提醒，AI 摘要完成也没有提示。状态机会卡在「等对方发现」。用户表有微信号字段但从未使用。

### 设计决定

- **待办实时计算，不落库**。`GET /inbox` 每次从业务表计算，处理完成就自然消失，不存在「待办与事实不一致」的问题。每一类都对应调用者确实有权执行的动作，不会出现点进去 403。
- **通知落库为附属数据**（迁移 013 `notifications`）。所有通知钩子都在业务事务提交之后执行，写入失败只记日志，绝不回滚或阻塞转交与交接。
- **外发只用于需要对方行动的事件**：转交、创建交接（给接收人）、确认交接（给负责人）。其余只发站内，避免刷屏。
- **外发通道全部可选**：企业微信 / 钉钉群机器人或通用 JSON webhook，SMTP 邮件。外发在独立 goroutine 中执行，有 30 秒超时，不拖慢接口响应。
- **密钥不入日志**：webhook URL 自带机器人密钥。单元测试最初发现网络错误信息会带出完整 URL，已改为只保留操作名和底层错误。
- 交接页支持 `/handovers?id=` 直接打开对应交接单，通知和待办都用这个链接。
- 新增统一时间格式函数 `formatDateTime`（总是带年份），通知、待办和总览已改用，其余页面在 M6 收口。

### 改动

- 后端：迁移 013；`NotificationRepository`（postgres / memory）；`NotificationService` 及转交、接收、交接创建/确认/完成/取消、AI 摘要结果、成员加入七类钩子；`internal/notify` 外发通道；`/inbox` 与 `/notifications*` 五个接口；配置 `PUBLIC_BASE_URL`、`NOTIFY_WEBHOOK_*`、`SMTP_*`。
- 前端：顶栏通知铃铛（未读角标、抽屉列表、只看未读、全部已读、点击跳转）；总览「我的待办」；交接页按链接打开。
- 文档：API 设计 8.6、数据库设计 3.15、项目定义 Phase 2 实现状态、部署说明通知外发、用户使用说明总览章节、`.env` 模板与 compose。

### 验证

- 单元测试：收件人去重并排除触发人；转交外发带绝对链接；接收转交仅站内；非转交动作不通知；交接确认通知负责人；仅摘要类 AI 结果通知；站内写入失败时不外发；零值服务无副作用。外发通道测试覆盖企业微信、钉钉、通用三种载荷，非 2xx 报错，URL 不泄露，邮件头注入被中和。
- 真实 PostgreSQL 集成测试 `TestPostgresIntegrationNotificationsAndInbox` 走完整链路：添加成员→被加人收到通知；转交→接收人收到通知且待办出现；接收→原转交人收到通知且待办消失；他人不能标记本人通知；创建交接→接收人「待确认」、负责人「待补清单」；补清单后「待补清单」消失；确认→负责人「待完成」并收到通知；普通成员看不到负责人类待办。
- 前端：`vue-tsc`、`vitest`（新增时间格式测试，共 19 项）、生产构建通过。`make verify` 通过。
- 隔离栈 `make smoke-e2e` 全绿，新增 `/inbox`、`/notifications/unread-count`、`/notifications` 三项检查。
- 2026-11-10 15:10 部署：迁移 013 已应用，`backend-go`、`frontend` 均 healthy，未鉴权访问 `/inbox` 返回 401，启动日志显示外发通道为空（未配置，符合预期）。
- 生产只读核验：在只读事务里对 8 个活跃用户逐一执行待办 SQL，全部成功返回，无报错。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,frontend}:before-opt-m2`；部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m2/db.sql.gz`。
- 迁移 013 只新增 `notifications` 表。应用回滚后旧代码不读这张表，可以保留；如需彻底撤销，执行 `DROP TABLE notifications; DELETE FROM schema_migrations WHERE version = 13;`。

### 启用外发的方法

在生产 `.env` 填写 `PUBLIC_BASE_URL` 与 `NOTIFY_WEBHOOK_URL`（或 `SMTP_*`），然后 `docker compose --profile app up -d --no-deps backend-go`。启动日志会列出启用的通道。目前未配置，属负责人决定事项。

## M3 交接归集、风险规则与命名

### 问题

- 设计第 7 节要求系统按成员归集候选文档，实际创建的是空单，清单全靠负责人手工勾选。
- 风险只有「30 天未更新」一条，看不到责任人已停用、转交没人接、交接拖着不办。
- `pending_handover` 显示为「待交接」，总览「待交接」却统计的是毕业交接单，两件事共用一个词。交接单状态「已生成 / 待确认」也看不出该谁动手。
- M1 发现交接确认、完成事件的审计没有课题归属，非管理员看不到。

### 设计决定

- **归集在创建事务内完成**，默认勾选。理由是负责人取消比勾选少得多，且漏交比多交代价高。只按「当前责任人」与「上传人」归集，不猜测协作关系。
- **风险定义只有一份 SQL**，总览计数与列表共用，保证数字和列表一致。同一文档只按最严重的一条出现，避免重复计数。阈值写成常量并在文档列明，属于产品定义，不做成配置。
- **改名不改枚举**。状态机和数据库不动，只改界面文案，并把四个页面各自维护的状态文案收拢到 `utils/labels.ts`，从根上消除文案漂移。交接单状态按「下一步由谁处理」命名。
- 总览「待交接」改为「进行中交接」，统计口径扩大到 `generated` + `pending_confirm`。
- 风险项与待办项都由后端给出 `link`，前端不再自行拼接地址。

### 改动

- 后端：`collectHandoverCandidates`；创建响应新增 `document_item_count`、`data_item_count`；交接动作审计带 `project_id`；`riskItemsSQL` 四类规则；`RiskDocumentItem` 新增 `related_type`、`related_id`、`link`；`handover_pending_count` 口径调整。
- 前端：`utils/labels.ts`；文档列表、详情、总览、交接页改用共享文案；总览风险项显示类型标签并按 `link` 跳转；交接页改名「成员交接」，创建后提示带入数量并直接打开清单。
- 文档：API 设计 7.1、8.1、8.3；项目定义 6.1 命名说明与 7 实现状态；用户使用说明总览与交接章节。

### 验证

- 真实 PostgreSQL 集成测试 `TestPostgresIntegrationHandoverCandidates`：只带入被交接人本课题未删除、未归档的文档和其上传的数据；跨课题交接带入全部；交接确认审计带课题号。
- `TestPostgresIntegrationRiskRules`：停用责任人、转交 8 天未接收、40 天未更新、20 天未完成交接各命中一次；定稿文档、2 天内的转交、已完成交接不报；最严重的排第一；总览计数等于列表长度、进行中交接计数正确。
- 无头浏览器在隔离栈上截图核对总览、通知抽屉、交接页、文档详情（桌面与手机宽度），无页面脚本错误。截图发现两处问题并已修正：待办里「X 转交给你」在管理员代为转交时指错了人，改为「来自 X（原责任人）」；跨课题转交后的当前责任人看不到该文档的访问记录，改为按文档查看权限放开单文档审计查询。
- M2 的集成测试同步更新：交接单创建后带入 1 份文档，「待补清单」只在被交接人名下为空时出现，取消后消失。

- `make verify`、真实 PostgreSQL 集成测试、隔离栈 `make smoke-e2e` 全部通过。
- 2026-11-10 15:35 部署 `backend-go`、`frontend`，均 healthy，无数据库迁移。生产只读核验：风险 SQL 在生产数据上命中 1 条「转交未接收」、12 条「长期未更新」，执行无报错。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,frontend}:before-opt-m3`；部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m3/db.sql.gz`。
- 无 schema 变更。已自动写入的交接清单项是普通数据，回滚后旧版本照常读取。

## 需负责人在仓库外执行的事项

这些事项无法在仓库内完成，本轮不处理，保留在 `TASKS.md` 待办：

- 公网 TLS：需要证书与 frp/nginx 链路调整。
- 演示环境默认口令 `admin123`：线上账号操作。
- 群晖共用账号凭据轮换：DSM 改密后更新本机 `.env`。
- `main` 分支保护：仓库设置变更。
- 通知外发通道：需要提供企业微信/钉钉群机器人地址或 SMTP 账号。
