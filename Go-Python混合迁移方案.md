# DigiDocs Mgt Go-Python 混合迁移方案

## 1. 目标

本方案用于指导 DigiDocs Mgt 从当前 `Python 单体后端骨架` 迁移到 `Go 主业务后端 + Python AI/文档处理 Worker` 的混合架构。

迁移目标：

- 保留现有数据库设计与 API 契约方向不变；
- 将主业务事实写入、事务、状态机、存储适配迁移至 Go；
- 将 AI 生成、文档提取、摘要与建议类能力保留在 Python；
- 确保“AI 不直接写主账本”的设计原则不被破坏；
- 在尽量不打断前端联调的前提下完成后端替换。

---

## 2. 总体原则

- Go 负责正式业务事实。
- Python 负责 AI 附属结果和文档处理任务。
- PostgreSQL 仍为唯一主账本。
- 不改变既有枚举、角色、状态定义，除非先更新设计文档。
- 不在业务代码中散落群晖 DSM / File Station 请求，统一走存储适配器。
- 不允许 Python Worker 直接改写 `documents.current_status`、`documents.current_owner_id` 等主业务字段。

---

## 3. 推荐目录结构草案

建议在仓库内逐步演进为以下结构：

```text
.
├── AGENTS.md
├── API设计.md
├── Go-Python混合迁移方案.md
├── README.md
├── SKILLS/
├── TASKS.md
├── backend-go/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   ├── configs/
│   ├── migrations/
│   ├── internal/
│   │   ├── app/
│   │   ├── config/
│   │   ├── transport/
│   │   │   └── http/
│   │   │       ├── handlers/
│   │   │       ├── middleware/
│   │   │       └── router/
│   │   ├── domain/
│   │   │   ├── auth/
│   │   │   ├── audit/
│   │   │   ├── document/
│   │   │   ├── flow/
│   │   │   ├── handover/
│   │   │   ├── project/
│   │   │   ├── teamspace/
│   │   │   └── dashboard/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── storage/
│   │   │   ├── minio/
│   │   │   └── synology/
│   │   ├── queue/
│   │   └── shared/
│   ├── sql/
│   ├── scripts/
│   ├── tests/
│   ├── go.mod
│   └── README.md
├── backend-py-worker/
│   ├── app/
│   │   ├── core/
│   │   ├── tasks/
│   │   ├── clients/
│   │   │   ├── openclaw_client.py
│   │   │   └── callback_client.py
│   │   ├── services/
│   │   ├── models/
│   │   └── schemas/
│   ├── tests/
│   ├── pyproject.toml
│   └── README.md
├── frontend/
├── docker-compose.yml
├── 数据库设计.md
└── 项目定义与技术架构.md
```

---

## 4. 各目录职责

### 4.1 `backend-go/`

职责：

- 对外提供主 REST API；
- 承担认证、权限、文档、版本、流转、交接、审计、负责人总览；
- 负责 PostgreSQL 事务；
- 负责 MinIO / 群晖 NAS 存储适配；
- 负责向异步队列投递 AI / 文档处理任务。

细分职责：

- `cmd/api/`
  - Go 服务入口。
- `internal/transport/http/`
  - 路由、处理器、中间件、请求响应封装。
- `internal/domain/`
  - 业务聚合边界与领域对象。
- `internal/repository/`
  - 面向 PostgreSQL 的数据访问。
- `internal/service/`
  - 事务编排、状态机、审计写入。
- `internal/storage/`
  - 统一对象存储和群晖适配器。
- `internal/queue/`
  - 向 Python Worker 投递任务。
- `migrations/`
  - Go 侧迁移脚本。
- `sql/`
  - `sqlc` 查询定义，若采用 `sqlc`。

### 4.2 `backend-py-worker/`

职责：

- 消费异步任务；
- 调用 OpenClaw；
- 执行文档摘要、交接摘要、标签建议、归档建议等 AI 任务；
- 执行文档文本提取、预处理、后续 OCR 等任务；
- 将结果以附属信息回写数据库或通过受控接口写回 Go 后端。

边界限制：

- 不直接承担前端主 API；
- 不直接修改主状态字段；
- 不直接承接文档流转、交接确认、归档等正式动作。

