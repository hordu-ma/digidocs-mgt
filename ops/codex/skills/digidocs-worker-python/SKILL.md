---
name: digidocs-worker-python
description: 约束 backend-py-worker/ 与 Python 辅助能力层的职责边界，确保 AI 与文档处理能力不侵入主业务账本。
---

# Skill: digidocs-worker-python

## 目的

约束 `backend-py-worker/` 与 Python 辅助能力层的职责边界，确保 AI 与文档处理能力不侵入主业务账本。

## 触发条件

- 需求涉及 `backend-py-worker/`
- 需求涉及 OpenClaw、异步任务、摘要、建议、回写结果

## 执行顺序

1. 先读 `docs/项目定义与技术架构.md`、`docs/数据库设计.md`、`docs/API设计.md`
2. 再读 `README.md`、`TASKS.md`，确认当前 AI 链路阶段和已知限制
3. 明确本次改动属于哪类能力：任务消费、上下文装配、OpenClaw 调用、正文抽取、结果回写
4. 先确认输入/输出契约，再修改 Worker 逻辑
5. 改动后至少跑 `uv run pytest -q`，如涉及联调再补 `make smoke`

## 必读信息源

1. `docs/项目定义与技术架构.md`
2. `docs/数据库设计.md`
3. `docs/API设计.md`
4. `AGENTS.md`
5. `TASKS.md`

## 强约束

- Worker 只承接 AI 与文档处理，不承接主业务状态机主写入。
- AI 输出必须落在附属结果表或附属结果结构，不得伪装成正式动作。
- 请求 OpenClaw 必须带范围控制。
- 长耗时任务放入异步任务系统，不走同步主链路。
- Worker 只消费业务侧显式装配的 `scope`、`context`、`memory`，不得自行扩大访问范围。
- 禁止依赖宿主机 persona、本地知识文件、长期隐式记忆或其他会污染业务结果的外部状态。

## 常见落点

- OpenClaw 客户端与回调：`app/clients/`
- 任务调度与消费：`app/services/`、`app/worker/`
- 文本抽取/OCR：`app/extractors/` 或同类模块
- 输入输出 schema：优先复用现有请求/结果结构，不额外发明主账本字段

## 输出要求

- 在总结中说明本次改动属于哪条 AI 链路，以及结果最终落到哪个附属结构
- 若新增能力影响阶段判断或待办项，同时更新 `README.md` 与 `TASKS.md`

## 最小验证

```bash
cd backend-py-worker
uv run pytest -q
```

如涉及真实网关或任务轮询，补充说明是否执行了 `make smoke` 或为何未执行。

## 完成检查

- AI 与主业务事实仍分离
- 测试已运行或已说明未运行原因
- `README.md` 与 `TASKS.md` 已同步
