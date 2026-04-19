# DigiDocs Mgt

面向课题组的文档资产管理与智能助理平台。

本项目用于构建一套面向科研团队的文档资产管理系统，核心关注点是文件级版本管理、显式流转、毕业交接、负责人总览、审计留痕，以及部署在 DGX Spark 上的 OpenClaw 深度助手能力。

## 项目目标

- 把课题组文档从个人电脑中沉淀为团队资产
- 用固定状态机管理文档责任和流转
- 支持毕业交接闭环
- 为负责人提供项目管理总览而不是纯审计事件堆叠
- 与群晖 NAS、DGX Spark / OpenClaw 做明确的系统级对接预留

## 当前仓库结构

```text
.
├── AGENTS.md          # 多轮协作与代理执行约束
├── README.md          # 项目入口与阶段快照
├── TASKS.md           # 执行任务账本
├── Makefile
├── .github/
├── .githooks/
├── docs/
│   ├── INDEX.md
│   ├── 项目定义与技术架构.md
│   ├── 数据库设计.md
│   ├── API设计.md
│   ├── 异步任务消息契约.md
│   ├── adr/
│   │   └── 001-go-python-hybrid.md
├── backend-go/
├── backend-py-worker/
├── docker-compose.yml
├── frontend/
├── ops/
└── scripts/
```

## 核心文档

详见 [docs/INDEX.md](docs/INDEX.md)，快速入口：

- [项目定义与技术架构](docs/项目定义与技术架构.md)
- [数据库设计](docs/数据库设计.md)
- [API 设计](docs/API设计.md)
- [异步任务消息契约](docs/异步任务消息契约.md)
- [部署准备与运行说明](docs/部署准备与运行说明.md)
- [backend-go 核心源码学习导读](docs/backend-go核心源码学习导读.md)
- [ADR-001 Go-Python 混合迁移方案](docs/adr/001-go-python-hybrid.md)
- [.github 协作资产索引](.github/INDEX.md)

## 技术栈

### 后端

- Go 1.25（主业务后端，标准库 net/http + database/sql）
- Python 3.12（Worker，承接 AI 与文档处理任务）
- PostgreSQL 17
- Go 内置 SQL migrations（`backend-go/migrations/`）
- uv（Python 依赖管理）

### 前端

- Vue 3
- TypeScript
- Vite
- Pinia
- Vue Router
- Element Plus

### 外部系统

- Synology DS925+ DSM / File Station Web API（文件存储）
- DGX Spark / p14s 上部署的 OpenClaw Gateway（AI 能力层，当前按 OpenAI 兼容 HTTP 接口接入）

## 协作规范

- 先维护文档和数据模型，再推进实现。
- 所有新增接口、表结构和跨系统集成，都应先对照三份设计文档。
- AI 建议与主业务事实必须分离，禁止让 AI 直接改写主状态。
- 优先走薄业务层自研，不重复造通用底座。
- 涉及群晖 NAS 和 OpenClaw 的调用，必须统一经由适配层/客户端模块，不允许散落在业务代码中。
- `TASKS.md` 作为执行账本，`README.md` 作为阶段快照；两者阶段判断必须保持一致。
- 项目级 Codex 可执行 skills 统一维护在 `ops/codex/skills/`。
- 新机器或新会话接力开发时，优先运行 `./scripts/codex/doctor.sh` 做环境体检。
- 若需让 Codex 在运行时发现项目技能，执行 `./scripts/codex/install-project-skills.sh`。
- GitHub 协作资产默认从 `.github/INDEX.md` 进入，不直接扫描整个 `.github/`。

## 近期开发顺序

1. 完成仓库协作资产与运行骨架
2. 安装后端和前端依赖
3. 跑通数据库迁移
4. 将占位 API 替换为真实数据库读写
5. 打通文档上传、版本、流转、交接、总览的最小闭环
6. 接入 OpenClaw 摘要链路
7. 接入群晖 NAS 存储适配器

## OpenClaw 当前实现

