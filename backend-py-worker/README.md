# Python Worker

从 Go API 领取任务、读取获准上下文、抽取正文、调用 OpenAI 兼容模型端点并回写附属结果。任务类型、回调凭据、领取次数和业务边界见[异步任务契约](../docs/异步任务消息契约.md)。

本目录执行 `uv sync --group dev` 安装开发依赖，执行 `uv run pytest -q` 验证。连接开发 Go API 与模型网关后可运行 `uv run python -m app.main`。生产通过 Compose 启动，配置见[部署参考](../docs/部署准备与运行说明.md)。

容器包含 Office 文件正文解析所需的代码和 PDF、OCR、旧版 Word 的外部工具。支持 TXT、Markdown、CSV、JSON、DOCX、XLSX、PPTX、DOC、PDF 和常见图片 OCR。扫描 PDF 默认最多处理 30 页，上限由 `MAX_OCR_PAGES` 控制。提取失败时，请求错误或摘要的元数据级标记须如实展示。