### 4.3 `frontend/`

职责保持不变：

- 只对接 Go 主 API；
- AI 结果通过 Go 聚合接口读取，不直接对接 Python Worker。

---

## 5. 模块归属划分

### 5.1 迁移到 Go 的模块

- 登录、当前用户、退出登录
- 团队空间、项目、目录树
- 文档创建、列表、详情、更新、删除、恢复
- 文档版本上传与版本记录
- 文档流转状态机
- 毕业交接生成、确认、完成、取消
- 审计事件统一写入与查询
- 负责人总览接口
- 文件上传下载
- 对象存储 / 群晖 NAS 适配

### 5.2 保留在 Python 的模块

- `assistant.ask`
- 文档摘要生成
- 交接摘要生成
- 标签建议、风险提示、结构建议
- 文档文本提取
- 后续 OCR、向量化、批量处理

### 5.3 共享契约

以下内容迁移过程中必须稳定：

- PostgreSQL 表结构语义
- 枚举值
- API 响应结构
- 异步任务消息结构
- suggestion 类结果与主业务事实的分离规则

---

## 6. 推荐技术选型

### 6.1 Go

- Web 框架：`Gin` 或 `Echo`
- 数据访问：优先 `sqlc + pgx`
- 迁移工具：`golang-migrate`
- 认证：JWT
- 队列：Redis + `Asynq` 或自定义轻量任务协议
- 配置：环境变量 + 配置结构体

推荐理由：

- 本项目数据库结构和事务边界明确，`sqlc` 比 ORM 更适合控制 SQL、索引和状态机更新。

### 6.2 Python

- 保留现有 `Python 3.12`
- 保留 Celery 或逐步过渡到与 Go 统一的 Redis 队列
- 保留轻量客户端封装与任务执行器

---

## 7. 迁移清单

### 阶段 0：迁移前冻结

目标：

- 冻结当前数据库设计和 API 契约，避免边迁移边变形。

清单：

- [ ] 确认三份设计文档为当前唯一依据
- [ ] 确认当前枚举、角色、状态不再随意新增
- [ ] 确认前端优先以 `API设计.md` 为对接契约
- [ ] 明确 Python 现有接口仅作为过渡实现

### 阶段 1：建立 Go 主后端骨架

目标：

- 先让 Go 服务具备可运行、可连库、可返回基础接口的能力。

清单：

- [ ] 新建 `backend-go/`
- [ ] 初始化 `go.mod`
- [ ] 搭建 HTTP 路由、配置加载、健康检查
- [ ] 建立数据库连接与事务封装
- [ ] 引入迁移工具并接管后续 schema migration
- [ ] 建立统一错误码和响应结构
- [ ] 建立 JWT 认证中间件

### 阶段 2：迁移基础主业务模块

目标：

- 先迁最稳定、依赖最少的主业务能力。

清单：

- [ ] 迁移 `auth`
- [ ] 迁移 `team_spaces`
- [ ] 迁移 `projects`
- [ ] 迁移 `folders/tree`
- [ ] 迁移文档列表与详情查询
- [ ] 增加集成测试覆盖基础查询链路

### 阶段 3：迁移文档与版本核心链路

目标：

- 完成文档上传、版本记录、对象存储写入的主路径。

清单：

- [ ] 实现创建文档并上传首版本
- [ ] 实现文档更新、删除、恢复
- [ ] 实现版本上传
- [ ] 落地 MinIO 存储适配器
- [ ] 为群晖 NAS 预留统一存储接口
- [ ] 写入版本审计日志

### 阶段 4：迁移流转、交接、审计

目标：

- 将业务状态机和正式动作全部收口到 Go。

清单：

- [ ] 实现流转状态机服务
- [ ] 实现流转记录写入
- [ ] 实现毕业交接生成
- [ ] 实现毕业交接确认和完成
- [ ] 实现审计事件统一写入
- [ ] 实现负责人总览聚合接口

### 阶段 5：拆出 Python Worker

目标：

- 将 Python 从“主 API 骨架”转为“AI / 文档处理 Worker”。

清单：

