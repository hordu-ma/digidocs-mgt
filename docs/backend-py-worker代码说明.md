# backend-py-worker 代码说明

本文档解释 `backend-py-worker/` 目录中每个文件的职责，以及这些文件如何组合成 Python Worker，支撑项目里的 AI 和文档处理能力。

## 1. backend-py-worker 最终解决什么问题

`backend-py-worker` 是平台的异步 AI 执行器，负责：

- 从 Go 后端轮询待处理任务；
- 按任务类型拉取项目、文档、交接单的业务上下文；
- 需要时下载文档原始文件并做正文抽取；
- 调用 OpenClaw Gateway 获取问答、摘要和结构化建议；
- 把处理结果回写给 Go 主后端。

它不直接对前端提供 API，也不直接写主业务状态；它只生产附属 AI 结果。

## 2. 整体协作链路

完整 Worker 链路如下：

1. `main.py` 启动调度器。
2. `TaskPollerClient` 从 Go 后端轮询待处理任务。
3. `dispatcher.py` 根据任务类型决定处理流程。
4. `BackendContextClient` 获取文档、项目、交接单上下文，必要时下载原始版本文件。
5. `document_text_extractor.py` 把 `txt/md/csv/json/docx` 等文件转成纯文本。
6. `OpenClawClient` 把任务提示词、业务上下文和正文发给 OpenClaw Gateway。
7. `CallbackClient` 把结果回写到 Go 后端的内部回调接口。

因此 `backend-py-worker` 是“异步 AI 执行层”。

## 3. 根目录与配置文件

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/Dockerfile` | 构建 Worker 镜像。 |
| `backend-py-worker/pyproject.toml` | Python 项目元数据与依赖声明。 |
| `backend-py-worker/pyrightconfig.json` | Pylance / Pyright 类型检查配置。 |
| `backend-py-worker/README.md` | Worker 局部说明、运行方式与范围说明。 |
| `backend-py-worker/uv.lock` | `uv` 解析后的依赖锁文件。 |

## 4. app 目录文件说明

### 4.1 入口与包定义

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/app/__init__.py` | 包初始化占位文件。 |
| `backend-py-worker/app/main.py` | Worker 入口。根据 `worker_mode` 决定是只打印启动信息还是进入无限轮询循环。 |

### 4.2 clients 子目录

这些文件负责与外部系统通信。

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/app/clients/__init__.py` | clients 包初始化占位文件。 |
| `backend-py-worker/app/clients/backend_context_client.py` | 调用 Go 后端内部接口，获取项目/文档/交接单上下文，并下载文档版本原文件。 |
| `backend-py-worker/app/clients/callback_client.py` | 将任务处理结果回写到 Go 后端 `/internal/worker-results`。 |
| `backend-py-worker/app/clients/openclaw_client.py` | OpenClaw Gateway 客户端，封装问答、文档摘要、交接摘要、建议生成逻辑。 |
| `backend-py-worker/app/clients/task_poller.py` | 轮询 Go 后端待处理任务，把原始 JSON 解析成 `WorkerTask`。 |

### 4.3 core 子目录

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/app/core/__init__.py` | core 包初始化占位文件。 |
| `backend-py-worker/app/core/config.py` | 通过环境变量加载 Worker 的运行配置，如 OpenClaw 地址、回调地址、token、轮询间隔等。 |

### 4.4 services 子目录

service 层是 Worker 的内部编排逻辑。

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/app/services/__init__.py` | services 包初始化占位文件。 |
| `backend-py-worker/app/services/dispatcher.py` | Worker 核心调度器。按任务类型组织“取上下文 → 调 OpenClaw / 抽正文 → 回写结果”的完整流程。 |
| `backend-py-worker/app/services/document_text_extractor.py` | 将原始文档内容抽取成纯文本，供摘要任务使用。 |

### 4.5 tasks 子目录

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/app/tasks/__init__.py` | tasks 包初始化占位文件。 |
| `backend-py-worker/app/tasks/contracts.py` | 定义 `TaskType`、`WorkerTask`、`TaskResult` 等 Worker 内部消息契约。 |

## 5. tests 目录文件说明

| 文件 | 作用 |
| --- | --- |
| `backend-py-worker/tests/__init__.py` | 测试包初始化占位文件。 |
| `backend-py-worker/tests/test_dispatcher.py` | 调度器测试，覆盖问答、摘要、建议、正文抽取等任务分发。 |
| `backend-py-worker/tests/test_openclaw_client.py` | OpenClaw 客户端测试，覆盖正常响应、JSON 代码块解析、HTTP 错误处理。 |

## 6. 各模块组合后支撑的功能

把以上文件组合起来，`backend-py-worker` 最终支撑了以下项目能力：

- AI 问答：根据项目或文档上下文回答问题；
- 文档摘要：基于业务上下文和抽取到的正文生成摘要与建议；
- 交接摘要：根据交接单信息生成概览和风险提示；
- 建议生成：围绕项目/文档/交接对象生成结构化建议；
- 文本抽取：把文档原始文件转成可供模型使用的纯文本；
- 异步执行：把耗时 AI 处理从主业务 API 中解耦。

## 7. 与 backend-go 的边界关系

为了理解项目整体，可以把两个后端这样区分：

- `backend-go` 负责主业务状态、权限、数据库、文件存储适配、AI 结果落库；
- `backend-py-worker` 负责执行 AI 任务和正文处理，不拥有主业务写入权；
- 两者通过内部 HTTP 接口和异步任务契约连接。

也就是说，Worker 是“AI 执行层”，而不是“业务主后端”。