- Python Worker 当前通过 `POST /v1/chat/completions` 调用 OpenClaw Gateway。
- Worker 先从 Go 主后端读取受控内部上下文，再把结构化上下文提交给 OpenClaw。
- 当前已落地的内部上下文范围：项目总览、文档详情/版本/流转、交接单详情。
- 当前已支持正文抽取：`txt`、`md`、`csv`、`json`、`docx`、`pdf`。
- 当前已支持图片 / 扫描 PDF OCR，但要求 Worker 主机存在 `tesseract`；若缺失会返回明确错误而非伪成功。
- `openclaw status` 返回 Gateway 正常，不等于 OpenAI 兼容 HTTP 端点已启用；部署前仍需单独验证 `GET /v1/models`。
- 假设/待确认：若 p14s 上 OpenClaw Gateway 未启用 OpenAI 兼容 HTTP 端点，需要先在网关配置中显式开启。
- 当前已补 Assistant 问答历史列表与筛选接口，前端助手页可直接回看历史请求并查看模型、OpenClaw 响应 ID 与耗时。
- 当前限制：图片 / 扫描 PDF 的 OCR 依赖 Worker 主机安装 `tesseract`；若未安装，AI 摘要仍会退化为结构化业务上下文摘要。

## 开发命令约定

### 后端

```bash
cd backend-go
go run ./cmd/api
```

### Go 主后端骨架

```bash
cd backend-go
go run ./cmd/api
```

### Python Worker 骨架

```bash
cd backend-py-worker
uv sync
uv run python -m app.main
```

### 前端

```bash
cd frontend
npm install
npm run dev
```

### 编排

```bash
docker compose up -d postgres
```

启动容器内联调路径：

```bash
docker compose --profile app up -d backend-go backend-py-worker frontend
```

说明：

- 上述 `docker compose` 主要用于本地单机联调；
- 当前目标部署分层为：`DGX Spark / P14s` 承载应用层与 OpenClaw，`群晖 DS925+` 承载数据库层（`Container Manager` 中的 PostgreSQL 容器）和文件层（DSM / File Station 原生服务）。

部署前配置模板：

- 本地开发模板：[.env.example](.env.example)
- 生产部署模板：[.env.production.example](.env.production.example)
- 部署说明：[部署准备与运行说明](docs/部署准备与运行说明.md)
- 同一局域网用户联调访问示例：`http://192.168.1.142:18080`，实际地址以宿主机 `hostname -I` 输出的局域网 IP 为准。
- RustDesk 或本机浏览器访问 `18080/18081` 超时时，按部署说明中的“前端 18080 / 后端 18081 访问超时排查”处理。

### Codex 协作环境

```bash
./scripts/codex/doctor.sh
./scripts/codex/check-doc-sync.sh
make status
make smoke
make verify
./scripts/codex/install-hooks.sh
./scripts/codex/install-persistent-routing.sh
./scripts/codex/install-project-skills.sh
```

项目级 Codex 协作资产：

- `ops/codex/skills/`：可执行 skill 包
- `ops/codex/skills/index.yaml`：技能索引
- `scripts/codex/install-project-skills.sh`：把项目技能安装到 `~/.codex/skills/`
- `scripts/codex/install-hooks.sh`：启用仓库内 `pre-commit` / `pre-push` hooks
- `scripts/codex/install-persistent-routing.sh`：为 Linux 宿主机安装 `tailscaled` 持久化路由修正
- `scripts/codex/doctor.sh`：环境与协作资产体检
- `scripts/codex/check-doc-sync.sh`：检查 `README.md` / `TASKS.md` / `AGENTS.md` 的阶段与协作约束一致性
- `scripts/codex/report.sh`：输出当前分支、阶段、技能安装与 hooks 状态
- `scripts/codex/smoke-local.sh`：本地容器联调烟测骨架
- `Makefile`：统一 `make doctor`、`make verify`、`make check-doc-sync`
- `.githooks/`：本地提交前和推送前门禁
- `.github/INDEX.md`：GitHub 协作资产入口
- `.github/workflows/verify.yml`：远端 GitHub Actions 验证门禁

## 跨机器接力开发

在另一台机器拉取仓库后，需要重新安装本地依赖和运行环境，不能直接复用其他机器的 `.venv`、`node_modules` 或本地容器状态。

建议执行：

