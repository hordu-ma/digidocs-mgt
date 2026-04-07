# backend-go 代码说明

本文档面向阅读源码的人，解释 `backend-go/` 中各文件承担的职责，以及这些文件组合后如何支撑 DigiDocs Mgt 的核心业务。

说明范围：

- 重点覆盖仓库自有 Go 代码、迁移、种子数据和后端 README。
- `vendor/` 下是第三方依赖源码，不属于本项目业务实现，本文只按依赖分组说明，不逐文件展开。

## 1. backend-go 最终解决什么问题

`backend-go` 是平台的主业务后端，负责：

- 提供前端访问的 REST API；
- 维护用户认证、文档、版本、流转、交接、审计、仪表盘等主业务账本；
- 通过统一存储适配层把文件写入内存存储或群晖 File Station；
- 通过内部接口为 Python Worker 提供受控上下文和文件下载能力；
- 负责 AI 请求的排队、结果落库和建议确认/忽略的主业务闭环。

它不直接做大模型推理，AI 调用由 `backend-py-worker` 承担；但所有 AI 结果的业务归档、展示和状态管理，仍然由 `backend-go` 控制。

## 2. 整体协作链路

典型请求链路如下：

1. 前端调用 HTTP API，进入 `transport/http/router`。
2. 路由把请求分发给对应 handler。
3. handler 解析参数、读取上下文用户信息，调用 service。
4. service 编排业务规则，必要时访问 repository、storage、queue。
5. repository 负责读写 PostgreSQL 或 memory 仓储。
6. storage 负责文件上传、下载、目录管理、分享链接等。
7. assistant 相关请求写入 `assistant_requests`，由 Worker 异步消费。
8. Worker 处理完成后回调 `backend-go`，结果落入 `assistant_requests` / `assistant_suggestions` / 文档摘要字段。

因此 `backend-go` 是整个项目的业务中枢和主账本。

## 3. 启动与基础设施文件

| 文件 | 作用 |
| --- | --- |
| `backend-go/cmd/api/main.go` | Go API 进程入口，负责加载配置、构建容器、启动 HTTP 服务。 |
| `backend-go/internal/app/server.go` | 定义 HTTP Server 启动和关闭逻辑，是应用运行壳层。 |
| `backend-go/internal/bootstrap/container.go` | 依赖装配中心，按配置选择 `postgres` / `memory` 仓储、`synology` / `memory` 存储，并构造所有 service。 |
| `backend-go/internal/config/config.go` | 从环境变量加载运行配置，包括 API 前缀、数据库地址、JWT、Worker token、群晖配置等。 |
| `backend-go/internal/db/postgres.go` | 打开 PostgreSQL 连接池并做基础连接管理。 |
| `backend-go/internal/db/migrate.go` | 启动时自动执行 SQL 迁移文件。 |
| `backend-go/internal/db/postgres_connectivity_test.go` | 校验 PostgreSQL 连通性的测试。 |
| `backend-go/Dockerfile` | 构建 Go API 镜像。 |
| `backend-go/go.mod` | Go 模块定义与依赖声明。 |
| `backend-go/go.sum` | Go 依赖校验和。 |
| `backend-go/README.md` | backend-go 的局部运行说明与范围说明。 |

## 4. 领域类型文件

这些文件不承载业务流程，而是给 service / repository / handler 提供统一的数据结构。

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/domain/auth/types.go` | 认证相关结构定义，例如登录结果、当前用户等。 |
| `backend-go/internal/domain/command/types.go` | 写操作输入模型，承载创建文档、更新文档、流转动作、交接动作等命令数据。 |
| `backend-go/internal/domain/query/types.go` | 查询返回模型和过滤条件，覆盖文档详情、版本、审计、建议列表、仪表盘等查询结果。 |
| `backend-go/internal/domain/task/types.go` | Go 与 Worker 之间的任务消息、回调结果等异步任务结构。 |

## 5. 队列层文件

这一层解决“主业务系统如何把 AI/异步任务交给 Worker”。

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/queue/contracts.go` | 队列发布与消费接口契约。 |
| `backend-go/internal/queue/memory/publisher.go` | 开发模式下的内存队列实现，适合本地和测试。 |
| `backend-go/internal/queue/noop/publisher.go` | 空发布器，主要用于 postgres 模式下避免重复发布。 |
| `backend-go/internal/queue/postgres/consumer.go` | 通过 `assistant_requests` 表轮询待处理任务的消费者，实现持久化任务队列。 |

## 6. Repository 契约与实现

