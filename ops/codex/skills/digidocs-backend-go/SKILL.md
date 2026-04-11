---
name: digidocs-backend-go
description: 约束 backend-go/ 中主业务后端的实现方式，确保 Go 迁移继续遵循契约优先与分层设计。
---

# Skill: digidocs-backend-go

## 目的

约束 `backend-go/` 中主业务后端的实现方式，确保 Go 迁移继续遵循契约优先与分层设计。

## 触发条件

- 需求涉及 `backend-go/`
- 需求涉及 REST API、repository、事务写链、dashboard 聚合、flow / handover / version

## 执行顺序

1. 先读 `docs/项目定义与技术架构.md`、`docs/数据库设计.md`、`docs/API设计.md`
2. 再读 `README.md`、`TASKS.md`，确认当前阶段和已知限制
3. 明确本次改动落在哪个边界：`handler`、`service`、`repository`、`storage`、`migration`
4. 先收紧契约，再改实现；若契约不成立，先更新文档
5. 变更后至少跑 `go test ./...`，必要时再补 `make smoke` 或相关 handler 集成验证

## 必读信息源

1. `docs/项目定义与技术架构.md`
2. `docs/数据库设计.md`
3. `docs/API设计.md`
4. `AGENTS.md`
5. `TASKS.md`

## 所有权范围

- `backend-go/cmd/`
- `backend-go/internal/`
- `backend-go/tests/`
- `backend-go/migrations/`

## 强约束

- 路由层只做请求解析、鉴权入口和响应拼装。
- 查询链与写链都应清晰落在 `handler -> service -> repository`。
- 涉及数据库结构变化时，必须走迁移，不允许手工跳过契约层。
- 涉及状态流转时，必须依据设计文档中的固定状态集合。
- 审计写入必须与正式动作分离，但动作完成后要有对应审计证据。
- 不允许在业务代码中直接拼装群晖 DSM / File Station 请求；统一走 `SynologyStorageProvider` 或等价适配层。
- 不允许让 AI 结果直接改写主业务状态；AI 相关字段只能作为附属结果落库。

## 常见落点

- 新增/修改 API：优先检查 `internal/transport/http/handlers/`、`internal/service/`、`internal/repository/`
- 新增事务写链：优先检查现有 workflow / service 是否已可复用，避免在 handler 内串联多个仓储
- 新增查询：优先补 service 输入校验和 repository 过滤条件，不在 handler 内做业务拼装
- 新增表或字段：先加 `backend-go/migrations/`，再补 repository 和测试

## 输出要求

- 在总结中说明本次改动影响的契约、目录和验证命令
- 若修改会改变阶段判断或任务状态，同时更新 `README.md` 与 `TASKS.md`

## 最小验证

```bash
cd backend-go
go test ./...
```

如改动了事务链路或接口返回，补充对应 smoke test 或调用示例。

## 完成检查

- 契约未漂移
- `go test ./...` 通过或明确说明阻塞原因
- `README.md` 与 `TASKS.md` 已同步
