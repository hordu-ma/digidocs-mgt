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
| M4 | AI：正文截断与分块、自动摘要、流式输出、结构化输出、建议可执行、向量检索 | 已完成并部署（向量检索因网关无嵌入接口改为词面检索，见下） |
| M5 | 全文检索与服务端分页筛选 | 已完成并部署 |
| M6 | 体验收口：权限可解释、时间与空值统一、版本折叠、命名统一、登录页品牌 | 已完成并部署 |
| M7 | 工程与运维：任务死信与幂等、/metrics、Office 预览、孤儿对象对账、低危安全项 | 已完成并部署 |
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

## M4 AI 能力层

### 网关能力探测（2026-11-10）

动手前先对本机 OpenClaw 网关逐项实测，决定实现方式：

| 能力 | 结果 | 决定 |
| --- | --- | --- |
| `stream=true` | 正常返回 SSE 增量 | 问答改为流式 |
| `response_format: json_schema` | 400，后端模型不支持 | 不用 |
| `response_format: json_object` | 正常 | 结构化输出使用，带 400 降级 |
| `POST /v1/embeddings` | 500 | 向量检索暂不可行 |
| 数据库扩展 | 有 `pg_trgm`，无 `vector` | 词面检索 + 三元组索引 |

顺带发现：一句话的请求网关也计入约 2.5 万提示 token，说明网关每次注入大量自身上下文。这违背项目定义 12.5.1「无隐式环境记忆」，并拖慢所有 AI 调用。属网关配置，已记入项目定义「已知限制」，需负责人在网关侧处理。

### 问题

- Worker 把整篇正文 `json.dumps` 进提示词，不截断、不分块，`document_context.extracted_text` 与 `document_text` 还重复发送一遍。长文会超出上下文或触发 180 秒超时。
- 正文只存在 `assistant_requests.output` 的 JSON 里，课题范围问答拿不到任何正文，只能基于元数据回答。
- 回答要等全部生成完才显示，最长 4 分钟只有动画。
- 上传后不自动摘要；结构化输出全靠提示词；AI 建议只能确认或忽略，确认后不会发生任何事。

### 设计决定

- **正文入库分块**（迁移 014 `document_text_chunks`）：抽取或摘要结果回写时在同一事务写入，只索引当前版本，旧版本迟到结果被忽略。迁移回填了历史正文。分块规则在 Go、Python、迁移 SQL 三处一致，并有测试互相锁定。
- **词面检索**：中文按两字组、英文按单词，按命中词数排序。不引入分词库或向量库，因为当前基础设施不支持向量，而两字组对中文召回足够实用。分块表为向量列预留了位置。
- **预算而非截断**：文档范围按问题选片段，课题范围走后端检索，总长受 `MAX_CONTEXT_CHARS` 约束；回答用 `[片段N]` 引用，引用随会话消息保存，前端可跳转核对。
- **长文分段提炼再汇总**，超出段数上限时均匀抽样并在摘要中如实说明，输出 `summary_basis`。
- **流式用「增量回传 + 1 秒轮询」实现**，而非 SSE 直推浏览器。理由：Worker 与浏览器之间隔着后端和数据库，SSE 需要新增长连接通道和 nginx 缓冲配置；现有轮询改成 1 秒并渲染部分回答即可达到同样体验，改动面小。
- **建议可执行但 AI 不执行**：建议可附带白名单动作，用户点「确认并执行」时后端以用户身份走正常流转服务，失败则建议保持待确认。白名单在 Worker 和后端双重校验。
- **自动摘要**默认开启，可用 `AUTO_SUMMARIZE_ON_UPLOAD=false` 关闭。

### 改动

- 迁移 014；`internal/textsearch`；`TextSearchRepository` / `SearchService`；正文入库；摘要版本回退；`/internal/worker-progress`、`/internal/assistant-context/projects/{id}/search`；建议 `action_payload` 与 `ConfirmAndExecuteSuggestion`；自动摘要链；会话消息保存引用。
- Worker：`context_budget`；流式 `_chat_stream`；`json_object` 降级与修复；长文分段摘要；建议动作白名单；进度回传节流；课题检索客户端。
- 前端：助手页实时显示生成中的回答、1 秒轮询、引用来源标签；文档详情「确认并执行」。
- 文档：API 设计 9.1.1、9.2、9.5.5、9.5.6、9.6；数据库设计 3.11、3.16；异步任务契约；项目定义 AI 升级与已知限制；部署说明与环境模板；用户使用说明助手与 AI 建议。

