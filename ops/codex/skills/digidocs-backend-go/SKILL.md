# Skill: digidocs-backend-go

## 目的

约束 `backend-go/` 中主业务后端的实现方式，确保 Go 迁移继续遵循契约优先与分层设计。

## 触发条件

- 需求涉及 `backend-go/`
- 需求涉及 REST API、repository、事务写链、dashboard 聚合、flow / handover / version

## 必读信息源

1. `项目定义与技术架构.md`
2. `数据库设计.md`
3. `API设计.md`
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
