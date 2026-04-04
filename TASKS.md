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

## 进行中

- 环境可运行化
  - 后端依赖安装确认
  - 数据库迁移执行
  - 后端服务启动验证
- Go-Python 混合迁移方案制定
  - 完成目标架构、目录结构草案
  - 完成分阶段迁移清单
- `backend-go/` 主业务基础设施补齐
  - 数据库连接与配置设计
  - API v1 模块路由骨架
  - 认证与统一错误结构
  - 文档、项目、团队空间占位接口迁移
  - 查询链路改为 `handler -> service -> repository`
  - 补齐 `postgres repository` 查询骨架
  - 增加 `postgres` 装配入口与基础错误映射
  - 增加 `projects/{id}/folders/tree` 查询链路
  - 增加 `assistant` 路由迁移占位与任务投递骨架
  - 增加 Worker 结果回写入口与简单鉴权
  - `auth/login` 按请求体解析
  - 增加版本上传接口、存储服务和审计落点
  - 迁入 `versions / flows / handovers / dashboard` 路由骨架
  - 为 `versions / flows / handovers` 补齐 memory repository 与 query/action service
  - 为 `versions / flows / handovers` 接入 postgres repository 读链
  - 为 `flow / handover / version upload` 接入 repository 写链骨架
  - 为 `version upload -> documents.current_version_id/current_status -> audit_events` 接入事务工作流骨架
  - 增加 `audit-events` Go 查询接口与 summary 聚合
  - 为 `flow` 动作接入 `documents.current_status/current_owner_id` 联动更新
  - 为 `flow / handover` 写路径接入 `audit_events` 落库
  - 为 `handovers/{id}/items` 接入持久化写链
  - 为 `handover complete` 接入选中文档责任人和状态联动
  - 为 `dashboard/overview`、`dashboard/recent-flows`、`dashboard/risk-documents` 接入真实聚合查询
  - 为 `flow / handover` 接口增加非法状态跳转校验和 `400 invalid_transition` 响应
- Docker 网络内真实数据库链路验证
  - 宿主机直连 PostgreSQL 仍异常
  - 改走 compose 网络内联调路径
  - 宿主机访问 `18081` 仍待单独排查
  - 已通过 Docker 网络内 Alembic 路径完成 PostgreSQL 初始 schema 初始化
  - 已验证 `dashboard/overview` 与 `audit-events` 在容器网络内可正常返回
- `backend-py-worker/` 职责收口
  - 明确 Worker 任务类型
  - 从旧 Python Web API 中抽离 AI 能力边界
- Codex 项目技能运行时接入
  - 待在本机执行 `./scripts/codex/install-project-skills.sh`
  - 待按实际使用反馈继续补 `backend-go` / `worker` / `verify` skill 内容
- 联调 smoke 验证
  - 待排查宿主机 `18081/healthz` 不可达问题

## 待办

- 跑通 Alembic 初始迁移
- 将占位 API 改为真实数据库读写
- 增加文档上传与版本管理服务层
- 增加流转状态机服务层
- 增加毕业交接服务层
- 增加更完整的审计事件过滤条件与统计聚合
- 增加 dashboard 聚合相关基础测试
- 补充数据库种子数据与真实业务链路联调
- 增加 OpenClaw 客户端真实调用
- 增加群晖 NAS 适配器
- 增加基础测试
- 增加更细粒度的 smoke test 和分层验证矩阵
- 将 smoke 验证进一步细化到关键业务闭环接口

## 更新规则

- 每次完成一个可感知阶段后更新本文件。
- 如果开发中断，下次继续前先检查本文件和 `README.md` 的当前开发进度。
