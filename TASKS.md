# TASKS

## 当前阶段

Go 主业务迁移与协作环境固化阶段

## 已完成

- 完成项目定义文档正式版
- 完成数据库设计文档正式版
- 完成 API 设计文档正式版
- 初始化后端骨架
- 初始化前端骨架
- 初始化协作文件：`README.md`、`AGENTS.md`、`SKILLS/`
- 完成 `backend-go/` 最小骨架落库
  - 服务入口与配置加载
  - 基础路由与健康检查
  - 统一 JSON 响应结构
  - 已完成 `go test ./...` 编译验证
- 完成 `backend-py-worker/` 最小骨架落库
  - 配置加载
  - OpenClaw / Callback 客户端骨架
  - 任务消息结构
  - Worker 调度器骨架
  - 已完成 `.venv/bin/python -m pytest -q` 验证
- 完成 Python Worker 基础测试样例
  - 覆盖 `assistant.ask` 调度路径
- 完成 Go/Python 异步任务契约初稿
  - 统一任务类型
  - 统一任务消息与结果消息结构
  - 约定 Worker 结果内部回写入口
- 完成 Go 工具链与 PostgreSQL 驱动接入
  - 安装 `go` / `gofmt`
  - 接入 `github.com/lib/pq`
  - 已完成 `go version` 与 `go test ./...` 验证
- 完成第三类基础阻塞的最小落点
  - JWT token 生成与解析
  - 文件上传服务与内存存储实现
  - 审计写入服务骨架
  - 队列发布接口与内存实现
- 完成容器化运行路径初稿
  - `backend-go` Dockerfile
  - `backend-py-worker` Dockerfile
  - compose profile `app`
  - `backend-go` / `backend-py-worker` 镜像构建通过
  - compose 网络内部 `backend-go:8080/healthz` 验证通过
- 完成项目级 Codex 协作资产首轮固化
  - 增加 `ops/codex/skills/` 真实 skill 包
  - 增加 `ops/codex/skills/index.yaml` 技能索引
  - 增加 `scripts/codex/install-project-skills.sh`
  - 增加 `scripts/codex/doctor.sh`
  - 增强 `AGENTS.md` 的会话启动清单、状态同步规则和验证约束
  - 新增 `docs/Harness Engineering 学习笔记.md`
- 完成第二阶段统一验证入口
  - 增加 `Makefile`
  - 增加 `scripts/codex/check-doc-sync.sh`
  - `doctor.sh` 接入文档一致性检查
  - 增加 `make verify` 聚合验证入口
  - 已实际跑通 `make verify`
- 完成第三阶段自动化门禁骨架
  - 增加 `.github/workflows/verify.yml`
  - 增加 `.githooks/pre-commit` 与 `.githooks/pre-push`
  - 增加 `scripts/codex/install-hooks.sh`
  - 增加 `scripts/codex/report.sh`
  - 增加 `scripts/codex/smoke-local.sh`
  - `Makefile` 增加 `status`、`smoke`、`install-hooks`
  - 已启用 `.githooks`
  - 已跑通 `make status`
  - 已执行 `make smoke`，当前以非严格模式跳过宿主机 `18081/healthz`
- 完成代码审查与安全修复（一期）
  - JWT 改为真实 HMAC-SHA256 签名，淘汰伪造 base64 token
  - login 接入 bcrypt 密码校验与独立 AuthService
  - 全业务路由接入 JWT 鉴权中间件
  - 前端登录接入真实 API，增加 localStorage 持久化与路由守卫
  - 修复 `transfer` 动作状态映射错误（应为 `pending_handover`）
  - 补齐 memory ActionRepository 状态机跳转校验逻辑
  - version_no 自增改为 `SELECT ... FOR UPDATE` 防并发写冲突
  - 修复 audit-events 分页参数无下界约束
  - 修复 handovers 错误判断改用 `errors.Is`
  - 修复 `newID()` 随机数失败时返回零值 UUID（改为 panic）
  - 删除与标准库冲突的自定义 `max()` 函数
  - 访问日志补充 HTTP 响应状态码记录
- 完成数据库种子数据与 PostgreSQL 端到端验证
  - 创建 `backend-go/sql/seed.sql`（5 用户、2 团队空间、3 项目、5 文件夹、7 文档、8 版本、4 流转记录、9 审计事件）
  - 已在 Docker 容器内加载种子数据并验证
- 完成 JWT 用户 ID 透传至审计事件写入
  - `auth.go` 中间件解析 Claims 注入请求上下文
  - `handlers/flows.go`、`handlers/handovers.go`、`handlers/versions.go` 从上下文提取 ActorID
  - `postgres/action_repository.go`、`postgres/version_repository.go`、`postgres/version_workflow.go` 使用真实 ActorID 写入 audit_events / flow_records / handovers