### 验证

- Go：`textsearch` 分块与迁移回填数量一致、查询词规则、转义、摘录；建议动作白名单；确认并执行以用户身份运行、失败保持待确认、非可执行建议拒绝；自动摘要只在开启且抽取成功时触发。
- 真实 PostgreSQL `TestPostgresIntegrationTextChunksSearchProgressAndActions`：长文分块；检索首条命中答案并带摘录；非成员检索为空；新版本替换索引、旧版本迟到结果被忽略；进度只写入运行中的请求；只保留白名单动作；无版本号摘要写到输出版本。
- Worker：12 项新测试覆盖流式增量、首 token 前失败降级、中途断开报错、`json_object` 400 后关闭、非 JSON 修复、长文分段抽样、片段预算、课题检索与引用、进度节流、进度回传不抛错，以及与 Go 一致的分词和分块规则。

- 真实网关端到端（隔离数据库）：上传约 7000 字、关键句埋在中间的版本后，正文自动入库为 10 个分块并自动生成摘要（`summary_basis=full_text`）；课题范围提问「奖励函数怎么设计、折扣因子多少」，回答准确给出 PPO、两部分奖励与 0.98，并以 `[片段1][片段2]` 引用《课题申报书》v5；流式调用 4.9 秒完成，后端收到增量回传。
- 隔离栈的 Worker 容器无法访问网关：本机防火墙只放行生产网段（compose 烟测文件已有说明）。本轮未改防火墙，改为在宿主机临时运行一份 Worker，连接隔离栈后端与本机网关完成验证。若要让 `RUN_ASSISTANT_SMOKE=1` 在容器内直接通过，需负责人为 172.30.0.0/24 放行网关端口。
- 2026-11-10 15:52 部署 `backend-go`、`backend-py-worker`、`frontend`，均 healthy。迁移 014 已应用：回填 15 份文档共 61 个分块，`pg_trgm` 索引创建成功。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,backend-py-worker,frontend}:before-opt-m4`；部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m4/db.sql.gz`。
- 迁移 014 新增一张表和一个可空列，旧代码不读取，回滚应用时可保留。彻底撤销：`DROP TABLE document_text_chunks; ALTER TABLE assistant_suggestions DROP COLUMN action_payload; DELETE FROM schema_migrations WHERE version = 14;`。
- Worker 与后端必须同批回滚：旧后端没有 `/internal/worker-progress` 和检索接口，新 Worker 对此降级处理（进度丢弃、检索返回空），不会失败，但旧 Worker 配新后端同样可用。

## M5 全文检索与服务端分页

### 问题

- 文档页一次拉 200 条再在浏览器里筛选与排序，「我负责的」「待接收」等筛选漏掉第 200 条以后的文档，左侧课题计数也只统计已加载部分。
- 搜索只匹配标题；正文已在 M4 入库但用户用不上。
- 行菜单根据 `is_archived` 决定显示「恢复」还是「移出工作台」，但列表接口从不返回 `is_archived`，于是永远显示「移出工作台」，点了实际是**软删除**，文案与行为不符。
- 删除的文档界面上无处恢复，用户手册却写着可以通过「显示已归档」恢复。
- 命令面板（Ctrl+K）直接显示英文状态枚举。

### 设计决定

- 筛选、排序、计数全部下沉到服务端，排序用白名单映射到 SQL，平局按 id 保证翻页稳定；列表与计数共用一个 WHERE 构造函数。
- 关键词按字面匹配，转义 `%` 与 `_`。
- 回收站是一个筛选条件而非新页面，只对课题 owner/manager 与管理员返回数据，与「恢复」的权限一致。
- 全文检索复用 M4 的分块表与分词规则，每篇文档只返回最相关的一段。
- 行菜单改为明确的「删除文档」（二次确认）与「恢复文档」。

