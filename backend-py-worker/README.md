# Backend Python Worker

Python AI 与文档处理 Worker。

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
- OpenClaw Gateway OpenAI 兼容客户端
- Go 后端内部上下文读取客户端
- Worker 调度器与真实 HTTP 轮询消费
- 与 `异步任务消息契约.md` 对齐
- 结果回写目标约定为 `/api/v1/internal/worker-results`
- OpenClaw 调用统一走 `POST /v1/chat/completions`
- 已补调度器与 OpenClaw 客户端测试样例

## Current Notes

- 当前 OpenClaw 对接使用官方 Gateway 的 OpenAI 兼容 HTTP 接口。
- 当前文档摘要和交接摘要优先消费“结构化业务上下文”；如果缺少正文内容，结果会明确标注为元数据级摘要。
- `document.extract_text` 已支持 `txt / md / csv / json / docx / pdf` 文本抽取。
- 若 Worker 主机存在 `tesseract`，还可进一步支持图片 OCR 与扫描 PDF OCR；若不存在，会返回明确的运行时错误提示。