- 完成前端页面接入真实后端 API
  - `DashboardView.vue` 接入 `/dashboard/overview`、`/dashboard/recent-flows`、`/dashboard/risk-documents`
  - `DocumentsView.vue` 接入 `/documents`，增加分页和搜索
  - `DocumentDetailView.vue` 接入 `/documents/{id}`、`/documents/{id}/versions`、`/documents/{id}/flows`
  - `HandoversView.vue` 接入 `/handovers`
- 完成宿主机 Docker 端口转发问题排查与修复
  - 根因：Tailscale 路由表 52 与 Docker 网桥 172.18.0.0/16 冲突
  - 修复：添加高优先级 ip rule 确保 Docker 子网走 main 路由表
  - UFW 增加 Docker 网桥接口入站规则
  - 已验证所有 API 端点从宿主机可达
- 完成环境可运行化
  - 后端依赖安装、数据库迁移执行、后端服务启动验证
- 完成 Go-Python 混合迁移方案制定
  - 目标架构、目录结构草案与分阶段迁移清单
- 完成 `backend-go/` 主业务基础设施全量补齐
  - 数据库连接与配置、API v1 路由骨架、认证与统一错误结构
  - 文档/项目/团队空间接口迁移、`handler → service → repository` 查询链路
  - postgres repository 查询/写链骨架、装配入口与错误映射
  - `projects/{id}/folders/tree`、`assistant` 路由迁移、Worker 回写入口
  - `auth/login` 请求体解析、版本上传接口与存储服务
  - `versions / flows / handovers / dashboard` 路由骨架与 memory/postgres repository
  - 事务工作流骨架（version upload → documents 联动 → audit_events）
  - `audit-events` 查询接口与 summary 聚合
  - `flow` 动作 documents 状态/责任人联动、`flow / handover` 审计落库
  - `handovers/{id}/items` 持久化写链、`handover complete` 文档联动
  - `dashboard` 三个聚合接口真实查询
  - `flow / handover` 非法状态跳转校验
- 完成 Docker 网络内真实数据库链路验证
  - Alembic 初始 schema 初始化
  - 容器内 `dashboard/overview` 与 `audit-events` 烟测通过
  - 宿主机 Docker 端口转发修复（Tailscale 路由冲突）
  - `18081/healthz` 宿主机可达验证通过
- 完成联调 smoke 验证
  - `make smoke` 宿主机 `18081/healthz` 已修复通过
- 完成前后端联调验证
  - Vite dev server 代理至后端，所有页面 API 端点验证通过
  - 修复 `DocumentDetail` API 返回 `current_owner`（含 display_name）替代原 `current_owner_id`
- 完成 Go 后端核心业务服务层单元测试
  - `auth_service_test.go`：登录成功、用户不存在、密码错误、仓库层错误传播（4 用例）
  - `action_service_test.go`：ApplyFlow/CreateHandover/UpdateHandoverItems/ApplyHandover 委托验证与错误传播（5 用例）
  - `dashboard_query_service_test.go`：Overview/RecentFlows/RiskDocuments 委托验证与错误传播（4 用例）

## 进行中

- `backend-py-worker/` 职责收口
  - 明确 Worker 任务类型
  - 从旧 Python Web API 中抽离 AI 能力边界
- Codex 项目技能运行时接入
  - 待在本机执行 `./scripts/codex/install-project-skills.sh`
  - 待按实际使用反馈继续补 `backend-go` / `worker` / `verify` skill 内容

## 待办

- ~~跑通 Alembic 初始迁移~~ ✅ 已完成
- ~~将占位 API 改为真实数据库读写~~ ✅ 已完成
- ~~前端页面接入真实后端 API~~ ✅ 已完成
- **Python Worker 实现真实队列消费**，`run_forever()` 当前为空循环占位，需对接 Redis/内存队列并驱动真实任务处理
- 增加文档上传与版本管理服务层
- 增加流转状态机服务层
- 增加毕业交接服务层
- ~~将 JWT 用户 ID 透传至所有审计事件写入~~ ✅ 已完成
- **Python CallbackClient / OpenClawClient 实现真实 HTTP 调用**，当前两个客户端均返回硬编码 dict，无任何网络请求
- 增加更完整的审计事件过滤条件与统计聚合
- ~~增加 dashboard 聚合相关基础测试~~ ✅ 已完成
- ~~补充数据库种子数据与真实业务链路联调~~ ✅ 已完成
- 增加 OpenClaw 客户端真实调用
- 增加群晖 NAS 适配器
- ~~增加基础测试~~ ✅ 已完成
- 增加更细粒度的 smoke test 和分层验证矩阵
- 将 smoke 验证进一步细化到关键业务闭环接口
- **前端 Element Plus 改为按需引入**以减小 bundle 体积（当前打包产物 >500 KB，需配置 `unplugin-vue-components` + `unplugin-auto-import`）

## 更新规则

- 每次完成一个可感知阶段后更新本文件。
- 如果开发中断，下次继续前先检查本文件和 `README.md` 的当前开发进度。