### 改动

- 后端：`ListDocuments` 重写（`sort`、`mine`、`updated_within_days`、`deleted_only`、`with_counts`、转义、返回 `is_archived`/`is_deleted`/`project_id`）；`CountDocumentsByProject`；`SearchDocuments` 与 `GET /search`。
- 前端：文档页按页加载、筛选变化回到第一页并丢弃过期响应、课题栏用服务端计数、正文命中区、回收站、删除确认与恢复、分页；命令面板加正文检索与中文状态。
- 文档：API 设计 4.2、4.2.1；用户使用说明文档列表、文档详情与快速导航。

### 验证

- 真实 PostgreSQL `TestPostgresIntegrationDocumentListFilters`：标题排序加分页、我负责的、近 7 天、`%` 按字面匹配、普通成员看不到回收站、课题负责人能看到并带 `is_deleted`、课题计数忽略 `project_id`。
- `SearchDocuments` 一篇文档只返回一行且带摘录，非成员检索为空。
- `make verify` 与隔离栈 `make smoke-e2e` 通过；无头浏览器在桌面和手机宽度下核对文档页与「我负责的」筛选，课题栏计数随筛选变化，无页面脚本错误。
- 2026-11-10 16:10 部署 `backend-go`、`frontend`，均 healthy，无数据库迁移；未鉴权访问 `/search` 返回 401。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,frontend}:before-opt-m5`；部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m5/db.sql.gz`。无 schema 变更。

## M6 体验收口

### 问题

- 详情页动作按钮由前端写死的状态表决定，与后端状态机不一致：「已交接」的文档后端允许开始处理、定稿、归档，前端一个按钮都不给，交接完成后文档在界面上卡死。按钮也不看权限，普通成员看得到「定稿」，点了才报 403。
- 详情接口不返回更新时间、所属目录、当前版本号，页面上「更新时间」永远是「-」，「所属目录」永远是「未挂接目录」。
- 时间格式有 7 处各自实现；代码页把 UTC 时间原样打印，比本地时间慢 8 小时；成员页、交接页格式又各不相同。
- 空值有「-」「时间待同步」「尚未推送」等多种写法。
- 成员页自助注册用户没有用户名，「账号」列显示「-」，负责人分不清是谁。
- 版本历史无折叠；登录与注册页只有一张卡片，没有品牌信息。
- 前端仍有 `any` 类型。

### 设计决定

- 状态机只保留一份（`service.ValidFlowTransition`），流转事务和详情按钮共用；按钮的可执行性复用流转接口的同一套权限检查，保证「亮着的按钮不会 403，灰着的按钮说明原因」。
- 时间统一为 `YYYY-MM-DD HH:mm` 本地时间，始终带年份；空值统一为「暂无 X」。
- 成员账号显示由数据库计算：有用户名显示用户名，否则显示脱敏手机号（前三后四）或脱敏邮箱，不暴露完整联系方式。
- 登录与注册共用 `AuthShell` 品牌框架，手机宽度下品牌区收窄、隐藏卖点列表。

### 改动

- 后端：`flow_rules.go`；详情补字段与 `actions`；成员列表 `account`。
- 前端：详情页按 `actions` 渲染并带禁用提示、版本历史折叠、「暂无版本」；全部视图改用 `formatDateTime` / `formatDate`；空值文案；成员页账号；`AuthShell`；API 类型定义，`any` 清零。
- 文档：API 设计 4.3；用户使用说明流转操作区与版本历史。

### 验证

- 单元测试：状态机规则（已交接可定稿/恢复/归档，草稿不可归档等）。
- 真实 PostgreSQL `TestPostgresIntegrationDocumentDetailActions`：已交接文档对当前责任人（参与人）返回 3 个动作，「开始处理」可执行，「定稿」「归档」不可执行并带原因；课题负责人全部可执行；详情带更新时间与课题名。
- `make verify`、隔离栈 `make smoke-e2e` 通过。无头浏览器核对登录页（桌面与手机）、成员视角的文档详情与成员页，无页面脚本错误；接口核对：参与人在本人已定稿文档上「归档」返回禁用及原因。
- 2026-11-10 16:30 部署 `backend-go`、`frontend`，均 healthy，无数据库迁移。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,frontend}:before-opt-m6`；部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m6/db.sql.gz`。无 schema 变更。

