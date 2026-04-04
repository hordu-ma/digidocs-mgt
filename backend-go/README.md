# Backend Go

Go 主业务后端骨架。

## Run

```bash
cd backend-go
go run ./cmd/api
```

或使用 Docker：

```bash
docker compose --profile app up -d backend-go
```

## Env

复制 `.env.example` 为 `.env`，或直接通过环境变量注入：

```bash
APP_NAME=DigiDocs\ Mgt\ Go\ API
HTTP_ADDR=:8080
APP_ENV=development
```

## Current Scope

- 服务入口
- 配置加载
- 基础路由
- 健康检查
- 统一 JSON 响应结构
- API v1 主模块占位路由
- `handler -> service -> repository` 查询链路
- `memory repository` 运行模式
- `postgres repository` 查询骨架
- `DATA_BACKEND=postgres` 的装配入口
- 统一任务消息类型与内存队列发布骨架
- `assistant` 路由迁移占位
- Worker 结果内部回写入口与简单鉴权
- `auth/login` 请求体解析已接入
- 已接入 PostgreSQL 驱动
- JWT、上传存储、审计服务已有最小落点
- `versions / flows / handovers / dashboard` 路由已迁入 Go 骨架