- [ ] 新建 `backend-py-worker/`
- [ ] 迁移 OpenClaw 客户端
- [ ] 迁移摘要任务
- [ ] 迁移交接摘要任务
- [ ] 迁移建议生成任务
- [ ] 迁移文档文本提取任务
- [ ] 约束 Worker 仅写附属结果

### 阶段 6：打通异步协作链路

目标：

- 让 Go 与 Python 通过稳定协议协同。

清单：

- [ ] 定义任务消息结构
- [ ] 定义任务类型枚举
- [ ] Go 完成任务投递
- [ ] Python 完成任务消费与重试
- [ ] 定义任务结果回写方式
- [ ] 增加失败补偿和幂等机制

### 阶段 7：切换前端与退役旧 Python API

目标：

- 前端正式切到 Go 主 API，旧 Python API 下线。

清单：

- [ ] 前端环境变量切到 Go API 地址
- [ ] 验证登录、文档、流转、交接、总览闭环
- [ ] 验证 AI 建议展示与正式动作分离
- [ ] 标记旧 `backend/app/api/routes/` 为废弃
- [ ] 删除或归档旧 Python Web API 占位实现

---

## 8. 模块迁移映射表

| 当前模块 | 目标归属 | 说明 |
| --- | --- | --- |
| `backend/app/api/routes/auth.py` | `backend-go` | 主认证接口迁移到 Go |
| `backend/app/api/routes/team_spaces.py` | `backend-go` | 主业务查询接口 |
| `backend/app/api/routes/projects.py` | `backend-go` | 主业务查询接口 |
| `backend/app/api/routes/documents.py` | `backend-go` | 核心文档主链路 |
| `backend/app/api/routes/versions.py` | `backend-go` | 版本记录与上传 |
| `backend/app/api/routes/flows.py` | `backend-go` | 状态机与流转 |
| `backend/app/api/routes/handovers.py` | `backend-go` | 毕业交接主流程 |
| `backend/app/api/routes/dashboard.py` | `backend-go` | 负责人总览聚合 |
| `backend/app/api/routes/audit_events.py` | `backend-go` | 审计查询与写入 |
| `backend/app/api/routes/assistant.py` | `backend-py-worker` + `backend-go` | Python 执行 AI，Go 提供聚合展示接口 |
| `backend/app/services/assistant_client.py` | `backend-py-worker` | OpenClaw 客户端保留在 Python |
| `backend/app/models/*` | `backend-go` 重新实现 | 表结构语义保留，代码重建 |
| `backend/alembic/*` | 短期保留，长期迁至 `backend-go/migrations/` | 迁移工具逐步切换 |

---

## 9. 关键风险与控制

### 风险 1：双后端职责混乱

控制：

- 只允许 Go 对外提供主 API；
- Python 仅作为内部 Worker；
- 主状态字段只能由 Go 写入。

### 风险 2：迁移期间 API 契约漂移

控制：

- 所有接口以 `API设计.md` 为准；
- Go 实现前先补齐接口清单；
- 前端不直接绑定旧 Python 占位返回结构。

### 风险 3：迁移体系断档

控制：

- Alembic 到 Go migration 的切换要有明确交接点；
- 切换前最后一次 Python migration 需要打标；
- 后续 schema 变更统一走新体系。

### 风险 4：AI 结果越权写主账本

控制：

- AI 结果单独入 suggestion / summary 类结构；
- 所有正式动作必须人工确认后由 Go 执行。

---

## 10. 建议的近期执行顺序

1. 建立 `backend-go/` 骨架与健康检查
2. 完成 Go 数据访问和认证中间件
3. 迁移组织结构与文档查询接口
4. 迁移文档上传、版本与存储链路
5. 迁移流转、交接、审计
6. 拆出 `backend-py-worker/`
7. 打通异步任务与 AI 回写
8. 前端切换到 Go 主 API
9. 退役旧 Python Web API

---

## 11. 完成定义

本迁移方案完成时，应满足：

- Go 已承接全部主业务 API；
- Python 仅承担 AI / 文档处理 Worker；
- 前端只对接 Go；
- 存储适配和审计写入已收口到 Go；
- AI 建议与正式业务状态仍严格分离；
- `README.md` 与 `TASKS.md` 已同步更新；
- 新增测试能覆盖至少核心主链路。