## M7 工程与运维

### 问题

- 卡死任务每 10 分钟重新排队，没有次数上限；一个总能卡死 Worker 的文件会被无限重试，反复占用 Worker 并重复调用模型。结果回写不幂等，重复回调会覆盖答案、重建已确认的建议、追加第二条聊天消息。
- 改密和停用账号都不会让已签发的令牌失效，最长 2 小时内仍可用。没有 `exp` 的令牌被视为永不过期。改密时输错当前密码，前端会把用户直接登出。
- 推送令牌用 `==` 比较；随机数失败时退化为固定字符串 `"token"`；请求体大小豁免用 `Contains("/internal/")` 判断；代码仓库接口返回宿主机路径。
- 公网可随意自助注册。
- 没有任何运行指标；存储孤儿对象无法发现；网关不可达只会在 AI 任务失败时才暴露。
- Office 文件无法在线查看。

### 设计决定

- 任务最多领取 3 次，第 3 次后仍卡死就标为失败（死信）；已结束的请求忽略迟到的结果。
- 会话失效靠 `users.password_changed_at` 加令牌 `iat` 比较实现，不建黑名单；用户状态缓存 30 秒，查库失败时放行。改密接口返回新令牌，当前会话不中断。内存模式（开发与处理器测试）不启用此检查。
- 自助注册三种模式，默认 `open`，保持现有行为；仅凭邀请模式复用现有邀请令牌。
- `/metrics` 不引入客户端库，自写文本格式；必须配置令牌才开启。路由标签用路由模板，序列数量有上限。
- Office 预览不引入 LibreOffice（镜像会增大约 500 MB），改为用 M4 已入库的正文拼回全文，在抽屉中只读展示，并计入预览审计。
- 存储对账只读，不做自动清理：误删客户文件的代价远高于孤儿文件占用的空间。

### 改动

- 后端：迁移 015；队列死信；结果幂等；`SessionValid` 与中间件钩子、`ReissueToken`、无 `exp` 令牌拒绝；推送令牌常量时间比较、随机数失败时 panic、请求体豁免改为前缀匹配、隐藏 `repo_storage_path`；`SELF_REGISTRATION` 与 `/system/info`；`internal/metrics` 与 `GET /metrics`；`GET /documents/{id}/text`；`cmd/storage-audit` 并打进镜像。
- Worker：启动时探测网关。
- 前端：改密保存新令牌，改密接口不触发统一登出；登录页、邀请页按注册模式显示入口，注册时带邀请令牌；详情页「在线查看正文」。
- 文档：API 设计 2.1.1、2.4、4.2.2、8.7；数据库设计 3.17；部署说明安全与运维；环境模板与 compose；用户使用说明。

### 验证

- 单元测试：会话规则六种情形与缓存；无 `exp` 令牌拒绝；三种注册模式与 `/system/info`；指标路由归并、未匹配计数、令牌门槛；对账走树、孤儿与缺失计算；网关探测；邀请重定向解析。
- 真实 PostgreSQL：改密后旧令牌失效、新令牌有效、停用账号令牌失效；所有指标查询在迁移后的库上可执行；正文拼回结果与原文完全一致。
- `make verify`、集成测试、隔离栈 `make smoke-e2e` 通过。隔离栈接口实测：改密后旧令牌 401、新令牌 200；输错当前密码返回 401；未配置令牌时 `/metrics` 返回 404。
- 2026-11-10 19:47 部署三个容器，均 healthy。迁移 015 已应用；Worker 启动日志显示 `openclaw gateway reachable: HTTP 200`；注册模式保持默认 `open`，`/metrics` 未开启。
- 生产只读对账：存储中有 21 个文件，数据库引用 19 个，发现 2 个孤儿文件，都是早期烟测写入的 `smoke-v2.txt`（共 42 字节），没有缺失文件。按设计未自动删除，留给负责人决定。

### 部署与回滚记录

