# Backend Python Worker

Python AI 与文档处理 Worker 骨架。

## Scope

- OpenClaw 客户端封装
- AI 任务调度
- 摘要、建议、文档预处理等异步任务
- 不直接承接前端主 API
- 不直接修改主业务状态

## Run

```bash
cd backend-py-worker
uv run python -m app.main
```

或使用 Docker：

```bash
docker compose --profile app up -d backend-py-worker
```

## Current Modules

- 配置加载
- 任务消息结构
- OpenClaw / Callback 客户端骨架
- Worker 调度器骨架
- 与 `异步任务消息契约.md` 对齐
- 结果回写目标约定为 `/api/v1/internal/worker-results`
- 已补基础调度器测试样例
