---
name: digidocs-verify
description: 为仓库提供统一的环境诊断与基础验证入口，避免每次依赖人工回忆命令。
---

# Skill: digidocs-verify

## 目的

为仓库提供统一的环境诊断与基础验证入口，避免每次依赖人工回忆命令。

## 触发条件

- 新机器接力开发
- 仓库环境异常
- 大改造后需要快速体检
- 项目级 skills / hooks / 协作脚本有新增或调整

## 推荐用法

- 会话启动：先跑 `./scripts/codex/doctor.sh`
- 需要确认阶段、skills 安装和 hooks 状态：再跑 `./scripts/codex/report.sh`
- 改完文档账本：补跑 `./scripts/codex/check-doc-sync.sh`
- 改完代码：再按子系统补跑 `go test ./...`、`uv run pytest -q`、`npm run build`
- 需要端到端联调：最后跑 `make smoke`

## 验证顺序

1. 运行 `./scripts/codex/doctor.sh`
2. 运行 `./scripts/codex/report.sh` 查看阶段、技能安装和 hooks 状态
3. 如有代码改动，再按相关子系统补跑测试
4. 若需要联调，再运行 `./scripts/codex/smoke-local.sh`
5. 在总结中说明哪些检查通过，哪些未通过

## 检查范围

- 基础工具存在性：`git`、`go`、`uv`、`node`、`npm`、`docker`
- 关键工程文件存在性
- 文档主账本一致性：`./scripts/codex/check-doc-sync.sh`
- 项目 skill 索引与安装脚本存在性
- 项目 skill 是否已实际安装到 `~/.codex/skills/`
- 本地 hooks、CI workflow、状态报告脚本存在性
- 主文档一致性检查

## 交付要求

- 总结里至少说明执行了哪些检查、哪些通过、哪些未执行
- 若发现 doctor/report 与账本状态不一致，先修正文档再推进实现

## 完成检查

- 基础诊断已执行
- 阻塞点已明确记录到总结或任务账本