### 6.1 契约

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/repository/contracts.go` | 定义所有仓储接口，是 service 与底层存储之间的边界。 |

### 6.2 memory 实现

这些文件用于开发测试或无数据库模式。

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/repository/memory/action_repository.go` | 内存版流转/交接动作写入与状态变更。 |
| `backend-go/internal/repository/memory/assistant_repository.go` | 内存版 AI 请求与建议记录。 |
| `backend-go/internal/repository/memory/audit_repository.go` | 内存版审计事件查询。 |
| `backend-go/internal/repository/memory/dashboard_repository.go` | 内存版仪表盘聚合数据。 |
| `backend-go/internal/repository/memory/document_repository.go` | 内存版文档主记录 CRUD。 |
| `backend-go/internal/repository/memory/flow_repository.go` | 内存版文档流转查询。 |
| `backend-go/internal/repository/memory/handover_repository.go` | 内存版交接单和交接项查询/写入。 |
| `backend-go/internal/repository/memory/project_repository.go` | 内存版项目查询。 |
| `backend-go/internal/repository/memory/team_space_repository.go` | 内存版团队空间查询。 |
| `backend-go/internal/repository/memory/user_auth_repository.go` | 内存版用户认证仓储，提供开发账号。 |
| `backend-go/internal/repository/memory/version_command_repository.go` | 内存版版本写命令仓储。 |
| `backend-go/internal/repository/memory/version_repository.go` | 内存版版本查询仓储。 |
| `backend-go/internal/repository/memory/version_workflow.go` | 内存版版本事务工作流。 |

### 6.3 postgres 实现

这些文件是真实数据库模式下的持久化实现。

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/repository/postgres/dbtx.go` | 抽象 `*sql.DB` / `*sql.Tx` 共同接口，便于事务内外复用查询逻辑。 |
| `backend-go/internal/repository/postgres/action_repository.go` | 文档流转和交接动作的真实落库实现。 |
| `backend-go/internal/repository/postgres/assistant_repository.go` | AI 请求、结果、建议、确认/忽略等持久化逻辑。 |
| `backend-go/internal/repository/postgres/audit_repository.go` | 审计事件列表与 summary 聚合查询。 |
| `backend-go/internal/repository/postgres/dashboard_repository.go` | 仪表盘 overview / recent flows / risk documents 聚合查询。 |
| `backend-go/internal/repository/postgres/document_repository.go` | 文档主记录查询、创建、更新、软删除、恢复。 |
| `backend-go/internal/repository/postgres/flow_repository.go` | 文档流转历史列表查询。 |
| `backend-go/internal/repository/postgres/handover_repository.go` | 交接单与交接项持久化和查询。 |
| `backend-go/internal/repository/postgres/project_repository.go` | 项目与目录树相关查询。 |
| `backend-go/internal/repository/postgres/team_space_repository.go` | 团队空间列表查询。 |
| `backend-go/internal/repository/postgres/user_auth_repository.go` | 用户认证、登录校验。 |
| `backend-go/internal/repository/postgres/version_repository.go` | 文档版本查询与版本详情读取。 |
| `backend-go/internal/repository/postgres/version_workflow.go` | 版本上传事务：写版本、更新文档当前版本、写审计。 |

## 7. Service 层文件

service 层是业务规则中心，也是理解项目需求最重要的一层。

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/service/assistant_service.go` | 创建 AI 任务、查询 AI 请求状态、查询建议、确认/忽略建议。 |
| `backend-go/internal/service/audit_query_service.go` | 面向审计查询的只读服务。 |
| `backend-go/internal/service/audit_service.go` | 审计事件写入辅助服务。 |
| `backend-go/internal/service/auth_service.go` | 登录认证逻辑，校验用户名密码并生成 token。 |
| `backend-go/internal/service/auth_service_test.go` | 认证服务测试。 |
| `backend-go/internal/service/dashboard_query_service.go` | 仪表盘聚合数据查询服务。 |
| `backend-go/internal/service/dashboard_query_service_test.go` | 仪表盘服务测试。 |
| `backend-go/internal/service/document_service.go` | 文档创建、更新、删除、恢复，以及首版本上传等写入逻辑。 |
| `backend-go/internal/service/document_service_test.go` | 文档服务测试。 |
| `backend-go/internal/service/errors.go` | 统一业务错误定义，如 `ErrNotFound`、`ErrForbidden` 等。 |
| `backend-go/internal/service/flow_service.go` | 文档流转动作（开始处理、转交、接受、定稿、归档、取消归档）和流转历史查询。 |
| `backend-go/internal/service/flow_service_test.go` | 流转服务测试。 |
| `backend-go/internal/service/handover_service.go` | 交接单创建、更新交接项、确认、完成、取消和查询。 |
| `backend-go/internal/service/handover_service_test.go` | 交接服务测试。 |
| `backend-go/internal/service/query_service.go` | 团队空间、项目、目录树等基础查询服务。 |
| `backend-go/internal/service/request_id.go` | 生成内部请求 ID，用于异步任务等场景。 |
| `backend-go/internal/service/task_service.go` | 任务相关辅助逻辑，围绕异步消息结构做封装。 |
| `backend-go/internal/service/token_service.go` | JWT 签发和解析。 |
| `backend-go/internal/service/version_service.go` | 版本上传、版本列表、版本详情、文件下载/预览。 |
| `backend-go/internal/service/version_service_test.go` | 版本服务测试。 |