```bash
git pull
cd backend-go
# configure .env as needed
cd ../frontend
npm install
cd ..
docker compose up -d postgres
```

## 开发进度

当前阶段已进入 交付闭环收口与部署验收准备阶段：

- 已完成 `versions` 事务上传工作流
- 已完成 `audit-events` Go 查询接口与 summary 聚合骨架
- 已完成 `flow / handover` 写链的首轮事务化与审计联动
- 已完成 `dashboard` 三个聚合接口的 Go 真实查询接入
- 已完成 `flow / handover` 非法状态跳转校验
- 已通过 Go 自动迁移完成 PostgreSQL 初始 schema 初始化
- 已完成 `dashboard/overview` 与 `audit-events` 容器网络烟测
- 已完成项目级 Codex skills、技能索引、安装脚本与 doctor 体检脚本初版
- 已完成 `make verify` 与 `check-doc-sync.sh` 统一验证入口
- 已实际跑通 `make verify`，当前通过 Go 测试、Python Worker 测试与前端构建验证
- 已完成第三阶段自动化门禁骨架：本地 hooks、GitHub Actions、状态报告与联调 smoke 脚本
- 已启用仓库本地 `.githooks`，并已跑通 `make status`
- 已跑通 `STRICT_SMOKE=1 make smoke`，当前覆盖登录、文档查询、Dashboard、交接、审计、版本上传/下载/预览、Assistant 完成态与群晖前置验收
- 已完成 `AGENTS.md` 会话启动清单、状态账本同步规则与验证约束增强
- 已完成 Harness Engineering 学习笔记初版（后续已迁移至 Obsidian）
- 已完成数据库种子数据创建与加载（`backend-go/sql/seed.sql`）
- 已新增 `wuhao-ai` 测试项目 fixture（`backend-go/sql/wuhao_ai_seed.sql`、`scripts/codex/load-wuhao-ai-fixtures.sh`），覆盖五好爱学 11 份文档元数据、版本、流转与审计事件
- 已完成测试用户实名与拼音账号调整，并新增 `users.wechat` 字段；用户列表接口已返回 `username / email / phone / wechat / status`
- 已完成首期权限矩阵落地：新增 `project_members` 项目成员关系，后端写操作按全局角色、项目角色与文档责任人执行权限校验
- 已完成 JWT 用户 ID 透传至所有审计事件写入（middleware → handlers → repositories）
- 已完成前端四个页面接入真实后端 API（仪表盘、文档列表、文档详情、交接单）
- 已修复宿主机 Docker 端口转发问题（Tailscale 路由表与 Docker 网桥冲突）
- 已完成前端商业化产品体验升级（二期）
  - 统一全局设计 token、Element Plus 皮肤与导航壳层
  - 新增全局搜索/跳转命令面板与快捷工作入口
  - 重构 Dashboard、文档工作台、文档档案页、交接工作台与助手工作区
  - 补强数据资产页、系统管理页与个人信息页的一致性体验
- 已完成前端移动端壳层与响应式基础收口
  - `AppLayout` 改为桌面侧栏 + 移动端抽屉导航，手机端顶栏信息密度明显下降
  - `GlobalCommandDialog` 和全局 Dialog 宽度改为适配手机视口
  - 当前预览已同步到运行中的 `digidocs-frontend` 容器，可直接通过 `18080` 查看效果
- 已完善客户介绍文档展示素材，补充系统架构图与系统截图，便于客户侧方案说明与 PDF 输出准备
- 已通过宿主机验证所有 API 端点：login、auth/me、documents、dashboard、versions、flows、handovers
- 已完成前后端联调验证（Vite proxy → backend-go → PostgreSQL 全链路通过）
- 已修复 `DocumentDetail` API 返回 `current_owner` 含 display_name（与 `DocumentListItem` 对齐）
- 已清理 TASKS.md，将已完成的大量"进行中"子项归档至"已完成"
- 已完成 Go 后端核心业务服务层单元测试（auth/action/dashboard，共 13 个用例）
- 已完成 OpenClaw Gateway 真实 HTTP 对接与 Worker 内部上下文桥接
- 已完成 Assistant 问答结果查询闭环与最小正文抽取链路
- 已完成前端 Element Plus 按需引入（JS bundle 从 1,041 KB 降至 470 KB）
- 已完成 Python Worker 真实队列消费链路（Go/HTTP 轮询 → 处理 → 回写）
- 已完成前端写操作 UI（文档流转操作、AI 助手问答提交、交接单创建）
- 已完成“代码”模块第一版
  - 新增代码仓库配置、Git remote 地址、push token 和 push 事件记录
  - 后端新增 `code_repositories` / `code_push_events` 迁移、REST API 与 Git smart HTTP 接收入口
  - 前端新增独立“代码”导航页，位于“数据”之后、“助手”之前，展示推送命令、同步目标和事件历史
  - 已移除“数据”模块中的“代码”文件小类别，避免与独立代码模块混淆
