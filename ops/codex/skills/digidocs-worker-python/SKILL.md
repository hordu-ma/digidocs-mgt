# Skill: digidocs-worker-python

## 目的

约束 `backend-py-worker/` 与 Python 辅助能力层的职责边界，确保 AI 与文档处理能力不侵入主业务账本。

## 触发条件

- 需求涉及 `backend-py-worker/`
- 需求涉及 OpenClaw、异步任务、摘要、建议、回写结果

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

## 最小验证

```bash
cd backend-py-worker
uv run pytest -q
```

## 完成检查

- AI 与主业务事实仍分离
- 测试已运行或已说明未运行原因
- `README.md` 与 `TASKS.md` 已同步