## 8. Shared 与 Storage 文件

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/shared/ctxkeys.go` | 请求上下文 key 定义，避免字符串 key 冲突。 |
| `backend-go/internal/shared/upload.go` | 文件上传通用结构和上传大小等共享逻辑。 |
| `backend-go/internal/storage/contracts.go` | 存储抽象接口，定义上传、下载、删除、建目录、分享链接等能力。 |
| `backend-go/internal/storage/memory/provider.go` | 内存存储实现，适合本地开发和测试。 |
| `backend-go/internal/storage/synology/provider.go` | 群晖 File Station API 适配器，实现真实文件存储。 |

## 9. HTTP 入口层文件

### 9.1 handlers

handler 负责把 HTTP 请求映射成 service 调用。

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/transport/http/handlers/assistant.go` | AI 问答、摘要、建议列表、确认、忽略等接口。 |
| `backend-go/internal/transport/http/handlers/audit_events.go` | 审计事件列表和统计接口。 |
| `backend-go/internal/transport/http/handlers/auth.go` | 登录、当前用户、退出登录接口。 |
| `backend-go/internal/transport/http/handlers/dashboard.go` | 仪表盘 overview / recent flows / risk documents 接口。 |
| `backend-go/internal/transport/http/handlers/documents.go` | 文档创建、详情、列表、更新、删除、恢复接口。 |
| `backend-go/internal/transport/http/handlers/flows.go` | 文档流转动作与流转列表接口。 |
| `backend-go/internal/transport/http/handlers/handlers_test.go` | handler 层集成测试。 |
| `backend-go/internal/transport/http/handlers/handovers.go` | 交接单 CRUD 与动作接口。 |
| `backend-go/internal/transport/http/handlers/internal_assistant_context.go` | Worker 专用内部上下文接口和版本文件下载接口。 |
| `backend-go/internal/transport/http/handlers/internal_auth.go` | 内部鉴权辅助逻辑。 |
| `backend-go/internal/transport/http/handlers/internal_worker.go` | Worker 轮询任务和回写结果接口。 |
| `backend-go/internal/transport/http/handlers/projects.go` | 项目列表与目录树接口。 |
| `backend-go/internal/transport/http/handlers/system.go` | 健康检查和系统信息接口。 |
| `backend-go/internal/transport/http/handlers/team_spaces.go` | 团队空间列表接口。 |
| `backend-go/internal/transport/http/handlers/version_query.go` | 版本查询辅助 handler。 |
| `backend-go/internal/transport/http/handlers/versions.go` | 版本上传、列表、详情、下载、预览接口。 |

### 9.2 middleware

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/transport/http/middleware/access_log.go` | HTTP 访问日志。 |
| `backend-go/internal/transport/http/middleware/auth.go` | JWT 鉴权并把用户信息注入上下文。 |
| `backend-go/internal/transport/http/middleware/chain.go` | 中间件链组合器。 |
| `backend-go/internal/transport/http/middleware/cors.go` | CORS 处理。 |
| `backend-go/internal/transport/http/middleware/json_content_type.go` | 默认响应头 Content-Type 为 JSON。 |
| `backend-go/internal/transport/http/middleware/request_id.go` | 为请求生成 request ID。 |

### 9.3 request / response / router

| 文件 | 作用 |
| --- | --- |
| `backend-go/internal/transport/http/request/auth.go` | 认证请求体解析结构。 |
| `backend-go/internal/transport/http/response/response.go` | 统一成功/失败 JSON 响应格式。 |
| `backend-go/internal/transport/http/router/router.go` | 注册所有公开接口、受保护接口、Worker 内部接口，并挂中间件。 |

## 10. 迁移、种子数据和其他文件

| 文件 | 作用 |
| --- | --- |
| `backend-go/migrations/001_initial_schema.sql` | 初始化核心表、枚举、索引。 |
| `backend-go/migrations/002_assistant_request_output.sql` | 为 AI 请求结果补充 `output` 等字段。 |
| `backend-go/sql/seed.sql` | 本地联调和演示用种子数据。 |

## 11. 第三方 vendored 代码说明

| 路径 | 作用 |
| --- | --- |
| `backend-go/vendor/github.com/lib/pq/**` | PostgreSQL 驱动源码。 |
| `backend-go/vendor/golang.org/x/crypto/**` | bcrypt 等密码学依赖。 |
| `backend-go/vendor/modules.txt` | vendored 依赖索引。 |

这些文件是依赖副本，不是 DigiDocs 的业务实现。阅读项目逻辑时通常不需要逐个进入。

## 12. 组合后支撑的项目功能

把以上文件组合起来，`backend-go` 最终支撑了以下用户功能：

- 用户登录、鉴权和当前用户识别；
- 团队空间、项目、目录树的浏览；
- 文档创建、编辑、软删除、恢复；
- 文档版本上传、下载、预览和当前版本更新；
- 文档流转、接收转交、定稿、归档；
- 交接单创建、交接项维护、确认与完成；
- 审计事件查询和仪表盘聚合展示；
- AI 问答、文档摘要、交接摘要、建议确认/忽略；
- 向 Python Worker 提供受控上下文和文件下载能力；
- 对接群晖存储或本地内存存储。

如果把整个项目看成一套“文档主业务平台 + AI 助手”的系统，`backend-go` 负责其中“主业务平台”和“AI 结果主账本”。