- 已完成 P0 文档/契约收口首轮修复（验证命令、README/AGENTS 迁移描述、前后端错误读取、转交流程、AI 问答契约）
- 已完成群晖前置验收 smoke 收口：`make smoke` 现可选直连 DSM / File Station API 做登录、上传、下载、共享链接与清理
- 已补群晖 HTTPS 受控兼容开关：自签名证书环境可通过 `SYNOLOGY_INSECURE_SKIP_VERIFY=true` 临时接入 Go 服务
- 已完成版本文件链路强化验证：`make smoke` 已覆盖现有文档的版本上传、下载与预览闭环
- 已修正 smoke 对 `assistant.ask` 的默认轮询容忍度，避免 OpenClaw 正常但响应稍慢时被误判失败
- 已补 Synology provider 分层契约测试与 memory 存储版本仓储联动，上传后再查询/下载/预览不再依赖静态占位返回
- 已完成 P1 AI 持久化闭环首轮修复（`assistant_requests` 落库、Worker 回调幂等更新、`assistant_suggestions` 查询/确认/忽略真实接线）
- 已完成 P2 持久化任务消费与 AI 结果展示首轮修复（PostgreSQL 任务轮询、摘要结果回写、文档详情页 AI 建议展示）
- 已补 `.github/INDEX.md` 与 GitHub 协作资产入口校验
- 已新增 `docs/backend-go核心源码学习导读.md`，面向 Go 初学者解释 backend-go 核心文件中的典型函数与关键代码块
- 已完成前端 P0/P1 产品化体验改版首轮：统一视觉 token、应用外壳、状态/文件徽标，并重构负责人总览、文档资产库和文档档案页
- 已完成前端 P2/P3 产品化体验改版首轮：毕业交接改为任务卡与流程工作台，OpenClaw 助手改为成熟对话工作区，并通过路由懒加载消除前端构建 chunk 过大提示
- 已完成 p14s/Linux compose 首轮部署适配（Worker 宿主机 OpenClaw 访问改用 `host-gateway`）
- 已修复 `backend-go` 运行镜像未携带 `migrations/` 导致容器内自动迁移失效的问题
- 已完成 Assistant 问答历史列表 / 筛选与 Markdown 结果展示首轮接线
- 已完成 PDF 正文抽取与 AI 可观测性首轮补强（模型、上游响应 ID、耗时、Worker 日志）
- 已将 `make smoke` 扩展到 `assistant.ask -> completed` 的 AI 闭环验证
- 已修复文档详情页 AI 建议展示口径，重开页面后不再把已处理建议重复展示为待处理提示
- 已完成文档与任务账本首轮纠偏，统一 OCR / Assistant 历史筛选的完成状态，并新增 `/api/v1/users` 只读查询契约
- 已完成前端 P1 交付收口：文档创建改为团队空间/课题/目录/责任人选择器，文档转交改为成员选择，交接页补齐创建后管理、清单编辑、确认、完成、取消闭环
- 已完成 AI 记忆架构（一期）最小闭环：新增 Assistant 会话/消息持久化、`conversation_id` 兼容问答接口、Go 侧显式记忆装配（最近会话/历史回答/已确认建议）
- 已完成 Assistant 前端会话化改造：助手页改为“会话列表 + 消息流 + 追问”模式，并展示 AI 响应的记忆来源提示
- 已完成项目级 Codex skills 运行时接入：本机已安装 `ops/codex/skills/` 到 `~/.codex/skills/`，并补齐 `backend-go` / `worker` / `verify` skill 执行说明
- 已完成 OpenClaw skill 复用策略（一期）：Worker 新增白名单 `skill_registry` / `skill_adapter`，AI 结果与会话元数据可追踪 `skill_name`、`skill_version`、`source_scope`、`memory_sources`
- 已完成 2026-04-13 第二轮部署验收排查：`SynologyStorageProvider` 与 `make smoke` 已兼容 `SYNO.API.Info -> entry.cgi`，`backend-go` 运行镜像已确认携带 `003_assistant_conversations.sql`，数据库 `schema_migrations` 已落到 `003`
- 已完成 `assistant.ask` 500 修复：PostgreSQL `assistant_suggestions.status / suggestion_type` enum 过滤已改为 `::text` 比较，`POST /assistant/ask` 与 `GET /api/v1/assistant/requests/{id}` 已恢复可排队、可查询
- 已完成 smoke 脚本收口：`.env` 加载不再覆盖命令行环境变量，Synology preflight 改为宿主机直连，`true/false` 布尔环境变量可直接识别
- 已确认 `STRICT_SMOKE=1 RUN_SYNOLOGY_PREFLIGHT=1 make smoke` 当前结果：核心业务接口通过，DSM preflight 通过，`assistant.ask` 可入队；剩余阻塞为 `documents/{id}/versions` 上传仍返回 500，以及 Assistant 请求在当前机器上停留 `pending`
- 已定位当前机器的主机级网络阻塞：`ip route show table 52` 仍把 `172.17.0.0/16`、`172.18.0.0/16` 指向 `tailscale0`，导致 Docker 容器既无法直连群晖 Tailscale 地址，也无法访问宿主机 `host.docker.internal`
- 已完成 2026-04-13 第三轮部署验收修复：宿主机 `table 52` 已对 Docker / 局域网网段改为 `throw`，`backend-go / backend-py-worker` 已重建到正确环境，`assistant.ask -> completed` 与版本上传/下载/预览严格 smoke 已恢复通过
- 已修复 Synology provider 目录幂等问题：版本上传前父目录已存在时，不再因 DSM `CreateFolder` 返回 400 而中断
- 已修复 `make smoke` 的 `healthz` 误报逻辑，宿主机直连成功时不再重复报 unreachable
- 已重建 `frontend` 运行容器到当前仓库版本，并完成一轮前端无头联调：已覆盖总览、文档列表、文档详情、交接页、助手页与问答提交主路径
- 已记录最终部署口径：TLS 终止在应用层反向代理，前端/后端容器仅走内网转发；群晖 `5001/5432` 仅对应用层主机放通
- 已完成宿主机持久化运维配置安装与验证：`install-persistent-routing.sh`、`digidocs-route-fix.sh` 与 `tailscaled` drop-in 已在当前机器落地，`systemctl restart tailscaled` 后 `table 52` 仍保持 `throw 172.17/172.18/192.168.1`
- 已补充 RustDesk / 本机浏览器访问 `18080/18081` 超时的排查与临时修复说明，复用现有 `digidocs-route-fix.sh` 路由修正脚本
- 已补充登录后个人信息维护能力：当前用户可自助更新显示姓名、手机号、微信号和邮箱，账号、角色、状态与权限保持只读
- 已完成 Assistant 助手页面产品化改造：范围选择改为项目/文档下拉选择器，会话列表显示人类可读范围标签，消息流隐藏技术字段并折叠技术详情，记忆来源改为中文友好文案，时间戳改为相对时间
- 已完成 AI 思考中交互优化（三点呼吸动画占位气泡 + 按钮状态）与全局空状态轻量化替换（AssistantView、HandoversView）
- 已完成总览页近期流转操作中文化、文档页按课题分组显示、数据库测试数据清理
- 已完成管理员功能模块：后端 9 个 admin API 端点（团队空间/项目/用户/成员 CRUD），RequireAdmin 中间件，前端四标签页管理界面与侧边栏角色感知入口
- 已修复管理页面布局（AppLayout 包裹）与总览页近期流转空状态中文化
- 已完成权限过滤：非 admin 用户仅可见其所属项目及关联空间；文档页空项目也显示占位卡片
- 已完成 AI 助手文档内容读取能力修复：`assistant.ask` 自动内联抽取文档正文（P0）、文档上传自动触发文本抽取任务（P1）、Worker 新增 PDF 正文抽取支持（`poppler-utils`）
- 已完成 .xlsx/.pptx/.doc 文档正文抽取支持，Worker 新增 `antiword` 依赖，全部 15 份文档抽取成功
- 已完成助手会话安全隔离：修复 `ListConversations` 未使用认证用户过滤的安全漏洞，每个用户只能看到自己的会话
- 已完成会话归档管理：新增归档/恢复 API 与前端交互（归档按钮、归档列表切换、恢复操作）
- 已完成文档摘要轮询显示：生成摘要后启动轮询并显示动画等待提示，完成后自动刷新建议列表
- 已修复文档摘要回调失败（P0 阻塞性 Bug）：Worker 成功调用 OpenClaw 但回调返回 500，根因是 OpenClaw 返回的 `suggestion_type` 不在 PostgreSQL 枚举范围内，新增 `normalizeSuggestionType()` 验证并回退到默认值；增强 Worker 与 Go 端的回调错误日志
- 已完成全局表格空状态中文化：文档详情（版本/流转）、管理页面（空间/项目/用户/成员）的空表格提示从 "No Data" 改为中文友好文案
- 已修复流转状态机细节缺陷：`unarchive` 现从 `archived` 正确恢复到 `finalized`，不再错误回退到 `in_progress`
- 已完成文档流转与毕业交接接口实测：验证 `finalize / archive / unarchive / transfer / accept-transfer` 与 `handover create / items / confirm / complete` 全链路通过
- 已完成**数据资产模块**首期实现（T1–T5）：新增 `data_folders` / `data_assets` / `graduation_handover_data_items` 数据库表，后端存储层、权限层、domain、repository、service、handler、路由全栈落地；前端新增"数据"页面（位于文档与交接之间），含文件列表、课题/文件夹过滤、XHR 大文件上传进度条、文件夹管理；已通过 `go test ./...` + 前端构建验证
- 当前下一步聚焦：把当前 Linux 单机运维方案复制到目标正式主机，并复验一次正式环境联调
- 已完成**数据资产模块 T6/T7 与 UI 收口**：
  - T6：交接详情弹窗新增文档/数据资产双 Tab，支持数据资产交接清单编辑
  - T7：`docs/数据库设计.md` 与 `docs/API设计.md` 补充数据资产模块全量内容
  - DataView 页面 UI 全面产品化：改用 `page-shell / page-header / page-card / asset-workspace` 布局，左侧课题 Rail、文件类型徽标色系、`--dd-*` 设计变量、自定义行布局替代 el-table，消除"样品感"
  - DocumentDetailView 补充下载按钮：hero-actions 新增"下载当前版本"，版本历史每行新增下载图标按钮，使用 blob 流方式下载，复用已有 `GET /api/v1/versions/{versionID}/download` 端点
- 已完成**数据页面二次 UI 优化**：
  - 文件夹导航移入左侧 Rail（分隔线 + 缩进树形 + hover 删除按钮），取消原横向 chip strip
  - 上传对话框文件选择器改为自定义拖拽上传区（拖放/点击二合一，选中后显示文件信息）
  - 修复切换团队空间残留、clearUploadFile 名称残留、dragleave 闪烁等细节 bug
- 已完成**文档页面交互细节修复**：
  - DocumentDetailView 按钮顺序调整：流转 → 档案编辑 → 下载当前版本 → 上传新版本；"编辑信息"更名为"档案编辑"
  - 修复归档后文档消失问题：DocumentsView 新增"显示已归档"开关，关闭时隐藏归档文档，开启后归档文档以灰色半透明 + 删除线标注
  - 修复管理交接弹窗加载失败：`HandoversView.loadProjectDataAssets` 响应解析从 `res.data?.data` 修正为 `res.data?.data?.items`，消除 `TypeError: assets.map is not a function`

详细任务状态持续维护在 [TASKS.md](TASKS.md)。
