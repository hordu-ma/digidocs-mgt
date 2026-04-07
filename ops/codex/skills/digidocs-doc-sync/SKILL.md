---
name: digidocs-doc-sync
description: 确保项目的双账本与协作文档持续同步，避免实现前进、文档停留在旧阶段。
---

# Skill: digidocs-doc-sync

## 目的

确保项目的双账本与协作文档持续同步，避免实现前进、文档停留在旧阶段。

## 触发条件

- 任何功能开发、阶段性交付、环境改造
- 任何会影响“当前阶段”“已完成”“进行中”“待办”的工作

## 主账本规则

- `TASKS.md` 是执行账本。
- `README.md` 是对外可读的阶段快照。
- 两者内容允许粒度不同，不允许阶段判断冲突。

## 同步要求

- 如果 `README.md` 的开发进度更新了，必须检查 `TASKS.md` 当前阶段。
- 如果 `TASKS.md` 中 `已完成/进行中/待办` 变化了，必须判断 `README.md` 是否需要同步。
- `AGENTS.md` 的流程、DoD 和环境约束变化后，必要时也同步到 `README.md` 协作规范。

## 最小验证

```bash
./scripts/codex/check-doc-sync.sh
```

## 完成检查

- `README.md` 与 `TASKS.md` 阶段一致
- 新增环境能力已写入账本
