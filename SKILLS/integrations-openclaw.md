# Skill: integrations-openclaw

## 适用范围

- DGX Spark 上的 OpenClaw 服务对接
- 文档摘要、问答、建议链路实现

## 规则

- OpenClaw 视为独立服务，不视为本地库。
- 所有请求必须带范围控制，至少限定到项目或文档。
- 失败必须可降级，不能影响文档主流程。
- AI 输出进入 `assistant_requests` 和 `assistant_suggestions`，不直接写业务主状态。
- 长耗时任务必须放进 Celery。

## 最小闭环

1. 上传新版本
2. 异步提交 OpenClaw 摘要任务
3. 保存摘要和标签建议
4. 在文档详情和总览中展示
5. 支持确认或忽略建议

