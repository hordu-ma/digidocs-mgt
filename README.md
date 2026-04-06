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
│   └── Harness Engineering 学习笔记.md
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
- [ADR-001 Go-Python 混合迁移方案](docs/adr/001-go-python-hybrid.md)

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
- 当前已支持最小正文抽取：`txt`、`md`、`csv`、`json`、`docx`。
- 假设/待确认：若 p14s 上 OpenClaw Gateway 未启用 OpenAI 兼容 HTTP 端点，需要先在网关配置中显式开启。
- 当前限制：`pdf`、图片 OCR、扫描件仍未接入真实正文抽取链路，因此这些文件的摘要结果仍可能退化为结构化业务上下文摘要。

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

部署前配置模板：

- 本地开发模板：[.env.example](/Users/liguo/code-repos/digidocs-mgt/.env.example)
- 生产部署模板：[.env.production.example](/Users/liguo/code-repos/digidocs-mgt/.env.production.example)
- 部署说明：[部署准备与运行说明](/Users/liguo/code-repos/digidocs-mgt/docs/部署准备与运行说明.md)

### Codex 协作环境

```bash
./scripts/codex/doctor.sh
./scripts/codex/check-doc-sync.sh
make status
make smoke
make verify
./scripts/codex/install-hooks.sh
./scripts/codex/install-project-skills.sh
```

项目级 Codex 协作资产：

- `ops/codex/skills/`：可执行 skill 包
- `ops/codex/skills/index.yaml`：技能索引
- `scripts/codex/install-project-skills.sh`：把项目技能安装到 `~/.codex/skills/`
- `scripts/codex/install-hooks.sh`：启用仓库内 `pre-commit` / `pre-push` hooks
- `scripts/codex/doctor.sh`：环境与协作资产体检
- `scripts/codex/check-doc-sync.sh`：检查 `README.md` / `TASKS.md` / `AGENTS.md` 的阶段与协作约束一致性
- `scripts/codex/report.sh`：输出当前分支、阶段、技能安装与 hooks 状态
- `scripts/codex/smoke-local.sh`：本地容器联调烟测骨架
- `Makefile`：统一 `make doctor`、`make verify`、`make check-doc-sync`
- `.githooks/`：本地提交前和推送前门禁
- `.github/workflows/verify.yml`：远端 GitHub Actions 验证门禁
- `docs/Harness Engineering 学习笔记.md`：本项目的 Harness Engineering 学习文档

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

当前阶段已进入 Go 主业务迁移与协作环境固化阶段：

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
- 已执行 `make smoke`，当前容器存在但宿主机 `18081/healthz` 不可达，脚本按非严格模式跳过
- 已完成 `AGENTS.md` 会话启动清单、状态账本同步规则与验证约束增强
- 已完成 [Harness Engineering 学习笔记](/home/liguoma/code-repos/digidocs-mgt/docs/Harness Engineering 学习笔记.md) 初版
- 已完成数据库种子数据创建与加载（`backend-go/sql/seed.sql`）
- 已完成 JWT 用户 ID 透传至所有审计事件写入（middleware → handlers → repositories）
- 已完成前端四个页面接入真实后端 API（仪表盘、文档列表、文档详情、交接单）
- 已修复宿主机 Docker 端口转发问题（Tailscale 路由表与 Docker 网桥冲突）
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
- 已完成 P0 文档/契约收口首轮修复（验证命令、README/AGENTS 迁移描述、前后端错误读取、转交流程、AI 问答契约）
- 已完成 P1 AI 持久化闭环首轮修复（`assistant_requests` 落库、Worker 回调幂等更新、`assistant_suggestions` 查询/确认/忽略真实接线）
- 已完成 P2 持久化任务消费与 AI 结果展示首轮修复（PostgreSQL 任务轮询、摘要结果回写、文档详情页 AI 建议展示）

详细任务状态持续维护在 [TASKS.md](/home/liguoma/code-repos/digidocs-mgt/TASKS.md)。
