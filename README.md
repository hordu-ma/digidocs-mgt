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
├── AGENTS.md
├── API设计.md
├── README.md
├── SKILLS/
├── TASKS.md
├── backend/
├── backend-go/
├── backend-py-worker/
├── docker-compose.yml
├── frontend/
├── 数据库设计.md
└── 项目定义与技术架构.md
```

## 核心文档

- [项目定义与技术架构.md](/home/liguoma/code-repos/digidocs-mgt/项目定义与技术架构.md)
- [数据库设计.md](/home/liguoma/code-repos/digidocs-mgt/数据库设计.md)
- [API设计.md](/home/liguoma/code-repos/digidocs-mgt/API设计.md)
- [Go-Python混合迁移方案.md](/home/liguoma/code-repos/digidocs-mgt/Go-Python混合迁移方案.md)
- [异步任务消息契约.md](/home/liguoma/code-repos/digidocs-mgt/异步任务消息契约.md)

## 技术栈

### 后端

- Go（迁移中，承接主业务后端）
- Python 3.12
- Python Worker（迁移中，承接 AI 与文档处理任务）
- FastAPI
- SQLAlchemy 2.x
- Alembic
- Celery
- Redis
- PostgreSQL
- uv

### 前端

- Vue 3
- TypeScript
- Vite
- Pinia
- Vue Router
- Element Plus

### 外部系统

- Synology DSM / File Station Web API
- DGX Spark 上部署的 OpenClaw 服务
- MinIO 作为首期默认对象存储实现

## 协作规范

- 先维护文档和数据模型，再推进实现。
- 所有新增接口、表结构和跨系统集成，都应先对照三份设计文档。
- AI 建议与主业务事实必须分离，禁止让 AI 直接改写主状态。
- 优先走薄业务层自研，不重复造通用底座。
- 涉及群晖 NAS 和 OpenClaw 的调用，必须统一经由适配层/客户端模块，不允许散落在业务代码中。

## 近期开发顺序

1. 完成仓库协作资产与运行骨架
2. 安装后端和前端依赖
3. 跑通数据库迁移
4. 将占位 API 替换为真实数据库读写
5. 打通文档上传、版本、流转、交接、总览的最小闭环
6. 接入 OpenClaw 摘要链路
7. 接入群晖 NAS 存储适配器

## 开发命令约定

### 后端

```bash
cd backend
uv sync
uv run alembic upgrade head
uv run uvicorn app.main:app --reload
```

### Go 主后端骨架

```bash
cd backend-go
go run ./cmd/api
```

### Python Worker 骨架

```bash
cd backend-py-worker
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
docker compose up -d postgres redis minio
```

启动容器内联调路径：

```bash
docker compose --profile app up -d backend-go backend-py-worker
```

## 跨机器接力开发

在另一台机器拉取仓库后，需要重新安装本地依赖和运行环境，不能直接复用其他机器的 `.venv`、`node_modules` 或本地容器状态。

建议执行：

```bash
git pull
cp .env.example backend/.env
cd backend
uv sync
cd ../frontend
npm install
cd ..
docker compose up -d postgres redis minio
```

## 开发进度

当前开发进度只在 [TASKS.md](/home/liguoma/code-repos/digidocs-mgt/TASKS.md) 维护，README 不再重复记录。