- 回滚镜像标签：`digidocs-mgt-{backend-go,backend-py-worker,frontend}:before-opt-m7`；部署前数据库备份：`/home/pgx/ops/digidocs-reviews/rollback-20261110-m7/db.sql.gz`。
- 迁移 015 只新增两个带默认值的列，旧代码不读取，回滚应用时可以保留。

## 后续处理（2026-11-10 晚）

### 注册模式与孤儿文件

- 生产 `.env` 设置 `SELF_REGISTRATION=invite_only` 并重启后端（改前备份 `rollback-20261110-m7/env.before-invite-only`）。`/system/info` 返回 `invite_only`；不带邀请令牌的注册探测返回 `403 invite_required`，未创建账号。
- `digidocs-storage-audit` 新增 `-delete-orphan`：删除前重新对账，只删除当前仍是孤儿、位于 `documents/` 或 `data/` 下的文件；任一目标不合格则一个都不删；先把内容备份到 `-backup-dir`。用它删除了两个 `smoke-v2.txt`（各 21 字节，内容均为 "smoke second version"），备份在 `/home/pgx/ops/digidocs-reviews/orphan-backup-20261110/`。复核：存储 19 个文件，数据库引用 19 个，孤儿与缺失均为 0。
- 负责人决定：演示口令与群晖密码不轮换。

### OpenClaw 网关问题

排查结论：

- 18789 端口上是个人 OpenClaw **智能体**网关（带长期记忆、定时任务、WhatsApp 日报）。`/v1/chat/completions` 请求进入的是这个智能体，每次都被注入人设、记忆与工具说明，所以一句话的请求也计约 25,157 个提示 token。
- 该智能体的**主模型是 `deepseek/deepseek-flash`（DeepSeek 公网 API）**，本地 Qwen 只是备用。DigiDocs 的摘要和问答都携带文档正文，因此课题文档一直被发往外部云服务，与「私有化部署、数据留在本单位」相冲突；文档内容也可能进入个人智能体的会话与记忆。
- 本机 `llama-server`（Qwen3.8 27B，`127.0.0.1:11500`）提供 OpenAI 兼容接口。直连实测同样的问题只计 21 个提示 token、0.6 秒返回（关闭思考模式），`json_object` 可用。

方案（最佳实践）：业务系统不经过个人智能体，直连本地模型；网关只留给个人助手用途。

- 新增 `ops/systemd/digidocs-llm-bridge.py` 与 `scripts/codex/install-llm-bridge.sh`：照搬现有 `openclaw-loopback-proxy` 的模式，把 `172.29.0.1:11501/11502` 转发到本地 Qwen 和 Ollama。
- Worker 新增 `LLM_EXTRA_BODY`，用于关闭 Qwen 的思考模式。
- 向量检索：Ollama 目前只有对话模型。直连打通后，下一步可拉取中文嵌入模型（如 `bge-m3`）并经 11502 端口实现混合检索；在此之前仍用词面检索。

状态：代码与脚本已提交，新 Worker 已部署但仍走原网关（行为不变）。安装桥接服务和放行防火墙属于系统持久化变更，被本会话的权限策略拦截，需负责人执行（命令见部署说明「模型端点」），之后切换 `.env` 并验证。

## 需负责人在仓库外执行的事项

这些事项无法在仓库内完成，本轮不处理，保留在 `TASKS.md` 待办：

- 公网 TLS：需要证书与 frp/nginx 链路调整。
- 演示环境默认口令 `admin123`：线上账号操作。
- 群晖共用账号凭据轮换：DSM 改密后更新本机 `.env`。
- `main` 分支保护：仓库设置变更。
- 通知外发通道：需要提供企业微信/钉钉群机器人地址或 SMTP 账号。
- 公网演示环境的 `SELF_REGISTRATION` 取值（建议 `invite_only` 或 `closed`），以及是否设置 `METRICS_TOKEN` 开启指标。
- 是否删除对账发现的 2 个烟测遗留孤儿文件。
- 改为直连本地 Qwen：执行 `scripts/codex/install-llm-bridge.sh` 与一条 `sudo ufw` 放行，然后切换 Worker 的模型端点（见部署说明「模型端点」）。
